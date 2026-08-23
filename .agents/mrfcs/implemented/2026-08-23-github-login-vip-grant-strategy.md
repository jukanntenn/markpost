# MRFC: GitHub-login VIP grant strategy

Status: implemented

English | [中文](2026-08-23-github-login-vip-grant-strategy.zh.md)

## Problem

The strategy "users who log in via GitHub become VIP" must be turn-off-able by an admin as an operational act — immediate, no deploy — and its off state must stop granting without revoking what was already granted. Two things were undecided because nothing like them existed: the repository had no runtime settings mechanism at all (config was Viper/TOML loaded once at startup; admin endpoints were read-only metrics or per-user governance), and the grant itself needed exact semantics — when it fires, what it does to returning users, and what logins do once the strategy is closed. The requester framed this as the *first* operations strategy, so more were expected to follow the same path.

## Decision

The switch lives in a keyed `settings` table (`key TEXT PRIMARY KEY, value JSONB NOT NULL, updated_by, updated_at`), created and seeded by migration `000009_settings` with `vip = {"enabled": true}` so the strategy launches on; future strategies land in the same home. The domain package is `backend/internal/domain/settings` (`SettingValue` carries the boolean via `driver.Valuer`/`sql.Scanner`), the repository is `backend/internal/infra/settings_repo.go` (upsert through `ON CONFLICT (key)`), and its `VIPStrategyEnabled` read doubles as the auth service's port — logins read it directly, uncached.

Admins drive it through `GET /api/v1/admin/settings` and `PUT /api/v1/admin/settings/:key` behind `RequireAdmin` (v1 admits only the seeded `vip` key; anything else is a 400 `unknown_setting`), each write audited as `setting.set` with key and value. The grant fires in `LoginWithGitHub` (`backend/internal/service/auth/auth.go`) after `GetOrCreateFromGitHub`: while enabled, any GitHub login — new or returning — idempotently sets the user's vip true (the write is skipped when vip is already true); while disabled, logins leave vip untouched in both directions. A settings-read failure fails toward not-granting: the login completes, nothing is granted, and the error is logged. Manual admin writes remain the only revocation path, and while the strategy is enabled a revoked user's next GitHub login re-grants — the recorded trade-off; close first, then curate.

## Alternatives considered

**A config.toml/Viper flag.** No new table or endpoints — but toggling becomes a release plus deploy, an ops lever for a growth experiment turns into an engineering action, and the value drifts between environments instead of living in one audited place.

**A single-row typed settings table (one boolean column).** Simplest typed read, and every future setting is another migration; the requester said "first strategy", so the keyed shape amortizes immediately.

**Grant only at account creation (first login).** Avoids the revoke-then-regrant fight — but it abandons the existing GitHub-linked users who were the whole point of "first-batch users become VIP by logging in", and matches no reading of "any user who logs in via GitHub".

**Revoke on login while the strategy is off.** Directly violates "already-granted vip is never taken back"; once off, logins must be side-effect-free on vip.

## Consequences

The strategy is an operational act: one audited PUT flips it, effective on the very next login, no deploy; the keyed JSONB table amortizes for the strategies the requester said would follow. Costs accepted: a generic settings table invites misuse as a config dumping ground, held in check by the unknown-key 400 (nothing lands without a migration seeding it) and the MRFC/Ask-first path for schema and behavior; the uncached per-login read adds one query to a path already running several; and the revoke-regrant fight under an enabled strategy is deliberate — a superseding MRFC adds an opt-out bit if practice demands it, rather than quietly patching this one. Verified by: migration up/down cycling, repository tests (seed read-back, upsert, unknown-key `ErrNotFound`), grant-hook tests covering enabled/disabled/read-failure/unwired, and handler tests for list/upsert/unknown-key through the real admin chain.
