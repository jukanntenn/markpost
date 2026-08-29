# Development Guide

English | [中文](development.zh.md)

How to set up and run markpost locally. The documentation rules live in [AGENTS.md](AGENTS.md); deployment is covered by [deployment.md](deployment.md); the agent-driven development loop runbook (activation checklist and platform constraints) is [agent-loop-runbook.md](agent-loop-runbook.md).

## Prerequisites

| Tool                    | Version     | Description                          | Install                                                                 |
| ----------------------- | ----------- | ------------------------------------ | ----------------------------------------------------------------------- |
| Go                      | 1.26+       | Backend language                     | [go.dev/dl](https://go.dev/dl/)                                         |
| Node.js                 | 24+         | Frontend runtime                     | [nodejs.org](https://nodejs.org/)                                       |
| pnpm                    | 11+         | Frontend package manager             | [pnpm.io/installation](https://pnpm.io/installation)                    |
| Docker & Docker Compose | Compose v2+ | Dev environment services             | [docs.docker.com/get-docker](https://docs.docker.com/get-docker/)       |
| Python 3                | 3.12+       | Dev environment orchestration script | [python.org](https://www.python.org/downloads/)                         |
| golangci-lint           | latest      | Go linter                            | [golangci-lint.run/install](https://golangci-lint.run/welcome/install/) |
| air                     | latest      | Go hot-reload during development     | [github.com/cosmtrek/air](https://github.com/cosmtrek/air#installation) |

Swagger generation uses `swag` pinned in `backend/go.mod` via `go tool` — no standalone install. PostgreSQL runs as a container; no local database install is needed.

## Quick Start

### Option 1 — `dev.py` (recommended)

Starts PostgreSQL, the backend, and the frontend, all in Docker Compose:

```bash
python3 devops/dev.py start   # start all services
python3 devops/dev.py stop    # stop all services
python3 devops/dev.py logs [backend|frontend|postgres]
```

- Frontend: <http://localhost:3034>
- Backend: <http://localhost:7330>
- Database: `docker exec markpost-postgres psql -U markpost` — the dev compose intentionally publishes no Postgres port; use `docker exec`.

### Option 2 — VS Code / Cursor / Trae / compatible IDEs

The project ships `.vscode/tasks.json` with three tasks:

- **Start All** — runs backend and frontend in parallel
- **Start Backend** — launches `air` in `backend/` with dev JWT keys
- **Start Frontend** — launches `pnpm dev` in `frontend/`

Open the Command Palette (`Ctrl+Shift+P`) → **Tasks: Run Task** → pick a task. To bind a shortcut (e.g. `Alt+R` → "Start All"), open keyboard shortcuts JSON (`Ctrl+Shift+P` → **Preferences: Open Keyboard Shortcuts (JSON)**) and add:

```json
{
  "key": "alt+r",
  "command": "workbench.action.tasks.runTask",
  "args": "Start All"
}
```

Note: make sure `air` and `pnpm` are in your PATH.

### Option 3 — Manual

Running services on the host requires your own PostgreSQL 17 reachable at the configured DSN (the dev compose does not publish 5432).

**Backend** (air hot-reload):

```bash
cd backend
cp config.example.toml config.toml   # edit [db] dsn to point at your Postgres
air
```

The dev server starts at <http://localhost:7330>. Set `debug = true` in `config.toml` to enable debug mode.

**Frontend:**

```bash
cd frontend
pnpm dev
```

The dev server starts at <http://localhost:3034>.

## Install Dependencies

`python3 devops/dev.py start` auto-installs dependencies on first run. To install manually:

**Backend:**

```bash
cd backend
go mod download
```

**Frontend:**

```bash
cd frontend
pnpm install
```

## Lint

prek owns every format/lint invocation (see `prek.toml` at the root, `backend/prek.toml`, `frontend/prek.toml`). After `prek install`, `git commit` runs the checks; on demand:

```bash
prek run --all-files          # everything CI's Lint job runs
prek run --group fmt --files <path>   # just the fixers, for specific files
```

The per-tree commands remain: `golangci-lint run` in `backend/`, `pnpm lint` in `frontend/`.

## Run Tests

**Backend** (requires a running Docker daemon — tests start a real PostgreSQL container via testcontainers-go):

```bash
cd backend
go test ./...                        # all tests
go test ./internal/service/post/...  # specific package
```

Set `TESTCONTAINERS_SKIP=1` to skip container-backed tests when Docker is unavailable.

**Frontend:**

```bash
cd frontend
pnpm test          # Vitest in watch mode
pnpm test:run      # single run (CI)
```

**E2E** (separate workspace; Playwright, chromium only):

```bash
cd e2e
pnpm test                                                        # local run
dagger call -m e2e all --source ..                               # from repo root: full CI-fidelity run
dagger call -m e2e test --test-file=login.spec.ts --source ..    # single spec
```

Each dagger spec runs in an isolated sandbox (PostgreSQL + backend + frontend + Playwright containers).

## Build

**Backend:**

```bash
cd backend
go build ./cmd/server
```

**Frontend** (static export to `out/`):

```bash
cd frontend
pnpm build
```

## Generate Swagger Docs

```bash
cd backend
go generate ./...
```

This is the single source of regeneration (`go tool swag` pinned in go.mod, plus the embedded-CSS build). Swagger UI is available at `/swagger/index.html` when the backend runs with `debug = true`.

## API Testing with yaak

The backend ships a Swagger 2.0 spec at `backend/docs/swagger.json`, which [yaak](https://yaak.app/) imports natively — no conversion script is needed.

### Import into yaak

1. Open yaak
2. **File** → **Import Into Workspace**
3. Select `backend/docs/swagger.json` (yaak auto-detects Swagger 2.0)
4. Set the workspace base URL to `http://localhost:7330`
5. For authenticated endpoints, set an `Authorization` header with your access token

### Workflow

When the backend API changes:

1. Update Swagger annotations in Go code
2. Run `go generate ./...` to regenerate `backend/docs/swagger.json`
3. Re-import into yaak (your environment configs are preserved)

## Agent MCP tooling

`.mcp.json` defines MCP servers for coding agents working in this repo, including serena for semantic code navigation. Serena starts without an explicit project argument and uses its working directory at launch as the active project.

Serena's web dashboard opens a browser tab on every launch; to disable it, set `web_dashboard_open_on_launch: False` in `~/.serena/serena_config.yml` — a user-side setting that lives outside the repo.

## Configuration

The backend reads config from three sources (highest priority wins):

1. **Environment variables** — prefix `MARKPOST_`, nested keys use `__`

   ```bash
   MARKPOST_DEBUG=true
   MARKPOST_SERVER__PORT=8080
   MARKPOST_DB__DSN="postgres://user:pass@localhost:5432/markpost?sslmode=disable"
   ```

2. **TOML file** — `config.toml` next to the binary, or via `-c /path/to/config.toml`
3. **Built-in defaults** — see `backend/config.example.toml` for a full reference

Environment variables are the recommended way to override defaults.

The frontend has one environment variable: `BACKEND_URL` (default `http://127.0.0.1:7330`). It feeds the dev-server rewrites in `next.config.ts`, which proxy `/api/v1` and `/swagger` to the backend during development. Production has no frontend server — Caddy reverse-proxies those paths — so `BACKEND_URL` affects only local development. Override it in `frontend/.env.local` (gitignored).
