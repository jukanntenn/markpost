# MRFC: Per-user history retention policy

Status: proposed

English | [中文](2026-08-31-per-user-history-retention-policy.zh.md)

## Problem

Retention in markpost is one-size-fits-all: `[post] retention_days` (default 7, 0 = never) and `[delivery] history_retention` (default 168h) are global config keys, swept by two cron-invoked CLI commands (`prune-expired-posts`, `prune-delivery-history`). Operations now needs per-user promises — "this user's history is kept forever", "these users keep 30 days" — typically tied to the VIP honorific, and nothing user-scoped exists anywhere in the schema, the API, or the admin UI. Two adjacent debts force themselves into the same decision: the repo schedules no cron for the prune commands at all (deployments that skip the manual step simply never delete), and shortening a retention window is irreversible deletion at the next sweep, so admin ergonomics here are a correctness concern, not polish.

## Proposal

Add a single per-user retention policy that drives **both** `posts` and `delivery_history`, preserving the standing invariant that history outlives no post by more than the post's own lifetime ([delivery scheduler spec](../../../specs/backend/delivery-scheduler.md)).

**Data model — `users.retention_days INT NULL`** (migration `000010_user_retention`), one column by the same reasoning that put `vip` on `users` ([VIP flag MRFC](../implemented/2026-08-23-user-vip-flag.md)):

| Value  | Meaning                                                                      |
| ------ | ---------------------------------------------------------------------------- |
| NULL   | inherit — each table's global config applies (today = 7 days)                |
| 0      | keep forever (reuses `[post] retention_days`'s existing 0 = never encoding)  |
| 1–3650 | keep N days                                                                  |

Effective cutoffs are per row: posts use `created_at < now() − COALESCE(user override, global)`; delivery_history the same but via `LEFT JOIN`, since rows orphaned by `ON DELETE SET NULL` fall back to the global window — an anonymous row can carry no personal policy. Global defaults stay in config.toml: changing them is deploy-weight, not an operations act.

**Prune commands keep their shape, change their predicate.** Both CLIs keep `--dry-run`/`--batch-size`, the batched subquery-LIMIT loops ([delivery queue MRFC](../implemented/2026-07-10-persistent-best-effort-delivery-queue.md)), and the render-cache drop by QID; the single cutoff becomes a per-row `CASE` (`retention_days = 0` → never eligible; the NULL comparison excludes the row). A daily cron job deployed by Ansible invokes both commands, closing the scheduling debt — without it the policy is theater on fresh deployments.

**VIP class default, materialized at grant time.** A new runtime settings key `vip_retention_days` (value shape `{"days": …}`, generalizing the settings table beyond `{"enabled": …}` from the [grant-strategy MRFC](../implemented/2026-08-23-github-login-vip-grant-strategy.md)) holds the class default. Whenever a user is granted VIP — the manual `PATCH /admin/users/:id/vip` or the GitHub-login auto-grant strategy — and their retention is still inherit, the class default is written onto the user in the same act. Revoking VIP keeps the materialized value: an honorific demotion must never expose two years of data to the 7-day sweep. `scope:"vip"` bulk writes remain for one-shot realignment of the existing VIP population.

**Admin API**, mirroring the `/vip` endpoint end to end ([VIP badge MRFC](../implemented/2026-08-23-vip-badge-and-admin-management.md)):

- `PATCH /api/v1/admin/users/:id/retention` — `{retention_days: null | 0 | 1–3650}`; `null` clears back to inherit.
- `POST /api/v1/admin/users/retention/bulk` — `{user_ids: […]}` (≤ 200) or `{scope: "vip"}`, plus the value; returns `{updated}`; one audit entry (`user.set_retention_bulk`) with scope and count rather than N rows.
- `POST /api/v1/admin/retention/impact` — same target shape plus a candidate value; returns `{users_affected, posts_to_delete, history_to_delete}`. Impact preview is a first-class endpoint because it feeds the destructive-confirm UI.
- Audit `user.set_retention` (single, old→new narrative) on the PATCH.

**Admin UX** (four surfaces, one shared dialog):

- The users list gains a Retention column showing the *effective* value — Forever (badge) / N days / Default · 7 days (resolved, so a config change moves the display) — plus a toolbar bulk-select toggle revealing a checkbox column, header select-all, and a floating action bar (N selected · Set retention · Exit). The row ⋮ menu gains Retention… as the single-user path.
- The shared dialog: a three-segment picker (Inherit default / Keep forever / Keep N days) with the third segment expanding to preset chips (7/30/90/365) and a free input validating 1–3650.
- Shortening a window is a destructive flow: the dialog fetches impact and, when the deletion count > 0, shows the counts and demands confirmation (type-to-confirm `DELETE` at ≥ 1000 posts), extending the UserGovernance pattern.
- The users-page header grows a VIP policy bar around the existing grant-strategy toggle: a class-default picker (Follow global / Forever / N days) and an Apply to all VIP users (N) button opening the same dialog in vip-align mode with aggregate impact. User detail gets a Profile row with the current value and a Set… action.

User-facing visibility of one's own retention (a badge on /posts) is deliberately out of scope; it is a follow-up if operations asks for it.

## Alternatives considered

**Cover only posts, or only delivery_history.** Posts-only strands VIP users with a 7-day notification record next to forever content; history-only preserves a record whose subject is deleted after 7 days. Either breaks the same-lifetime invariant; the doubled table surface is the price of a coherent promise, and both prune loops already share a shape.

**A live VIP class default in the resolution chain** (user explicit > class default > global). It covers future VIPs without touching grant paths, but revoking VIP would instantly re-expose old data to the global sweep — a governance side effect — and every reader (prune SQL, admin display) would eat a derivation chain. Materializing at grant keeps the stored value explicit and the demotion harmless; the class promise still self-maintains because both grant paths funnel through the same hook.

**One-shot bulk only** (no stored class default). Simplest surface, but VIPs arrive automatically via the GitHub strategy with nobody at the keyboard; each new VIP silently stays at 7 days and the promise decays invisibly — exactly the silent degradation the project refuses.

**A per-user policy table or settings composite keys** (`retention:user:123`). One dimension, one writer: the VIP MRFC already rejected a membership table for a single dimension, and the keyed settings store is global by design.

**Moving the global defaults into runtime settings.** Runtime-changeable global retention is deploy-level power wearing an ops hat; config.toml matches the weight of the act. The settings table still grows the `vip_retention_days` key — a class promise operations re-decides per cohort, which is runtime-weight.

## Acceptance criteria

- `000010_user_retention` up/down applies cleanly; with every user NULL, both prune commands delete exactly what they delete today.
- testcontainers coverage: explicit forever, explicit N, inherit, global `retention_days = 0`, and history rows orphaned by user deletion (SET NULL) falling back to the global window.
- Grant-time materialization fires on both grant paths and is a no-op when the user already carries a value; revocation leaves the value untouched; a third grant path cannot land without meeting the hook (test-level enforcement).
- Bulk endpoint: ids and `scope:"vip"` both write per-user values, respect the 200 cap, and produce one audit entry; impact endpoint returns correct counts for single, multi, and vip scopes.
- The Ansible playbook deploys the daily cron job on a scratch host and both prune runs are observable in logs; failures surface, not silence.
- Admin flows (single, multi via bulk-select, VIP align, shorten-with-impact) verified with Playwright against the dev stack; shorten confirm blocks when deletion impact > 0; i18n keys present in all four locale files.
- Every UI surface in this design carries screenshot evidence in the delivery PR: retention column and bulk-select mode, the shared dialog (all three segments plus the day input), the shorten-with-impact confirm, and the VIP policy bar.

## Risks

- Shortening retention is irreversible deletion at the next sweep; the UI gates it with impact counts and confirmation, but the API cannot — an admin with a token bypasses the dialog (audited; admin trust boundary).
- Data already swept before a policy is set cannot be resurrected; "keep forever" saves only what still exists.
- The settings value shape generalizes from `{"enabled"}` to include `{"days"}` — the settings API contract must absorb this backward-compatibly (the existing `vip` key stays untouched).
- Materialization couples retention to the grant paths; the hook must survive refactors of the VIP grant strategy, and the coupling is recorded here and at the hook site.
- The per-row `CASE` cutoff makes prune SQL heavier than a single timestamp compare; at ~0.12 writes/second this is noise, and batching bounds lock scope regardless.
