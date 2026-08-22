# AGENTS.md — backend

Go 1.26 service: Gin + GORM over PostgreSQL, JWT auth, Swagger via swag, Viper config, OpenTelemetry. Repo-wide orders live in the [root AGENTS.md](../AGENTS.md); this file adds this tree's own.

## Commands (run in `backend/`)

- `go test ./...` — tests (testcontainers-go; requires a running Docker daemon)
- `go test -race ./...` — with the race detector (mirrors CI)
- `go test -cover ./...` — per-package coverage summary; `-coverprofile=coverage.out` + `go tool cover -html=coverage.out` to view it
- `go test -fuzz=^FuzzXxx$ -fuzztime=120s ./internal/pkg/...` — fuzz one target for a bounded duration (crash seeds land in `testdata/fuzz/`)
- `go build ./...` — compile
- `golangci-lint run ./...` — lint; `golangci-lint fmt` — format (gofmt + goimports; no standalone gofmt/goimports calls)
- `golangci-lint config verify` — validate `.golangci.yml` against the embedded schema
- `go run ./cmd/server migrate up` — apply database migrations (run before serve on deploys)
- `go generate ./...` — regenerate Swagger docs (`go tool swag`, pinned in go.mod) + embedded CSS (`cmd/buildcss`); the single source of regeneration

## Layout

```
cmd/server/           HTTP server entry + CLI subcommands (serve, migrate, reset-password, ...)
cmd/<sub>.go          one Run* func per subcommand
internal/api/rest/v1/ REST handlers
internal/config/      Viper + TOML config (validate tags)
internal/domain/      domain models + repository interfaces (post/, user/, delivery/)
internal/infra/       GORM repos + migrations/ (embedded SQL) + migrate.go + testdb.go
internal/middleware/  auth, CORS, rate limiting
internal/service/     business logic (auth/, post/, delivery/, admin/)
pkg/                  shared packages (apierr/, auth/, crypto/, i18n/, utils/)
docs/                 generated Swagger (DO NOT edit)
```

## Database migrations

Schema changes go through `golang-migrate` with versioned SQL files in `internal/infra/migrations/` (embedded in the binary):

- To change schema: create `NNNNNN_description.up.sql` + `.down.sql`, run `markpost migrate up`.
- Migrations are PostgreSQL-only (the only supported driver).
- Never edit a migration file after it's applied to any shared DB; write a new one instead.
- `infra.New` only opens the connection; migrations run via the `migrate` subcommand, called by the deploy pipeline before `serve`.
- Pair every GORM struct tag change with a new migration file.

## Testing

Tests use testcontainers-go (a real PostgreSQL container) — repository mocks in CI hide SQL drift. Any package calling `infra.SetupTestDB` routes its `TestMain` through `infra.RunTestMain` (see `internal/infra/main_test.go`): the shared container outlives individual tests, and the testcontainers reaper is deliberately disabled (it is shared across packages by parent pid and used to kill containers mid-run), so the package terminates it itself. `TESTCONTAINERS_SKIP=1` skips container tests when Docker is unavailable. Fuzz targets are scheduled daily by `.github/workflows/fuzz.yml`; commit any `testdata/fuzz/` crash seeds as regression cases.

## Style and boundaries

- Self-documenting code; comments only to explain _why_, never _what_.
- Never edit generated files: `docs/` (Swagger), `go.sum`.
- Never reintroduce sqlite/mysql database drivers — PostgreSQL is the only supported DB.
