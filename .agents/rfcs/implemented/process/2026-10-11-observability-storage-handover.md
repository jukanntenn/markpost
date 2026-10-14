# RFC: Observability storage handover to the external devops repo

Status: implemented

English | [中文](2026-10-11-observability-storage-handover.zh.md)

## Problem

The three observability stores on the NAS (192.168.5.57) — Jaeger (traces), VictoriaMetrics `vmsingle` (metrics), VictoriaLogs (logs) — are shared platform infrastructure under [the OTLP observability stack MRFC](2026-09-19-otlp-observability-stack.md): multiple business systems send telemetry to them, and the external operators run them from handoff material. The three stores' configurations carry zero markpost-specific content — no secrets, no tenant config, no `.env` (audited); every value in them is platform-level: image pins, data paths, retention, disk-space protection.

That fact poses two questions the stack MRFC does not answer. First, a repository home: configuration standardization for operator-owned platform services belongs with the operator, and a tenant repository carrying platform copies means every tenant grows a drifting copy of configuration it does not own. Second, a rationale record: the configurations embed decisions with future value — the Jaeger 2.20.0 pin, the disk-space protection flags — and without a written record the pin reads as stale and the flags as arbitrary. markpost needs those facts preserved even though the configuration itself lives elsewhere.

## Decision

The stores' configuration standardization and repository reclamation live entirely in the external devops ops repo, as one-service-per-playbook Ansible playbooks: `ansible/victoriametrics.yml`, `ansible/victorialogs.yml`, `ansible/jaeger.yml`. The [issue #134](https://github.com/jukanntenn/markpost/issues/134) body is the content specification for the devops side.

markpost is a pure telemetry producer. This repository carries no storage configurations — no playbooks, no pins, no storage secrets. The producer side and its query consumption ([the observability spec](../../../../specs/backend/observability.md)) are the only markpost-owned surfaces; the transport chain stays external under [the frp transport MRFC](2026-09-19-otlp-transport-frp-exposure.md).

The stores and versions the playbooks pin: Jaeger 2.20.0 (traces), VictoriaMetrics v1.151.0 (`vmsingle`, metrics), VictoriaLogs v1.52.0 (logs). Publishing surfaces stay `0.0.0.0` (8428 / 9428 / 16686) under the single-operator home-LAN trust model.

This record is the storage half of the observability ownership redraw; the collector handover ([#133](https://github.com/jukanntenn/markpost/issues/133)) is its sibling and revises the stack MRFC's wording in its own change.

## Storage pins and protection flags

What the playbooks hold and why, verified against the upstream sources on 2026-10-11; operational deployment lives in the devops repo:

- **Jaeger stays pinned at 2.20.0.** Upstream 2.21.0 ships the breaking change "Feat(query)!: remove v1 http endpoints the ui no longer calls" (PR #9260): the v1 REST API underpins both markpost's agent query surface (`/api/traces`, [the observability spec](../../../../specs/backend/observability.md)) and Grafana's built-in Jaeger datasource. Pin-lift condition: re-evaluate after the Grafana datasource migration. The `jaegertracing/jaeger:2.20.0` image runs as USER 10001, and the badger data directory's ownership follows it.
- **VictoriaMetrics runs `-storage.minFreeDiskSpaceBytes=1GB`** — the protection valve on the shared 128GB SSD; the upstream default is 100MB, and below the threshold the storage switches itself to read-only and stops accepting new data. The rest of the flag form: `storageDataPath`, `retentionPeriod=90d`, `memory.allowedBytes=300MB`, `selfScrapeInterval=10s`.
- **VictoriaLogs runs `-storage.minFreeDiskSpaceBytes=1GB`** (upstream default 10MB) **and `-retention.maxDiskSpaceUsageBytes=20GiB`** (10GiB → 20GiB; the upstream default is 0 — no byte cap). The 20GiB cap binds alongside the 30d retention, whichever hits first, dropping the oldest per-day partitions automatically; 20GiB is ~2x headroom for tenant growth, ≈16% of the SSD.
- **Devops-side follow-ups**, owned by that repo and recorded here so the boundary does not lose them: VictoriaLogs `-deleteAuthKey` guards the destructive `/delete/*` endpoints (OSS-supported); further tightening of the unauthenticated 8428 / 9428 / 16686 surfaces, if ever needed, happens at the network layer.

## Alternatives considered

**Keep the three stores' configs inside the markpost repo.** Lost on the audited fact at the center of this record: the configurations carry zero markpost-specific content, so repository ownership would not follow the operational owner — and every tenant repository would grow its own copy of platform configuration, each drifting against the operator's truth.

**A single combined observability playbook in the devops repo.** One playbook for all three stores is one file to review. Lost on lifecycle independence: Jaeger, VictoriaMetrics, and VictoriaLogs release on deliberately different cadences, and one-service-per-playbook keeps each upgrade's blast radius to one service — mirroring the per-service compose form the stack MRFC already settled for the NAS.

**Jaeger 2.21.0 upgrade now.** Staying current beats carrying a pin. Lost on the consumption surface: 2.21.0 removes the v1 REST endpoints that Grafana's built-in Jaeger datasource and the spec's agent query surface still depend on; the upgrade lands only together with the Grafana datasource migration.

## Consequences

Bought: platform configuration follows the operator — one home, the devops ops repo, for every store playbook — while tenant repositories stay pure producers; the pin and protection-flag rationale survives in this record instead of dissolving into another repo's YAML. markpost's query interfaces are unchanged: Prometheus querying API, LogsQL, Jaeger REST ([the observability spec](../../../../specs/backend/observability.md)).

Cost and obligations: the storage rationale is split across two repos — this record holds the why, the devops repo holds the live flag values, and the two update independently; lifting the Jaeger pin waits on the Grafana datasource migration, a devops-side step. The devops side delivers the three playbooks specified by the issue #134 body, with `-deleteAuthKey` and any network-layer tightening open there. Telemetry-arrival verification after storage changes remains the pipeline's own no-data alert (telemetry-gap, per the alert ownership boundary in [the observability wiring overhaul MRFC](2026-10-04-observability-wiring-overhaul.md)) — this repo carries no per-store monitor for it.
