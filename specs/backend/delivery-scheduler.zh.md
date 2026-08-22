# 投递调度器与认领路径

[English](delivery-scheduler.md) | 中文

投递子系统是一个三层管线：一张用于持久化的 PostgreSQL 表、一个单 goroutine 的 ticker 负责调度、一个有界的 pond v2 worker 池负责并发。本页规定分发器（`internal/service/delivery/dispatcher.go`）—— 入队、调度器 tick、原子认领、worker 执行与清理。数据模型见 [`delivery-queue.zh.md`](./delivery-queue.zh.md)；重试时序与失败分类见 [`delivery-retry.zh.md`](./delivery-retry.zh.md)；投递语义与崩溃恢复见 [`delivery-recovery.zh.md`](./delivery-recovery.zh.md)。决策理由（选 pond 而非 ants、无消息代理、分批清扫）见[投递 MRFC](../../.agents/mrfcs/implemented/2026-07-10-persistent-best-effort-delivery-queue.zh.md)。

<a id="capacity-envelope-the-saas-reference-instance"></a>

## 容量包络（SaaS 参考实例）

硬件包络是一台 2 核 / 2 GB / 3 Mbps 的 VPS（完整表格见 [`caching.zh.md`](./caching.zh.md)）。在最坏负载模型下 —— 10 000 名用户全部打满 L2 写入上限（每人每天 1000 篇）—— 持续投递上限约为 116 篇/秒（10 000 × 1000 ÷ 86 400）。

两个事实决定了设计：

1. **CPU 不是瓶颈。** 投递是 I/O 密集型（阻塞在飞书 HTTP 响应上的 goroutine 几乎不耗 CPU）。在 116 任务/秒时，整个投递子系统（过滤 + DB 写入 + HTTP 扇出）的花费低于两个核心的 7%。过滤器基准（`internal/service/delivery/filter/filter_bench_test.go`）显示 256 字节标题的编译+匹配约 ~1.6 µs/op；即使每篇文章带 10 个渠道，也只占 CPU 的不到 1%。
2. **单 goroutine 分发器扛不住这个负载。** 一个发起同步 HTTP 调用（5 秒超时）的 goroutine 在飞书 300 ms 响应下顶多 ~3 任务/秒 —— 比上限低两个数量级。有界缓冲会被填满并静默丢弃投递。

<a id="three-layers-three-responsibilities"></a>

## 三层，三份职责

| 层        | 职责                                       | 机制                                                           | 缘由                                                                                        |
| --------- | ------------------------------------------ | -------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| 1. 持久化 | 在进程重启后存活；不丢失待办工作           | `delivery_attempts` 表（PostgreSQL）                           | worker 池库是内存中的并发原语；持久化是数据库的职责，不是池的。                             |
| 2. 调度   | 扫描到期任务；认领且不重复投递；施加过期墙 | Go `time.Ticker`（1 秒）+ 原子认领（`FOR UPDATE SKIP LOCKED`） | 简单的轮询调度器易于推理，且不需要外部消息代理。                                            |
| 3. 并发   | 并行发出至多 N 个飞书 HTTP 调用，有界      | pond v2 worker 池                                              | 有界并发同时控制出站速率（飞书 QPS 限制）与带宽（3 Mbps 上限）；无界 goroutine 两者皆失守。 |

并发层是 `github.com/alitto/pond/v2`（零外部依赖），配置为 `WithQueueSize(1024)` 与 `WithNonBlocking(true)`：队列已满时 `pool.Go` 返回 `ErrQueueFull`，任务带着一条日志被丢弃，尝试行保持已认领状态直至其 `next_at` 预留到期 —— 届时下一个 tick 重新认领并重试。池还提供优雅排空（`StopAndWait`，由 `Dispatcher.Stop` 使用）与默认的 panic 恢复。

<a id="enqueue-in-createpost-synchronous"></a>

## 入队（在 `CreatePost` 中，同步）

```
CreatePost succeeds (post.ID known)
  channels := channelRepo.GetByUserID(userID)
  for each enabled channel:
      if keywordFilter(channel.Keywords).Match(post.Title):
          INSERT delivery_attempts
              (user_id, post_id, channel_id, status=StatusPending,
               attempts=0, next_at=now, created_at=now, updated_at=now)
```

关键词过滤器（[`keyword-filter.zh.md`](./keyword-filter.zh.md)）在**持久化之前**运行，因此只有真正匹配的渠道才产生尝试行；关键词表达式编译失败的渠道带着一条日志被跳过。`Dispatcher.Enqueue` 实现 `post.DeliveryEnqueuer`：它是同步但尽力而为的 —— 任何错误（渠道列表、编译、批量插入）都被记录并吞掉，且它从不返回错误，因此投递失败永远不能使文章创建失败。成功时它把 `delivery_pending` gauge 按插入行数递增。

<a id="the-scheduler-tick"></a>

## 调度器 tick

一个 goroutine 每 `[delivery] scan_interval`（默认 1 秒）tick 一次，每个 tick 依次执行两项职责：先是过期墙清扫，然后是认领。两者都是**分批的**（每条语句有界的批大小），并使用**子查询 `LIMIT` 形式**（`WHERE id IN (SELECT ... LIMIT N)`），绝不是裸 `UPDATE/DELETE ... LIMIT` —— PostgreSQL 不支持在 UPDATE/DELETE 上直接 `LIMIT`。两项职责的批大小均为 64。

**过期墙清扫** —— 把超过墙的 pending 行迁移为 `expired`，随后归档：

```sql
-- MarkExpired; the dispatcher loops until a batch comes back short:
UPDATE delivery_attempts
   SET status = <expired>, updated_at = $now
 WHERE id IN (
     SELECT id FROM delivery_attempts
      WHERE status = <pending> AND created_at < $now - $wall_ms
      ORDER BY created_at
      LIMIT 64
 )
RETURNING *;
-- each returned row → ArchiveAndDelete(status=<expired>) by the scheduler itself
```

分批限定每条语句的锁范围与死元组量：在大量 `pending` 积压下（例如飞书故障累积数万行停滞行），一条无界的 `UPDATE ... RETURNING *` 会锁住整个匹配行区间，并在 1 秒 tick 之下爆发死元组。tick 内的循环仍会迅速排空积压 —— 墙是分钟级的，不需要亚秒级排空。

**认领到期任务** —— 原子地认领并预留：

```sql
UPDATE delivery_attempts SET next_at = $reserve_until
 WHERE id IN (
     SELECT id FROM delivery_attempts
      WHERE status = <pending> AND next_at <= $now
      ORDER BY next_at
      LIMIT 64
      FOR UPDATE SKIP LOCKED
 )
RETURNING *;
-- each returned row → pool.Go(execute)
```

`$reserve_until` 是 `now + [delivery] request_timeout + 500 ms`（`claimReserveBuffer` 常量）。两个性质：

- **`FOR UPDATE SKIP LOCKED`** 是 PostgreSQL 文档记载的队列模式：并发的认领者 —— 或重叠的调度器 tick —— 取得不相交的行集而不是阻塞，因此认领吞吐随 worker 数扩展而不是串行化。
- **预留即重复认领的修复。** 把 `next_at` 推进到请求超时之后，使在途行对下一个 1 秒 tick 不可见。没有它，一行投递耗时超过 1 秒就会被重新认领、重复投递。如果 worker 在投递中途死亡，预留到期后该行重新可被认领 —— 重试自然恢复（见 [`delivery-recovery.zh.md`](./delivery-recovery.zh.md)）。

<a id="worker-execution"></a>

## worker 执行

```
execute(attempt):
  post := posts.GetByID(attempt.post_id)                        -- PK lookup; post guaranteed alive (< 40m < 7d retention)
  channel := channels.GetByIDAndUserID(attempt.channel_id, attempt.user_id)
  err := sender.Send(ctx, post, channel)                        -- PostDeliveryService → Feishu card

  if err == nil:
      archiveAndDelete(attempt, status=delivered, last_error='', error_category='')
  else:
      handleSendError(attempt, err)                             -- classify → retry with backoff, or fail (see delivery-retry.md)
```

取回错误（文章或渠道查询）与发送错误走同一条 `handleSendError` 路径：它们被分类为 `internal`（可重试）并遵循标准退避序列。`PostDeliveryService`（`post_delivery.go`）是 `Sender` 的实现：它从文章 + 渠道构建飞书卡片，并经 `FeishuClient`（`feishu.go`）发起 webhook 调用；后者施加配置的请求超时，并把非 2xx 响应与飞书业务码分类为 `DeliveryError` 类别（见 [`delivery-retry.zh.md`](./delivery-retry.zh.md)）。`SendTest` 发送一张固定的诊断卡片，让用户不发布文章即可验证 webhook —— 它是发射后不管的，从不进入重试队列，也不写历史。

**webhook URL 安全。** `webhook_url` 是用户控制的服务端请求目标。渠道创建/更新（用户与管理员路径 alike）经 SSRF 防护（`ssrf.go`）校验它：仅允许 http/https scheme、host 不得是私有/保留 IP 段（含云元数据段 169.254.0.0/16）、DNS 解析结果不得落入私有/保留段（挫败 DNS rebinding）。违规返回 `webhook_url_forbidden`（422）。

**`archiveAndDelete`（单事务）：**

```
BEGIN
  INSERT delivery_history
    (user_id, post_id, channel_id, status, last_error, error_category, created_at)
    VALUES (attempt.user_id, attempt.post_id, attempt.channel_id, status, last_error, error_category, attempt.created_at)
  DELETE FROM delivery_attempts WHERE id = attempt.id
COMMIT
```

终态归档 + 删除是一个事务，历史记录与尝试行删除因此是原子的。

<a id="cleanup"></a>

## 清理

没有集中式的批量 DELETE 去清扫终态尝试。每个尝试行由它自己的投递操作在到达终态的那一刻删除 —— worker 在 `delivered`/`failed` 时的 `archiveAndDelete`，以及调度器在 `expired` 时的同一操作。清理因此分布在所有在途投递操作之中，从不产生大规模死元组爆发；这正是 `delivery_attempts` 保持小体量、autovacuum 负载保持轻量的原因。`delivery_history` 的保留期由 `prune-delivery-history` cron 命令单独清扫（见 [`delivery-queue.zh.md`](./delivery-queue.zh.md) _保留期_）。

<a id="metrics"></a>

## 指标

分发器经可选的 `Metrics` 接口（`WithMetrics`）记录；`cmd/server/main.go` 中的生产接线把它接到 OTel 工具（见 [`observability.zh.md`](./observability.zh.md)）：`delivery_pending`（gauge，入队 +N，每次终态迁移 −1）、`delivery_dispatched`（counter，每次成功发送）、`delivery_failed`（带错误类别标签的 counter，每次失败尝试）。`AttemptRepository.CountByStatus` 暴露按状态分组的尝试计数，供可观测性使用。

<a id="configuration"></a>

## 配置

```toml
[delivery]
body_preview_chars = 200                 # Feishu card body preview length
request_timeout = "5s"                   # per Feishu HTTP call; also drives the claim reservation
workers = 32                             # pond pool size (concurrent Feishu sends)
queue_size = 1024                        # pond task queue depth
scan_interval = "1s"                     # scheduler tick
history_retention = "168h"               # 7 days; delivery_history prune threshold
```

重试序列与过期墙**不在**本节：它们硬编码在 `internal/service/delivery/backoff.go`（见 [`delivery-retry.zh.md`](./delivery-retry.zh.md)）。默认值理由：

- `workers = 32` 在飞书 300 ms 响应下覆盖 116 任务/秒的持续上限（32/0.3 ≈ 106/秒），并为突发留出队列余量。
- `queue_size = 1024` 给池提供超出 worker 数的突发余量（pond v2 的 `WithQueueSize` 把"worker 忙"与"队列满"解耦）。
- `scan_interval = 1s` 在投递延迟（亚秒到秒级）与 DB 负载（每秒一次索引查询微不足道）之间取得平衡。
- `history_retention = 168h`（7 天）与 `post.retention_days` 对齐；历史不会比文章自身寿命活得更久，因此读取时的 JOIN 在保留窗口内总能找到活文章。
