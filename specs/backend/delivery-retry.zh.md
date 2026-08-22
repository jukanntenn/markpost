# 投递重试与退避

[English](delivery-retry.md) | 中文

投递队列的重试策略：硬编码的固定退避序列加自动计算的过期墙，再加上让永久被拒的发送快速失败、而不是烧光重试预算的错误分类。常量位于 `internal/service/delivery/backoff.go` 与 `internal/service/delivery/delivery_error.go`。序列为何硬编码而非可配置，以及落败的替代方案，记录在[投递 MRFC](../../.agents/mrfcs/implemented/2026-07-10-persistent-best-effort-delivery-queue.zh.md)；它改写的队列表见 [`delivery-queue.zh.md`](./delivery-queue.zh.md)，驱动它的调度器见 [`delivery-scheduler.zh.md`](./delivery-scheduler.zh.md)。

<a id="the-backoff-sequence"></a>

## 退避序列

```go
var backoffSequence = [...]time.Duration{
    1 * time.Minute,
    5 * time.Minute,
    10 * time.Minute,
    20 * time.Minute,
}
```

该序列列出**每次失败尝试之后**施加的延迟。第一次投递尝试在认领时立即发生（无前置等待）；后续每次尝试等待 `backoffSequence[attempts]` 再重试（`NextBackoff(attempts)` 返回第（attempts+1）次尝试之前的延迟；计数覆盖整个序列后返回 `ok=false`）。

时间线（最坏情况，每次尝试都失败）：

```
t=0     create attempt. scheduler claims, first delivery attempt.
        fail → attempts=1, wait backoff[0]=1m
t=1m    retry 1. fail → attempts=2, wait backoff[1]=5m
t=6m    retry 2. fail → attempts=3, wait backoff[2]=10m
t=16m   retry 3. fail → attempts=4, wait backoff[3]=20m
t=36m   retry 4. fail → NextBackoff(4) not ok → FAILED (sequence exhausted)
```

该序列给出 **1 次立即尝试 + 4 次重试 = 至多 5 次投递尝试**，在 t=36m 耗尽。

**为何硬编码而非可配置。** markpost 今天恰好只交付一种投递渠道类型（飞书）。没有需要满足的按渠道重试需求，把 `backoff_sequence` 暴露给运维者会创造一个没有消费者的调优面 —— 过早的可配置性。序列经代码变更 + 发布来改变，而发布本就轮换进程并清空在途状态。如果加入重试语义不同的第二种渠道类型，届时再引入可配置性。

<a id="the-expiry-wall-auto-computed-from-the-sequence"></a>

## 过期墙（由序列自动计算）

```
wall = round_up_to_10min( sum(backoffSequence) )
```

对上面的序列：sum(1+5+10+20) = 36 分钟 → 向上取整为 **40 分钟**。4 分钟余量（40−36）确保最后一次重试（t=36m）不与墙相撞。`computeExpiryWall` 是被单元测试覆盖的包级函数；它不对运维者开放调节。

- 一个在 `created_at + wall` 过去时仍 `pending` 的投递被调度器清扫迁移为 **expired**（见 [`delivery-scheduler.zh.md`](./delivery-scheduler.zh.md)）。
- 空序列会禁用重试（第一次失败 → `failed`，墙为 0 且不参与）—— 发射后不管的退化模式，作为代码级开关保留，不是配置项。

<a id="terminal-states"></a>

## 终态

| 状态        | int8 | 含义                     | 触发时机                               |
| ----------- | ---- | ------------------------ | -------------------------------------- |
| `pending`   | 0    | 等待中或投递中           | 初始；在途                             |
| `delivered` | 1    | 一次飞书发送成功         | 任一次尝试成功                         |
| `failed`    | 2    | 重试序列耗尽仍未成功     | `NextBackoff` 返回 not ok，或不可重试  |
| `expired`   | 3    | 时间墙在序列耗尽之前过去 | `pending` 且 `created_at + wall < now` |

`failed` 与 `expired` 是不同的状态，让用户/历史能区分"我们试过了一切"与"我们没时间了"。对上面的序列，`failed` 在 t=36m 触发而 `expired` 几乎从不触发（序列在 40 分钟墙之前耗尽）；`expired` 只在调度器停滞时才变得相关。数值映射由 `Status` 枚举固定（见 [`delivery-queue.zh.md`](./delivery-queue.zh.md)），其顺序永远只允许追加。

<a id="error-classification"></a>

## 错误分类

`DeliveryError`（`delivery_error.go`）为每个发送失败包装一个 `ErrorCategory` 与一个 `Retryable` 决定；`Error()`/`Unwrap()` 转发给原因错误，因此 `last_error` 文本与既有的字符串断言不变。分类有三个消费者：重试策略（下文）、`delivery_failed` 指标标签、管理员历史过滤器（`delivery_history.error_category`）。

| 类别                      | 来源                              | 可重试 | 含义                                                       |
| ------------------------- | --------------------------------- | ------ | ---------------------------------------------------------- |
| `card_rejected`           | 飞书业务码 11246                  | 否     | 上游拒绝了卡片内容（例如无效的图片 key）。                 |
| `upstream_client_error`   | 除 429 外的 HTTP 4xx              | 否     | webhook 配置错误或已被吊销。                               |
| `upstream_server_error`   | HTTP 5xx 或 429                   | 是     | 上游故障或限流 —— 瞬态。                                   |
| `upstream_business_error` | 其他任何非零飞书业务码            | 是     | 未识别的上游码；保守默认保持可重试。                       |
| `network`                 | DNS、超时、连接被拒               | 是     | 请求从未完成。                                             |
| `internal`                | 文章/渠道查询失败、载荷序列化错误 | 是     | 我们自己的数据或逻辑错误；可重试，让瞬态抖动获得一次机会。 |

只有具体已知的永久性飞书码（11246）不可重试；其他所有业务码默认可重试，使最终可成功的消息不被过早丢弃。分发器在 `handleSendError` 中应用该策略：

```
nextAttempts := attempt.attempts + 1
category, retryable := classify(sendErr)

if NextBackoff(nextAttempts - 1) not ok   -- sequence exhausted
   or not retryable:                     -- permanent failure
    archiveAndDelete(attempt, status=failed,
                     last_error=truncate(err, 200), error_category=category)
else:
    MarkRetry(attempt.id, attempts=nextAttempts,
              last_error=truncate(err, 200), next_at=now + backoff)
```

对永久类别快速失败正是分类的意义所在：一张被上游因内容拒绝的卡片重试不可能成功，因此该尝试立即归档为 `failed`，而不是在队列里占用 36 分钟。`delivered` 与 `expired` 行携带空的 `error_category`。
