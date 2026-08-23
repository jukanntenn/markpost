# MRFC: VIP badge and admin management surface

Status: proposed

English | [中文](2026-08-23-vip-badge-and-admin-management.zh.md)

## Problem

With vip stored ([the flag MRFC](2026-08-23-user-vip-flag.md)) and granted by strategy ([the grant MRFC](2026-08-23-github-login-vip-grant-strategy.md)), the strategy still has no surface: a VIP user cannot see their own standing — and exclusive-status visibility is the entire product of a growth strategy — and admins have no per-user lever and no switch in the UI. The frontend renders the current user's username in the dashboard welcome and the app-shell user menu, and admin renders usernames in the users list and detail pages; the repo has its own Badge component and a governance-dialog pattern to copy. What the badge says, where it appears, and how admins drive both levers is what this layer decides.

## Proposal

**Badge.** Reuse the in-repo `Badge` (`frontend/src/components/ui/badge.tsx`, `variant="accent"`) with localized text — `VIP` in English and Japanese, `尊享 VIP` in both Chinese variants — placed immediately after the username in four spots: the dashboard welcome line (`DashboardPage`), the app-shell user menu label (`AppShell`), and the admin users list and user detail pages. Non-VIP users see nothing; the badge is purely honorific and grants no permissions today. Public post pages are out of scope: they do not render an author username, and inventing one would widen the strategy beyond what was asked.

**Per-user management.** `PATCH /api/v1/admin/users/:id/vip` with body `{"vip": <bool>}`, mirroring the `/active` endpoint end to end: handler in the admin REST layer (bind, `parseIDParam`, audit action `user.set_vip` with the value as metadata, response `AdminUserItem`), service `SetUserVIP` behind the `UserMutator` port, repository setter through `updateByID`. Two deliberate departures from `/active`: no self-targeting guard (an admin setting their own vip harms no invariant) and no `token_version` bump — vip rides no claim and carries no authority, and the auth middleware re-reads the row each request, so a toggle is visible immediately without invalidating anyone's session. The admin UI adds the action to the existing per-row governance menu (`UserGovernance`) with the same confirm-dialog and invalidation pattern.

**Strategy switch UI.** A single toggle on the admin users page header — "GitHub login auto-VIP" on/off — calling `PUT /admin/settings/github-vip-strategy`; no separate settings page in v1, because one strategy does not earn a navigation surface of its own.

**Localization and mocks.** All four locale files (`en`, `zh-Hans`, `zh-Hant`, `ja`) gain the badge text, governance strings, and toggle labels together; `audit-action-text` maps `user.set_vip` and `site_setting.set`; MSW handlers extend for the new endpoints.

## Alternatives considered

**A Badge component from @base-ui/react.** The installed @base-ui/react has no Badge; the repo's own Badge is the established component and already varies by `variant`.

**A dedicated vip service or nested resource endpoints.** More "RESTful" surface for one boolean; the single PATCH copies a proven pattern that reviewers and tests already know.

**Bump `token_version` on vip change.** Treats an honorific like a security state; it would force re-login for every managed user with zero authority actually changing.

**A dedicated admin settings page hosting the toggle.** Anticipates a settings surface with one item; the users page header keeps the lever where the operator already works, and a page can be extracted when settings actually accumulate.

**Badge on public post pages.** No author username renders there today; adding one to display a badge invents a feature the strategy never asked for.

## Acceptance criteria

A VIP user sees the badge beside their own username in the welcome line and user menu; a non-VIP user sees nothing extra. The admin users list and detail show each user's vip, and the row menu offers grant/revoke with confirmation, toast, and list invalidation. The header toggle flips `github-vip-strategy` and the next GitHub login behaves per the grant MRFC's semantics. Both new audit actions render with localized text in the audit view. All four locale files carry every new key, and MSW covers the new endpoints.

## Risks

Four-locale synchronization is manual and drift-prone; the implementation layer lists every added key so a reviewer can diff the four files at a glance. The badge invites scope creep toward meaning something — if vip ever grants permissions, that change arrives with its own MRFC, and until then the copy stays honorific. Hosting the strategy toggle on the users page couples it visually to user management; acceptable while it is the only strategy, revisit when a second lands in `site_settings`.
