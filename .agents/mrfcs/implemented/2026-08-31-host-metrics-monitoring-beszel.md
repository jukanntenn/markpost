# MRFC: Host metrics monitoring via Beszel

Status: implemented

English | [中文](2026-08-31-host-metrics-monitoring-beszel.zh.md)

## Problem

The availability layer ([availability-monitoring MRFC](./2026-08-30-availability-monitoring.md)) detects failures within ~4 minutes but explains nothing: a red readiness probe cannot say whether the origin ran out of disk, memory, or CPU. Nothing on ttyo records host or container resource history or raises threshold alerts before an outage — the classic single-VPS killers (disk filling up, memory exhaustion, sustained saturation) stay invisible until they become the outage that kuma then reports. The markpost container's Docker healthcheck only gates local restarts, and Postgres is invisible off its Unix socket.

## Decision

Beszel (MIT, Go, PocketBase + embedded SQLite, no external database) carries the host-metrics layer, split across two hosts so the monitor never dies with the monitored (#61):

- The agent runs on the production host ttyo as its own compose project at `~/docker/beszel-agent`, rendered by [`deploy.yml`](../../../devops/ansible/deploy.yml) from [`beszel-agent-compose.yml.j2`](../../../devops/ansible/templates/beszel-agent-compose.yml.j2): image pinned via `beszel_agent_version` in `group_vars/production/vars.yml`, host networking, read-only `docker.sock`. It collects host CPU/memory/disk/load/network and per-container stats for the markpost and postgres containers, and reaches the hub by outbound WebSocket (`HUB_URL`); the install tasks run only when `beszel_hub_url` is defined — the heartbeat's setup-order contract.
- The hub runs on a separate, operator-managed server — never on the monitored host. Its lifecycle is ops work outside this repository; the runbook ([`docs/monitoring.md`](../../../docs/monitoring.md)) carries the ops checklist.
- Threshold alerts (dual warning/critical) on disk, memory, CPU, load, bandwidth, and agent/system status notify through Feishu as primary channel, matching the availability layer's convention; the alert inventory and thresholds live in [`docs/monitoring.md`](../../../docs/monitoring.md).
- Scope boundary: availability probing, the reverse heartbeat, and certificate expiry stay with uptime-kuma. Beszel carries metrics and threshold alerting only; host death remains the kuma push-heartbeat's signal (silence), with the off-host hub additionally reporting the agent offline.

## Alternatives considered

- **Komari** (Go probe panel, ~18 MB agent RSS). Rejected: no Docker container stats — declined upstream as not planned (komari-agent issue #65) — which is precisely what markpost's containerized deployment needs; its agent ships a web terminal and batch execution and carries a C2-abuse disclosure history (CVE-2025-55300), too hot for the production origin; built-in notification channels are narrower (Telegram/Bark/SMTP/ServerChan/webhook only — Feishu would ride a generic webhook).
- **Co-locating the hub on ttyo with the agent.** Rejected on review: hub and host die together, so the resource layer goes dark exactly when it is needed most and resource history dies with the box; an off-host hub additionally turns agent silence into an offline alert.
- **Netdata.** Rejected for this layer despite the strongest engine (per-second collection, hundreds of built-in alerts, ML anomaly detection): official typical footprint is 250–350 MB RAM against Beszel's 128–256 Mi budget, and its depth serves fleet observability a single-box service cannot use.
- **Platform stacks** (Prometheus + Alertmanager + Grafana, VictoriaMetrics, Zabbix, HertzBeat, Nightingale, Coroot). Rejected on weight alone for a 1–2 GB VPS: 1–2 GB+ combined RAM (Zabbix's official "small" profile is 8 GiB; HertzBeat's JVM wants 4 GB), several with mandatory external databases.
- **Extending uptime-kuma instead of a second tool** (its docker/host monitor types). Rejected: kuma's docker monitor needs Docker exposed to its external vantage, already rejected in the availability MRFC for attack-surface reasons, and kuma has no host-metrics agent or resource-threshold alerting at all.

## Consequences

Bought: the "why" behind every red availability monitor — host and per-container resource curves with threshold alerts that fire before an outage — at a 128–256 Mi budget and zero new inbound ports; host death leaves the hub alive to report the agent offline, and history survives the outage. Cost: the hub host and the ttyo→hub route join the dependency chain (a hub outage silences threshold alerting while the availability layer stays independent); the agent's read-only `docker.sock` grants metadata visibility should it be compromised (a socket-proxy container is the recorded escalation); Beszel is 0.x under a single primary maintainer, so versions pin deliberately; minute-level granularity can miss sub-minute spikes. Verification: `ansible-playbook --syntax-check` and the template render gates cover the automation; activation follows the runbook's setup order — once ops sets `beszel_hub_url` + `beszel_agent_key` and deploys, `docker compose -f ~/docker/beszel-agent/docker-compose.yml ps` shows the agent running, the hub's system page shows live data, and a test alert with its recovery notice arrives on the Feishu channel.
