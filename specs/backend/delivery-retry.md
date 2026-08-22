# Delivery Retry and Backoff

English | [中文](delivery-retry.zh.md)

Retry policy for the delivery queue: a hardcoded fixed backoff sequence with an auto-computed expiry wall, plus error classification that fails permanently-rejected sends fast instead of burning the retry budget. The constants live in `internal/service/delivery/backoff.go` and `internal/service/delivery/delivery_error.go`. Why the sequence is hardcoded rather than configurable, and the alternatives that lost, are recorded in [the delivery MRFC](../../.agents/mrfcs/implemented/2026-07-10-persistent-best-effort-delivery-queue.md); the queue tables it mutates are specified in [`delivery-queue.md`](./delivery-queue.md) and the scheduler that drives it in [`delivery-scheduler.md`](./delivery-scheduler.md).

## The backoff sequence

```go
var backoffSequence = [...]time.Duration{
    1 * time.Minute,
    5 * time.Minute,
    10 * time.Minute,
    20 * time.Minute,
}
```

The sequence lists the delays applied **after each failed attempt**. The first delivery attempt happens immediately on claim (no leading wait); each subsequent attempt waits `backoffSequence[attempts]` before retrying (`NextBackoff(attempts)` returns the delay before the (attempts+1)-th attempt, or `ok=false` once the count covers the whole sequence).

Timeline (worst case, every attempt fails):

```
t=0     create attempt. scheduler claims, first delivery attempt.
        fail → attempts=1, wait backoff[0]=1m
t=1m    retry 1. fail → attempts=2, wait backoff[1]=5m
t=6m    retry 2. fail → attempts=3, wait backoff[2]=10m
t=16m   retry 3. fail → attempts=4, wait backoff[3]=20m
t=36m   retry 4. fail → NextBackoff(4) not ok → FAILED (sequence exhausted)
```

The sequence yields **1 immediate attempt + 4 retries = up to 5 delivery attempts**, exhausting at t=36m.

**Why hardcoded, not configurable.** markpost ships exactly one delivery channel kind today (Feishu). There is no per-channel retry requirement to satisfy, and exposing `backoff_sequence` to operators would create a tuning surface with no consumer — premature configurability. The sequence changes via a code change + release, which already rotates the process and clears in-flight state. If a second channel kind with different retry semantics is added, configurability can be introduced then.

## The expiry wall (auto-computed from the sequence)

```
wall = round_up_to_10min( sum(backoffSequence) )
```

For the sequence above: sum(1+5+10+20) = 36 min → round up to **40 min**. The 4-minute margin (40−36) ensures the last retry (t=36m) does not collide with the wall. `computeExpiryWall` is a package function exercised by a unit test; it is not operator-tunable.

- A delivery still `pending` when `created_at + wall` passes is transitioned to **expired** by the scheduler's sweep (see [`delivery-scheduler.md`](./delivery-scheduler.md)).
- An empty sequence would disable retry (first failure → `failed`, wall = 0 and does not participate) — the fire-and-forget degenerate mode, retained as a code-level switch, not a config flag.

## Terminal states

| Status      | int8 | Meaning                                                | When                                    |
| ----------- | ---- | ------------------------------------------------------ | --------------------------------------- |
| `pending`   | 0    | awaiting or mid-delivery                               | initial; in-flight                      |
| `delivered` | 1    | a Feishu send succeeded                                | any attempt succeeds                    |
| `failed`    | 2    | the retry sequence was exhausted without success       | `NextBackoff` not ok, or non-retryable  |
| `expired`   | 3    | the time wall passed before the sequence was exhausted | `pending` and `created_at + wall < now` |

`failed` and `expired` are distinct so the user/history can tell "we tried everything" apart from "we ran out of time." With the sequence above, `failed` fires at t=36m and `expired` almost never fires (the sequence exhausts before the 40m wall); `expired` becomes relevant only when the scheduler is stalled. The numeric mapping is fixed by the `Status` enum (see [`delivery-queue.md`](./delivery-queue.md)) and its order is append-only forever.

## Error classification

`DeliveryError` (`delivery_error.go`) wraps every send failure with an `ErrorCategory` and a `Retryable` decision; `Error()`/`Unwrap()` forward to the cause so `last_error` text and existing string assertions are unchanged. Classification has three consumers: retry policy (below), the `delivery_failed` metric label, and the admin history filter (`delivery_history.error_category`).

| Category                  | Source                                            | Retryable | Meaning                                                                 |
| ------------------------- | ------------------------------------------------- | --------- | ----------------------------------------------------------------------- |
| `card_rejected`           | Feishu business code 11246                        | no        | The upstream rejected the card content (e.g. invalid image keys).       |
| `upstream_client_error`   | HTTP 4xx except 429                               | no        | The webhook is misconfigured or revoked.                                |
| `upstream_server_error`   | HTTP 5xx or 429                                   | yes       | Upstream fault or rate limit — transient.                               |
| `upstream_business_error` | any other non-zero Feishu business code           | yes       | Unrecognized upstream code; conservative default keeps it retryable.    |
| `network`                 | DNS, timeout, connection refused                  | yes       | The request never completed.                                            |
| `internal`                | post/channel fetch failure, payload marshal error | yes       | Our own data or logic error; retryable so transient hiccups get a shot. |

Only the specifically-known permanent Feishu code (11246) is non-retryable; every other business code defaults to retryable so an eventually-succeedable message is not dropped prematurely. The dispatcher applies the policy in `handleSendError`:

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

Fail-fast for permanent categories is the point of the classification: a card the upstream rejected for its content cannot succeed on retry, so the attempt archives as `failed` immediately instead of occupying the queue for 36 minutes. `delivered` and `expired` rows carry an empty `error_category`.
