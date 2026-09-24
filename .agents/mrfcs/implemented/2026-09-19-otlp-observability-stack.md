# MRFC: Adopt the OTLP observability stack on the NAS and retire the files-only pipeline

English | [中文](2026-09-19-otlp-observability-stack.zh.md)

Status: implemented

## Problem

The observability pipeline was deliberately built files-only: [the observability spec](../../../specs/backend/observability.md) pinned a hard constraint that all three pillars land on disk as JSONL, analyzed with `jq`. That interface serves AI agents and single-service development well, but two forces broke the premise:

1. markpost entered SaaS operations. Human operators need dashboards, alerting, and service-level views; JSONL files serve neither. The operators and the agents have different consumption interfaces — humans get a UI, agents get APIs — and the files-only design provided only the second.
2. The observability infrastructure becomes shared: business systems beyond markpost send telemetry to the same stack. It must run within hard limits — a 2-core/4GB RAM NAS with a 128GB SSD, reached only through a 3Mbps public relay — and must avoid SQLite-class stores whose single-writer model corrupts under concurrent multi-writer load. Deciding this architecture late is expensive; the stores and query languages chosen now are what later migration costs.

## Decision

markpost is a **consumer** of an externally deployed observability stack; the repo ships the producer side only. The stack — deployed and operated by the external operators from handoff material recorded in the [frp transport MRFC](./2026-09-19-otlp-transport-frp-exposure.md) — is Jaeger v2 (traces, embedded badger), VictoriaMetrics `vmsingle` (metrics), VictoriaLogs (logs), and Grafana with a dedicated PostgreSQL backend, fronted by one `otelcol-contrib` (per-service bearer-token auth, memory_limiter, fan-out). Its selection rationale and resource envelopes live in Alternatives below; its deployment form (one compose project per service under `docker/<service>/`, one shared external docker network, no cross-service volumes) is handoff material, not repo deliverables.

The producer side in this repo (`backend/internal/observability/otel.go`):

- **Exporters are environment-driven and dual-mode**: with `OTEL_EXPORTER_OTLP_ENDPOINT` set, traces ship via `otlptracehttp`, metrics via `otlpmetrichttp` (60s PeriodicReader), and logs via `otlploghttp` behind an `otelslog` bridge; without it the stdout exporters write the timberjack JSONL files unchanged — the mode the loadtest/capacity stack keeps, so `scripts/loadtest/capacity/analyze.py` is untouched.
- **App logs are dual-written** in OTLP mode: the default slog logger fans out to the timberjack file handler (the crash channel) and the OTLP logs pipeline; every request-scoped log call uses the `*Context` form so both channels carry trace correlation.
- **Agent consumption moves from jq-over-files to the stores' HTTP JSON APIs** (Prometheus querying API, LogsQL, Jaeger REST); the [observability spec](../../../specs/backend/observability.md) is rewritten to current state, superseding the files-only hard constraint with this record.
- **Three environments are wired per env** (`devops/ansible`): `markpost-dev` reaches the collector directly over the LAN, `markpost-staging` and `markpost` push to the public OTLP ingress; each env's bearer token is vaulted (`otel_otlp_token`), the compose template falls back to file mode until endpoint and token are both defined.

## Alternatives considered

**Keep the files-only pipeline.** Zero new infrastructure, and the agent interface already worked. Lost because it cannot serve human operators at any reasonable cost: building dashboards over JSONL means building a visualization product, and the SaaS operations need is now, not eventually. The constraint was right for its era; the era changed.

**SigNoz (all-in-one APM).** One UI for traces/metrics/logs/dashboards/alerts/exceptions, MIT-licensed community parts, native OTLP — the strongest feature fit. Lost on the envelope and the operations model: the full deployment is ClickHouse + ZooKeeper + signoz-otel-collector + query service + frontend, which does not fit 4GB (ClickHouse alone typically claims 1-2GB); its docker-compose install path was deprecated in favor of the vendor's Foundry installer with self-managed rollback (`deploy/README.md`, `deploy/MIGRATION.md` in SigNoz/signoz, explored 2026-09-19); and its legacy metastore was SQLite, conflicting with the no-SQLite constraint.

**Prometheus + Loki (all-de-facto-standard composable stack).** Every component a CNCF graduate or Grafana-ecosystem standard. Lost on the resource envelope: Prometheus's own docs caution that its OTLP receiver is for low-volume use cases, and VictoriaMetrics — Apache-2.0, biweekly releases — implements the Prometheus querying API and PromQL, so Grafana dashboards and alert rules stay portable between them. The API, not the binary, is the standard markpost depends on; choosing the lighter implementation of the same API keeps the migration cost near zero if it ever has to be reversed. Loki stays the documented fallback for logs if VictoriaLogs' youth becomes a problem.

**Grafana Tempo for traces.** The Grafana-native OTLP trace backend. Lost because its design centers on object storage — a 4GB box would have to add and feed a MinIO service — while Jaeger v2 (itself an OpenTelemetry Collector distribution) ingests OTLP natively on 4317/4318 and persists to embedded badger with zero additional services.

**Younger all-in-ones (Uptrace, OpenObserve, HyperDX).** Rejected before deep exploration: none is a community-adopted standard, and the operator required officially-adopted or de-facto-standard projects under active maintenance as a hard criterion.

**One compose file for the whole NAS stack.** A single compose project gives the default network for free, cross-service `depends_on` and healthcheck gating, and a one-command stack bring-up. Lost on lifecycle coupling: six services with deliberately different release cadences (VictoriaMetrics ~biweekly, Grafana ~monthly, Jaeger ~6-weekly) would share every `pull`/`up`/`down` and every YAML mistake as one blast radius, while its real advantages are replaceable — a shared external network replaces the default one, and `restart: unless-stopped` plus the exporters' retry/queue semantics replace boot ordering. (The claim that any change restarts everything overstates it — `up -d` diffs per service — but the project-level lifecycle coupling is real.) Feasibility of the per-service form, the reviewer's specific question: inter-service networking is one `docker network create` plus an `external: true` default network in each file, with Docker's embedded DNS resolving service names across projects; data sharing is nil by design — no volume is shared, the stores' data directories are private, so nothing to coordinate. Raised in review of this record.

## Consequences

Human operators get Grafana dashboards with an environment selector; AI agents get three query APIs with strictly more power than jq-over-files (time ranges, filters, indexed `trace_id`). The stack serves multiple business systems within its envelope, keyed on `service.name`.

Costs and obligations that shipped with it:

- **VictoriaLogs is young** (first release 2023-06) and LogsQL differs from Loki's LogQL: the switching cost is the query language, not the data (logs re-ship from files). Loki is the documented fallback under the same Grafana surface.
- **Grafana is AGPL-3.0**: running it unmodified, internal-only, triggers no obligations; it is treated as a black-box container, never embedded or modified.
- **Trace volume grows linearly with services** under `ParentBased(AlwaysOn)`: the sampling config slot stays reserved; collector-side tail sampling is the escalation path.
- **The spec's manual child spans** (`post.Create`, `post.RenderHTML`, …) remain unimplemented — traces carry exactly one otelgin server span per request; the spec now states this truthfully and the spans are follow-up work, as are trimming the crash-channel file retention and an ingest-absence alert (a stale-env token gap during rollout was invisible to everyone because SDK-side failures are silent).
- **The stack's operational quality sits with the external operators**: this repo's controls are the handoff material and black-box acceptance (401/400 auth states, datasource health) — verified during rollout — not the deployment itself.
