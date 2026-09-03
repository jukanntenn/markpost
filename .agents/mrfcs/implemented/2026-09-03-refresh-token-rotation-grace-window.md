# MRFC: Refresh token rotation grace window

Status: implemented

English | [中文](2026-09-03-refresh-token-rotation-grace-window.zh.md)

## Problem

A tab that refreshes its token pair can die between the backend's rotation and the localStorage persist, and the frontend mutex cannot close that gap. Rotation happens server-side (`RefreshToken`, `backend/internal/service/auth/auth.go`): the presented token is soft-revoked, a successor pair is issued, and only afterwards does the client persist the successor (`setTokens` → localStorage, `frontend/src/lib/api/base.ts`). If the tab crashes in that gap, localStorage still holds the already-revoked token — and because localStorage is shared, every sibling tab holds the same stale copy. The next replay, whether a sibling's 401-driven refresh or the crashed tab's reload, hits the reuse-detection path (auth.md §2.3): a resubmitted revoked token is judged theft, and `RevokeAllByUserID` revokes every refresh token of the user. One client-side crash therefore logs the user out on all devices. The Web Locks mutex (`frontend/src/lib/auth/refresh-lock.ts`) serializes refreshes across live tabs but cannot help: the lock dies with the tab, and the server-side rotation has already happened.

The reuse-detection reaction is disproportionate to this failure mode. A thief replaying a consumed token gains nothing either way — the token is dead, and the response reveals no successor (tokens are stored only as SHA-256 hashes). It is the user, not the attacker, whom the strict semantics punish. The decision the issue (#38) forces: whether to introduce a bounded rotation grace window that treats replay of a freshly rotated token as a race rather than theft — and, if adopting one, fix its length and the revocation semantics inside the window.

## Decision

markpost runs a **reject-without-revocation** grace window.

**Semantics.** When a token whose revocation is younger than the window is replayed, the request is rejected with 401 `ErrInvalidToken` but the family is left intact — no `RevokeAllByUserID`. Replays of tokens revoked before the window, or of legacy rows with no revocation timestamp, keep the strict behavior: user-wide revocation plus the reuse-detected rejection. Every replay — inside or outside the window — is still detected and rejected, so RFC 9700 §4.14.2's detection requirement stays satisfied; only the revocation _reaction_ is deferred inside the window, and an in-window replay mints nothing, so it yields an attacker no credential. The in-window replay is logged at info with the token hash, distinct from the `refresh token reuse detected` theft warning. The issue's own quantification — "a thief replaying inside the window succeeds at most once" — describes re-issue semantics (see alternatives); reject-without-revocation improves it to zero.

**Window: 30 seconds** — the `refreshGraceWindow` constant in the auth service (like `oauthStateTTL`), not a config knob. Under reject-without-revocation the marginal theft value of a wider window is zero — an in-window replay yields nothing for an attacker to keep alive — while coverage of crash→reload gaps grows. 30 s matches the WorkOS default and the upper end of the issue's 10–30 s range.

**Schema.** `refresh_tokens` carries a nullable `revoked_at TIMESTAMPTZ` column (migration `000011_refresh_token_revoked_at`), stamped by every soft-revocation write (`RevokeRefreshToken`, `RevokeAllByUserID`, `RevokeRefreshTokenByID` in `backend/internal/infra/token_repo.go`) alongside `revoked = true`; the GORM model (`backend/internal/domain/user/token.go`) carries the `RevokedAt *time.Time` field. NULL means "revoked before the column existed" and takes the strict path — the conservative direction. The reuse branch in `RefreshToken` reads the revoked row (`GetRevokedRefreshToken`, which already returns it) and branches on `withinGraceWindow`.

**No frontend change.** The client's refresh-failure handling (logout → re-login) is unchanged; the crashed device re-authenticates while other tabs and devices keep their sessions — the harm this decision removes.

Docs: `specs/auth.md` §2.2–2.5 document the columns and the window; `specs/backend/database-schema.md` carries the `revoked_at` row and the relaxed write-once wording; both bilingual twins updated in the same change.

## Alternatives considered

**Keep strict revocation (the pre-decision behavior).** The residual window is real, and the punishment lands on the user: any tab crash inside the rotation gap revokes every session of the user on every device, forcing a full re-login. Since a thief replaying a consumed token gains nothing either way, the strictness adds no attacker-facing value — it only converts a client crash into a self-inflicted denial of service.

**Return the same rotated tokens (Supabase's reuse interval, WorkOS's grace period).** The mainstream answer — replay inside the window is idempotent and returns the successor tokens — requires the server to recover the successor's plaintext. markpost stores only SHA-256 hashes (`TokenHash`); returning the same tokens would mean storing recoverable refresh tokens, weakening the storage design for a rare crash case. Lost on this constraint, not on merit.

**Re-issue a fresh pair inside the window (Auth0's rotation overlap period, `leeway`).** Feasible with hash-only storage, but it hands any in-window replaying thief a fresh, fully working pair — turning the window into a bounded period in which theft succeeds — and it re-opens the repeat-replay loophole Supabase's design exhibits (replay every < interval to mint tokens indefinitely, supabase/auth#1901). It also forks the rotation chain: re-issuing from the replayed token must either orphan or revoke the live successor, reintroducing exactly the multi-tab race the Web Locks mutex already solves.

**Drive the window from an in-memory "recently revoked" cache (ristretto) instead of a schema change.** Avoids the migration but builds a second source of truth beside the `revoked` column: entries vanish on restart, silently collapsing the window to the strict path (fail-safe, but inconsistent), and the revocation time — a fact about the row — lives outside the row. `revoked_at` on `refresh_tokens` is the one fact's one home; the migration is the named Ask-first cost.

**Expose the window as configuration.** No operator need to tune it has been shown, and under reject-without-revocation the value is not performance- or capacity-sensitive. A constant avoids a misconfiguration surface on a security-sensitive knob and mirrors `oauthStateTTL`.

## Consequences

The residual crash window (rotation done, successor not persisted) no longer escalates to user-wide revocation: the crashed device alone re-authenticates while sibling tabs and other devices keep their sessions. Theft detection outside the window is unchanged.

The trade-off costs:

- **Theft detection dulled inside the window**: if an attacker and the legitimate client race on the same stolen token within 30 s, the victim's replay no longer revokes the family and the attacker keeps the successor they already obtained. Bounded at the window; under reject-without-revocation the window can never mint the attacker a new credential.
- **The residual window is narrowed, not eliminated**: a crash whose replay arrives after 30 s still triggers user-wide revocation; wider windows trade detection latency for marginal recovery coverage — mainstream ships seconds-to-a-minute windows for the same reason (Auth0 defaults to disabled).
- **Legacy rows** (`revoked_at IS NULL`) take the strict path — conservative, and no backfill was needed.

Verification: testcontainers-backed tests pin the in-window branch (the replayed token is rejected while the legitimate successor still refreshes), the 29 s/31 s boundary pair around the 30 s window, and the NULL-timestamp legacy row (family revoked); repository tests pin that all three revocation writes stamp `revoked_at`. `go test ./...` is green, `-race` on the touched packages is green, and golangci-lint reports 0 issues.
