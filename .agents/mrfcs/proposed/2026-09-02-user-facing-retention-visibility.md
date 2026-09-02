# MRFC: User-facing retention visibility

Status: proposed

English | [中文](2026-09-02-user-facing-retention-visibility.zh.md)

## Problem

Retention in markpost is a promise made about a user's data without ever being shown to its owner. The windows live in global config (`[post] retention_days`, `[delivery] history_retention`), and since the per-user policy ([per-user retention MRFC](2026-08-31-per-user-history-retention-policy.md)) a single override can say "keep this person's data forever". The owner sees none of it: `/posts` and `/delivery/history` simply lose rows at sweep time. A user who assumed permanence meets silent deletion; one who assumed ephemerality leaves content sitting indefinitely. That MRFC deliberately deferred owner-facing visibility ("a badge on /posts… a follow-up if operations asks for it"); operations has now asked. The displayable value is the *effective* policy — override ?? global — which only the server can resolve per caller; static copy would lie to exactly the overridden users the policy exists to protect.

## Proposal

One authenticated endpoint and one shared page-level hint.

**`GET /api/v1/me/retention`** (JWT, plain read — no dedicated limiter) returns the caller's effective policy as `{posts_days, history_days}`, each `0` (keep forever, reusing the existing zero encoding) or a whole-day count. Resolution mirrors the prune predicate exactly (the per-row `CASE` in `post_repo`/`delivery_attempt_repo`): an explicit override wins for both kinds; an inheriting user reads each table's own global, so the two numbers can diverge only when the globals drift apart. `history_retention` is a Go duration; it renders as whole days, ceiling for a nonzero remainder — the display must never read "0 days" while meaning "not forever". The endpoint opens the `/me` namespace: self-scoped reads gather there from now on.

**A shared `RetentionHint`** — one muted line between `PageHeading` and the list on `/posts` and `/delivery/history`, fed by a single query with a 5-minute `staleTime` (the value moves only when an admin acts or a deploy changes config). Copy states the outcome, never the mechanics: forever → "Data is kept permanently"; N days → "Data is kept for N days, then removed automatically". `/posts` renders `posts_days`; `/delivery/history` renders `history_days`. Inherit/VIP mechanics stay admin-facing (the [per-user retention MRFC](2026-08-31-per-user-history-retention-policy.md) owns that surface); the owner reads the outcome only.

The schema prerequisite already ships — `users.retention_days` and the resolution semantics merged with the per-user stack's prune layer; the stack's remaining admin layers are orthogonal and impose no ordering.

## Alternatives considered

**Per-row expiry timestamps.** Declined with the maintainer: one user's rows share one policy, so a page-level line already answers "how long is my data kept"; per-row dates restate the same number as visual noise.

**Static copy from global config, no API.** Cheapest — and wrong for exactly the users the policy exists for: a forever-override VIP would read the global "7 days". Only the server can resolve override ?? global per caller.

**Fold the hint into the list endpoints' envelopes.** Couples the notice to the posts list's 3-second first-page polling and duplicates resolution across two payloads; a dedicated lightweight read mirrors the admin layer's `GET /admin/retention/defaults` and keeps one home.

**Cover the admin lists too.** Declined with the maintainer: the per-user stack's admin UX already gives the users list its effective Retention column; admin posts/history lists would restate governance data, not the owner's promise this feature exists for.

## Acceptance criteria

- testcontainers: explicit 0 → `{0, 0}`; explicit N → `{N, N}`; inherit under default globals → `{7, 7}`; inherit under `[post] retention_days = 0` → posts forever; 401 unauthenticated. Duration-to-days rounding unit-tested (168h → 7, 36h → 2).
- Both pages render the hint in forever and N-day variants; the four-locale key-consistency test passes.
- Playwright against the dev stack: screenshots of both pages × both variants.
- The [api-schema](../../../specs/backend/api-schema.md) bilingual pair gains the endpoint row; no new spec page, no `specs/index.md` change.

## Risks

- The copy promises removal ("then removed automatically"); the sweep itself is the daily cron the prune layer deploys. The hint states policy, not deletion timing — if operations disables the cron, the words outlive the mechanics. Recorded so nobody reads the badge as a timer.
- `/me` opens a namespace; future self-scoped endpoints gather there rather than sprouting new prefixes.
- The hint can lag a policy change by up to its 5-minute `staleTime`; acceptable for a reassurance line, stated so the number's freshness is understood.
- History displays whole days while the underlying cutoff is a timestamp; the sweep boundary can differ from the displayed day boundary by hours.
