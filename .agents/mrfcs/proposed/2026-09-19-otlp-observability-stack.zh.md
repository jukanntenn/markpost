# MRFC: 在 NAS 上采纳 OTLP 观测栈并退役仅文件管道

[English](2026-09-19-otlp-observability-stack.md) | 中文

Status: proposed

## Problem

可观测管道当初是刻意设计为仅文件的：[可观测 spec](../../../specs/backend/observability.zh.md) 把"三支柱全部落盘为 JSONL、用 `jq` 分析"定为硬约束。这个接口很好地服务了 AI agent 与单服务开发，但两个变化打破了前提：

1. markpost 进入 SaaS 运营阶段。人类运营人员需要仪表盘、告警与服务级视图，JSONL 文件两者都给不了。运营人员和 agent 的消费接口不同——人看 UI，agent 用 API——仅文件设计只提供了后者。
2. 可观测基建变为共享：markpost 之外的业务系统将向同一套栈发送遥测。它必须运行在硬限制内——2核4G 内存、128GB SSD 的 NAS，且只能经 3Mbps 公网中继到达——并必须避开 SQLite 这类单写者模型、在并发多写负载下会出问题的存储。这类架构决策拖到后期再定的代价极高：现在选定的存储与查询语言就是将来的迁移成本。

## Proposal

保持 OTel SDK 埋点不动（otelgin span、业务指标、手写 slog trace handler）；替换的是导出器，不是埋点：

- **Traces**：`stdouttrace` → `otlptracehttp`；**metrics**：`stdoutmetric` → `otlpmetrichttp`；均走 OTLP/HTTP + gzip，endpoint 与 headers 进配置。
- **Logs**：slog 继续写本地 timberjack 文件，同时经 OTLP 日志管道外发（`otelslog` bridge，它从 ctx 提取 trace 上下文的行为与现有 handler 完全一致）。双写：文件是崩溃通道，API 侧副本才是被检索的。
- **NAS 上的栈**：`otelcol-contrib` 作为唯一前门（按业务发放 bearer token 认证、`memory_limiter`、扇出）→ Jaeger v2 单二进制 + 内嵌 badger 存储（traces）、VictoriaMetrics `vmsingle`（metrics，OTLP 在 :8428）、VictoriaLogs（logs，OTLP 在 :9428，`trace_id` 自动建索引）；Grafana 作为唯一人类 UI，后端用一个小型专用 PostgreSQL 实例（绝不用 SQLite——Grafana 的 `conf/defaults.ini` 支持 mysql/postgres/sqlite3 三选一，postgres 与团队经验一致）。部署形态：每个服务一个 compose project，位于 `docker/<service>/docker-compose.yml`，全部接入同一个共享外部 docker 网络——网络是唯一的共享底座；跨服务不共享任何卷，各存储独占自己的数据目录。
- **文件处置**：
  - `traces-*.jsonl` / `metrics-*.jsonl`：生产环境退役。压测/容量栈保留 stdout 导出器，使 [`scripts/loadtest/capacity/analyze.py`](../../../scripts/loadtest/capacity/analyze.py)——文件格式唯一的程序化消费方——零改动。
  - `app-*.jsonl`：保留为本地崩溃通道，保留期 30d → 7d；可检索副本进 VictoriaLogs。
  - 历史文件：不回灌（Jaeger/badger 不是回填目标）；按现有保留策略自然过期，过渡期内仍可 jq。
- **Agent 接口**：从 jq 读文件转为后端的 HTTP JSON API——VictoriaMetrics 的 Prometheus querying API、VictoriaLogs 的 `/select/logsql/query`、Jaeger REST API。后续 markpost-mcp 增加 `observability` toolset 包装这三个 API；当前并不存在该工具。
- **Spec 改写**：[可观测 spec](../../../specs/backend/observability.zh.md) 中英双份同步改写，"仅文件"硬约束由本 MRFC supersede。[缓存清理可观测 MRFC](../implemented/2026-09-03-cache-purge-observability.zh.md) 中"从文件读取"相关事实在实施落地时一并更新。
- **资源 envelopes**（4GB 预算，留 ≈2GB 余量）：collector ~100MB（`memory_limiter` 封顶）、VictoriaMetrics ~200-400MB（`-memory.allowedPercent` 从默认 60% 下调）、Jaeger ~300-500MB、VictoriaLogs ~100-150MB、Grafana + PostgreSQL ~300-500MB、操作系统 ~300-400MB。磁盘按存储封顶：VictoriaMetrics `-retentionPeriod`、VictoriaLogs `-retentionPeriod` + `-retention.maxDiskSpaceUsageBytes`、Jaeger badger `ttl`。

遥测如何穿越公网中继到达 NAS——传输、暴露与认证——由配套的 frp 传输与暴露 MRFC 决定，该层直接叠加在本层之上，同属 [#102](https://github.com/jukanntenn/markpost/issues/102) 的 RFC 栈。

## Alternatives considered

**维持仅文件管道。** 零新增基础设施，agent 接口已经可用。败因是无法以合理成本服务人类运营：在 JSONL 上建仪表盘等于自研可视化产品，而运营需求是现在时的。该约束在其时代是对的；时代变了。

**SigNoz（一体化 APM）。** 单 UI 覆盖 traces/metrics/logs/仪表盘/告警/异常，社区部分 MIT，原生 OTLP——功能契合度最高。败在 envelope 与运维模型：完整部署是 ClickHouse + ZooKeeper + signoz-otel-collector + query service + 前端，4GB 放不下（ClickHouse 自身通常要 1-2GB）；其 docker-compose 安装路径已被废弃、改走厂商 Foundry 安装器且回滚自管（SigNoz/signoz 的 `deploy/README.md`、`deploy/MIGRATION.md`，2026-09-19 勘察）；旧版 metastore 是 SQLite，与无 SQLite 约束冲突。

**Prometheus + Loki（全事实标准组合栈）。** 每个组件都是 CNCF 毕业项目或 Grafana 生态标准。败在资源 envelope：Prometheus 官方文档明确其 OTLP receiver 仅建议低流量场景；而 VictoriaMetrics（Apache-2.0，双周发版）实现了 Prometheus querying API 与 PromQL，Grafana 仪表盘与告警规则在两者间可移植。markpost 依赖的标准是 API 而非二进制——选同一 API 的更轻实现，把"将来要换回"的迁移成本压到近零。Loki 作为日志的书面后备：若 VictoriaLogs 的年轻成为问题即切换。

**Grafana Tempo 做 traces。** Grafana 生态原生的 OTLP trace 后端。败因是设计以对象存储为中心——4GB 的机器还得再养一个 MinIO 服务；而 Jaeger v2（本身就是 OpenTelemetry Collector 发行版）原生在 4317/4318 收 OTLP，内嵌 badger 持久化，零额外服务。

**更年轻的一体化方案（Uptrace、OpenObserve、HyperDX）。** 深度探索前即被否：没有一个是社区采纳的标准，而运营方明确要求"官方采纳或事实标准、维护活跃"作为硬标准。

**整个 NAS 栈用一个 compose 文件。** 单一 compose project 白送默认网络、跨服务 `depends_on` 与健康检查门控、一条命令起全栈。败在生命周期耦合：六个发布节奏刻意不同的服务（VictoriaMetrics 约双周、Grafana 约月度、Jaeger 约 6 周）将共享每一次 `pull`/`up`/`down` 和每一个 YAML 笔误的全量爆炸半径，而它的真实优势都有替代品——共享外部网络替代默认网络，`restart: unless-stopped` 加导出器 retry/queue 语义替代启动排序。（"任何变更都重启全部"略有夸大——`up -d` 按服务 diff——但 project 级生命周期耦合是真实的。）审阅者特别问到的每服务形态可行性：服务间互联是一次 `docker network create` 加每个文件里的 `external: true` 默认网络，Docker 内嵌 DNS 跨 project 解析服务名；数据共享按设计为零——不共享任何卷，各存储数据目录私有，无可协调之物。于本层 review 中提出。

## Acceptance criteria

映射自 [#102](https://github.com/jukanntenn/markpost/issues/102)：

- markpost 三支柱经 OTLP 到达 Jaeger / VictoriaMetrics / VictoriaLogs，可在 Grafana 查询并按 `trace_id` 关联（logs↔traces↔metrics）。
- 文件处置按提案落地：app 日志双写、本地保留 7d，生产环境 traces/metrics 文件消失，压测文件模式不变（`analyze.py` 零改动）。
- NAS compose 栈在 RAM/磁盘 envelope 内运行，各存储封顶配置生效。
- [可观测 spec](../../../specs/backend/observability.zh.md) 中英双份改写，"仅文件"约束改为指向本 MRFC 的 supersede 说明。
- 传输与暴露的验收条件由叠加于本层之上的传输 MRFC 承担（同属 [#102](https://github.com/jukanntenn/markpost/issues/102) 栈）。

## Risks

- **VictoriaLogs 年轻**（首发 2023-06）且 LogsQL 不同于 Loki 的 LogQL：切换成本是查询语言而非数据（日志可从文件重发）。若 VL 停滞，Loki 在同一 Grafana 面下替换之；书面化的后备方案保持了可逆性。
- **Grafana 是 AGPL-3.0**：未修改、仅内部使用不触发开源义务；嵌入或修改其代码则会。永不嵌入，按黑盒容器对待。
- **trace 体量随业务数线性增长**（`ParentBased(AlwaysOn)` 下）：采样配置槽在 spec 中已预留；升级路径是 collector 侧 tail sampling，再到 Jaeger 远程/自适应采样。
- **badger 在 NAS 盘上**：SSD 无虞；风险是无界增长，由 badger `ttl` 与 `max_traces` 封顶。
- **跨 project 启动排序**：`depends_on` 不跨 compose project。首次启动排序（PostgreSQL 先于 Grafana）与一次性的外部网络引导步骤随实施写入运维 runbook；稳态依赖重启策略加推送侧 retry/queue 语义。
