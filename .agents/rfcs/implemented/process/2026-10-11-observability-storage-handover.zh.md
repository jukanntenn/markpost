# RFC: 观测存储移交外部 devops 仓库

Status: implemented

[English](2026-10-11-observability-storage-handover.md) | 中文

## Problem

NAS(192.168.5.57)上的三个观测存储——Jaeger(traces)、VictoriaMetrics `vmsingle`(metrics)、VictoriaLogs(logs)——是 [OTLP 观测栈 MRFC](2026-09-19-otlp-observability-stack.zh.md) 之下的共享平台基建:多个业务系统向其发送遥测,由外部运维方按交付材料运行。三个存储的配置中 markpost 专属内容为零——无秘密、无租户配置、无 `.env`(已审计);其中的每个值都是平台级的:镜像钉版、数据路径、保留期、磁盘空间保护。

这一事实提出了栈 MRFC 未回答的两个问题。其一是仓库归属:运维方持有的平台服务,其配置标准化归属于运维方;租户仓库携带平台配置的副本,意味着每个租户都要养一份自己并不拥有、且会漂移的平台配置。其二是理由记录:配置里嵌着有未来价值的决策——Jaeger 2.20.0 钉版、磁盘空间保护标志——没有书面记录,钉版会被读作陈旧、标志会被读作随意。markpost 需要这些事实得以留存,即使配置本身住在别处。

## Decision

三个存储的配置标准化与收编入库完全位于外部 devops 运维仓库,以一服务一 playbook 的 Ansible playbook 交付:`ansible/victoriametrics.yml`、`ansible/victorialogs.yml`、`ansible/jaeger.yml`。[issue #134](https://github.com/jukanntenn/markpost/issues/134) 正文即 devops 侧的内容规格。

markpost 是纯粹的遥测生产方。本仓库不携带任何存储配置——无 playbook、无钉版、无存储秘密。生产方一侧及其查询消费([可观测 spec](../../../../specs/backend/observability.zh.md))是 markpost 仅有的自有面;传输链路按 [frp 传输 MRFC](2026-09-19-otlp-transport-frp-exposure.zh.md) 保持在外部。

playbook 所钉的存储与版本:Jaeger 2.20.0(traces)、VictoriaMetrics v1.151.0(`vmsingle`,metrics)、VictoriaLogs v1.52.0(logs)。发布面保持 `0.0.0.0`(8428 / 9428 / 16686),依单运维者的家庭内网信任模型。

本记录是观测所有权重划的存储部分;collector 移交([#133](https://github.com/jukanntenn/markpost/issues/133))与之并列,在各自的变更中修订栈 MRFC 的措辞。

## Storage pins and protection flags

playbook 持有何物、为何持有——2026-10-11 已对照上游源码验证;运维部署位于 devops 仓库:

- **Jaeger 钉在 2.20.0。** 上游 2.21.0 载入破坏性变更"Feat(query)!: remove v1 http endpoints the ui no longer calls"(PR #9260):v1 REST API 同时支撑 markpost 的 agent 查询面(`/api/traces`,[可观测 spec](../../../../specs/backend/observability.zh.md))与 Grafana 内置的 Jaeger 数据源。解除钉版的条件:Grafana 数据源迁移之后再评估。`jaegertracing/jaeger:2.20.0` 镜像以 USER 10001 运行,badger 数据目录的属主随之而来。
- **VictoriaMetrics 运行 `-storage.minFreeDiskSpaceBytes=1GB`**——共享 128GB SSD 上的保护阀;上游默认仅 100MB,低于阈值时存储自行切换为只读、停止接收新数据。其余标志形态:`storageDataPath`、`retentionPeriod=90d`、`memory.allowedBytes=300MB`、`selfScrapeInterval=10s`。
- **VictoriaLogs 运行 `-storage.minFreeDiskSpaceBytes=1GB`**(上游默认 10MB),并以 `-retention.maxDiskSpaceUsageBytes=20GiB` 为界(10GiB → 20GiB;上游默认 0——即无字节上限)。20GiB 上限与 30d 保留期并行约束、先到者生效,自动丢弃最旧的按日分区;20GiB 约为租户增长留了 ~2x 余量,占 SSD 的 ≈16%。
- **devops 侧跟进项**,由该仓库承担、在此留档以免边界遗失:VictoriaLogs `-deleteAuthKey` 保护破坏性的 `/delete/*` 端点(OSS 支持);如有需要,对 8428 / 9428 / 16686 未认证面的进一步收紧在网络层进行。

## Alternatives considered

**把三个存储的配置留在 markpost 仓库内。** 败在本记录核心的已审计事实:配置中 markpost 专属内容为零,仓库归属不会跟随运维归属——且每个租户仓库都要养一份平台配置副本,各自对着运维方的真值漂移。

**devops 仓库内单个合并的观测 playbook。** 三存储一个 playbook,评审只看一个文件。败在生命周期独立:Jaeger、VictoriaMetrics、VictoriaLogs 的发布节奏刻意不同,一服务一 playbook 把每次升级的爆炸半径限制在单个服务——与栈 MRFC 已为 NAS 定下的每服务 compose 形态互为镜像。

**现在就升级 Jaeger 2.21.0。** 保持最新优于携带钉版。败在消费面:2.21.0 移除了 Grafana 内置 Jaeger 数据源与 spec 的 agent 查询面仍在依赖的 v1 REST 端点;该升级只能与 Grafana 数据源迁移一同落地。

## Consequences

买到:平台配置跟随运维者——每个存储 playbook 只有一个家,即 devops 运维仓库——租户仓库保持纯生产方;钉版与保护标志的理由留存于本记录,而非消散在另一个仓库的 YAML 里。markpost 的查询接口不变:Prometheus querying API、LogsQL、Jaeger REST([可观测 spec](../../../../specs/backend/observability.zh.md))。

代价与义务:存储理由分处两个仓库——本记录持有"为什么",devops 仓库持有在役标志值,两者各自独立更新;解除 Jaeger 钉版要等 Grafana 数据源迁移这一 devops 侧步骤。devops 侧按 issue #134 正文交付三个 playbook,`-deleteAuthKey` 与任何网络层收紧在该侧保持开放。存储变更后的遥测到达验证仍属管道自身的 no-data 告警(telemetry-gap,依[观测接线整修 MRFC](2026-10-04-observability-wiring-overhaul.zh.md)的告警归属边界)——本仓库不为其添加任何按存储的监控。
