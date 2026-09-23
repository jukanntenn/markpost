# MRFC: 在 NAS 上采纳 OTLP 观测栈并退役仅文件管道

[English](2026-09-19-otlp-observability-stack.md) | 中文

Status: implemented

## Problem

可观测管道当初是刻意设计为仅文件的：[可观测 spec](../../../specs/backend/observability.zh.md) 曾把"三支柱全部落盘为 JSONL、用 `jq` 分析"定为硬约束。这个接口很好地服务了 AI agent 与单服务开发，但两个变化打破了前提：

1. markpost 进入 SaaS 运营阶段。人类运营人员需要仪表盘、告警与服务级视图，JSONL 文件两者都给不了。运营人员和 agent 的消费接口不同——人看 UI，agent 用 API——仅文件设计只提供了后者。
2. 可观测基建变为共享：markpost 之外的业务系统向同一套栈发送遥测。它必须运行在硬限制内——2核4G 内存、128GB SSD 的 NAS，且只能经 3Mbps 公网中继到达——并必须避开 SQLite 这类单写者模型、在并发多写负载下会出问题的存储。这类架构决策拖到后期再定的代价极高：现在选定的存储与查询语言就是将来的迁移成本。

## Decision

markpost 是外部部署观测栈的**消费方**；仓库只交付生产方一侧。观测栈——由外部运维方按 [frp 传输 MRFC](./2026-09-19-otlp-transport-frp-exposure.zh.md) 记录的交付材料部署运维——为 Jaeger v2（traces、内嵌 badger）、VictoriaMetrics `vmsingle`（metrics）、VictoriaLogs（logs）、Grafana + 专用 PostgreSQL 后端，前置一个 `otelcol-contrib`（按业务发放 bearer token 认证、memory_limiter、扇出）。其选型理由与资源 envelope 见下文 Alternatives；其部署形态（每服务一个 compose project 于 `docker/<service>/`、单一共享外部 docker 网络、跨服务零共享卷）是交付材料而非仓库交付物。

仓库内的生产方一侧（`backend/internal/observability/otel.go`）：

- **导出器由环境驱动、双模式**：设置 `OTEL_EXPORTER_OTLP_ENDPOINT` 后，traces 走 `otlptracehttp`、metrics 走 `otlpmetrichttp`（60s PeriodicReader）、logs 经 `otelslog` bridge 走 `otlploghttp`；未设置时 stdout 导出器照旧写 timberjack JSONL 文件——压测/容量栈沿用该模式，`scripts/loadtest/capacity/analyze.py` 零改动。
- **OTLP 模式下 app 日志双写**：默认 slog logger 扇出到 timberjack 文件 handler（崩溃通道）与 OTLP 日志管道；所有请求作用域的日志调用一律 `*Context` 形式，两个通道都携带 trace 关联。
- **Agent 消费从 jq 读文件转为各存储的 HTTP JSON API**（Prometheus querying API、LogsQL、Jaeger REST）；[可观测 spec](../../../specs/backend/observability.zh.md) 已改写为当前状态，"仅文件"硬约束由本记录 supersede。
- **三套环境按环境接线**（`devops/ansible`）：`markpost-dev` 经内网直连 collector，`markpost-staging` 与 `markpost` 推送公网 OTLP 入口；各环境 bearer token 已 vault（`otel_otlp_token`），compose 模板在 endpoint 与 token 未同时定义时回落文件模式。

## Alternatives considered

**维持仅文件管道。** 零新增基础设施，agent 接口已经可用。败因是无法以合理成本服务人类运营：在 JSONL 上建仪表盘等于自研可视化产品，而运营需求是现在时的。该约束在其时代是对的；时代变了。

**SigNoz（一体化 APM）。** 单 UI 覆盖 traces/metrics/logs/仪表盘/告警/异常，社区部分 MIT，原生 OTLP——功能契合度最高。败在 envelope 与运维模型：完整部署是 ClickHouse + ZooKeeper + signoz-otel-collector + query service + 前端，4GB 放不下（ClickHouse 自身通常要 1-2GB）；其 docker-compose 安装路径已被废弃、改走厂商 Foundry 安装器且回滚自管（SigNoz/signoz 的 `deploy/README.md`、`deploy/MIGRATION.md`，2026-09-19 勘察）；旧版 metastore 是 SQLite，与无 SQLite 约束冲突。

**Prometheus + Loki（全事实标准组合栈）。** 每个组件都是 CNCF 毕业项目或 Grafana 生态标准。败在资源 envelope：Prometheus 官方文档明确其 OTLP receiver 仅建议低流量场景；而 VictoriaMetrics（Apache-2.0，双周发版）实现了 Prometheus querying API 与 PromQL，Grafana 仪表盘与告警规则在两者间可移植。markpost 依赖的标准是 API 而非二进制——选同一 API 的更轻实现，把"将来要换回"的迁移成本压到近零。Loki 作为日志的书面后备：若 VictoriaLogs 的年轻成为问题即切换。

**Grafana Tempo 做 traces。** Grafana 生态原生的 OTLP trace 后端。败因是设计以对象存储为中心——4GB 的机器还得再养一个 MinIO 服务；而 Jaeger v2（本身就是 OpenTelemetry Collector 发行版）原生在 4317/4318 收 OTLP，内嵌 badger 持久化，零额外服务。

**更年轻的一体化方案（Uptrace、OpenObserve、HyperDX）。** 深度探索前即被否：没有一个是社区采纳的标准，而运营方明确要求"官方采纳或事实标准、维护活跃"作为硬标准。

**整个 NAS 栈用一个 compose 文件。** 单一 compose project 白送默认网络、跨服务 `depends_on` 与健康检查门控、一条命令起全栈。败在生命周期耦合：六个发布节奏刻意不同的服务（VictoriaMetrics 约双周、Grafana 约月度、Jaeger 约 6 周）将共享每一次 `pull`/`up`/`down` 和每一个 YAML 笔误的全量爆炸半径，而它的真实优势都有平替——共享外部网络替代默认网络，`restart: unless-stopped` 加导出器 retry/queue 语义替代启动排序。（"任何变更都重启全部"略有夸大——`up -d` 按服务 diff——但 project 级生命周期耦合是真实的。）审阅者特别问到的每服务形态可行性：服务间互联是一次 `docker network create` 加每个文件里的 `external: true` 默认网络，Docker 内嵌 DNS 跨 project 解析服务名；数据共享按设计为零——不共享任何卷，各存储数据目录私有，无可协调之物。于本记录的 review 中提出。

## Consequences

人类运营获得带环境选择器的 Grafana 仪表盘；AI agent 获得三个查询 API，能力严格强于 jq 读文件（时间范围、过滤、`trace_id` 索引）。观测栈在 envelope 内服务多个业务系统，以 `service.name` 区分。

随之落地的代价与义务：

- **VictoriaLogs 年轻**（首发 2023-06）且 LogsQL 不同于 Loki 的 LogQL：切换成本是查询语言而非数据（日志可从文件重发）。Loki 是同一 Grafana 面下的书面后备。
- **Grafana 是 AGPL-3.0**：未修改、仅内部使用不触发开源义务；按黑盒容器对待，永不嵌入、永不修改。
- **trace 体量随业务数线性增长**（`ParentBased(AlwaysOn)` 下）：采样配置槽保持预留；升级路径是 collector 侧 tail sampling。
- **spec 中的手写子 span**（`post.Create`、`post.RenderHTML`……）仍未实现——每条 trace 恰好只有一个 otelgin server span；spec 现已如实陈述，补 span、削减崩溃通道文件保留期、摄取断流告警均为后续工作（rollout 期间一次 env 失配导致的断流对所有人不可见，正因 SDK 侧失败是静默的）。
- **栈的运维质量在外部运维方手里**：本仓库的控制手段是交付材料与黑盒验收（401/400 认证状态、数据源健康）——rollout 期间已验证——而非部署本身。
