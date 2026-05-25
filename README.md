# Markpost

**A lightweight Markdown-to-HTML publishing service.** Upload Markdown via API, get a rendered HTML page back. Simple, self-hosted, and fast.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

English | [简体中文](README_zh.md)

---

## Features

- **API-First** — Upload Markdown with a single `POST` request, get back a unique URL
- **Web Dashboard** — Manage posts, view analytics, and configure delivery channels
- **Self-Hosted** — Runs anywhere: Docker, bare metal, or cloud
- **Multi-Database** — SQLite for simplicity, PostgreSQL for scale
- **Delivery Channels** — Forward posts to webhooks (Feishu, Slack, custom) with keyword filtering
- **OAuth Support** — Login with GitHub or username/password

## Quick Start

### Docker Compose (recommended)

Create a `docker-compose.yml`:

```yaml
services:
  frontend:
    image: jukanntenn/markpost-web:latest
    container_name: markpost-frontend
    ports:
      - "7330:3000"
    environment:
      - API_PROXY_TARGET=http://backend:7330
    depends_on:
      backend:
        condition: service_healthy
    restart: unless-stopped

  backend:
    image: jukanntenn/markpost:latest
    container_name: markpost-backend
    volumes:
      - ./data:/app/data
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://127.0.0.1:7330/api/v1/health"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s
    restart: unless-stopped
```

```bash
docker compose up -d
```

Open `http://localhost:7330` and log in with:
- **Username:** `markpost` (default)
- **Password:** `markpost` (change in production!)

Your **post key** is displayed on the dashboard after login.

### Backend Only (headless / API-only)

If you don't need the web dashboard, you can run the backend alone:

```bash
docker run -d \
  --name markpost \
  -p 7330:7330 \
  -v ./data:/app/data \
  --restart unless-stopped \
  jukanntenn/markpost:latest
```

This provides the API endpoints and post rendering (`GET /:id`) but no web UI.

## API Reference

### Create a Post

```bash
curl -X POST http://localhost:7330/YOUR_POST_KEY \
  -H "Content-Type: application/json" \
  -d '{"title": "My Post", "body": "# Hello World\nThis is **Markdown**."}'
```

Response:
```json
{ "id": "p-abc123" }
```

### View a Post

Navigate to `http://localhost:7330/p-abc123` — renders as a styled HTML page.

## Configuration

Markpost uses a TOML config file or environment variables.

| Variable | Description | Default |
|----------|-------------|---------|
| `MARKPOST_SERVER__HOST` | Bind address | `127.0.0.1` |
| `MARKPOST_SERVER__PORT` | HTTP port | `7330` |
| `MARKPOST_DB__DRIVER` | Database driver | `sqlite` |
| `MARKPOST_DB__DSN` | Connection string | `file:./data/markpost.db` |
| `MARKPOST_JWT__ACCESS_SIGNING_KEY` | **Required.** JWT signing key | — |

See [config.example.toml](backend/config.example.toml) for the full reference.

## Development

Prerequisites: Go 1.26+, Node.js 24+, pnpm, Docker

```bash
# Start dev environment (PostgreSQL, backend with hot-reload)
python3 devops/dev.py start

# Frontend dev server
cd frontend && pnpm install && pnpm dev

# Run backend tests
cd backend && go test ./...

# Run frontend tests
cd frontend && pnpm test
```

See [Development Guide](docs/development.md) for detailed instructions.

## Deployment

See [Deployment Guide](docs/deployment.md) for:
- Production Docker setup
- Ansible automation
- Reverse proxy configuration
- PostgreSQL configuration

## License

[MIT](LICENSE)
