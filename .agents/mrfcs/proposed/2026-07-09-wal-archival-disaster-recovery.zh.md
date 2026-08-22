# MRFC: WAL-archival disaster recovery to B2

Status: proposed

[English](2026-07-09-wal-archival-disaster-recovery.md) | 中文

## Problem

markpost 以单实例运行：一台 VPS、一个 Postgres 容器、无副本。服务器宕机或宿主丢失数据，一切皆失 —— 部署管线完全没有安排任何备份。数据是 7 天保留的瞬时内容，写入约 0.12 次/秒，因此恢复设计必须相称：以最小成本换最小损失，不带副本运维的复杂度。

## Proposal

采用**WAL 归档到对象存储**作为 DR 架构：持续上传 WAL 段，加上周期性全量基础备份到 **Backblaze B2**，恢复时从基础备份向前重放 WAL。从最简单的一档起步 —— cron 每小时把 `pg_dump` 上传到 B2（RPO ≤ 1 小时，RTO 约 10 分钟） —— 待每小时的 RPO 被证明不足时，升级为 **pgBackRest** 持续 WAL 归档（带 PITR 的秒级 RPO）。供给是在既有 Ansible 管理实例上的运维者工作；任何东西都不落进应用代码。

| 属性                 | WAL 归档（选定）                                | 在线流式副本（拒绝）                                   |
| -------------------- | ----------------------------------------------- | ------------------------------------------------------ |
| RPO（数据丢失）      | 秒级（WAL）或 ≤1 h（dump）                      | 约 0（同步）或秒级（异步）                             |
| RTO（停机）          | 约 30 分钟（开通 VPS、拉取基础备份、重放 WAL）  | 秒到分钟（自动故障转移）                               |
| 额外基础设施         | 无 —— 仅对象存储，40 GB 约每月 $0.20            | 第二台常开 VPS（约每月 $5）                            |
| 运维成本             | 低 —— pgBackRest 一次配置，无人值守运行         | 高 —— 复制延迟监控、故障转移自动化、脑裂               |

## Alternatives considered

**带自动故障转移的在线流式副本。** RPO/RTO 的收益配不上 25 倍的成本与副本运维的复杂度：写入率约 0.12/s，数据在 7 天视界上衰减，且故障期间读路径在 CDN 边缘存活、只有写在等。（单实例韧性 —— 无 Redis、无第二台 VPS —— 已由[性能优化 MRFC](../implemented/2026-07-09-read-path-performance-pass.zh.md)裁定；本记录覆盖那个裁决留下的备份层。）

**用 Cloudflare R2 替代 B2。** 备份写多读少：B2 存储便宜 3 倍（$0.005 对 $0.015/GB/月），一次性的恢复出口流量（40 GB 约 $0.40）可忽略。B2 还把备份留在 Cloudflare 伞外，一个失陷的 Cloudflare 账号无法同时删掉在线路径与备份。

**用 `wal-g` 替代 pgBackRest。** 两者都是主流并讲 B2 实现的 S3 API；pgBackRest 专精 Postgres，带增量备份、并行处理与完整性校验，社区文档更强。两者皆可；推荐 pgBackRest。

## Acceptance criteria

- 一条 cron 驱动的 `pg_dump` 到 B2 的备份（或 pgBackRest 全量备份）在生产实例上无人值守运行，且其失败可观测。
- 存在一套成文的恢复流程，并已在一台临时 VPS 上执行过一次，在声明的 RTO 内恢复到一致的数据库。
- 升级到持续 WAL 归档是一次配置变更，不是重设计。

## Risks

B2 是一个外部依赖，有自己的故障画像（罕见；多副本）；每小时 `pg_dump` 一档最多丢一小时的写入；而在本提案实现之前，实例完全在没有自动备份的状态下运行 —— 当前状态姿态记录在 [`specs/backend/disaster-recovery.md`](../../../specs/backend/disaster-recovery.zh.md)。
