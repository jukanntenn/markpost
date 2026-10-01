# Observability

English | [中文](observability.zh.md)

The three observability pillars (Logs / Traces / Metrics) in one specification. Logging, as one pillar, is described here together with traces and metrics.

## Tech Stack and Hard Constraints

### Hard constraints

**Telemetry ships via OTLP to the externally deployed observability stack; the local filesystem is the fallback and the crash channel.** With `OTEL_EXPORTER_OTLP_ENDPOINT` set, all three pillars export OTLP/HTTP + gzip to the collector (bearer-token authenticated). Without it, the process falls back to the files-only pipeline (stdout exporters → JSONL) — the mode the loadtest/capacity stack still uses, so `scripts/loadtest/capacity/analyze.py` keeps working unchanged. The stack, its transport, and the exposure architecture are owned by [the OTLP observability stack MRFC](../../.agents/mrfcs/implemented/2026-09-19-otlp-observability-stack.md) and [the frp transport MRFC](../../.agents/mrfcs/implemented/2026-09-19-otlp-transport-frp-exposure.md); this spec describes markpost's producer side only.

### Route A: slog + a hand-written trace Handler

The logging pillar uses `log/slog` (Go standard library) plus a hand-written slog Handler that pulls the trace_id from ctx into every log entry. In OTLP mode the default logger is a fan-out: the timberjack file handler **and** an `otelslog` bridge into the OTLP logs pipeline (the bridge extracts trace context from ctx exactly like the file handler). **The OTel Logs SDK is not used directly** — only through the bridge.

**`slog-otel` is not used** (inactive maintenance) — the hand-written Handler implements the trace↔log correlation instead. This decision record is kept.

### The three pillars

| Pillar      | Collection                                                                            | OTLP mode                                         | File mode (fallback)               |
| ----------- | ------------------------------------------------------------------------------------- | ------------------------------------------------- | ---------------------------------- |
| **Logs**    | `log/slog`, hand-written Handler injecting trace_id/span_id from ctx into every entry | fan-out: timberjack file + `otelslog` → OTLP logs | timberjack → `app-*.jsonl`         |
| **Traces**  | OTel Go SDK + `otelgin.Middleware` (automatic HTTP spans)                             | `otlptracehttp` → collector                       | `stdouttrace` → `traces-*.jsonl`   |
| **Metrics** | OTel Go metric SDK (counter/gauge/histogram) + automatic runtime collection           | `otlpmetrichttp` → collector (60s PeriodicReader) | `stdoutmetric` → `metrics-*.jsonl` |

Exporter selection is purely environment-driven (`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_HEADERS` carrying the bearer token, `OTEL_EXPORTER_OTLP_COMPRESSION`, `OTEL_SERVICE_NAME`); instrumentation is identical in both modes.

## File Layout and Rotation

In OTLP mode only the app log file is written (the crash channel: evidence survives collector/tunnel outages). Traces and metrics files exist only in file mode.

```
/app/data/logs/
├── app-2026-07-14.jsonl          business events + HTTP access + errors (slog)
├── app-2026-07-14T00-00-00.000-time.jsonl.zst   midnight rotation archive
├── traces-2026-07-14.jsonl       OTel spans (file mode only)
└── metrics-2026-07-14.jsonl      OTel metric data points (file mode only)
```

### timberjack rotation config (hybrid strategy, shared by all files)

| Setting            | Value                       | Purpose                                                               |
| ------------------ | --------------------------- | --------------------------------------------------------------------- |
| `RotateAt`         | `["00:00"]`                 | Rotate at midnight daily (primary)                                    |
| `MaxSize`          | 100 MB                      | Mid-day fallback cut on incident days (keeps single files bounded)    |
| `MaxBackups`       | 14                          | Keep 14 old files (about two weeks)                                   |
| `MaxAge`           | 30                          | Delete past 30 days (stricter of MaxBackups/MaxAge wins)              |
| `Compression`      | `"zstd"`                    | zstd-compress old files                                               |
| `BackupTimeFormat` | `"2006-01-02T15-04-05.000"` | Millisecond format; avoids name collisions on a second size-based cut |

## Logs (slog)

### Log level conventions

- **Error**: unexpected errors, panics, boundary errors that are not service.Error, unknown error codes
- **Warn**: recoverable anomalies (rate limiting, degradation, retries)
- **Info**: lifecycle events (startup / shutdown / config loading), key business events (post creation, login, delivery dispatch)
- **Debug**: development-time detail, off in production by default

### When to log

- **Startup lifecycle**: config loaded / db init / server start / listening address
- **Unexpected boundary errors**: when `apierr.RespondError` meets a non-service.Error or an unknown error code, it logs **with `slog.Error` and trace fields** (not `log.Printf`)
- **panic recovery**: after the fallback middleware recovers, `slog.Error` records it (with trace_id, path, error)
- **Key business events**: post creation, login, delivery dispatch, etc., with structured fields (user_id, post_id, session_id, ...)

**Service-layer errors are not logged one by one** — logging happens at the boundary (handler / apierr) where they surface.

**Every request-scoped log call uses the `*Context` form** (`slog.InfoContext(ctx, ...)`, `s.logger().InfoContext(ctx, ...)`); ctx-less calls break trace correlation in both channels and are a defect.

### Sensitive data that is never logged

- Passwords (plaintext or hashed)
- JWT tokens (access or refresh)
- OAuth client secrets
- Post key values (in production logs)
- Full request bodies (may contain user content)

### Fatal logs

**Fatal logging is uniformly `slog.Error` + `os.Exit(1)`; `log.Fatalf` is unused.** Rationale: fatal entries land in the structured log (app.jsonl) with trace fields.

Fatal is reserved for unrecoverable startup errors (the process cannot continue):

- Config file loading failure
- Database connection failure
- Admin user initialization failure
- Trusted proxy configuration failure
- Server bind failure

### trace↔log correlation

The hand-written slog Handler pulls span context from ctx into every entry; in OTLP mode the `otelslog` bridge does the same for the exported copy. VictoriaLogs indexes `trace_id` as a first-class field, so a log line's `trace_id` resolves to the Jaeger trace directly.

```go
func (h *traceHandler) Handle(ctx context.Context, r slog.Record) slog.Record {
    spanCtx := trace.SpanContextFromContext(ctx)
    if spanCtx.IsValid() {
        r.AddAttrs(
            slog.String("trace_id", spanCtx.TraceID().String()),
            slog.String("span_id", spanCtx.SpanID().String()),
        )
    }
    return r
}
```

## Traces (OTel)

### Automatic spans (the otelgin middleware)

`otelgin.Middleware(serviceName)` registers as middleware and creates a span for every HTTP request automatically, recording HTTP method, route, status code, and latency. **These automatic HTTP spans are the only spans markpost emits today.**

### Manual child spans (reserved, not implemented)

DB-transaction, rendering, and delivery-loop child spans (`post.Create`, `post.RenderHTML`, `delivery.Schedule`, `auth.GitHubCallback`) are designed but **not implemented** — no `tracer.Start` call exists in the codebase. Until they are added, traces contain exactly one server span per request. When adding them: `tracer.Start(ctx, "operation.name")`, inherit the parent via ctx, and on error `span.SetStatus(codes.Error, msg); span.RecordError(err)`.

### Sampling policy

`ParentBased(AlwaysOn)` — sample everything by default.

Rationale: a single service with no cross-service propagation, so volume stays manageable. If QPS grows, switching to `ParentBased(TraceIDRatioBased(0.1))` is one line (a config slot is reserved); collector-side tail sampling is the escalation path after that.

## Metrics (OTel)

### Reader

`PeriodicReader(exporter, metric.WithInterval(60*time.Second))` — flushes to the collector (OTLP mode) or the metrics file (file mode) every 60 seconds.

### Naming style

OTel semantic conventions (semconv), dot-separated like `http.server.request.duration` — **not** the underscore style (`http_request_duration_seconds`). Stores may sanitize on ingestion (VictoriaMetrics keeps the dotted form).

### Metric inventory

The metrics adopted today, extended as needed:

| Layer    | Metric                               | Type      | Labels                | Purpose                                                        |
| -------- | ------------------------------------ | --------- | --------------------- | -------------------------------------------------------------- |
| HTTP     | `http.server.request.duration`       | histogram | method, route, status | Per-endpoint performance (otelgin automatic)                   |
| HTTP     | `http.server.active_requests`        | gauge     | —                     | In-flight request count                                        |
| Business | `markpost.posts.created_total`       | counter   | —                     | Posts created                                                  |
| Business | `markpost.auth.login_success_total`  | counter   | —                     | Successful logins                                              |
| Business | `markpost.auth.login_failure_total`  | counter   | —                     | Failed logins                                                  |
| Business | `markpost.auth.token_refresh_total`  | counter   | —                     | Token refresh count                                            |
| Business | `markpost.delivery.pending`          | gauge     | —                     | Pending dispatch count                                         |
| Business | `markpost.delivery.dispatched_total` | counter   | —                     | Dispatched count                                               |
| Business | `markpost.delivery.failed_total`     | counter   | error_category        | Dispatch failures (by reason)                                  |
| Business | `markpost.render_cache.hit_total`    | counter   | —                     | Render requests served from the render cache                   |
| Business | `markpost.render_cache.miss_total`   | counter   | —                     | Render requests that missed and entered the singleflight path  |
| Business | `markpost.cdn.purge_success_total`   | counter   | —                     | CDN cache-tag purges completed (HTTP < 300)                    |
| Business | `markpost.cdn.purge_failure_total`   | counter   | —                     | CDN purge attempts failed (marshal/build/transport/HTTP ≥ 300) |
| Business | `markpost.cdn.purge_skipped_total`   | counter   | —                     | CDN purges not attempted (no-op purger/unconfigured)           |
| System   | runtime metrics                      | —         | —                     | OTel Go runtime auto-collection (goroutines, GC, memory)       |

The five render-cache/CDN-purge counters are attribute-free — one series per outcome, hit rate and purge attempts derivable by aggregation (decision record: [the cache/purge observability MRFC](../../.agents/mrfcs/implemented/2026-09-03-cache-purge-observability.md); reading them against `CF-Cache-Status`: [`caching.md`](./caching.md)).

Counters produce their first data point only on first increment — dashboards and alert rules must not assume a series exists before the first business event.

### Log correlation fields

Every business log entry carries `trace_id` and `span_id` automatically, plus business fields where applicable (`user_id`, `post_id`, etc.).

## Initialization Wiring (cmd/server/main.go)

At startup, in order:

1. **Create the three timberjack Loggers** (app / traces / metrics) with the rotation settings
2. **Construct the pipeline** (`observability.Init`): OTLP mode builds `otlptracehttp` / `otlpmetrichttp` / `otlploghttp` exporters from the environment plus a shared resource (`service.name`); file mode builds the stdout exporters exactly as before
3. **Wire the providers**: tracer/meter providers → `otel.SetTracerProvider` / `SetMeterProvider`
4. **Register the otelgin middleware**: `r.Use(otelgin.Middleware("markpost"))`
5. **Install the default slog logger** (`providers.InstallSlogDefault`): file handler with trace correlation, fanned out to the OTLP logs pipeline in OTLP mode
6. **Graceful shutdown**: `Shutdown(ctx)` flushes the trace/metric/log exporters + `Close()` on the timberjack loggers

## Consuming the telemetry

The query interface is the stores' HTTP JSON APIs (also what AI agents consume), plus Grafana for humans:

```bash
# metrics — Prometheus querying API on VictoriaMetrics
curl -s 'http://<vm>:8428/api/v1/query?query=markpost.posts.created_total'
# logs — LogsQL on VictoriaLogs (trace_id is an indexed field)
curl -s 'http://<vl>:9428/select/logsql/query?query={service.name="markpost"}'
# traces — Jaeger REST API
curl -s 'http://<jaeger>:16686/api/traces?service=markpost&limit=1'
```

The file formats (file mode) remain JSONL, one JSON object per line, analyzable with `jq` during loadtest post-mortems.
