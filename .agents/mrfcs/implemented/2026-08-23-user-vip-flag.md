# MRFC: User VIP flag

Status: implemented

English | [中文](2026-08-23-user-vip-flag.zh.md)

## Problem

The VIP growth strategy (issue #10) needs a per-user, mutable vip state whose lifetime is independent of the strategy's own on/off switch: grants made while the strategy is enabled must survive the admin turning it off ("the first batch keeps it"), and admins must be able to override vip per user regardless of login activity. The `users` table today carries governance state (`role`, `is_active`) but nothing like a membership mark. Three data-shape questions forced a decision: what vip stores, where it lives, and which readers see it.

## Decision

`users` carries a boolean `vip` column (`BOOLEAN NOT NULL DEFAULT FALSE`) added by migration `000008_user_vip` (down: `DROP COLUMN`); existing rows landed on `false`. The domain model exposes it as `User.VIP` in `backend/internal/domain/user/user.go` — with an explicit `column:vip` GORM tag, because GORM's naming strategy derives `v_ip` from the initialism. All writes flow through `Repository.SetUserVIP` (`backend/internal/domain/user/repository.go`), implemented in `backend/internal/infra/user_repo.go` via the shared `updateByID` map helper. The value is durable at grant time and never re-derived from the login provider plus strategy state.

Readers see vip through the two user DTOs — `UserResponse` (login responses) and `AdminUserItem` (admin list/detail) in `backend/internal/api/rest/v1/types.go`, with swagger regenerated. It stays out of the JWT claims: the auth middleware reloads the full user row from the database on every request (`backend/internal/middleware/auth.go`), so every handler sees the current vip and an admin flipping vip never invalidates sessions.

## Alternatives considered

**Derive vip at read time (`github_id IS NOT NULL` and strategy on).** Zero schema change — and it breaks both requirements the strategy exists for: the moment the admin closes the strategy, the derived value flips and the first batch silently loses vip, and an admin's per-user override has nowhere to live. The point of the column is a mutable per-user fact independent of current strategy state.

**A tier enum instead of a boolean.** Anticipates VIP levels nobody has asked for and widens every consumer to interpret a level; a later tier need is a follow-up migration plus backfill, which is exactly when the requirement would first become real.

**A separate membership table (`user_memberships`).** Normalizes future multi-strategy state, but one boolean for one strategy does not earn a join on every user read; revisit if a second membership dimension actually appears.

**Carry vip in JWT claims.** Saves the per-request column read — but the middleware already reads the row, and claims would go stale on every admin toggle unless each vip change bumped `token_version` and forced re-login for an honorific.

## Consequences

The flag buys exactly what the strategy needs: state that survives the strategy's own closure, admin override independent of login activity, and immediate visibility to every handler without touching token lifecycle. It costs a frozen vocabulary — introducing tiers later means a migration with backfill, accepted as the price of not guessing at unknown tier semantics now — and the `ALTER TABLE … ADD COLUMN … NOT NULL DEFAULT` took a brief access-exclusive lock, a non-issue at markpost's user volume but worth revisiting for large-table add-columns. GORM's initialism quirk (`VIP` → `v_ip`) is pinned down by the explicit column tag; any future field named with consecutive capitals needs the same treatment. Verified by: `markpost migrate up`/`down`/`up` cycling cleanly, repository tests covering grant, revoke, and not-found (`TestUserRepository_SetUserVIP`), and both DTOs exposing `vip` in the regenerated swagger.
