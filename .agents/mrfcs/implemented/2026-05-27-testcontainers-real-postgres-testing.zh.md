# MRFC: Backend tests run against a real PostgreSQL container

Status: implemented

[English](2026-05-27-testcontainers-real-postgres-testing.md) | 中文

## Problem

仓储层 mock 与内存数据库会掩盖 SQL 漂移：一条在 mock 上通过的查询，对着真实数据库仍可能是错的 —— 方言专有运算符、JSON 行为、锁定子句和迁移状态从不执行。重度依赖 mock 的测试套件给出绿灯，绿灯却预测不了生产行为，而这个缺口要到部署时才暴露。

## Decision

后端集成测试对着一个经 testcontainers-go 启动的真实 PostgreSQL 17 容器运行（`backend/internal/infra/testdb.go`）。每个需要数据库的包都把它的 `TestMain` 路由经过 `infra.RunTestMain`（`backend/internal/infra/main_test.go`）：一个共享容器比单个测试活得久，并且由包 —— 而非 testcontainers 的 reaper —— 终止它。reaper 刻意保持禁用，因为它按父 pid 跨包共享，否则会在运行中途杀掉容器。在没有 Docker daemon 的地方，设置 `TESTCONTAINERS_SKIP=1` 跳过这些测试。

## Alternatives considered

**SQLite 内存数据库。** 快且无需 daemon，但演练的是与生产不同的 SQL 方言 —— 恰是本决策要消除的那类漂移；当 PostgreSQL 专有 SQL（部分索引、`FOR UPDATE SKIP LOCKED`）进入代码库后，这个分歧变得不可容忍。

**在服务边界 mock 仓储接口。** 让单元测试保持封闭，但仓储 SQL 从此不被任何测试执行；SQL 回归未经测试就抵达生产。mock 仅在被测单元不是仓储本身的地方继续使用。

**每个测试一个容器。** 隔离最强，但容器启动支配运行时长，套件慢到无法作为 pre-push 闸门运行。按包共享的容器是那个折中：让真实 SQL 留在可容忍的成本上。

## Consequences

运行完整后端套件（本地与 CI 中）需要一个运行中的 Docker daemon；没有它时，`TESTCONTAINERS_SKIP=1` 以覆盖换可运行性。套件比纯 mock 测试慢。作为交换，方言行为、迁移和锁定语义都在测试下执行，本地运行演练的正是 CI 与生产使用的同一引擎。这是把 [PRINCIPLES.md](../../../PRINCIPLES.zh.md) 的 "Minimal mock, maximal real" 应用于数据库边界。
