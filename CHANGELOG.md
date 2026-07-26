# Changelog

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
