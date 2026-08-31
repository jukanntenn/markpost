# MRFC: Host metrics monitoring via Beszel

Status: proposed

English | [中文](2026-08-31-host-metrics-monitoring-beszel.zh.md)

## Problem

The availability layer ([availability-monitoring MRFC](../implemented/2026-08-30-availability-monitoring.md)) detects failures within ~4 minutes but explains nothing: a red readiness probe cannot say whether the origin ran out of disk, memory, or CPU. Nothing on ttyo records host or container resource history or raises threshold alerts before an outage — the classic single-VPS killers (disk filling up, memory exhaustion, sustained saturation) stay invisible until they become the outage that kuma then reports. The markpost container's Docker healthcheck only gates local restarts, and Postgres is invisible off its Unix socket.

## Proposal

A second, complementary layer: Beszel (MIT, Go, PocketBase + embedded SQLite, no external database), split across two hosts so the monitor never dies with the monitored (#61):

- The agent runs on the production host ttyo. It collects host metrics (CPU, memory, disk, load, network, temperatures where exposed) and per-container CPU/memory/network stats for the markpost and postgres containers via a read-only `docker.sock` mount; it reaches the hub by outbound WebSocket (`HUB_URL`, default agent port 45876), so the firewall opens nothing new.
- The hub runs on a separate, operator-managed server — never on the monitored host. Host death leaves the hub alive to report the agent offline, and resource history survives the outage; host death itself remains the kuma push-heartbeat's signal (silence). The hub's lifecycle is ops work and outside this repository's automation.
- Threshold alerts (dual warning/critical) on disk, memory, CPU, load, bandwidth, and agent/system status notify through Feishu as primary channel, matching the availability layer's convention; the alert inventory and thresholds live in [`docs/monitoring.md`](../../../docs/monitoring.md).
- Scope boundary: availability probing, the reverse heartbeat, and certificate expiry stay with uptime-kuma. Beszel carries metrics and threshold alerting only.

Deployment topology and the repo's automation boundary are decided by the follow-up topology MRFC in this stack.

## Alternatives considered

- **Komari** (Go probe panel, ~18 MB agent RSS). Rejected: no Docker container stats — declined upstream as not planned (komari-agent issue #65) — which is precisely what markpost's containerized deployment needs; its agent ships a web terminal and batch execution and carries a C2-abuse disclosure history (CVE-2025-55300), too hot for the production origin; built-in notification channels are narrower (Telegram/Bark/SMTP/ServerChan/webhook only — Feishu would ride a generic webhook).
- **Co-locating the hub on ttyo with the agent.** Rejected on review: hub and host die together, so the resource layer goes dark exactly when it is needed most and resource history dies with the box; an off-host hub additionally turns agent silence into an offline alert. The hub is therefore operator-managed on a separate server, outside the repo's automation.
- **Netdata.** Rejected for this layer despite the strongest engine (per-second collection, hundreds of built-in alerts, ML anomaly detection): official typical footprint is 250–350 MB RAM against Beszel's 128–256 Mi budget, and its depth serves fleet observability a single-box service cannot use. Revisit if post-mortems ever need per-second granularity.
- **Platform stacks** (Prometheus + Alertmanager + Grafana, VictoriaMetrics, Zabbix, HertzBeat, Nightingale, Coroot). Rejected on weight alone for a 1–2 GB VPS: 1–2 GB+ combined RAM (Zabbix's official "small" profile is 8 GiB; HertzBeat's JVM wants 4 GB), several with mandatory external databases. The two failure classes — availability and resources — are covered by kuma + Beszel at a fraction of that footprint.
- **Extending uptime-kuma instead of a second tool** (its docker/host monitor types). Rejected: kuma's docker monitor needs Docker exposed to its external vantage, already rejected in the availability MRFC for attack-surface reasons, and kuma has no host-metrics agent or resource-threshold alerting at all.

## Acceptance criteria

- The agent runs on ttyo (pinned image, ansible-managed) and reports host metrics plus both containers' stats to the off-host hub; the hub itself is deployed and operated manually by ops — no repo automation for it.
- Threshold alerts configured (disk / memory / CPU / load + agent/system status); a test alert and its recovery notice arrive on the Feishu channel.
- `docs/monitoring.md` and its zh pair gain the metrics-layer section: alert inventory, thresholds, channel, setup order (agent automated, hub ops checklist), removal.
- Firewall rules on ttyo unchanged (`ufw status` identical before and after) — the agent's connection is outbound.

## Risks

- Beszel is 0.x under a single primary maintainer; versions pin deliberately, and upgrades are conscious acts recorded in the runbook.
- A compromised agent reads container metadata through its read-only `docker.sock` — read-only limits mutation, not visibility; a socket-proxy container is the recorded escalation path.
- The hub's host and the ttyo→hub route join the dependency chain: a hub outage silences threshold alerting (the availability layer stays independent), and a route outage surfaces as agent-offline alerts.
- Minute-level history granularity can miss sub-minute spikes; accepted — spikes that matter become outages, and outages are the availability layer's job.
