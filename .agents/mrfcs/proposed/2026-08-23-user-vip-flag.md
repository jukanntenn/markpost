# MRFC: User VIP flag

Status: proposed

English | [中文](2026-08-23-user-vip-flag.zh.md)

## Problem

The VIP growth strategy (issue #10) needs a per-user, mutable vip state whose lifetime is independent of the strategy's own on/off switch: grants made while the strategy is enabled must survive the admin turning it off ("the first batch keeps it"), and admins must be able to override vip per user regardless of login activity. The `users` table today carries governance state (`role`, `is_active`) but nothing like a membership mark. Three data-shape questions force a decision: what vip stores, where it lives, and which readers see it.

## Proposal

Add a boolean column `users.vip` (`BOOLEAN NOT NULL DEFAULT FALSE`) through a versioned migration — `000008_user_vip`, following the `ALTER TABLE` pattern of `000005_token_version` — with `DROP COLUMN` as its down side; existing rows all land on `false`. The domain model gains `User.VIP bool` in `backend/internal/domain/user/user.go`, and all writes flow through a repository setter using the existing `updateByID` map pattern (`backend/internal/infra/user_repo.go`). The value is durable at grant time: vip is written when granted and never re-derived later from the login provider plus the strategy's current state.

For readers, `vip` joins the two user DTOs — `UserResponse` (login responses carry the current user) and `AdminUserItem` (admin list/detail) in `backend/internal/api/rest/v1/types.go`, with swagger regenerated. It deliberately stays out of the JWT claims: the auth middleware reloads the full user row from the database on every request (`backend/internal/middleware/auth.go`), so every handler already sees the current vip, and keeping it out of tokens means an admin flipping vip never needs to invalidate sessions.

## Alternatives considered

**Derive vip at read time (`github_id IS NOT NULL` and strategy on).** Zero schema change — and it breaks both requirements the strategy exists for: the moment the admin closes the strategy, the derived value flips and the first batch silently loses vip, and an admin's per-user override has nowhere to live. The point of the column is a mutable per-user fact independent of current strategy state.

**A tier enum instead of a boolean.** Anticipates VIP levels nobody has asked for and widens every consumer to interpret a level; a later tier need is a follow-up migration plus backfill, which is exactly when the requirement would first become real.

**A separate membership table (`user_memberships`).** Normalizes future multi-strategy state, but one boolean for one strategy does not earn a join on every user read; revisit if a second membership dimension actually appears.

**Carry vip in JWT claims.** Saves the per-request column read — but the middleware already reads the row, and claims would go stale on every admin toggle unless each vip change bumped `token_version` and forced re-login for an honorific.

## Acceptance criteria

`markpost migrate up` then `down` then `up` applies cleanly; the column exists with default `false` and existing rows untouched. `UserResponse` and `AdminUserItem` expose `vip` and swagger reflects it. Handlers read vip from the request-context user (database-loaded), never from token claims.

## Risks

A boolean freezes the vocabulary — introducing tiers later means a migration with backfill, accepted deliberately as the cost of not guessing at unknown tier semantics now. `ALTER TABLE … ADD COLUMN … NOT NULL DEFAULT` takes a brief access-exclusive lock; at markpost's user volume this is a non-issue, noted so a future large-table add-column revisits it deliberately.
