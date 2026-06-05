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

Starts PostgreSQL (Docker), the backend (Docker), and the frontend (local `pnpm dev`) in one command:

```bash
python3 devops/dev.py start
```

停止全部已启动的服务:

```bash
python3 devops/dev.py stop
```

### Option 2 — VS Code / Cursor / Trae / compatible IDE

The project ships `.vscode/tasks.json` with three tasks:

- **Start All** — runs both backend and frontend in parallel
- **Start Backend** — launches `air` in `backend/` with dev JWT keys
- **Start Frontend** — launches `pnpm dev` in `frontend/`

**Run a task:**

- Open Command Palette (`Ctrl+Shift+P`) → **Tasks: Run Task** → choose a task
- Or bind a shortcut (e.g. `Alt+R` to run "Start All"):
  1. Open keyboard shortcuts JSON (`Ctrl+Shift+P` → **Preferences: Open Keyboard Shortcuts (JSON)**)
  2. Add:

     ```json
     {
       "key": "alt+r",
       "command": "workbench.action.tasks.runTask",
       "args": "Start All"
     }
     ```

### Option 3 — Manual start

Run each service individually. Requires PostgreSQL or a local SQLite setup.

**Backend (with air hot-reload):**

```bash
cd backend
cp config.example.toml ../.local/config.toml
mkdir -p data
air
```

the backend defaults to SQLite (`data/markpost.db`).

**Frontend:**

```bash
cd frontend
pnpm dev
```

The dev server starts on [http://localhost:3034](http://localhost:3034) with hot module replacement.

## Install Dependencies

`python3 devops/dev.py start` auto-installs dependencies on first run. You can also install manually:

```bash
cd backend && go mod download
cd frontend && pnpm install
```

## Lint

```bash
cd backend && golangci-lint run
cd frontend && pnpm lint
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
dagger call all --source=..                                                    # run all spec files
dagger call test --test-file=login.spec.ts --source=..                        # run a single spec file
dagger call test --test-file=login.spec.ts --test-file=posts.spec.ts --source=..  # run multiple spec files
```

Each spec runs in an isolated sandbox (PostgreSQL + backend + frontend + Playwright containers).

## Build

```bash
cd backend && go build -o markpost-server ./cmd/server/
cd frontend && pnpm build
```

## Generate Swagger Docs

```bash
cd backend
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

Swagger UI is available at `/swagger/index.html` when running backend with `debug = true`.

## Environment Configuration

The backend reads config from three sources (highest priority wins):

1. **Environment variables** — prefix `MARKPOST_`, nested keys use `__`

   ```bash
   MARKPOST_DEBUG=true
   MARKPOST_SERVER__PORT=8080
   MARKPOST_DB__DSN="postgres://user:pass@localhost:5432/markpost?sslmode=disable"
   ```

2. **TOML file** — `markpost.toml` next to the binary, or via `-c /path/to/config.toml`
3. **Built-in defaults** — see `backend/config.example.toml` for a full annotated reference

The dev Docker Compose setup provides PostgreSQL with these defaults:

- Host: `postgres` (container network) / `localhost` (host)
- Database / User / Password: `markpost`

The frontend ships a committed `frontend/.env` with the default dev backend address:

```
BACKEND_URL=http://127.0.0.1:7330
```

The proxy (`src/proxy.ts`) uses this to forward `/api/*` requests to the backend. To override, create `frontend/.env.local` (gitignored):

```
BACKEND_URL=http://your-backend:7330
```
