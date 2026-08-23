# MRFC: GitHub-login VIP grant strategy

Status: proposed

English | [中文](2026-08-23-github-login-vip-grant-strategy.zh.md)

## Problem

The strategy "users who log in via GitHub become VIP" must be turn-off-able by an admin as an operational act — immediate, no deploy — and its off state must stop granting without revoking what was already granted. Two things are undecided because nothing like them exists: the repository has no runtime settings mechanism at all (config is Viper/TOML loaded once at startup, `backend/internal/config/config.go`; admin endpoints today are read-only metrics or per-user governance), and the grant itself needs exact semantics — when it fires, what it does to returning users, and what logins do once the strategy is closed. The requester framed this as the *first* operations strategy, so more are expected to follow the same path.

## Proposal

**Switch storage — a keyed settings table.** A new `settings` table (`key TEXT PRIMARY KEY, value JSONB NOT NULL, updated_by INTEGER, updated_at TIMESTAMPTZ`), seeded by its migration with `vip = {"enabled": true}` so the strategy launches on. This is the shared home future strategies land in, bought once. Admin surface: `GET /api/v1/admin/settings` returns all rows, `PUT /api/v1/admin/settings/:key` writes one (v1 only `vip`, body `{"enabled": <bool>}`), both behind `RequireAdmin` with the audit action `setting.set` recording key and value. Reads go straight to the table per use — login is a low-frequency path that already runs several queries; no cache until measurement says otherwise.

**Grant semantics.** In `LoginWithGitHub` (`backend/internal/service/auth/auth.go`), after `GetOrCreateFromGitHub`: when the strategy is enabled, set the user's vip to true — idempotent, for newly created and returning users alike, so the existing GitHub user base joins the first batch by logging in during the window. When disabled, logins leave vip untouched in both directions: no grant, no revoke. Manual admin writes (the per-user PATCH endpoint of the VIP badge and management layer, stacked above this one) remain the only revocation path, and the [flag itself](../implemented/2026-08-23-user-vip-flag.md) is durable at grant time, so closing the strategy never un-vests the first batch. If the settings read fails mid-login, the login proceeds without a grant and the error is logged — fail toward not-granting, since a wrongly granted vip is harder to walk back than a missed one.

**A trade-off recorded in the open.** While the strategy is enabled, an admin's revocation of a user's vip is undone by that user's next GitHub login — the strategy re-asserts itself. An admin who wants a revoke to stick closes the strategy first, then curates. A per-user opt-out bit was considered and deliberately deferred: no requirement has asked for it, and it would add a second mutable fact to every user row.

## Alternatives considered

**A config.toml/Viper flag.** No new table or endpoints — but toggling becomes a release plus deploy, an ops lever for a growth experiment turns into an engineering action, and the value drifts between environments instead of living in one audited place.

**A single-row typed settings table (one boolean column).** Simplest typed read, and every future setting is another migration; the requester said "first strategy", so the keyed shape amortizes immediately.

**Grant only at account creation (first login).** Avoids the revoke-then-regrant fight — but it abandons the existing GitHub-linked users who were the whole point of "first-batch users become VIP by logging in", and matches no reading of "any user who logs in via GitHub".

**Revoke on login while the strategy is off.** Directly violates "already-granted vip is never taken back"; once off, logins must be side-effect-free on vip.

## Acceptance criteria

With the strategy enabled, a GitHub login — new user or returning `vip=false` user — ends with vip true; a password login never writes vip. With it disabled, neither direction of login changes vip. The admin toggle takes effect on the very next login without restart; toggles are audited under `setting.set`. A failed settings read during login still completes the login, grants nothing, and logs the error.

## Risks

A generic `settings` table invites misuse as a config dumping ground; scope it to operational strategies — schema and behavior changes still go through MRFCs and Ask-first. The uncached per-login read adds one query to a path that already runs several; negligible now, cache only with a measurement. The revoke-regrant fight under an enabled strategy is intentional and documented here; if practice shows admins need durable per-user revokes while the strategy runs, a superseding MRFC adds the opt-out bit rather than quietly patching this one.
