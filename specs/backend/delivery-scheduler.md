# Delivery Scheduler and Claim Path

The delivery subsystem is a three-layer pipeline: a PostgreSQL table for persistence, a single-goroutine ticker for scheduling, and a bounded pond v2 worker pool for concurrency. This page specifies the dispatcher (`internal/service/delivery/dispatcher.go`) — enqueue, the scheduler tick, the atomic claim, worker execution, and cleanup. The data model is specified in [`delivery-queue.md`](./delivery-queue.md); retry timing and failure classification in [`delivery-retry.md`](./delivery-retry.md); delivery semantics and crash recovery in [`delivery-recovery.md`](./delivery-recovery.md). Decision rationale (pond over ants, no broker, batched sweeps) lives in [the delivery MRFC](../../mrfc/implemented/2026-07-10-persistent-best-effort-delivery-queue.md).

## Capacity envelope (the SaaS reference instance)

The hardware envelope is a 2-core / 2 GB / 3 Mbps VPS (see [`caching.md`](./caching.md) for the full table). Under the worst-case load model — 10 000 users all saturating the L2 write limit (1000 posts/day each) — the sustained delivery ceiling is ~116 posts/s (10 000 × 1000 ÷ 86 400).

Two facts shape the design:

1. **CPU is not the bottleneck.** Delivery is I/O-bound (goroutines blocked on Feishu HTTP responses consume near-zero CPU). At 116 jobs/s the entire delivery subsystem (filter + DB writes + HTTP fan-out) costs under 7% of two cores. The filter benchmarks (`internal/service/delivery/filter/filter_bench_test.go`) show compile+match at ~1.6 µs/op for a 256-byte title; even with 10 channels per post this is sub-percent CPU.
2. **A single-goroutine dispatcher cannot carry the load.** One goroutine issuing synchronous HTTP calls (5 s timeout) tops out at ~3 jobs/s at a 300 ms Feishu response — two orders of magnitude below the ceiling. A bounded buffer would fill and silently drop deliveries.

## Three layers, three responsibilities

| Layer          | Responsibility                                                       | Mechanism                                                        | Why                                                                                                                              |
| -------------- | -------------------------------------------------------------------- | ---------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| 1. Persistence | survive process restart; do not lose pending work                    | `delivery_attempts` table (PostgreSQL)                           | A worker-pool library is an in-memory concurrency primitive; persistence is the database's job, not the pool's.                  |
| 2. Scheduling  | scan due tasks; claim without double-delivery; apply the expiry wall | Go `time.Ticker` (1 s) + atomic claim (`FOR UPDATE SKIP LOCKED`) | A simple polling scheduler is easy to reason about and needs no external broker.                                                 |
| 3. Concurrency | issue up to N Feishu HTTP calls in parallel, bounded                 | pond v2 worker pool                                              | Bounded concurrency controls outbound rate (Feishu QPS limits) and bandwidth (3 Mbps cap); unbounded goroutines would risk both. |

The concurrency layer is `github.com/alitto/pond/v2` (zero external dependencies) configured with `WithQueueSize(1024)` and `WithNonBlocking(true)`: when the queue is full, `pool.Go` returns `ErrQueueFull`, the task is dropped with a log line, and the attempt row stays claimed until its `next_at` reservation elapses — at which point the next tick re-claims it and retries. The pool also provides graceful drain (`StopAndWait`, used by `Dispatcher.Stop`) and default panic recovery.

## Enqueue (in `CreatePost`, synchronous)

```
CreatePost succeeds (post.ID known)
  channels := channelRepo.GetByUserID(userID)
  for each enabled channel:
      if keywordFilter(channel.Keywords).Match(post.Title):
          INSERT delivery_attempts
              (user_id, post_id, channel_id, status=StatusPending,
               attempts=0, next_at=now, created_at=now, updated_at=now)
```

The keyword filter ([`keyword-filter.md`](./keyword-filter.md)) runs **before** persistence, so only channels that actually match produce attempt rows; a channel whose keyword expression fails to compile is skipped with a log line. `Dispatcher.Enqueue` implements `post.DeliveryEnqueuer`: it is synchronous but best-effort — any error (channel list, compile, batch insert) is logged and swallowed, and it never returns an error, so a delivery failure can never fail post creation. On success it bumps the `delivery_pending` gauge by the number of inserted rows.

## The scheduler tick

One goroutine ticks every `[delivery] scan_interval` (default 1 s) and runs two duties per tick, in order: the expiry-wall sweep, then the claim. Both are **batched** (bounded batch size per statement) and use the **subquery-`LIMIT` form** (`WHERE id IN (SELECT ... LIMIT N)`), never bare `UPDATE/DELETE ... LIMIT` — PostgreSQL does not support `LIMIT` directly on UPDATE/DELETE. The batch size for both duties is 64.

**Expire wall sweep** — transition pending rows past the wall to `expired`, then archive them:

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

Batching bounds the lock scope and dead-tuple volume per statement: under a large `pending` backlog (e.g. a Feishu outage accumulating tens of thousands of stalled rows), one unbounded `UPDATE ... RETURNING *` would lock the whole matched row range and burst dead tuples under the 1 s tick. The within-tick loop still drains the backlog promptly — the wall is minutes-scale, so sub-second drain is not required.

**Claim due tasks** — atomically claim and reserve:

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

`$reserve_until` is `now + [delivery] request_timeout + 500 ms` (the `claimReserveBuffer` constant). Two properties:

- **`FOR UPDATE SKIP LOCKED`** is PostgreSQL's documented queue pattern: concurrent claimers — or overlapping scheduler ticks — take disjoint row sets instead of blocking, so claim throughput scales with worker count rather than serializing.
- **The reservation is the double-claim fix.** Advancing `next_at` past the request timeout makes in-flight rows invisible to the next 1-second tick. Without it, a row whose delivery takes longer than 1 s would be re-claimed and re-delivered. If the worker dies mid-delivery, the reservation elapses and the row becomes re-claimable — the retry resumes naturally (see [`delivery-recovery.md`](./delivery-recovery.md)).

## Worker execution

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

Fetch errors (post or channel lookup) go through the same `handleSendError` path as send errors: they are classified `internal` (retryable) and follow the standard backoff sequence. `PostDeliveryService` (`post_delivery.go`) is the `Sender` implementation: it builds the Feishu card from the post + channel and issues the webhook call through `FeishuClient` (`feishu.go`), which applies the configured request timeout and classifies non-2xx responses and Feishu business codes into `DeliveryError` categories (see [`delivery-retry.md`](./delivery-retry.md)). `SendTest` sends a fixed diagnostic card so a user can verify a webhook without publishing a post — it is fire-and-forget and never enters the retry queue or writes history.

**Webhook URL security.** `webhook_url` is a user-controlled server-side request target. Channel create/update (user and admin paths alike) validates it through the SSRF guard (`ssrf.go`): http/https scheme only, host must not be a private/reserved IP range (including cloud-metadata 169.254.0.0/16), and DNS resolution must not land in a private/reserved range (defeating DNS rebinding). Violations return `webhook_url_forbidden` (422).

**`archiveAndDelete` (single transaction):**

```
BEGIN
  INSERT delivery_history
    (user_id, post_id, channel_id, status, last_error, error_category, created_at)
    VALUES (attempt.user_id, attempt.post_id, attempt.channel_id, status, last_error, error_category, attempt.created_at)
  DELETE FROM delivery_attempts WHERE id = attempt.id
COMMIT
```

The terminal-state archive + delete is one transaction so the history record and the attempt removal are atomic.

## Cleanup

There is **no centralized batch DELETE** sweeping terminal attempts. Each attempt row is deleted by its own delivery operation at the moment it reaches a terminal state — `archiveAndDelete` from the worker on `delivered`/`failed`, and from the scheduler on `expired`. Cleanup is therefore distributed across all in-flight delivery operations and never produces a large dead-tuple burst; this is why `delivery_attempts` stays small and its autovacuum load stays light. `delivery_history` retention is swept separately by the `prune-delivery-history` cron command (see [`delivery-queue.md`](./delivery-queue.md) _Retention_).

## Metrics

The dispatcher records through an opt-in `Metrics` interface (`WithMetrics`); the production wiring in `cmd/server/main.go` connects it to the OTel instruments (see [`observability.md`](./observability.md)): `delivery_pending` (gauge, +N on enqueue, −1 on each terminal transition), `delivery_dispatched` (counter, per successful send), `delivery_failed` (counter with the error-category label, per failed attempt). `AttemptRepository.CountByStatus` exposes per-status attempt counts for observability.

## Configuration

```toml
[delivery]
body_preview_chars = 200                 # Feishu card body preview length
request_timeout = "5s"                   # per Feishu HTTP call; also drives the claim reservation
workers = 32                             # pond pool size (concurrent Feishu sends)
queue_size = 1024                        # pond task queue depth
scan_interval = "1s"                     # scheduler tick
history_retention = "168h"               # 7 days; delivery_history prune threshold
```

The retry sequence and the expiry wall are **not** in this section: they are hardcoded in `internal/service/delivery/backoff.go` (see [`delivery-retry.md`](./delivery-retry.md)). Defaults rationale:

- `workers = 32` covers the 116 jobs/s sustained ceiling at a 300 ms Feishu response (32/0.3 ≈ 106/s) with queue headroom for bursts.
- `queue_size = 1024` gives the pool burst headroom beyond the worker count (pond v2's `WithQueueSize` decouples "workers busy" from "queue full").
- `scan_interval = 1s` balances delivery latency (sub-second-to-second) against DB load (one indexed query per second is negligible).
- `history_retention = 168h` (7 days) matches `post.retention_days`; history outlives no post by more than the post's own lifetime, so the JOIN at read time always finds a live post within the retention window.
