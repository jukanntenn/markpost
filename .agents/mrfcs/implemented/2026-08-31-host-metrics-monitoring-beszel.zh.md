# MRFC: Host metrics monitoring via Beszel

Status: implemented

[English](2026-08-31-host-metrics-monitoring-beszel.md) | 中文

## Problem

可用性层（[availability-monitoring MRFC](./2026-08-30-availability-monitoring.zh.md)）能在 ~4 分钟内发现故障，但解释不了故障：readiness 探针变红说不出源站究竟是磁盘满了、内存耗尽还是 CPU 饱和。ttyo 上没有任何组件记录主机与容器的资源历史、或在故障发生前发出阈值告警——单台小 VPS 的经典杀手（磁盘写满、内存耗尽、持续饱和）在酿成事故前完全不可见，直到变成 kuma 随后上报的那次宕机。markpost 容器的 Docker healthcheck 只管本地重启，Postgres 在 Unix socket 之外不可见。

## Decision

Beszel（MIT、Go、PocketBase + 内嵌 SQLite、无外部数据库）承载主机指标层，跨两台主机拆分，让监控不死在被监控者手里（#61）：

- agent 运行在生产主机 ttyo 上，独立 compose 项目位于 `~/docker/beszel-agent`，由 [`deploy.yml`](../../../devops/ansible/deploy.yml) 从 [`beszel-agent-compose.yml.j2`](../../../devops/ansible/templates/beszel-agent-compose.yml.j2) 渲染：镜像经 `group_vars/production/vars.yml` 的 `beszel_agent_version` 钉版、host 网络、只读 `docker.sock`。它采集主机 CPU/内存/磁盘/负载/网络与 markpost、postgres 两容器的统计，经出站 WebSocket（`HUB_URL`）连 hub；安装任务仅在 `beszel_hub_url` 已定义时运行——沿用心跳的配置顺序契约。
- hub 运行在独立的、运维管理的服务器上——绝不同驻被监控主机。其生命周期是本仓库之外的运维事务；运维手册（[`docs/monitoring.zh.md`](../../../docs/monitoring.zh.md)）承载运维清单。
- 阈值告警（warning/critical 双阈值）覆盖磁盘、内存、CPU、负载、带宽与 agent/系统状态，主通知渠道用飞书，与可用性层的约定一致；告警清单与阈值落在 [`docs/monitoring.zh.md`](../../../docs/monitoring.zh.md)。
- 范围边界：可用性拨测、反向心跳、证书到期仍归 uptime-kuma。Beszel 只承载指标与阈值告警；主机死亡仍由 kuma push 心跳的静默负责，异置 hub 额外把 agent 离线上报出来。

## Alternatives considered

- **Komari**（Go 探针面板，agent RSS ~18 MB）。落选：不支持 Docker 容器级统计——上游明确拒绝（komari-agent issue #65）——而这恰是 markpost 容器化部署的核心诉求；其 agent 自带 Web 终端与批量执行能力，且有被滥用作 C2 的披露史（CVE-2025-55300），对生产源站过热；内置通知渠道更窄（仅 Telegram/Bark/SMTP/ServerChan/webhook——飞书只能走通用 webhook）。
- **hub 与 agent 同驻 ttyo。** 评审中落选：hub 与主机同死，资源层恰在最需要时熄火，资源历史也随机器消失；异置的 hub 还能把 agent 静默转为离线告警。
- **Netdata**。本层落选，尽管引擎最强（秒级采集、数百条内置告警、ML 异常检测）：官方典型占用 250–350 MB RAM，对比 Beszel 128–256 Mi 的预算；其深度服务的是单机用不上的舰队级可观测。
- **平台型全家桶**（Prometheus + Alertmanager + Grafana、VictoriaMetrics、Zabbix、HertzBeat、夜莺、Coroot）。在 1–2 GB VPS 上仅重量一项即出局：合计 1–2 GB+ 内存（Zabbix 官方"小型"规格 8 GiB；HertzBeat 的 JVM 要 4 GB 起），且多数强制外接数据库。
- **扩展 uptime-kuma 而非引入第二个工具**（其 docker/host 监控类型）。落选：kuma 的 docker 监控要求把 Docker 暴露给外部探针——可用性 MRFC 已因攻击面拒绝；且 kuma 根本没有主机指标 agent 与资源阈值告警。

## Consequences

买到：每个红色可用性监控项背后的"为什么"——主机与容器资源曲线、事故前触发的阈值告警——预算 128–256 Mi、入站端口零新增；主机死亡时 hub 幸存上报 agent 离线，历史跨故障留存。代价：hub 主机与 ttyo→hub 路径进入依赖链（hub 故障使阈值告警静默，可用性层保持独立）；agent 的只读 `docker.sock` 一旦被攻破即暴露容器元数据可见性（socket-proxy 容器是记录在案的升级路径）；Beszel 还是 0.x、单一主维护者，版本刻意钉住；分钟级粒度可能漏掉亚分钟尖峰。验证：`ansible-playbook --syntax-check` 与模板渲染门禁覆盖自动化；激活按运维手册的配置顺序——运维设好 `beszel_hub_url` + `beszel_agent_key` 并部署后，`docker compose -f ~/docker/beszel-agent/docker-compose.yml ps` 显示 agent 运行中、hub 的 system 页出现实时数据、测试告警与恢复通知送达飞书渠道。
