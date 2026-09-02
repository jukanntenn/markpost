# MRFC: User-facing retention visibility

Status: implemented

English | [中文](2026-09-02-user-facing-retention-visibility.zh.md)

## Problem

Retention in markpost is a promise made about a user's data without ever being shown to its owner. The windows live in global config (`[post] retention_days`, `[delivery] history_retention`), and since the per-user policy ([per-user retention MRFC](2026-08-31-per-user-history-retention-policy.md)) a single override can say "keep this person's data forever". The owner sees none of it: `/posts` and `/delivery/history` simply lose rows at sweep time. A user who assumed permanence meets silent deletion; one who assumed ephemerality leaves content sitting indefinitely. That MRFC deliberately deferred owner-facing visibility ("a badge on /posts… a follow-up if operations asks for it"); operations then asked. The displayable value is the *effective* policy — override ?? global — which only the server can resolve per caller; static copy would lie to exactly the overridden users the policy exists to protect.

## Decision

`GET /api/v1/me/retention` (JWT, plain read — no dedicated limiter) returns the caller's effective policy as `{posts_days, history_days}`, each `0` (keep forever, reusing the existing zero encoding) or a whole-day count. Resolution mirrors the prune predicate exactly (the per-row `CASE` in `post_repo`/`delivery_attempt_repo`): an explicit override drives both kinds; an inheriting user reads each table's own global, so the two numbers diverge only when the globals drift apart. The auth middleware reloads the user row on every request, so the endpoint resolves from the context user with no repository round-trip; `history_retention` renders as whole days, ceiling for a nonzero remainder — the display never reads "0 days" while meaning "not forever". The endpoint opened the `/me` namespace: self-scoped reads gather there.

A shared `RetentionHint` renders one muted line between `PageHeading` and the list on `/posts` and `/delivery/history`, fed by a single query with a 5-minute `staleTime` (the value moves only when an admin acts or a deploy changes config). Copy states the outcome, never the mechanics: forever → "Data is kept permanently"; N days → "Data is kept for N days, then removed automatically" — shipped in all four locales. `/posts` renders `posts_days`; `/delivery/history` renders `history_days`; inherit/VIP mechanics stay admin-facing.

## Alternatives considered

**Per-row expiry timestamps.** Declined with the maintainer: one user's rows share one policy, so a page-level line already answers "how long is my data kept"; per-row dates restate the same number as visual noise.

**Static copy from global config, no API.** Cheapest — and wrong for exactly the users the policy exists for: a forever-override VIP would read the global "7 days". Only the server can resolve override ?? global per caller.

**Fold the hint into the list endpoints' envelopes.** Couples the notice to the posts list's 3-second first-page polling and duplicates resolution across two payloads; a dedicated lightweight read mirrors the admin layer's `GET /admin/retention/defaults` and keeps one home.

**Cover the admin lists too.** Declined with the maintainer: the per-user stack's admin UX already gives the users list its effective Retention column; admin posts/history lists would restate governance data, not the owner's promise this feature exists for.

## Consequences

What the trade-off bought: the data owner reads the promise the system makes about their data, resolved per caller — a forever-override user no longer reads a global "7 days" that does not apply to them, and an ephemeral-by-default user knows deletion is scheduled. The read is one cheap JSON response with no repository round-trip. Verification: the resolution matrix (explicit forever, explicit N, inherit under default globals, inherit under global posts 0) and the duration ceiling (168h → 7, 36h → 2, sub-day → 1) are pinned in service unit tests; the handler runs the same matrix through the gin engine plus the no-context-user fail-closed path, while the unauthenticated 401 stays owned by `AuthWithBlacklist`'s middleware suite — the shipped resolution is pure, so the matrix runs in fast tests rather than testcontainers; `RetentionHint` variants are covered by component tests against MSW; the [api-schema](../../../specs/backend/api-schema.md) pair documents the endpoint; and the two pages are verified with Playwright screenshots — both pages in the forever and N-day variants, carried as acceptance-evidence comments on the delivery PR (#81) instead of committed binaries. What it costs: the copy promises removal while the sweep itself is the daily cron the prune layer deploys — the hint states policy, not deletion timing; the hint can lag a policy change by up to its 5-minute `staleTime`; history displays whole days while the underlying cutoff is a timestamp, so the sweep boundary can differ from the displayed day boundary by hours; and `/me` is now a namespace that future self-scoped endpoints join instead of sprouting new prefixes.
