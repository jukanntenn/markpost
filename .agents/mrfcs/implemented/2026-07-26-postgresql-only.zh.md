# MRFC: PostgreSQL is the only supported database

Status: implemented

[English](2026-07-26-postgresql-only.md) | 中文

## Problem

同时支持 PostgreSQL、MySQL 和 SQLite 给每个数据层决策上税：SQL 必须保持方言可移植，索引与表调优按方言分支，迁移需要三向验证，测试矩阵翻三倍。MySQL 与 SQLite 连一个生产部署都没有，成本什么也没换来 —— 税交给了没人使用的能力。

## Decision

markpost 只在 PostgreSQL 17 上运行。`backend/internal/infra/db.go` 用 `gorm.io/driver/postgres`（pgx v5）打开连接，别无其他；`db.driver` 校验为 `oneof=postgresql` 且默认 `postgresql`（`backend/internal/config/config.go`）；sqlite/mysql 驱动已从 `go.mod` 消失。schema 变更走嵌入二进制的版本化 PostgreSQL 专有 SQL 迁移（`backend/internal/infra/migrations/`，由 `migrate` 子命令在 `serve` 之前应用）。自由使用 PostgreSQL 专有特性 —— 部分索引、`FOR UPDATE SKIP LOCKED`、`fillfactor`、lz4 TOAST 压缩、Unix-domain socket —— 如今是默认，不是分支。

## Alternatives considered

**保留全部三个驱动。** 部署者灵活性最大，但每个 schema 与查询变更永远交可移植性税，测试矩阵持续膨胀；从来没有部署用过 MySQL。

**PostgreSQL 加 SQLite 供 homelab/开发。** 保住零依赖的小部署，却让双方言 SQL 纪律和分歧的锁定/并发论证（单连接串行化对行锁）活在每个投递路径设计里。

**在按方言的仓储之上加一层抽象。** 把方言差异局部化，但复制了数据层，且仍需完整测试矩阵来证明每个实现。

## Consequences

任何部署，包括 homelab，都运行一个 PostgreSQL 17 实例 —— 单一 Docker 镜像与 dev compose 都提供一个。作为回报，数据层为每项工作使用可得的最佳 PostgreSQL 机制，测试对着唯一真实引擎运行，迁移只写一次。彻底删掉驱动而不是带着不用，是 [PRINCIPLES.md](../../../PRINCIPLES.zh.md) "Design from first principles" 的应用。
