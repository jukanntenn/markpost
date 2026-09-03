# MRFC: Refresh token rotation grace window

Status: proposed

English | [中文](2026-09-03-refresh-token-rotation-grace-window.zh.md)

## Problem

A tab that refreshes its token pair can die between the backend's rotation and the localStorage persist, and the frontend mutex cannot close that gap. Rotation happens server-side (`RefreshToken`, `backend/internal/service/auth/auth.go`): the presented token is soft-revoked, a successor pair is issued, and only afterwards does the client persist the successor (`setTokens` → localStorage, `frontend/src/lib/api/base.ts`). If the tab crashes in that gap, localStorage still holds the already-revoked token — and because localStorage is shared, every sibling tab holds the same stale copy. The next replay, whether a sibling's 401-driven refresh or the crashed tab's reload, hits the reuse-detection path (auth.md §2.3): a resubmitted revoked token is judged theft, and `RevokeAllByUserID` revokes every refresh token of the user. One client-side crash therefore logs the user out on all devices. The Web Locks mutex (`frontend/src/lib/auth/refresh-lock.ts`) serializes refreshes across live tabs but cannot help: the lock dies with the tab, and the server-side rotation has already happened.

The reuse-detection reaction is disproportionate to this failure mode. A thief replaying a consumed token gains nothing either way — the token is dead, and the response reveals no successor (tokens are stored only as SHA-256 hashes). It is the user, not the attacker, whom the current semantics punish. The decision the issue (#38) forces: whether to introduce a bounded rotation grace window that treats replay of a freshly rotated token as a race rather than theft — and, if adopting one, fix its length and the revocation semantics inside the window.

## Proposal

Adopt a **reject-without-revocation** grace window.

**Semantics.** When a token whose revocation is younger than the window is replayed, the request is rejected with today's error (401 `ErrInvalidToken`) but the family is left intact — no `RevokeAllByUserID`. Replays of tokens revoked before the window, or of legacy rows with no revocation timestamp, keep today's behavior: user-wide revocation plus the reuse-detected rejection. Every replay — inside or outside the window — is still detected and rejected, so RFC 9700 §4.14.2's detection requirement stays satisfied; only the revocation *reaction* is deferred inside the window. The in-window replay is logged server-side with a line distinct from the theft warning. The issue's own quantification — "a thief replaying inside the window succeeds at most once" — describes re-issue semantics (see alternatives); reject-without-revocation improves it to zero: an in-window replay mints nothing.

**Window: 30 seconds**, a named constant in the auth service (like `oauthStateTTL`), not a config knob. Under reject-without-revocation the marginal theft value of a wider window is zero — an in-window replay yields nothing for an attacker to keep alive — while coverage of crash→reload gaps grows. 30 s matches the WorkOS default and the upper end of the issue's 10–30 s range.

**Schema.** `refresh_tokens` gains a nullable `revoked_at TIMESTAMPTZ` column (a versioned SQL migration, next sequential number), stamped by every soft-revocation write (`RevokeRefreshToken`, `RevokeAllByUserID`, `RevokeRefreshTokenByID`) alongside `revoked = true`; the GORM struct tag change pairs with the migration per `backend/AGENTS.md`. NULL means "revoked before the column existed" and takes the strict path — the conservative direction. The reuse branch reads the revoked row (`GetRevokedRefreshToken`, which already returns it) and branches on `revoked_at`. This migration is an Ask-first item and is named in the PR body.

**No frontend change.** The client's refresh-failure handling (logout → re-login) is unchanged; the crashed device re-authenticates while other tabs and devices keep their sessions — the harm this decision removes.

Docs: `specs/auth.md` §2 gains the grace-window semantics, with its bilingual twin updated in the same change.

## Alternatives considered

**Keep strict revocation (status quo).** The residual window is real, and the punishment lands on the user: any tab crash inside the rotation gap revokes every session of the user on every device, forcing a full re-login. Since a thief replaying a consumed token gains nothing either way, the strictness adds no attacker-facing value — it only converts a client crash into a self-inflicted denial of service.

**Return the same rotated tokens (Supabase's reuse interval, WorkOS's grace period).** The mainstream answer — replay inside the window is idempotent and returns the successor tokens — requires the server to recover the successor's plaintext. markpost stores only SHA-256 hashes (`TokenHash`); returning the same tokens would mean storing recoverable refresh tokens, weakening the storage design for a rare crash case. Lost on this constraint, not on merit.

**Re-issue a fresh pair inside the window (Auth0's rotation overlap period, `leeway`).** Feasible with hash-only storage, but it hands any in-window replaying thief a fresh, fully working pair — turning the window into a bounded period in which theft succeeds — and it re-opens the repeat-replay loophole Supabase's design exhibits (replay every < interval to mint tokens indefinitely, supabase/auth#1901). It also forks the rotation chain: re-issuing from the replayed token must either orphan or revoke the live successor, reintroducing exactly the multi-tab race the Web Locks mutex already solves.

**Drive the window from an in-memory "recently revoked" cache (ristretto) instead of a schema change.** Avoids the migration but builds a second source of truth beside the `revoked` column: entries vanish on restart, silently collapsing the window to the strict path (fail-safe, but inconsistent), and the revocation time — a fact about the row — lives outside the row. `revoked_at` on `refresh_tokens` is the one fact's one home; the migration is the named Ask-first cost.

**Expose the window as configuration.** No operator need to tune it has been shown, and under reject-without-revocation the value is not performance- or capacity-sensitive. A constant avoids a misconfiguration surface on a security-sensitive knob and mirrors `oauthStateTTL`.

## Acceptance criteria

- A versioned migration adds nullable `revoked_at TIMESTAMPTZ` to `refresh_tokens`; all three soft-revocation write paths stamp it; the GORM model change pairs with the migration.
- Replaying a token revoked within the last 30 s returns 401 `ErrInvalidToken` and leaves every other refresh token of the user untouched.
- Replaying a token revoked more than 30 s ago, or a legacy revoked row (`revoked_at IS NULL`), revokes all of the user's refresh tokens and returns the reuse-detected error — behavior identical to today.
- A testcontainers-backed test pins both branches and the 30 s boundary.
- The in-window replay logs a line distinct from the theft warning, carrying the token hash.
- `specs/auth.md` §2 documents the grace window (en + zh in the same change).

## Risks

- **Theft detection dulled inside the window**: if an attacker and the legitimate client race on the same stolen token within 30 s, the victim's replay no longer revokes the family and the attacker keeps the successor they already obtained. Bounded at the window; out-of-window replay keeps today's behavior. Accepted because the window can never mint the attacker a new credential under reject-without-revocation.
- **The residual window is narrowed, not eliminated**: a crash whose replay arrives after 30 s still triggers user-wide revocation. Wider windows trade detection latency for marginal recovery coverage; mainstream ships seconds-to-a-minute windows for the same reason (Auth0 defaults to disabled).
- **Legacy rows** (`revoked_at IS NULL`) take the strict path, which is conservative and needs no backfill.
