# 投递语义与崩溃恢复

[English](delivery-recovery.md) | 中文

投递子系统的产品契约及其跨进程崩溃的行为。机制细节位于兄弟页面 —— 表见 [`delivery-queue.zh.md`](./delivery-queue.zh.md)，分发器见 [`delivery-scheduler.zh.md`](./delivery-scheduler.zh.md)，重试策略见 [`delivery-retry.zh.md`](./delivery-retry.zh.md)。恰好一次与外部消息代理为何被拒绝，记录在[投递 MRFC](../../.agents/mrfcs/implemented/2026-07-10-persistent-best-effort-delivery-queue.zh.md)。

<a id="product-semantics"></a>

## 产品语义

markpost 的投递在文章创建时向文章作者配置的每个投递渠道发送一条通知（一张可交互的飞书卡片）。关键词过滤器表达式（[`keyword-filter.zh.md`](./keyword-filter.zh.md)）逐渠道决定一篇给定文章是否被推送。

- **触发：** 仅文章创建（`CreatePost`）。没有更新触发或删除触发的投递；文章不可变且一次写入（[`caching.zh.md`](./caching.zh.md)）。
- **目标：** 当前的飞书 webhook URL（`delivery.ChannelKindFeishu`）。发送路径中的 `switch channel.Kind` 为更多类型留出空间，无需 schema 变更。
- **投递是尽力而为，不保证送达。** 契约是"在有界窗口内尽力尝试"（40 分钟过期墙），而非"恰好一次投递"。重试序列耗尽后仍无法到达飞书的通知记为 `failed` 并展示给用户；它不会被永远重试。
- **消息延迟可接受。** 文章创建与通知之间数秒到数分钟的延迟没有问题。这允许持久化 + 调度重试，而非同步发送。

<a id="at-least-once-delivery"></a>

## 至少一次投递

worker 可能在飞书发送成功与 `archiveAndDelete` 提交之间崩溃。重启后该行仍是 `pending`（归档从未提交），调度器重新认领它，飞书收到一张重复卡片。这是至少一次的固有代价；飞书 webhook 容忍偶发重复，且卡片内容是幂等的 —— 重发同一篇文章令人困扰，但无害。恰好一次需要事务性发件箱或以幂等键为键的去重表，为每次投递的热路径增加一次写入；重复卡片的代价低于那套机制。

<a id="crash-recovery"></a>

## 崩溃恢复

所有 pending 状态位于 `delivery_attempts`，不在进程内存中。进程重启后重新运行调度器，它重新认领每一个 `next_at <= now` 的 `pending` 行。崩溃时唯一丢失的投递工作是一个尚未返回的在途 HTTP 调用 —— 而该行在下一个 tick 被重新认领。优雅停机（`Dispatcher.Stop`）关闭调度器并经 `StopAndWait` 排空 worker 池；在途发送要么完成并归档，要么进程退出、该行在重启后被重新认领。

<a id="double-claim-prevention"></a>

## 重复认领防护

认领查询把 `next_at` 推进到 `now + request_timeout + 500 ms`（见 [`delivery-scheduler.zh.md`](./delivery-scheduler.zh.md)），因此正在投递的行对下一个调度器 tick 不可见。如果 worker 在投递中途死亡，预留的 `next_at` 到期后该行重新可被认领 —— 重试自然恢复。没有 `running` 中间状态：在途去重仅凭 `next_at` 预留实现，少一个需要迁移和索引的状态（被拒绝的 `running` 状态替代方案记录在 MRFC 中）。
