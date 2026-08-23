# MRFC: VIP badge and admin management surface

Status: implemented

English | [中文](2026-08-23-vip-badge-and-admin-management.zh.md)

## Problem

With vip stored ([the flag MRFC](2026-08-23-user-vip-flag.md)) and granted by strategy ([the grant MRFC](2026-08-23-github-login-vip-grant-strategy.md)), the strategy still had no surface: a VIP user could not see their own standing — and that visibility is the entire product of a growth strategy — and admins had no per-user lever and no switch in the UI. The frontend renders the current user's username in the dashboard welcome and the app-shell user menu, and admin renders usernames in the users list and detail pages; the repo has its own Badge component and a governance-dialog pattern to copy. What the badge says, where it appears, and how admins drive both levers was this layer's decision.

## Decision

The badge is the in-repo `Badge` (`frontend/src/components/ui/badge.tsx`, `variant="accent"`) wrapped as `VipBadge` (`frontend/src/components/ui/vip-badge.tsx`) with the locale-invariant text `VIP`, rendered immediately after the username in four spots: the dashboard welcome line (`DashboardPage`), the app-shell user menu (`AppShell`), and the admin users list and user detail pages; non-VIP users see nothing extra. Public post pages stay out — no author username renders there.

Per-user management is `PATCH /api/v1/admin/users/:id/vip` with body `{"vip": <bool>}`, mirroring the `/active` endpoint end to end: handler in the admin REST layer with the audit action `user.set_vip` (value as metadata) and `AdminUserItem` response, service `SetUserVIP` behind the `UserMutator` port, repository setter through `updateByID`. Its two deliberate departures from `/active`: no self-targeting guard (an admin setting their own vip breaks no invariant) and no `token_version` bump (vip rides no claim; the per-request row reload makes the toggle visible immediately). The admin UI adds the action to the per-row governance menu (`UserGovernance`) with the same confirm-dialog and invalidation pattern, open to self as well.

The strategy switch is one toggle on the admin users page header (`VipStrategyToggle` in `AdminUsersPage`) calling `PUT /admin/settings/vip` — no separate settings page in v1. All four locale files carry the governance strings, toggle labels, and audit narratives together (`admin.users.vip*`, `admin.users.vipStrategy.*`, `admin.audit.action.user.vipGrant/vipRevoke`, `admin.audit.action.setting.*`); `audit-action-text` maps the two new actions; MSW handlers cover the new endpoints.

## Alternatives considered

**A Badge component from @base-ui/react.** The installed @base-ui/react has no Badge; the repo's own Badge is the established component and already varies by `variant`.

**A dedicated vip service or nested resource endpoints.** More "RESTful" surface for one boolean; the single PATCH copies a proven pattern that reviewers and tests already know.

**Bump `token_version` on vip change.** Treats an honorific like a security state; it would force re-login for every managed user with zero authority actually changing.

**A dedicated admin settings page hosting the toggle.** Anticipates a settings surface with one item; the users page header keeps the lever where the operator already works, and a page can be extracted when settings actually accumulate.

**Badge on public post pages.** No author username renders there today; adding one to display a badge invents a feature the strategy never asked for.

## Consequences

A VIP user sees the mark beside their own username in the welcome line and user menu, and admins see and drive each user's vip plus the strategy switch in one screen; both levers are audited with localized narratives. The costs: four-locale synchronization is manual (the implementation lists every added key so a reviewer can diff the four files at a glance), and the badge invites scope creep toward meaning something — if vip ever grants permissions, that change arrives with its own MRFC and until then the copy stays honorific. Hosting the strategy toggle on the users page couples it visually to user management; acceptable while it is the only strategy, to revisit when a second lands in `settings`. Verified by: handler tests for grant/revoke/404, the frontend suite covering badge rendering and the toggle, and swagger regenerated for the new endpoint.
