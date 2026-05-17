# Development Guide

## Prerequisites

- **Go 1.26+** — Backend language
- **Node.js 24+** — Frontend runtime
- **pnpm** — Frontend package manager
- **Docker & Docker Compose** — Dev environment services (PostgreSQL)
- **Python 3** — Dev environment orchestration script
- **golangci-lint** — Go linter
- **swag** — Swagger doc generator for Go

## Quick Start

1. Clone the repository
2. Start the dev environment:

```bash
python3 devops/dev.py start
```

This starts PostgreSQL, the backend server, and the frontend dev server via Docker Compose. The backend runs on port 7330 and the frontend on port 3034.

3. Access the application at `http://localhost:3034`

## Backend Development

Working directory: `backend/`

### Install Dependencies

```bash
cd backend
go mod download
```

### Run Tests

```bash
go test ./...
```

Run a specific package:

```bash
go test ./internal/service/post/...
```

### Lint

```bash
golangci-lint run
```

### Generate Swagger Docs

```bash
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

Swagger UI is available at `/swagger/index.html` when running with `debug = true`.

### Hot Reload

Use [air](https://github.com/cosmtrek/air) for live reload during development:

```bash
air
```

### Build

```bash
go build -o markpost-server ./cmd/server/
```

## Frontend Development

Working directory: `frontend/`

### Install Dependencies

```bash
cd frontend
pnpm install
```

### Dev Server

```bash
pnpm dev
```

Starts on port 3034 with hot module replacement.

### Run Tests

```bash
pnpm test          # Watch mode
pnpm test:run      # Single run (CI)
pnpm test:e2e      # Playwright E2E tests
```

### Lint

```bash
pnpm lint
```

### Build

```bash
pnpm build
```

## Environment Configuration

The dev Docker Compose setup provides PostgreSQL with these defaults:

- Host: `localhost`
- Port: `5432`
- Database: `markpost`
- User: `markpost`
- Password: `markpost`

No config file is needed for local development. The backend defaults to SQLite if no DSN is configured, which works for quick testing without Docker.

For the frontend, set `NEXT_PUBLIC_API_URL` if the backend is not accessible at the same origin:

```bash
NEXT_PUBLIC_API_URL=http://localhost:7330 pnpm dev
```

## Common Workflows

### Creating a New API Endpoint

1. Define the domain model and repository interface in `internal/domain/`
2. Implement the repository in `internal/infra/`
3. Create a service in `internal/service/`
4. Add a handler in `internal/api/rest/v1/`
5. Register the route in `cmd/server/main.go` in `SetupRoutes`

### Adding a UI Component

1. Create the component in the appropriate `src/components/` subdirectory
2. Use shadcn/ui primitives from `src/components/ui/`
3. Follow the design system in `specs/frontend/design.md`
4. Use Tailwind utilities with design tokens (`bg-background`, `text-foreground`)

### Adding a Translation

**Backend** (TOML files in `backend/locales/`):

```toml
["error.my_new_message"]
other = "My new message"
```

**Frontend** (JSON files in `frontend/src/i18n/locales/`):

Add keys to both `en.json` and `zh.json`:

```json
{
  "myFeature": {
    "title": "My Feature"
  }
}
```
