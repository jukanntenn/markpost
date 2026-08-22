# MRFC: Backend tests run against a real PostgreSQL container

Status: implemented

## Problem

Repository mocks and in-memory databases hide SQL drift: a query that passes against a mock can still be wrong against the real database — dialect-specific operators, JSON behavior, locking clauses, and migration state never execute. Mock-heavy suites give green tests that do not predict production behavior, and the gap only surfaces on deploy.

## Decision

Backend integration tests run against a real PostgreSQL 17 container started through testcontainers-go (`backend/internal/infra/testdb.go`). Every package needing a database routes its `TestMain` through `infra.RunTestMain` (`backend/internal/infra/main_test.go`): one shared container outlives individual tests, and the package — not the testcontainers reaper — terminates it. The reaper stays deliberately disabled because it is shared across packages by parent pid and would otherwise kill containers mid-run. Setting `TESTCONTAINERS_SKIP=1` skips these tests where no Docker daemon exists.

## Alternatives considered

**SQLite in-memory database.** Fast and daemon-free, but exercises a different SQL dialect than production — the exact drift this decision exists to eliminate; the divergence became intolerable once PostgreSQL-only SQL (partial indexes, `FOR UPDATE SKIP LOCKED`) entered the codebase.

**Repository interfaces mocked at the service boundary.** Keeps unit tests hermetic, but repository SQL is then never executed by any test; SQL regressions reach production untested. Mocks remain in use only where the unit under test is not the repository itself.

**One container per test.** Strongest isolation, but container startup dominates runtime and suites become too slow to run as a pre-push gate. The shared per-package container is the compromise that keeps real SQL at a tolerable cost.

## Consequences

A running Docker daemon is required to run the full backend suite (locally and in CI); without one, `TESTCONTAINERS_SKIP=1` trades coverage for runnability. The suite is slower than mock-only tests. In exchange, dialect behavior, migrations, and locking semantics execute under test, and local runs exercise the same engine CI and production use. This is [PRINCIPLES.md](../../../PRINCIPLES.md) "Minimal mock, maximal real" applied to the database boundary.
