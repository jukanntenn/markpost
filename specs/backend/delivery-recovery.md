# Delivery Semantics and Crash Recovery

English | [中文](delivery-recovery.zh.md)

The delivery subsystem's product contract and its behavior across process crashes. The mechanism references live in the sibling pages — [`delivery-queue.md`](./delivery-queue.md) for the tables, [`delivery-scheduler.md`](./delivery-scheduler.md) for the dispatcher, [`delivery-retry.md`](./delivery-retry.md) for retry policy. Why exactly-once and an external broker were rejected is recorded in [the delivery MRFC](../../.agents/mrfcs/implemented/2026-07-10-persistent-best-effort-delivery-queue.md).

## Product semantics

markpost's delivery sends a notification (an interactive Feishu card) to each of a post author's configured delivery channels when a post is created. The keyword filter expression ([`keyword-filter.md`](./keyword-filter.md)) decides, per channel, whether a given post is pushed.

- **Trigger:** post creation only (`CreatePost`). There is no update-triggered or deletion-triggered delivery; posts are immutable and write-once ([`caching.md`](./caching.md)).
- **Target:** Feishu webhook URLs today (`delivery.ChannelKindFeishu`). The `switch channel.Kind` in the send path leaves room for more kinds without a schema change.
- **Delivery is best-effort, not guaranteed.** The contract is "try hard within a bounded window" (the 40-minute expiry wall), not "deliver exactly once." A notification that cannot reach Feishu after the retry sequence is exhausted is recorded as `failed` and shown to the user; it is not retried forever.
- **Message latency is acceptable.** Seconds-to-minutes delay between post creation and notification is fine. This permits persistence + scheduled retry rather than synchronous send.

## At-least-once delivery

A worker may crash between a successful Feishu send and the `archiveAndDelete` commit. On restart the row is still `pending` (the archive never committed), the scheduler re-claims it, and Feishu receives a duplicate card. This is the inherent cost of at-least-once; Feishu webhooks tolerate occasional duplicates, and the card content is idempotent — re-sending the same post is annoying, not harmful. Exactly-once would require a transactional outbox or a dedup table keyed by an idempotency key, adding a write to the hot path of every delivery; the duplicate-card cost is lower than that machinery.

## Crash recovery

All pending state lives in `delivery_attempts`, not in process memory. A process restart re-runs the scheduler, which re-claims every `pending` row whose `next_at <= now`. The only delivery work lost on crash is an in-flight HTTP call that had not yet returned — and that row is re-claimed on the next tick. Graceful shutdown (`Dispatcher.Stop`) closes the scheduler and drains the worker pool via `StopAndWait`; in-flight sends either complete and archive, or the process exits and the rows are re-claimed after restart.

## Double-claim prevention

The claim query advances `next_at` to `now + request_timeout + 500 ms` (see [`delivery-scheduler.md`](./delivery-scheduler.md)), so a row being delivered is invisible to the next scheduler tick. If the worker dies mid-delivery, the reserved `next_at` elapses and the row becomes re-claimable — the retry resumes naturally. There is no `running` intermediate state: in-flight deduplication is achieved by the `next_at` reservation alone, with one state fewer to transition and index (the rejected `running`-state alternative is recorded in the MRFC).
