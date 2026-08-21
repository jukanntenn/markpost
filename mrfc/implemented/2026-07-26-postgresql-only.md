# MRFC: PostgreSQL is the only supported database

Status: implemented

## Problem

Supporting PostgreSQL, MySQL, and SQLite simultaneously taxed every data-layer decision: SQL had to stay dialect-portable, indexes and table tuning branched per dialect, migrations needed three-way validation, and the test matrix tripled. Neither MySQL nor SQLite had a single production deployment, so the cost bought nothing — the tax was paid for capabilities nobody used.

## Decision

markpost runs on PostgreSQL 17 only. `backend/internal/infra/db.go` opens the connection with `gorm.io/driver/postgres` (pgx v5) and nothing else; `db.driver` validates as `oneof=postgresql` and defaults to `postgresql` (`backend/internal/config/config.go`); the sqlite/mysql drivers are gone from `go.mod`. Schema changes go through versioned PostgreSQL-only SQL migrations embedded in the binary (`backend/internal/infra/migrations/`, applied by the `migrate` subcommand before `serve`). Free use of PostgreSQL specifics — partial indexes, `FOR UPDATE SKIP LOCKED`, `fillfactor`, lz4 TOAST compression, Unix-domain sockets — is now the default, not a branch.

## Alternatives considered

**Keep all three drivers.** Maximum deployer flexibility, but every schema and query change pays the portability tax forever and the test matrix keeps growing; no deployment ever used MySQL.

**PostgreSQL plus SQLite for homelab/dev.** Preserves a zero-dependency small deployment, but keeps the dual-dialect SQL discipline and the divergent locking/concurrency arguments (single-connection serialization vs. row locks) alive in every delivery-path design.

**An abstraction layer over per-dialect repositories.** Localizes dialect differences, but duplicates the data layer and still requires the full test matrix to prove each implementation.

## Consequences

Any deployment, including a homelab, runs a PostgreSQL 17 instance — the single Docker image and dev compose both provide one. In return, the data layer uses the best available PostgreSQL mechanism for each job, tests run against the one real engine, and migrations are written once. Dropping the drivers entirely rather than carrying them unused is [PRINCIPLES.md](../../PRINCIPLES.md) "Design from first principles" applied.
