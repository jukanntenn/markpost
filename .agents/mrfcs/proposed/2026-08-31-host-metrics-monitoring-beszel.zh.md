# MRFC: Host metrics monitoring via Beszel

Status: proposed

[English](2026-08-31-host-metrics-monitoring-beszel.md) | 中文

## Problem

可用性层（[availability-monitoring MRFC](../implemented/2026-08-30-availability-monitoring.zh.md)）能在 ~4 分钟内发现故障，但解释不了故障：readiness 探针变红说不出源站究竟是磁盘满了、内存耗尽还是 CPU 饱和。ttyo 上没有任何组件记录主机与容器的资源历史、或在故障发生前发出阈值告警——单台小 VPS 的经典杀手（磁盘写满、内存耗尽、持续饱和）在酿成事故前完全不可见，直到变成 kuma 随后上报的那次宕机。markpost 容器的 Docker healthcheck 只管本地重启，Postgres 在 Unix socket 之外不可见。

## Proposal

补第二层、与可用性层互补：Beszel（MIT、Go、PocketBase + 内嵌 SQLite、无外部数据库）以 hub + agent 成对运行在生产主机 ttyo 上（#61）：

- agent 采集主机指标（CPU、内存、磁盘、负载、网络、温度——在可暴露处），并经只读 `docker.sock` 挂载采集 markpost 与 postgres 两个容器的 CPU/内存/网络统计；它以出站 WebSocket 连接 hub（agent 默认端口 45876），防火墙零新增放行。
- hub 与 agent 同驻 ttyo。资源预算取官方 Helm 值——agent 128 Mi / hub 256 Mi requests——该层必须塞得进小 VPS 上 markpost 之外的余量。
- 阈值告警（warning/critical 双阈值）覆盖磁盘、内存、CPU、负载、带宽与系统状态，主通知渠道用飞书，与可用性层的约定一致；监控清单与阈值落在 [`docs/monitoring.md`](../../../docs/monitoring.zh.md)。
- 范围边界：可用性拨测、反向心跳、证书到期仍归 uptime-kuma。Beszel 只承载指标与阈值告警。hub 与主机同死是被接受的取舍：主机死亡由 kuma push 心跳的静默负责，不指望本地 hub。

部署拓扑、暴露方式与密钥流转由本栈后续的 topology MRFC 决定。

## Alternatives considered

- **Komari**（Go 探针面板，agent RSS ~18 MB）。落选：不支持 Docker 容器级统计——上游明确拒绝（komari-agent issue #65）——而这恰是 markpost 容器化部署的核心诉求；其 agent 自带 Web 终端与批量执行能力，且有被滥用作 C2 的披露史（CVE-2025-55300），对生产源站过热；内置通知渠道更窄（仅 Telegram/Bark/SMTP/ServerChan/webhook——飞书只能走通用 webhook）。
- **Netdata**。本层落选，尽管引擎最强（秒级采集、数百条内置告警、ML 异常检测）：官方典型占用 250–350 MB RAM，对比 Beszel 128–256 Mi 的预算；其深度服务的是单机用不上的舰队级可观测。若事后复盘需要秒级粒度再重议。
- **平台型全家桶**（Prometheus + Alertmanager + Grafana、VictoriaMetrics、Zabbix、HertzBeat、夜莺、Coroot）。在 1–2 GB VPS 上仅重量一项即出局：合计 1–2 GB+ 内存（Zabbix 官方"小型"规格 8 GiB；HertzBeat 的 JVM 要 4 GB 起），且多数强制外接数据库。可用性与资源两类故障由 kuma + Beszel 以零头开销覆盖。
- **扩展 uptime-kuma 而非引入第二个工具**（其 docker/host 监控类型）。落选：kuma 的 docker 监控要求把 Docker 暴露给外部探针——可用性 MRFC 已因攻击面拒绝；且 kuma 根本没有主机指标 agent 与资源阈值告警。

## Acceptance criteria

- hub + agent 运行于 ttyo（镜像钉版、ansible 管理），上报主机指标与两个容器的统计。
- 阈值告警配好（磁盘/内存/CPU/负载 + 系统状态）；测试告警与恢复通知均送达飞书渠道。
- `docs/monitoring.md` 及中文对新增指标层小节：告警清单、阈值、渠道、安装顺序、移除。
- 防火墙规则不变（前后 `ufw status` 完全一致）。

## Risks

- Beszel 还是 0.x、单一主维护者；版本刻意钉住，升级是显式动作并记录在运维手册。
- hub 一旦被攻破可经 `docker.sock` 读取容器元数据——只读挡得住写入、挡不住可见性；暴露面由 topology 层把关，socket-proxy 容器是记录在案的升级路径。
- 分钟级历史粒度可能漏掉亚分钟尖峰；接受——酿成事故的尖峰会变成宕机，而宕机归可用性层管。
