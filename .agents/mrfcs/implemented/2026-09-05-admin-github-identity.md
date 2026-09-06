# MRFC: GitHub identity in the admin user pages

Status: implemented

English | [中文](2026-09-05-admin-github-identity.zh.md)

## Problem

The user model has carried GitHub identity since OAuth landed — `github_id` and `avatar_url` are written at GitHub-signup time — but none of it reached the admin console's user surfaces as *identity*. The user list showed bare usernames (plus VIP); the detail page had a text-only "GitHub: Linked as #N" row. An admin could not tell at a glance which auth method an account uses, could not visually recognize a user, and `avatar_url` was stored but never exposed through the admin API at all.

## Decision

The admin user pages render avatar + auth-method marker beside usernames:

- `AdminUserItem` (backend DTO, `newAdminUserItem`) gains `avatar_url` — an additive wire-format field assembled from the existing model column; no schema change, no migration.
- Two small `ui/` components own the presentation: `UserAvatar` (the GitHub avatar, falling back to an initial circle for password-registered users) and `GithubBadge` (an icon-only lucide `GithubIcon` marker with a native "GitHub" tooltip, locale-invariant like the VIP badge).
- They appear in the admin user list (desktop table and mobile cards) beside each username, in the detail page header, and the detail "GitHub" profile row carries the icon next to "Linked as #N".

No external link to a GitHub profile is offered: profile URLs need the GitHub login, which the schema does not store, and `github_id` has no public profile URL.

## Alternatives considered

**Icon badge only, no avatar.** The minimal reading of the ask (a GitHub identity mark). It marks the auth method but discards the identity signal a GitHub user actually recognizes — their avatar — which the database already holds. The avatar is the field's reason for existing; showing it is the essence fix, not scope creep.

**Store the GitHub login and link out.** Would give a real profile link but requires a schema migration (an ask-first change) so admins can navigate away from the console to a third-party site — little governance value for the cost. Revisit if admins ever need to inspect the upstream GitHub account itself.

**Show avatars app-wide (settings, menus).** The ask was the admin console's user-related pages; broadening it would touch surfaces nobody asked about.

## Consequences

Admins see auth method and visual identity in both list and detail views; password users keep a stable layout through the initial-circle fallback. The avatar loads from `avatars.githubusercontent.com`, so admin browsers make one external image request per GitHub user rendered — acceptable for an admin page, and the fallback circle renders regardless of load outcome. The DTO field is additive: older clients ignore it, and the Swagger docs regenerate with `go generate`.
