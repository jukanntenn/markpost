# MRFC: Adopt the OTLP observability stack on the NAS and retire the files-only pipeline

English | [中文](2026-09-19-otlp-observability-stack.zh.md)

Status: proposed

## Problem

The observability pipeline was deliberately built files-only: [the observability spec](../../../specs/backend/observability.md) pins a hard constraint that all three pillars land on disk as JSONL, analyzed with `jq`. That interface serves AI agents and single-service development well, but two forces break the premise:

1. markpost entered SaaS operations. Human operators need dashboards, alerting, and service-level views; JSONL files serve neither. The operators and the agents have different consumption interfaces — humans get a UI, agents get APIs — and the files-only design provided only the second.
2. The observability infrastructure becomes shared: business systems beyond markpost will send telemetry to the same stack. It must run within hard limits — a 2-core/4GB RAM NAS with a 128GB SSD, reached only through a 3Mbps public relay — and must avoid SQLite-class stores whose single-writer model corrupts under concurrent multi-writer load. Deciding this architecture late is expensive; the stores and query languages chosen now are what later migration costs.

## Proposal

Keep the OTel SDK instrumentation untouched (otelgin spans, business metrics, the hand-written slog trace handler); replace the exporters, not the instrumentation:

- **Traces**: `stdouttrace` → `otlptracehttp`; **metrics**: `stdoutmetric` → `otlpmetrichttp`; both OTLP/HTTP + gzip, endpoint and headers from configuration.
- **Logs**: slog keeps writing the local timberjack file AND ships through an OTLP log pipeline (`otelslog` bridge, which extracts trace context from ctx exactly like the current handler). Dual-write: the file stays as the crash channel, the API-bound copy is what gets searched.
- **Stack on the NAS** (docker compose): `otelcol-contrib` as the single front door (per-service bearer-token auth, `memory_limiter`, fan-out) → Jaeger v2 single binary with embedded badger storage (traces), VictoriaMetrics `vmsingle` (metrics, OTLP on :8428), VictoriaLogs (logs, OTLP on :9428, `trace_id` auto-indexed); Grafana as the only human UI, backed by a small dedicated PostgreSQL instance (never SQLite — Grafana's `conf/defaults.ini` offers mysql/postgres/sqlite3; postgres matches team experience).
- **Files disposition**:
  - `traces-*.jsonl` / `metrics-*.jsonl`: retire in production. The loadtest/capacity stack keeps the stdout exporters so [`scripts/loadtest/capacity/analyze.py`](../../../scripts/loadtest/capacity/analyze.py) — the only programmatic consumer of the file formats — is unchanged.
  - `app-*.jsonl`: keep as the local crash channel, retention 30d → 7d; the searchable copy lives in VictoriaLogs.
  - Historical files: no backfill (Jaeger/badger is not a backfill target); they age out under existing retention, still jq-readable during the transition.
- **Agent interface**: shifts from jq-over-files to the backends' HTTP JSON APIs — the Prometheus querying API on VictoriaMetrics, `/select/logsql/query` on VictoriaLogs, the Jaeger REST API. A future markpost-mcp `observability` toolset wraps these three; no such tool exists today.
- **Spec rewrite**: [the observability spec](../../../specs/backend/observability.md) is rewritten in both languages; its files-only hard constraint is superseded by this MRFC. The [cache-purge observability MRFC](../implemented/2026-09-03-cache-purge-observability.md) has its file-reading facts updated when the implementation lands.
- **Resource envelopes** (4GB budget, ≈2GB headroom): collector ~100MB (capped by `memory_limiter`), VictoriaMetrics ~200-400MB (`-memory.allowedPercent` tuned down from the 60% default), Jaeger ~300-500MB, VictoriaLogs ~100-150MB, Grafana + PostgreSQL ~300-500MB, OS ~300-400MB. Disk caps per store: VictoriaMetrics `-retentionPeriod`, VictoriaLogs `-retentionPeriod` + `-retention.maxDiskSpaceUsageBytes`, Jaeger badger `ttl`.

How telemetry crosses the public relay to reach the NAS — transport, exposure, and authentication — is decided by the companion frp transport and exposure MRFC, layered directly above this one in the [#102](https://github.com/jukanntenn/markpost/issues/102) RFC stack.

## Alternatives considered

**Keep the files-only pipeline.** Zero new infrastructure, and the agent interface already worked. Lost because it cannot serve human operators at any reasonable cost: building dashboards over JSONL means building a visualization product, and the SaaS operations need is now, not eventually. The constraint was right for its era; the era changed.

**SigNoz (all-in-one APM).** One UI for traces/metrics/logs/dashboards/alerts/exceptions, MIT-licensed community parts, native OTLP — the strongest feature fit. Lost on the envelope and the operations model: the full deployment is ClickHouse + ZooKeeper + signoz-otel-collector + query service + frontend, which does not fit 4GB (ClickHouse alone typically claims 1-2GB); its docker-compose install path was deprecated in favor of the vendor's Foundry installer with self-managed rollback (`deploy/README.md`, `deploy/MIGRATION.md` in SigNoz/signoz, explored 2026-09-19); and its legacy metastore was SQLite, conflicting with the no-SQLite constraint.

**Prometheus + Loki (all-de-facto-standard composable stack).** Every component a CNCF graduate or Grafana-ecosystem standard. Lost on the resource envelope: Prometheus's own docs caution that its OTLP receiver is for low-volume use cases, and VictoriaMetrics — Apache-2.0, biweekly releases — implements the Prometheus querying API and PromQL, so Grafana dashboards and alert rules stay portable between them. The API, not the binary, is the standard markpost depends on; choosing the lighter implementation of the same API keeps the migration cost near zero if it ever has to be reversed. Loki stays the documented fallback for logs if VictoriaLogs' youth becomes a problem.

**Grafana Tempo for traces.** The Grafana-native OTLP trace backend. Lost because its design centers on object storage — a 4GB box would have to add and feed a MinIO service — while Jaeger v2 (itself an OpenTelemetry Collector distribution) ingests OTLP natively on 4317/4318 and persists to embedded badger with zero additional services.

**Younger all-in-ones (Uptrace, OpenObserve, HyperDX).** Rejected before deep exploration: none is a community-adopted standard, and the operator required officially-adopted or de-facto-standard projects under active maintenance as a hard criterion.

## Acceptance criteria

Mapped from [#102](https://github.com/jukanntenn/markpost/issues/102):

- markpost's three pillars reach Jaeger / VictoriaMetrics / VictoriaLogs over OTLP and are queryable in Grafana, correlated by `trace_id` (logs↔traces↔metrics).
- Files disposition shipped as proposed: app logs dual-written with 7d local retention, traces/metrics files gone in production, loadtest file mode intact (`analyze.py` unchanged).
- The NAS compose stack runs within the RAM/disk envelopes with per-store caps configured.
- [The observability spec](../../../specs/backend/observability.md) rewritten in both languages; the files-only constraint superseded by a pointer to this MRFC.
- Transport and exposure criteria are owned by the transport MRFC layering above this one in the [#102](https://github.com/jukanntenn/markpost/issues/102) stack.

## Risks

- **VictoriaLogs is young** (first release 2023-06) and LogsQL differs from Loki's LogQL: the switching cost is the query language, not the data (logs re-ship from files). If VL stalls, Loki replaces it under the same Grafana surface; the documented fallback keeps this reversible.
- **Grafana is AGPL-3.0**: running it unmodified, internal-only, triggers no obligations; embedding or modifying its code in markpost would. Never embed; treat as a black-box container.
- **Trace volume grows linearly with services** under `ParentBased(AlwaysOn)`: the sampling config slot is already reserved in the spec; the escalation path is collector-side tail sampling, then Jaeger remote/adaptive sampling.
- **badger on the NAS disk**: fine on SSD; the risk is unbounded growth, bounded by badger `ttl` and `max_traces`.
