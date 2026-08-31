# MRFC: Host metrics monitoring via Beszel

Status: proposed

[English](2026-08-31-host-metrics-monitoring-beszel.md) | 中文

## Problem

可用性层（[availability-monitoring MRFC](../implemented/2026-08-30-availability-monitoring.zh.md)）能在 ~4 分钟内发现故障，但解释不了故障：readiness 探针变红说不出源站究竟是磁盘满了、内存耗尽还是 CPU 饱和。ttyo 上没有任何组件记录主机与容器的资源历史、或在故障发生前发出阈值告警——单台小 VPS 的经典杀手（磁盘写满、内存耗尽、持续饱和）在酿成事故前完全不可见，直到变成 kuma 随后上报的那次宕机。markpost 容器的 Docker healthcheck 只管本地重启，Postgres 在 Unix socket 之外不可见。

## Proposal

补第二层、与可用性层互补：Beszel（MIT、Go、PocketBase + 内嵌 SQLite、无外部数据库），跨两台主机拆分，让监控不死在被监控者手里（#61）：

- agent 运行在生产主机 ttyo 上。它采集主机指标（CPU、内存、磁盘、负载、网络、温度——在可暴露处），并经只读 `docker.sock` 挂载采集 markpost 与 postgres 两个容器的 CPU/内存/网络统计；它以出站 WebSocket 连接 hub（`HUB_URL`，agent 默认端口 45876），防火墙零新增放行。
- hub 运行在独立的、运维管理的服务器上——绝不同驻被监控主机。主机死亡时 hub 幸存并把 agent 静默上报为离线，资源历史也在故障后留存；主机死亡本身仍由 kuma push 心跳的静默负责。hub 的生命周期是运维事务，在本仓库自动化范围之外。
- 阈值告警（warning/critical 双阈值）覆盖磁盘、内存、CPU、负载、带宽与 agent/系统状态，主通知渠道用飞书，与可用性层的约定一致；告警清单与阈值落在 [`docs/monitoring.zh.md`](../../../docs/monitoring.zh.md)。
- 范围边界：可用性拨测、反向心跳、证书到期仍归 uptime-kuma。Beszel 只承载指标与阈值告警。

部署拓扑与仓库的自动化边界由本栈后续的 topology MRFC 决定。

## Alternatives considered

- **Komari**（Go 探针面板，agent RSS ~18 MB）。落选：不支持 Docker 容器级统计——上游明确拒绝（komari-agent issue #65）——而这恰是 markpost 容器化部署的核心诉求；其 agent 自带 Web 终端与批量执行能力，且有被滥用作 C2 的披露史（CVE-2025-55300），对生产源站过热；内置通知渠道更窄（仅 Telegram/Bark/SMTP/ServerChan/webhook——飞书只能走通用 webhook）。
- **hub 与 agent 同驻 ttyo。** 评审中落选：hub 与主机同死，资源层恰在最需要时熄火，资源历史也随机器消失；异置的 hub 还能把 agent 静默转为离线告警。因此 hub 由运维管理、独立于仓库自动化，放独立服务器。
- **Netdata**。本层落选，尽管引擎最强（秒级采集、数百条内置告警、ML 异常检测）：官方典型占用 250–350 MB RAM，对比 Beszel 128–256 Mi 的预算；其深度服务的是单机用不上的舰队级可观测。若事后复盘需要秒级粒度再重议。
- **平台型全家桶**（Prometheus + Alertmanager + Grafana、VictoriaMetrics、Zabbix、HertzBeat、夜莺、Coroot）。在 1–2 GB VPS 上仅重量一项即出局：合计 1–2 GB+ 内存（Zabbix 官方"小型"规格 8 GiB；HertzBeat 的 JVM 要 4 GB 起），且多数强制外接数据库。可用性与资源两类故障由 kuma + Beszel 以零头开销覆盖。
- **扩展 uptime-kuma 而非引入第二个工具**（其 docker/host 监控类型）。落选：kuma 的 docker 监控要求把 Docker 暴露给外部探针——可用性 MRFC 已因攻击面拒绝；且 kuma 根本没有主机指标 agent 与资源阈值告警。

## Acceptance criteria

- agent 运行于 ttyo（镜像钉版、ansible 管理），向异置 hub 上报主机指标与两个容器的统计；hub 本身由运维手工部署运维——仓库对其零自动化。
- 阈值告警配好（磁盘/内存/CPU/负载 + agent/系统状态）；测试告警与恢复通知均送达飞书渠道。
- `docs/monitoring.zh.md` 及英文对新增指标层小节：告警清单、阈值、渠道、安装顺序（agent 自动化、hub 运维清单）、移除。
- ttyo 防火墙规则不变（前后 `ufw status` 完全一致）——agent 连接是出站。

## Risks

- Beszel 还是 0.x、单一主维护者；版本刻意钉住，升级是显式动作并记录在运维手册。
- 被攻破的 agent 可经其只读 `docker.sock` 读取容器元数据——只读挡得住写入、挡不住可见性；socket-proxy 容器是记录在案的升级路径。
- hub 主机与 ttyo→hub 路径进入依赖链：hub 故障使阈值告警静默（可用性层保持独立），路径故障以 agent 离线告警浮出。
- 分钟级历史粒度可能漏掉亚分钟尖峰；接受——酿成事故的尖峰会变成宕机，而宕机归可用性层管。
