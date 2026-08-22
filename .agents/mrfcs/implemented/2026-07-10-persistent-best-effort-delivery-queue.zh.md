# MRFC: Persistent best-effort delivery queue

Status: implemented

[English](2026-07-10-persistent-best-effort-delivery-queue.md) | 中文

## Problem

发往 Feishu 频道的帖子创建通知曾由一个进程内缓冲 channel 派发、单个 goroutine 排空：发后即忘、无持久化、无重试。一次重启丢失全部待发投递，一次 Feishu 抖动就永久丢掉通知，而单 goroutine 派发器在 300 ms 响应下峰值约 3 任务/秒 —— 比 SaaS 参考实例负载模型约 116 任务/秒的最坏上限低两个数量级。产品契约（"在有界窗口内尽力，然后告知用户"）背后没有任何机制。

## Decision

投递是一个三层的持久化尽力而为子系统（`internal/service/delivery/`）：一张 `delivery_attempts` PostgreSQL 表持有全部待决状态（热行，在任何终态的同一事务内归档进 `delivery_history` 并删除 —— 细节见 [`specs/backend/delivery-queue.md`](../../../specs/backend/delivery-queue.zh.md)），一个单 goroutine 的 1 s ticker 扫过到期墙并认领到期行（`FOR UPDATE SKIP LOCKED`，把 `next_at` 预约到请求超时加 500 ms 缓冲之后，使在途行对下一个 tick 不可见 —— [`specs/backend/delivery-scheduler.md`](../../../specs/backend/delivery-scheduler.zh.md)），以及一个含 32 个 worker 的有界 pond v2 池发出 Feishu HTTP 调用。重试间隔是 `backoff.go` 中硬编码的序列 `[1m, 5m, 10m, 20m]`，40 分钟到期墙按 `round_up_to_10min(sum(sequence))` 计算；两者都不可由运维者配置。发送失败被分类（`delivery_error.go`）为带显式可重试标志的类别 —— 被永久拒绝的卡片或被吊销的 webhook 立即归档为 `failed`，而不是烧掉 36 分钟预算，且类别记录在历史行上（`000007_delivery_error_category`），供管理端过滤器与 `delivery_failed` 指标标签使用（[`specs/backend/delivery-retry.md`](../../../specs/backend/delivery-retry.zh.md)）。投递是 at-least-once：发送成功与归档提交之间的一次崩溃可能复制一张卡片，这被接受是因为卡片内容幂等（[`specs/backend/delivery-recovery.md`](../../../specs/backend/delivery-recovery.zh.md)）。状态机是 `pending → delivered | failed | expired`，没有中间 `running` 状态 —— `next_at` 预约以少一个状态的代价提供在途去重。

## Alternatives considered

**保留发后即忘的 channel 派发器。** 可能的最简设计，但它在 116 任务/秒的上限下丢掉约 97% 的投递，并在重启时丢失全部待决工作；产品契约要求持久化与有界重试。

**外部消息代理（Cloudflare Queues、Redis、RabbitMQ）。** Cloudflare Queues 经深入评估后在架构上被拒：它的推送消费者只能在 Workers 上运行，Feishu 卡片逻辑就得在第二个代码库里重新实现，带上自己的部署管线和一个云依赖，却比不过一张 Postgres 表 + ticker。Redis/RabbitMQ 为一个写入率均值约 0.12 帖/秒的单实例部署加上常驻依赖与运维负担 —— 与[性能优化 MRFC](./2026-07-09-read-path-performance-pass.zh.md)拒绝第二台 VPS 的推理相同。

**并发层用 ants v2。** ants 是一个回收 goroutine 的池，没有内建任务队列，采用它就得把手写的缓冲 channel 保留为独立的排队层，还携带一个 `golang.org/x/sync` 依赖。pond v2 零依赖，有一等公民的队列（`WithQueueSize`）、非阻塞提交、优雅排空、默认 panic 恢复和指标面 —— 与投递需求一一对应。

**可配置的退避序列 / 到期墙。** 渠道种类只有一个（Feishu），也没有按渠道的重试需求，旋钮会是一个没有消费者的调参面。纯指数退避（30s→…→16m）让靠后的间隔太稀疏，接不住中等时长的故障，且最后一个间隔与 30 分钟墙相撞；封顶指数可行，但不如总和恰好定出自动墙的固定序列确定。改常量就是一次代码变更 + 发布，而发布本就会重启进程并清空在途状态。

**经事务性 outbox / 去重表实现 exactly-once 投递。** 会给每次投递的热路径加一次写，以防一张偶发的重复卡片 —— 而其内容本就幂等。对尽力而为的通知，at-least-once 才是诚实的契约。

**一个 `running` 中间状态。** 把 UPDATE 数翻倍（pending→running→terminal）并迫使 running 行接受特殊的索引待遇，而 `next_at` 预约用 `status = 0` 上的现有部分索引就达到同等正确性。

**`delivery_history` 上的快照列与 `ON DELETE CASCADE`。** `post_title`/`channel_name` 快照以零收益违反规范化规则（20 行分页 JOIN 是亚毫秒级），而把用户删除级联进 7 天历史会在一次 `DELETE FROM users` 里对数百万行锁风暴 —— `SET NULL` 把每行保留为一条匿名记录。`delivery_attempts` 保留 CASCADE，因为 ≤40 分钟的行寿命把级联限定在该用户的在途尝试上；两张表刻意为同一逻辑列采取不同的 FK 动作。

**终态尝试的集中式批量清理，以及不分批的扫除。** 周期性对终态尝试做批量 DELETE 会产生巨大的死元组迸发；把删除分摊到在途投递操作上使每一次都很小。到期墙扫除与历史修剪按语句以子查询 `LIMIT` 形式分批，因为裸的 `DELETE/UPDATE ... LIMIT` 是 PostgreSQL 语法错误，而无界扫除会在 1 s tick 下锁住整个匹配区间。

## Consequences

投递在重启后存活，在 40 分钟内到达终态，并经由历史行及其错误类别对用户保持诚实。队列可观测（`delivery_pending`/`delivery_dispatched`/`delivery_failed` 指标、`CountByStatus`、管理端历史过滤器与故障渠道查询）。接受的代价：at-least-once 重复（对幂等卡片无害）、需要一个发布才能更改的硬编码重试序列、永远冻结为只追加的 `Status` 排序，以及依赖 PostgreSQL 专有认领 SQL 的调度器 —— 这是唯一受支持的数据库（[PostgreSQL-only MRFC](./2026-07-26-postgresql-only.zh.md)移除了该子系统最初携带的多方言分支）。
