# AGENTS.md

## Identity

You are a senior pair-programming partner for the markpost codebase: a Go (Gin/GORM) backend and a Next.js 16 + React 19 frontend, deployed as a single multi-arch Docker image. Write secure, maintainable, performant code that matches the patterns already in this repo.

## Commands

All commands assume the working directory noted in each section. Prefer running the dev environment in containers (see DevOps) over running services on the host.

**Backend** (`backend/`):
- `go test ./...` — run tests (uses testcontainers-go; requires a running Docker daemon)
- `go build ./...` — compile
- `golangci-lint run ./...` — lint
- `golangci-lint fmt` — format all Go files (gofmt + goimports)
- `go run ./cmd/server migrate up` — apply database migrations (run before serve on deploys)
- `swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal` — regenerate Swagger docs

**Frontend** (`frontend/`):
- `pnpm dev` — dev server (port 3034)
- `pnpm build` — production build (static export to `out/`)
- `pnpm lint` — ESLint
- `pnpm format` / `pnpm format:check` — Prettier write / check
- `pnpm test` — Vitest watch; `pnpm test:run` — run once

**E2E** (`e2e/`, separate workspace):
- `pnpm test` — Playwright (chromium only)
- `dagger -c e2e call all --source ..` — full e2e via dagger (from repo root)

**DevOps** (repo root):
- `python3 devops/dev.py start` — start backend + frontend + postgres in Docker Compose
- `python3 devops/dev.py stop`
- `python3 devops/dev.py logs [backend|frontend|postgres]`
- `docker exec markpost-postgres psql -U markpost` — inspect dev DB (postgres has no published port)

## Tech Stack

- **Frontend**: Next.js 16 (`output: "export"` static export), React 19, TypeScript, Tailwind CSS 4, Zustand, TanStack Query, next-intl, @base-ui/react, Prettier
- **Backend**: Go 1.26, Gin, GORM, JWT, Swagger (swag), Viper, OpenTelemetry
- **Database**: PostgreSQL 17 (the only supported driver; sqlite/mysql were removed)
- **Testing**: Vitest (frontend unit), testcontainers-go + postgres (backend), Playwright chromium (e2e)
- **Tooling**: golangci-lint v2 (lint+format), prek (pre-commit), air (Go hot reload)

## Project Structure

```
backend/
  cmd/server/        HTTP server entry + CLI subcommands (serve, migrate, reset-password, ...)
  cmd/<sub>.go       one Run* func per subcommand
  internal/api/rest/v1/  REST handlers
  internal/config/   Viper + TOML config (validate tags)
  internal/domain/   domain models + repository interfaces (post/, user/, delivery/)
  internal/infra/    GORM repos + migrations/ (embedded SQL) + migrate.go + testdb.go
  internal/middleware/  auth, CORS, rate limiting
  internal/service/  business logic (auth/, post/, delivery/, admin/)
  pkg/               shared packages (apierr/, auth/, crypto/, i18n/, utils/)
  docs/              generated Swagger (DO NOT edit)
frontend/
  src/app/           App Router ((auth), (dashboard): admin/dashboard/posts/settings)
  src/components/    ui/ (shadcn-style), auth/, layout/, dashboard/, admin/, posts/
  src/lib/           utils.ts, api/ fetchers
  src/i18n/          next-intl + locales (en, zh)
  src/stores/        Zustand
  next.config.ts     output: "export" + dev-only rewrites (proxy /api/v1 to backend)
e2e/                 Playwright workspace (separate package.json, chromium only)
devops/              dev.py, docker-compose.yml, *.Dockerfile, ansible/
docker/              production Dockerfile (s6 multi-process), build.py
.github/workflows/   CI (lint/test/build/e2e with path filters)
```

## Database Migrations (important)

Schema changes go through `golang-migrate` with versioned SQL files in `backend/internal/infra/migrations/` (embedded in the binary). Rules:
- To change schema: create `NNNNNN_description.up.sql` + `.down.sql`, run `markpost migrate up`.
- Migrations are PostgreSQL-only (the only supported driver).
- Never edit a migration file after it's applied to any shared DB; write a new one instead.
- `infra.New` only opens the connection; it does NOT migrate. Migrations run via the `migrate` subcommand, called by the deploy pipeline before `serve`.

## Code Style

- **Go**: golangci-lint handles format (gofmt + goimports, `local-prefixes: markpost`) and lint. No standalone gofmt/goimports calls. Self-documenting code; comments only to explain *why*, never *what*.
- **Frontend**: Prettier for formatting (`.prettierrc.json`), eslint-config-next for correctness. Function components + hooks only, never class components. PascalCase component files (`PostList.tsx`).
- **No comments unless explaining a non-obvious *why*.** Prefer clear names.
- **Never edit generated files**: `backend/docs/` (Swagger), lock files (`pnpm-lock.yaml`, `go.sum`).

## Git Workflow

- **Conventional Commits** (match existing history): `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`, `build:`, `style:`. Optional scope: `fix(test): ...`.
- Examples from this repo: `fix(test): relax singleflight burst assertion`, `chore(devops): fix ansible warnings`.
- Commits are signed off by the author; do not commit on behalf of others.
- `prek` runs on pre-commit (format + lint + AGENTS sync) and pre-push (tests); a `commit-msg` hook checks the Conventional Commits format.

## Testing

- **Backend**: `go test ./...` in `backend/`. Tests use testcontainers-go (real PostgreSQL container) — a Docker daemon is required. Set `TESTCONTAINERS_SKIP=1` to skip when Docker is unavailable.
- **Frontend**: `pnpm test:run` — Vitest with jsdom + v8 coverage.
- **E2E**: Playwright, chromium only. Local: `cd e2e && pnpm test`. CI/fidelity: `dagger -c e2e call all --source ..`.

## Boundaries

- **Always**:
  - Read a file in full before editing it.
  - Run the relevant formatter/linter before finishing (golangci-lint for Go, pnpm format+lint for frontend).
  - Pair every GORM struct tag change with a new migration file.
- **Ask first**:
  - Modifying database schema / writing migrations.
  - Adding a new dependency (go get / pnpm add).
  - Changes to CI workflows or Docker images.
- **Never**:
  - Edit generated files (Swagger docs in `backend/docs/`, lock files).
  - Commit secrets or `.env` files.
  - Reintroduce sqlite/mysql database drivers — PostgreSQL is the only supported DB.
  - Add a `middleware.ts`/`proxy.ts` to the frontend — it is a static export; dev `/api` proxying is via `rewrites` in `next.config.ts`, prod via Caddy.
