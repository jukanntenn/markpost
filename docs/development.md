# Development Guide

## Prerequisites

| Tool                    | Version     | Description                           | Install                                                                 |
| ----------------------- | ----------- | ------------------------------------- | ----------------------------------------------------------------------- |
| Go                      | 1.26+       | Backend language                      | [go.dev/dl](https://go.dev/dl/)                                         |
| Node.js                 | 24+         | Frontend runtime                      | [nodejs.org](https://nodejs.org/)                                       |
| pnpm                    | 11+         | Frontend package manager              | [pnpm.io/installation](https://pnpm.io/installation)                    |
| Docker & Docker Compose | Compose v2+ | Dev environment services (PostgreSQL) | [docs.docker.com/get-docker](https://docs.docker.com/get-docker/)       |
| Python 3                | 3.12+       | Dev environment orchestration script  | [python.org](https://www.python.org/downloads/)                         |
| golangci-lint           | latest      | Go linter                             | [golangci-lint.run/install](https://golangci-lint.run/welcome/install/) |
| air                     | latest      | Go hot-reload during development      | [github.com/cosmtrek/air](https://github.com/cosmtrek/air#installation) |
| swag                    | latest      | Swagger doc generator for Go          | [github.com/swaggo/swag](https://github.com/swaggo/swag#installation)   |

## Quick Start

### Option 1 — `dev.py` (recommended)

Starts PostgreSQL and the backend in Docker, plus the frontend dev server locally:

```bash
python3 devops/dev.py start   # start all services
python3 devops/dev.py stop    # stop all services
```

### Option 2 — VS Code / Cursor / Trae / compatible IDEs

The project ships `.vscode/tasks.json` with three tasks:

- **Start All** — runs backend and frontend in parallel
- **Start Backend** — launches `air` in `backend/` with dev JWT keys
- **Start Frontend** — launches `pnpm dev` in `frontend/`

Open the Command Palette (`Ctrl+Shift+P`) → **Tasks: Run Task** → pick a task.
To bind a shortcut (e.g. `Alt+R` → "Start All"), open keyboard shortcuts JSON (`Ctrl+Shift+P` → **Preferences: Open Keyboard Shortcuts (JSON)**) and add:

```json
{
  "key": "alt+r",
  "command": "workbench.action.tasks.runTask",
  "args": "Start All"
}
```

Note: make sure `air` and `pnpm` are in your PATH.

### Option 3 — Manual

**Backend** (air hot-reload):

```bash
cd backend
cp config.example.toml markpost.toml
air
```

The dev server starts at [http://localhost:7330](http://localhost:7330), defaulting to SQLite (`data/markpost.db`).
Set `debug = true` in `markpost.toml` to enable debug mode.

**Frontend:**

```bash
cd frontend
pnpm dev
```

The dev server starts at [http://localhost:3034](http://localhost:3034).

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

**Backend:**

```bash
cd backend
golangci-lint run
```

**Frontend:**

```bash
cd frontend
pnpm lint
```

## Run Tests

**Backend:**

```bash
cd backend
go test ./...                        # all tests
go test ./internal/service/post/...  # specific package
```

**Frontend:**

```bash
cd frontend
pnpm test          # Vitest in watch mode
pnpm test:run      # single run (CI)
```

**E2E (Dagger + Playwright):**

```bash
cd e2e
dagger call all --source=..                                                      # all specs
dagger call test --test-file=login.spec.ts --source=..                           # single spec
dagger call test --test-file=login.spec.ts --test-file=posts.spec.ts --source=.. # multiple specs
```

Each spec runs in an isolated sandbox (PostgreSQL + backend + frontend + Playwright containers).

## Build

**Backend:**

```bash
cd backend
go build ./cmd/server
```

**Frontend:**

```bash
cd frontend
pnpm build
```

## Generate Swagger Docs

```bash
cd backend
swag init -g main.go -d cmd/server,internal/api/rest/v1 -o docs --parseDependency --parseInternal
```

Swagger UI is available at `/swagger/index.html` when the backend runs with `debug = true`.

## Configuration

The backend reads config from three sources (highest priority wins):

1. **Environment variables** — prefix `MARKPOST_`, nested keys use `__`

   ```bash
   MARKPOST_DEBUG=true
   MARKPOST_SERVER__PORT=8080
   MARKPOST_DB__DSN="postgres://user:pass@localhost:5432/markpost?sslmode=disable"
   ```

2. **TOML file** — `markpost.toml` next to the binary, or via `-c /path/to/config.toml`
3. **Built-in defaults** — see `backend/config.example.toml` for a full reference

Environment variables are the recommended way to override defaults.

The frontend proxy (`src/proxy.ts`) forwards `/api/*` requests to the backend using `BACKEND_URL`, which defaults to `http://127.0.0.1:7330`. To override, create `frontend/.env.local` (gitignored):

```
BACKEND_URL=http://your-backend:7330
```
