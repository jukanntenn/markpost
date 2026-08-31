# MRFC: Host metrics monitoring via Beszel

Status: proposed

English | [中文](2026-08-31-host-metrics-monitoring-beszel.zh.md)

## Problem

The availability layer ([availability-monitoring MRFC](../implemented/2026-08-30-availability-monitoring.md)) detects failures within ~4 minutes but explains nothing: a red readiness probe cannot say whether the origin ran out of disk, memory, or CPU. Nothing on ttyo records host or container resource history or raises threshold alerts before an outage — the classic single-VPS killers (disk filling up, memory exhaustion, sustained saturation) stay invisible until they become the outage that kuma then reports. The markpost container's Docker healthcheck only gates local restarts, and Postgres is invisible off its Unix socket.

## Proposal

A second, complementary layer: Beszel (MIT, Go, PocketBase + embedded SQLite, no external database) runs on the production host ttyo as a hub + agent pair (#61):

- The agent collects host metrics (CPU, memory, disk, load, network, temperatures where exposed) and per-container CPU/memory/network stats for the markpost and postgres containers via a read-only `docker.sock` mount; it reaches the hub by outbound WebSocket (default agent port 45876), so the firewall opens nothing new.
- The hub co-locates on ttyo. Resource budget is the official Helm line — agent 128 Mi / hub 256 Mi requests — the layer must fit alongside markpost on a small VPS.
- Threshold alerts (dual warning/critical) on disk, memory, CPU, load, bandwidth, and system status notify through Feishu as primary channel, matching the availability layer's convention; the monitor inventory and thresholds live in [`docs/monitoring.md`](../../../docs/monitoring.md).
- Scope boundary: availability probing, the reverse heartbeat, and certificate expiry stay with uptime-kuma. Beszel carries metrics and threshold alerting only. A co-located hub dying with the host is accepted: host death is the kuma push-heartbeat's signal (silence), not the local hub's.

Deployment topology, exposure, and secrets are decided by the follow-up topology MRFC in this stack.

## Alternatives considered

- **Komari** (Go probe panel, ~18 MB agent RSS). Rejected: no Docker container stats — declined upstream as not planned (komari-agent issue #65) — which is precisely what markpost's containerized deployment needs; its agent ships a web terminal and batch execution and carries a C2-abuse disclosure history (CVE-2025-55300), too hot for the production origin; built-in notification channels are narrower (Telegram/Bark/SMTP/ServerChan/webhook only — Feishu would ride a generic webhook).
- **Netdata.** Rejected for this layer despite the strongest engine (per-second collection, hundreds of built-in alerts, ML anomaly detection): official typical footprint is 250–350 MB RAM against Beszel's 128–256 Mi budget, and its depth serves fleet observability a single-box service cannot use. Revisit if post-mortems ever need per-second granularity.
- **Platform stacks** (Prometheus + Alertmanager + Grafana, VictoriaMetrics, Zabbix, HertzBeat, Nightingale, Coroot). Rejected on weight alone for a 1–2 GB VPS: 1–2 GB+ combined RAM (Zabbix's official "small" profile is 8 GiB; HertzBeat's JVM wants 4 GB), several with mandatory external databases. The two failure classes — availability and resources — are covered by kuma + Beszel at a fraction of that footprint.
- **Extending uptime-kuma instead of a second tool** (its docker/host monitor types). Rejected: kuma's docker monitor needs Docker exposed to its external vantage, already rejected in the availability MRFC for attack-surface reasons, and kuma has no host-metrics agent or resource-threshold alerting at all.

## Acceptance criteria

- Hub + agent run on ttyo (pinned image, ansible-managed) and report host metrics plus both containers' stats.
- Threshold alerts configured (disk / memory / CPU / load + system status); a test alert and its recovery notice arrive on the Feishu channel.
- `docs/monitoring.md` and its zh pair gain the metrics-layer section: alert inventory, thresholds, channel, setup order, removal.
- Firewall rules unchanged (`ufw status` identical before and after).

## Risks

- Beszel is 0.x under a single primary maintainer; versions pin deliberately, and upgrades are conscious acts recorded in the runbook.
- A compromised hub reads container metadata through `docker.sock` — read-only limits mutation, not visibility; the topology layer gates exposure, and a socket-proxy container is the recorded escalation path.
- Minute-level history granularity can miss sub-minute spikes; accepted — spikes that matter become outages, and outages are the availability layer's job.
