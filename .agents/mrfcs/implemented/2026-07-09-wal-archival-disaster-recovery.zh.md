# MRFC: WAL-archival disaster recovery to B2

Status: implemented

[English](2026-07-09-wal-archival-disaster-recovery.md) | 中文

## Problem

markpost 以单实例运行：一台 VPS、一个 Postgres 容器、无副本。服务器宕机或宿主丢失数据，一切皆失 —— 部署管线完全没有安排任何备份。数据是按用户保留策略管理的瞬时内容（全局默认 7 天；VIP 子集可无限保留，见[保留策略 MRFC](../implemented/2026-08-31-per-user-history-retention-policy.zh.md)），写入约 0.12 次/秒；而 VPS 上行链路约 3 Mbps —— 出站字节正是[缓存规格](../../../specs/backend/caching.zh.md)已经围绕设计的约束瓶颈。因此恢复设计必须相称：以最小成本换最小损失，备份流量按**新增写入**而非存量数据定尺，且不带副本运维的复杂度。

## Decision

DR 档位是**pgBackRest WAL 归档到 Backblaze B2**，由部署管线供给、按环境经 vault 激活（`b2_repo_key_id` —— staging 与生产各一对、各对着自己的桶，因为 staging 是晋升门；与心跳、Beszel 代理共享的设置顺序契约；规程见 [`docs/backup.md`](../../../docs/backup.zh.md)）：

- 每月全量基础备份加每日增量，都在 postgres 容器内运行 —— 该镜像由 `postgres-archival.Dockerfile` 本地构建（postgres:17-alpine + Alpine 的 `pgbackrest` 包，因为官方 pgbackrest 镜像是 glibc，进不了 musl）。持续 WAL 归档依托 `archive_mode=on` / `archive_command` / `archive_timeout=300`，把 RPO 界定在 ≤ 5 分钟写入。
- 每日一份逻辑 `pg_dump`（03:30 UTC，zstd，rclone 限速 2 MB/s，B2 生命周期 14 天过期）提供格式多样性，兜底会打断 PITR 链的基础镜像或页级损坏。
- 备份不可变：桶版本化，上传密钥无删除能力，过期由服务端执行 —— 失陷的主机只能新增备份，不能删除它们。
- 失败被观测：每日 `pgbackrest check` 加 26 小时新鲜度探测加 `pg_wal` 增长绊线（128 段 / 2 GB）把判定推送到 uptime-kuma；每月 cron 演练在临时容器内恢复并断言行数落在 RPO 界内。

| 属性                 | pgBackRest WAL 归档（选定）                        | 每小时 pg_dump 起步（拒绝）                        | 在线流式副本（拒绝）                                |
| -------------------- | -------------------------------------------------- | -------------------------------------------------- | --------------------------------------------------- |
| RPO（数据丢失）      | ≤ 5 分钟（WAL 尾巴）                               | ≤ 1 小时                                           | 约 0（同步）或秒级（异步）                          |
| RTO（停机）          | ~30–45 分钟；前提：3 Mbps 恢复下载                 | 现量级 ~10 分钟，随数据线性增长                    | 秒到分钟（自动故障转移）                            |
| 日上传（~1 GB 库）   | ~0.1–0.2 GB                                        | 压缩后 ~1.2 GB；未压缩每小时饱和 ~13 分钟         | WAL 流 + 第二台常开 VPS（约每月 $5）                |
| 额外基础设施         | 无 —— 仅对象存储，≤ $0.10/月                       | 无                                                 | 第二台 VPS + 故障转移工具                           |
| 运维成本             | 低 —— 一次配置无人值守；归档停滞需告警             | 低但易静默失败                                     | 高 —— 复制延迟监控、故障转移自动化、脑裂            |

## Alternatives considered

**每小时 `pg_dump` 作为起步档。** 全量逻辑转储每次运行都是 O(全部存量数据)：~1 GB 库的未压缩每小时 dump 每小时要饱和 3 Mbps 链路约 13 分钟，且每天把整个语料重传 24 遍（压缩后 ~1.2 GB/天）；到设计上限 ~15 GB 时单次运行超过一小时，这一档直接死亡。在当前写入率下，RPO、带宽、静默失败面三轴全被 WAL + 增量支配 ——"最简单档起步"推迟的恰恰是正确的架构，而不是赚到了简单。

**带自动故障转移的在线流式副本。** RPO/RTO 的收益配不上 25 倍的成本与副本运维的复杂度：写入率约 0.12/s，数据大多在 7 天视界上衰减，且故障期间读路径在 CDN 边缘存活、只有写在等。（单实例韧性 —— 无 Redis、无第二台 VPS —— 已由[性能优化 MRFC](../implemented/2026-07-09-read-path-performance-pass.zh.md)裁定；本记录覆盖那个裁决留下的备份层。）

**用 Cloudflare R2 替代 B2。** 备份写多读少：B2 存储便宜 3 倍（$0.005 对 $0.015/GB/月），一次性的恢复出口流量可忽略。B2 还把备份留在 Cloudflare 伞外，一个失陷的 Cloudflare 账号无法同时删掉在线路径与备份。在 R2 免费层放第二份**副本**作为账号多样性加固曾被提出，暂未决定。

**用 `wal-g` 替代 pgBackRest。** 两者都是主流并讲 B2 实现的 S3 API；pgBackRest 专精 Postgres，带页级增量、保留管理与 `check` 往返校验。2026-04 的维护者更替（见 Consequences）削弱了"社区更强"的常规定调；若赞助联盟停滞，wal-g 仍是即插即用的后备。

## Consequences

这笔取舍买到的是：RPO ≤ 5 分钟，占用 3 Mbps 链路不足 1%，成本 ≤ $0.10/月，且传输量随写入缩放 —— 到设计上限 ~15 GB 时同一设计仍然装得下（每月全量变大，日流仍是 ~MB 级），而任何重复的全量 dump 装不下。它付出的代价是：激活随一次计划内 postgres 重启（`archive_mode` 是 postmaster 上下文）；本地构建的派生镜像由我们维护（Alpine 包更新随基础镜像重建到来）；归档停滞会撑满 `pg_wal` 直至写路径死亡 —— 依次以 `archive-push-queue-max=1GiB` 背压、每日 check/新鲜度告警、`pg_wal` 绊线和 40 GB 盘上按天计的余量缓解。恢复路径由每月演练执行，而非假设。pgBackRest 自身刚经历维护者更替 —— 2026-04-27 被唯一维护者归档，2026-05-19 起由付费赞助联盟（Percona、AWS、Supabase 等）接管 —— 仓库格式 MIT、对象自存于 B2，即使项目停滞，既有备份仍可被任何旧版本二进制恢复。损失界定：WAL 尾巴外加[调优规格](../../../specs/backend/postgres-tuning.zh.md)已接受的 `synchronous_commit=off` ~600 ms 窗口。运行时激活 —— 建桶、无删除键、生命周期规则、vault 密钥对、首次 stanza-create + 全量 —— 是运维者工作，记录在 [`docs/backup.md`](../../../docs/backup.zh.md)；当前态势见 [`specs/backend/disaster-recovery.md`](../../../specs/backend/disaster-recovery.zh.md)。
