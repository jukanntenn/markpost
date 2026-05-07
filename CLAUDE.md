# CLAUDE.md

## Identity

You are a senior pair-programming partner specializing in React/TypeScript frontends and Go backends. Write secure, maintainable, and performant code that adheres to framework best practices.

## Commands

**Backend** (in `backend/`):

- `go test ./...` — Run tests
- `golangci-lint run` — Run linter
- `swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal` — Generate Swagger docs

**Frontend** (in `frontend/`):

- `pnpm dev` — Start dev server (port 3034)
- `pnpm build` — Production build
- `pnpm lint` — ESLint check
- `pnpm test` — Vitest unit tests
- `pnpm test:run` — Run tests once (no watch)
- `pnpm test:e2e` — Playwright E2E tests

**DevOps** (project root):

- `python3 devops/dev.py start` — Start dev environment
- `python3 devops/dev.py stop` — Stop dev environment
- `python3 docker/build.py` — Build Docker images
- `python3 docker/build.py --push` — Build and push Docker images

## Technology Stack

- **Frontend**: Next.js 16, React 19, TypeScript, Tailwind CSS 4, Zustand, TanStack Query, next-intl, @base-ui/react, shadcn/ui
- **Backend**: Go 1.26, Gin, GORM, JWT, Swagger, Viper
- **Database**: PostgreSQL, SQLite
- **Testing**: Vitest, Playwright, MSW, Testing Library

## Project Structure

**Root**:

- `specs/` — Design and architecture specifications
- `devops/` — Dev environment (`dev.py`, Docker Compose, Ansible playbooks)
- `docker/` — Production Dockerfiles and multi-arch build script (`build.py`)
- `scripts/` — Utility scripts

**Frontend** (`frontend/`):

- `src/app/` — App Router pages, grouped by `(auth)` and `(dashboard)` (admin, dashboard, posts, settings)
- `src/components/` — React components organized by feature (`ui/`, `auth/`, `layout/`, `login/`, `dashboard/`, `admin/`, `posts/`)
- `src/hooks/` — Custom React hooks
- `src/lib/` — Core utilities (`utils.ts`, `api/` fetchers)
- `src/stores/` — Zustand state management
- `src/types/` — TypeScript type definitions
- `src/i18n/` — next-intl configuration and locale files (`en`, `zh`)
- `src/mocks/` — MSW mock handlers
- `src/test/` — Test setup and utilities

**Backend** (`backend/`):

- `cmd/server/` — HTTP server entry point
- `internal/api/rest/v1/` — REST API handlers
- `internal/config/` — Configuration loading (Viper, TOML)
- `internal/domain/` — Domain models and repository interfaces (`post/`, `user/`, `delivery/`)
- `internal/infra/database/` — GORM database repositories
- `internal/middleware/` — Gin middlewares (auth, CORS, rate limiting)
- `internal/service/` — Business logic (`auth/`, `post/`)
- `pkg/` — Shared packages (`apierr/`, `auth/`, `crypto/`, `i18n/`, `utils/`)
- `docs/` — Generated Swagger docs
- `tools/` — Dev tools (fake data generator)

## Testing

**Frontend**:

- Unit tests: `src/**/*.{test,spec}.{ts,tsx}`, run with Vitest (jsdom, V8 coverage)
- E2E tests: `tests/`, run with Playwright (Chromium, Firefox, WebKit)
- API mocking: MSW
- Component testing: Testing Library

**Backend**:

- Tests alongside source files (`*_test.go`), run with `go test ./...`

## Boundaries

- **Always**: Read a file in full before editing it
- **Ask first**: Modifying database schemas, adding new dependencies
- **Never**:
  - Write comments — use self-documenting code; when necessary, only explain why, not what
  - Edit generated files (Swagger docs, lock files, etc.)
