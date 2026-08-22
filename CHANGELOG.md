# Changelog

## [0.2.0-rc.5] - 2026-08-22

### Added

- Published post pages carry a strict content security policy as a second line of defense against malicious content

### Changed

- Repeat visits to pages, and lookups of missing posts, are now absorbed by the CDN edge instead of re-hitting the server, keeping things fast under heavy traffic

### Fixed

- Opening the app in multiple tabs no longer triggers surprise logouts when the session renews in one of the tabs

## [0.2.0-rc.4] - 2026-08-16

### Added

- A marketing landing page greets visitors at the root URL
- An admin dashboard with delivery trends, channel health, and pipeline status
- Audit logs can be filtered by actor, action, target, and time range
- Users can review and revoke their own login sessions from settings
- Delivery failures are classified by cause, and admins can filter history by error category
- A password strength meter helps you choose stronger passwords
- Password changes and session revokes take effect immediately everywhere

### Changed

- The post key page shows the full URL with a copy options menu
- Refined typography scale, brand icon, and dashboard layouts

### Fixed

- External links in rendered posts no longer leak the referring page
- Feishu card previews no longer lose inline images

## [0.2.0-rc.3] - 2026-08-08

### Added

- Admin console with a dashboard of system-wide counts and stats
- Admins can manage users end to end: create, change role, reset password, activate/deactivate, and delete
- Admins can create, enable, disable, and delete delivery channels from the web UI
- Admins can review active login sessions and revoke them in bulk
- An audit log records privileged admin actions for later review
- Pagination controls on all admin list pages
- The first admin account is created automatically on first startup, so a fresh deployment is ready to sign in to
- Email is now optional when creating a user
- Confirmation dialogs guard destructive actions (deleting posts, users, and channels)
- Post publishing and delivery dispatch now emit metrics and structured logs for monitoring

### Changed

- **Breaking:** PostgreSQL is now the only supported database — SQLite and MySQL have been removed. Schema changes apply through versioned migrations via `markpost migrate up`.
- **Breaking:** the config file must now be named `config.toml`; the legacy `markpost.toml` fallback was removed
- Timestamps are consistent everywhere now that the database connection and application clock share a configurable, pinned timezone
- Dates display using the correct regional format for the selected language
- The app shell shows a loading skeleton instead of a blank screen while starting up
- Logs persist to `/app/data/logs` and Docker log files are size-capped so they cannot fill the disk

### Fixed

- Emphasis such as `*bold*` and `_italic_` now closes correctly next to CJK fullwidth punctuation
- Copying a post link now works over plain HTTP via a fallback, and reports failure instead of silently doing nothing
- Posts sharing the same timestamp no longer shuffle between page loads
- Admin dashboard stats report real counts instead of placeholders
- Delivery links open in a new tab, and the cancel button in confirmation dialogs works

## [0.2.0-rc.2] - 2026-07-26

### Changed

- Docker image now defaults to plain HTTP on port 2053; self-hosted deployments work out of the box without built-in TLS

## [0.2.0-rc.1] - 2026-07-25

### Added

- Delivery channels with keyword filter expressions (AND/OR/NOT), send-test API, and per-channel history UI
- GFM markdown support (tables, strikethrough, autolinks) and hard wrap rendering
- CLI commands for fake data generation, load-test seeding, and expired post pruning
- Docker Compose quick start with PostgreSQL and `.env.example` for one-command deployment

### Fixed

- Sanitized rendered post HTML to block stored XSS attacks

## [0.1.0] - 2026-06-08

- ✍️ **Markdown Publishing** — Upload via a single `POST` request, get back a rendered HTML page with a unique URL
- 🌐 **Web Dashboard** — Manage posts, view analytics, and configure delivery channels
- 📬 **Delivery Channels** — Forward posts to webhooks (Feishu, Slack, custom) with keyword filtering
- 🏠 **Self-Hosted** — Single Docker container, runs anywhere with SQLite or PostgreSQL
