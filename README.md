English | [简体中文](README_zh.md)

<div align="center">

# Markpost

**A lightweight Markdown-to-HTML publishing service.** Upload Markdown via API, get a rendered HTML page back. Simple, self-hosted, and fast.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

</div>

---

## Features

- ✍️ **Markdown Publishing** — Upload via a single `POST` request, get back a rendered HTML page with a unique URL
- 🌐 **Web Dashboard** — Manage posts, view analytics, and configure delivery channels
- 📬 **Delivery Channels** — Forward posts to webhooks (Feishu, Slack, custom) with keyword filtering
- 🏠 **Self-Hosted** — Single Docker container, runs anywhere with SQLite or PostgreSQL

## Quick Start

For detailed deployment, see the [Deployment Guide](docs/deployment.md).

After deployment, open `http://<your-server-ip-or-domain-name>:7157`, then log in with the default credentials:

- **Username:** `markpost`
- **Password:** `<your-secret-admin-password>`, default is `markpost`

> ⚠️ Change the default password immediately after first login.

After logging in, your **Post Key** is displayed on the dashboard homepage. You'll need it to create posts via the API (see below).

## API Reference

### Create Post

POST /:post-key

```json
{ "title": "My Post", "body": "# Hello World\nThis is **Markdown**." }
```

**Response** `201 Created`

```json
{ "id": "p-abc123" }
```

Your post key (prefixed with `mpk-`) is generated on first login. You can find it on the dashboard homepage.

### View Post

**Rendered HTML:**

GET /:qid

**Raw Markdown:**

GET /:qid?format=raw

## Development

See the [Development Guide](docs/development.md).

## License

[MIT](LICENSE)
