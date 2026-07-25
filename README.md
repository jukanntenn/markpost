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
- 🏠 **Self-Hosted** — Single Docker Compose stack, runs anywhere with PostgreSQL

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/install/) (v2+)

### 1. Download the files

```bash
mkdir markpost && cd markpost
BASE=https://raw.githubusercontent.com/jukanntenn/markpost/main/docker
curl -o docker-compose.yml $BASE/docker-compose.yml
curl -o Caddyfile $BASE/Caddyfile
curl -o .env.example $BASE/.env.example
```

### 2. Configure

```bash
cp .env.example .env
```

Edit `.env` — at minimum, change the passwords and JWT signing keys:

```env
MARKPOST_ADMIN__INITIAL_PASSWORD=<your-password>
MARKPOST_JWT__ACCESS_SIGNING_KEY=<random-string>
MARKPOST_JWT__REFRESH_SIGNING_KEY=<random-string>
```

### 3. Start

```bash
docker compose up -d
```

### 4. Access

Open `http://<your-server-ip>:2053`, then log in with the credentials you set in `.env` (defaults: `markpost` / `markpost`).

> ⚠️ Change the default password immediately after first login.

Your **Post Key** is displayed on the dashboard homepage. You'll need it to create posts via the API (see below).

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

## Configuration

All configuration is done via environment variables. Set them in your `.env` file or pass them directly to the container.

| Variable | Description | Default |
|----------|-------------|---------|
| `MARKPOST_ADMIN__INITIAL_USERNAME` | Admin username (first boot only) | `markpost` |
| `MARKPOST_ADMIN__INITIAL_PASSWORD` | Admin password (first boot only) | `markpost` |
| `MARKPOST_JWT__ACCESS_SIGNING_KEY` | JWT access token signing key | `change-me` |
| `MARKPOST_JWT__REFRESH_SIGNING_KEY` | JWT refresh token signing key | `change-me` |
| `MARKPOST_SERVER__PUBLIC_URL` | Public URL of the service | *(empty)* |
| `MARKPOST_TIMEZONE` | Container timezone | `UTC` |
| `MARKPOST_POST__RETENTION_DAYS` | Days before posts expire | `7` |

For the full list, see [`.env.example`](docker/.env.example).

## License

[MIT](LICENSE)
