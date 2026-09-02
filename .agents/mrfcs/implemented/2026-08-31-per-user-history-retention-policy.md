# MRFC: Per-user history retention policy

Status: implemented

English | [中文](2026-08-31-per-user-history-retention-policy.zh.md)

## Problem

Retention in markpost is one-size-fits-all: `[post] retention_days` (default 7, 0 = never) and `[delivery] history_retention` (default 168h) are global config keys, swept by two cron-invoked CLI commands (`prune-expired-posts`, `prune-delivery-history`). Operations needs per-user promises — "this user's history is kept forever", "these users keep 30 days" — typically tied to the VIP honorific, and nothing user-scoped existed anywhere in the schema, the API, or the admin UI. Two adjacent debts forced themselves into the same decision: the repo schedules no cron for the prune commands at all (deployments that skip the manual step simply never delete), and shortening a retention window is irreversible deletion at the next sweep, so admin ergonomics here are a correctness concern, not polish.

## Decision

A single per-user retention policy drives **both** `posts` and `delivery_history`, preserving the standing invariant that history outlives no post by more than the post's own lifetime ([delivery scheduler spec](../../../specs/backend/delivery-scheduler.md)).

**Data model — `users.retention_days INT NULL`** (migration `000010_user_retention`), one column by the same reasoning that put `vip` on `users` ([VIP flag MRFC](./2026-08-23-user-vip-flag.md)):

| Value  | Meaning                                                                      |
| ------ | ---------------------------------------------------------------------------- |
| NULL   | inherit — each table's global config applies (today = 7 days)                |
| 0      | keep forever (reuses `[post] retention_days`'s existing 0 = never encoding)  |
| 1–3650 | keep N days                                                                  |

Effective cutoffs are per row: posts use `created_at < now() − COALESCE(user override, global)`; delivery_history the same but via `LEFT JOIN`, since rows orphaned by `ON DELETE SET NULL` fall back to the global window — an anonymous row can carry no personal policy. Global defaults stay in config.toml: changing them is deploy-weight, not an operations act.

**Prune commands keep their shape, carry a per-row predicate.** Both CLIs keep `--dry-run`/`--batch-size`, the batched subquery-LIMIT loops ([delivery queue MRFC](./2026-07-10-persistent-best-effort-delivery-queue.md)), and the render-cache drop by QID; the single cutoff became a per-row `CASE` (`retention_days = 0` → never eligible; the NULL comparison excludes the row). A daily cron job (`markpost-retention-prune`, `devops/ansible/files/prune-retention.sh`) invokes both commands from the Ansible-managed host, closing the scheduling debt. Dry-run output counts what the effective policies would delete.

**VIP class default, materialized at grant time.** The runtime settings key `vip_retention_days` (value shape `{"days": …}`, generalizing the settings table beyond `{"enabled": …}` from the [grant-strategy MRFC](./2026-08-23-github-login-vip-grant-strategy.md)) holds the class default. Whenever a user is granted VIP — the manual `PATCH /admin/users/:id/vip` or the GitHub-login auto-grant strategy — and their retention is still inherit, the class default is written onto the user in the same statement (`UPDATE … retention_days = COALESCE(retention_days, $default)`). Revoking VIP keeps the materialized value: an honorific demotion never re-exposes old data to the global sweep. `scope:"vip"` bulk writes realign the existing VIP population on demand.

**Admin API**, mirroring the `/vip` endpoint end to end ([VIP badge MRFC](./2026-08-23-vip-badge-and-admin-management.md)):

- `PATCH /api/v1/admin/users/:id/retention` — `{retention_days: null | 0 | 1–3650}`; `null` clears back to inherit; audited as `user.set_retention` (old→new).
- `POST /api/v1/admin/users/retention/bulk` — `{user_ids: […]}` (≤ 200) or `{scope: "vip"}`, plus the value; returns `{updated}`; one audit entry (`user.set_retention_bulk`) with scope and count.
- `POST /api/v1/admin/retention/impact` — same target shape plus a candidate value; returns `{users_affected, posts_to_delete, history_to_delete}` — the destructive-confirm dialog's data source. A candidate of null resolves against each table's global fallback; 0 (forever) matches nothing.
- `GET /api/v1/admin/retention/defaults` — the global fallback windows, so the UI renders the effective value of inherit policies.

**Admin UX** (four surfaces, one shared dialog):

- The users list carries a Retention column showing the *effective* value — Forever (badge) / N days / Default · 7 days (resolved via the defaults endpoint) — plus a toolbar bulk-select toggle revealing a checkbox column, header select-all, and a floating action bar (N selected · Set retention · Exit). The row ⋮ menu gains Retention… as the single-user path.
- The shared dialog: a three-segment picker (Inherit default / Keep forever / Keep N days) with the third segment expanding to preset chips (7/30/90/365) and a free input validating 1–3650.
- Shortening a window is a destructive flow: the dialog fetches the impact preview and, when the deletion count > 0, shows the counts and demands confirmation (type-to-confirm `DELETE` at ≥ 1000 posts), extending the UserGovernance pattern.
- The users-page header carries a VIP policy bar around the grant-strategy toggle: a class-default picker (Follow global / Forever / N days) and an Apply to all VIP users (N) button opening the same dialog in vip-align mode. User detail gets a Profile row with the current value and a Set… action.

User-facing visibility of one's own retention (a badge on /posts) is deliberately out of scope; it is a follow-up if operations asks for it.

## Alternatives considered

**Cover only posts, or only delivery_history.** Posts-only strands VIP users with a 7-day notification record next to forever content; history-only preserves a record whose subject is deleted after 7 days. Either breaks the same-lifetime invariant; the doubled table surface is the price of a coherent promise, and both prune loops already share a shape.

**A live VIP class default in the resolution chain** (user explicit > class default > global). It covers future VIPs without touching grant paths, but revoking VIP would instantly re-expose old data to the global sweep — a governance side effect — and every reader (prune SQL, admin display) would eat a derivation chain. Materializing at grant keeps the stored value explicit and the demotion harmless; the class promise still self-maintains because both grant paths funnel through the same hook.

**One-shot bulk only** (no stored class default). Simplest surface, but VIPs arrive automatically via the GitHub strategy with nobody at the keyboard; each new VIP silently stays at 7 days and the promise decays invisibly — exactly the silent degradation the project refuses.

**A per-user policy table or settings composite keys** (`retention:user:123`). One dimension, one writer: the VIP MRFC already rejected a membership table for a single dimension, and the keyed settings store is global by design.

**Moving the global defaults into runtime settings.** Runtime-changeable global retention is deploy-level power wearing an ops hat; config.toml matches the weight of the act. The settings table still grows the `vip_retention_days` key — a class promise operations re-decides per cohort, which is runtime-weight.

## Consequences

What the trade-off bought: operations can promise retention per user, per cohort, or per VIP class with a self-maintaining grant hook, a visible effective value in the admin list, and a deletion-count gate on every shortening; the prune jobs are actually scheduled, so the policy is real on fresh deployments. Verification: testcontainers cover explicit forever, explicit N, shorten-at-next-sweep, inherit, global 0, and orphaned history rows falling back to the global window; grant-time materialization is asserted on both grant paths (an inherit user takes the class default, an explicit value survives grant and revoke, revocation keeps it); the bulk/impact/defaults endpoints are covered at handler level; the admin surfaces are verified with Playwright screenshots — [`01` retention column](./2026-08-31-per-user-history-retention-policy/01-users-retention-column.png), [`02` bulk-select mode](./2026-08-31-per-user-history-retention-policy/02-bulk-select-mode.png), [`03` the shared dialog](./2026-08-31-per-user-history-retention-policy/03-retention-dialog.png), [`04` its day input](./2026-08-31-per-user-history-retention-policy/04-dialog-days-expanded.png), [`05` shorten-with-impact confirm](./2026-08-31-per-user-history-retention-policy/05-shorten-impact-confirm.png), [`06` the VIP policy bar](./2026-08-31-per-user-history-retention-policy/06-vip-policy-bar.png), [`07` vip-align](./2026-08-31-per-user-history-retention-policy/07-vip-apply-all-dialog.png), [`08` the detail row](./2026-08-31-per-user-history-retention-policy/08-user-detail-row.png); i18n keys ship in all four locale files.

What it cost and still risks: shortening retention is irreversible deletion at the next sweep — the UI gates it with impact counts and confirmation, but an admin with a token bypasses the dialog (audited; admin trust boundary); data already swept before a policy is set cannot be resurrected; the settings value shape generalized from `{"enabled"}` to include `{"days"}` (wrong-shape writes are rejected 400 in both directions); materialization couples retention to the grant paths — a third grant path must meet the same hook (enforced today by the service-level seam both paths call); and the per-row `CASE` cutoff makes prune SQL heavier than a single timestamp compare — at ~0.12 writes/second this is noise, and batching bounds lock scope regardless.
