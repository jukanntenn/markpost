# Observability

English | [中文](observability.zh.md)

The three observability pillars (Logs / Traces / Metrics) in one specification. Logging, as one pillar, is described here together with traces and metrics.

## Tech Stack and Hard Constraints

### Hard constraints

**All three pillars export to the local filesystem; no external services** (no Jaeger / Prometheus / Loki / OTLP collector). Every observability artifact (logs / spans / metrics) lands on disk as JSONL files, analyzed with `jq`.

### Route A: slog + a hand-written trace Handler

The logging pillar uses `log/slog` (Go standard library) plus a hand-written slog Handler that pulls the trace_id from ctx into every log entry; **the OTel Logs SDK is not used**.

**`slog-otel` is not used** (inactive maintenance) — a hand-written ~20-line slog Handler implements the trace↔log correlation instead. This decision record is kept.

### The three pillars on disk

| Pillar      | Collection                                                                              | On disk                                                        |
| ----------- | --------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| **Logs**    | `log/slog`, hand-written Handler injecting trace_id/span_id from ctx into every entry   | timberjack → `app-*.jsonl`                                     |
| **Traces**  | OTel Go SDK + `otelgin.Middleware` (automatic HTTP spans) + manual business child spans | `stdouttrace.New(WithWriter(timberjack))` → `traces-*.jsonl`   |
| **Metrics** | OTel Go metric SDK (counter/gauge/histogram) + automatic runtime collection             | `stdoutmetric.New(WithWriter(timberjack))` → `metrics-*.jsonl` |

### Technical feasibility basis

- timberjack's `Logger` implements `io.Writer` (the `Write(p []byte)` method in `timberjack.go`), so it serves directly as the output sink for logs and exporters.
- All three OTel stdout exporters offer a `WithWriter(io.Writer)` option (`stdouttrace/config.go`, `stdoutmetric/config.go`, `stdoutlog/config.go`), so a timberjack instance feeds an exporter directly, giving traces and metrics their own independently rotating files.

## File Layout and Rotation

### The three-file model

```
/app/data/logs/
├── app-2026-07-14.jsonl          business events + HTTP access + errors (slog)
├── app-2026-07-14T00-00-00.000-time.jsonl.zst   midnight rotation archive
├── traces-2026-07-14.jsonl       OTel spans
├── metrics-2026-07-14.jsonl      OTel metric data points
└── ...
```

The three files chain together through `trace_id`: spot an anomaly in app → read the call chain in traces → check the metrics of that moment.

### timberjack rotation config (hybrid strategy, shared by all three files)

| Setting            | Value                       | Purpose                                                               |
| ------------------ | --------------------------- | --------------------------------------------------------------------- |
| `RotateAt`         | `["00:00"]`                 | Rotate at midnight daily (primary)                                    |
| `MaxSize`          | 100 MB                      | Mid-day fallback cut on incident days (keeps single files bounded)    |
| `MaxBackups`       | 14                          | Keep 14 old files (about two weeks)                                   |
| `MaxAge`           | 30                          | Delete past 30 days (stricter of MaxBackups/MaxAge wins)              |
| `Compression`      | `"zstd"`                    | zstd-compress old files                                               |
| `BackupTimeFormat` | `"2006-01-02T15-04-05.000"` | Millisecond format; avoids name collisions on a second size-based cut |

**Hybrid strategy explained**: midnight rotation is the primary trigger, but if a day's logs surge (an incident storm), the 100 MB cap cuts a second file mid-day, giving that day 2 files. `BackupTimeFormat`'s millisecond precision keeps the second size-based cut from colliding names. If "strictly one file per day" matters more, drop MaxSize and rotate purely by date — at the cost of single-file size control.

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

### trace↔log correlation (the hand-written slog Handler)

A hand-written ~20-line slog Handler pulls trace information from `ctx` into every log entry:

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

API: `trace.SpanFromContext(ctx).SpanContext()` → `.TraceID()` / `.SpanID()` (from `go.opentelemetry.io/otel/trace`).

## Traces (OTel)

### Automatic spans (the otelgin middleware)

`otelgin.Middleware(serviceName, opts...)` registers as middleware and creates a span for every HTTP request automatically:

```go
r.Use(otelgin.Middleware("markpost"))
```

Recorded automatically: HTTP method, path, status code, latency.

### Manual child spans

Business-critical operations open child spans with `tracer.Start(ctx, "operation.name")`:

| Operation                                                | Span name                          |
| -------------------------------------------------------- | ---------------------------------- |
| DB write transactions (post creation, delivery dispatch) | `post.Create`, `delivery.Dispatch` |
| Markdown rendering                                       | `post.RenderHTML`                  |
| The delivery scheduling loop                             | `delivery.Schedule`                |
| External calls (OAuth callback to GitHub)                | `auth.GitHubCallback`              |

Child spans inherit the parent's trace_id through `trace.SpanFromContext(ctx)`, forming call chains. **On error, record an error attribute on the span**: `span.SetStatus(codes.Error, msg); span.RecordError(err)`.

### Sampling policy

`ParentBased(AlwaysOn)` — sample everything by default.

Rationale: a single service with no cross-service propagation, so the traces file stays a manageable size. If QPS grows, switching to `ParentBased(TraceIDRatioBased(0.1))` is one line (a config slot is reserved).

## Metrics (OTel)

### Reader

`PeriodicReader(stdoutmetricExporter, metric.WithInterval(60*time.Second))` — exports to the metrics file every 60 seconds.

### Naming style

OTel semantic conventions (semconv), dot-separated like `http.server.request.duration` — **not** the underscore style (`http_request_duration_seconds`).

### Metric inventory

The metrics adopted today, extended as needed:

| Layer    | Metric                               | Type      | Labels               | Purpose                                                        |
| -------- | ------------------------------------ | --------- | -------------------- | -------------------------------------------------------------- |
| HTTP     | `http.server.request.duration`       | histogram | method, path, status | Per-endpoint performance (otelgin automatic + additions)       |
| HTTP     | `http.server.active_requests`        | gauge     | —                    | In-flight request count                                        |
| Business | `markpost.posts.created_total`       | counter   | —                    | Posts created                                                  |
| Business | `markpost.auth.login_success_total`  | counter   | —                    | Successful logins                                              |
| Business | `markpost.auth.login_failure_total`  | counter   | —                    | Failed logins                                                  |
| Business | `markpost.auth.token_refresh_total`  | counter   | —                    | Token refresh count                                            |
| Business | `markpost.delivery.pending`          | gauge     | —                    | Pending dispatch count                                         |
| Business | `markpost.delivery.dispatched_total` | counter   | —                    | Dispatched count                                               |
| Business | `markpost.delivery.failed_total`     | counter   | error_category       | Dispatch failures (by reason)                                  |
| Business | `markpost.render_cache.hit_total`    | counter   | —                    | Render requests served from the render cache                   |
| Business | `markpost.render_cache.miss_total`   | counter   | —                    | Render requests that missed and entered the singleflight path  |
| Business | `markpost.cdn.purge_success_total`   | counter   | —                    | CDN cache-tag purges completed (HTTP < 300)                    |
| Business | `markpost.cdn.purge_failure_total`   | counter   | —                    | CDN purge attempts failed (marshal/build/transport/HTTP ≥ 300) |
| Business | `markpost.cdn.purge_skipped_total`   | counter   | —                    | CDN purges not attempted (no-op purger/unconfigured)           |
| System   | runtime metrics                      | —         | —                    | OTel Go runtime auto-collection (goroutines, GC, memory)       |

The five render-cache/CDN-purge counters are attribute-free — one series per outcome, hit rate and purge attempts derivable by aggregation (decision record: [the cache/purge observability MRFC](../../.agents/mrfcs/implemented/2026-09-03-cache-purge-observability.md); reading them against `CF-Cache-Status`: [`caching.md`](./caching.md)).

### Log correlation fields

Every business log entry carries `trace_id` and `span_id` automatically, plus business fields where applicable (`user_id`, `post_id`, etc.).

## Initialization Wiring (cmd/server/main.go)

At startup, in order:

1. **Create the three timberjack Loggers** (app / traces / metrics) with the rotation settings
2. **Construct the exporters**:
   - `stdouttrace.New(stdouttrace.WithWriter(appTracesLogger))`
   - `stdoutmetric.New(stdoutmetric.WithWriter(appMetricsLogger))`
3. **Wire the providers**:
   - `sdktrace.NewTracerProvider/sdktrace.WithBatcher(traceExporter)` → `otel.SetTracerProvider`
   - `sdkmetric.NewMeterProvider/sdkmetric.WithReader(metric.NewPeriodicReader(metricExporter))` → `otel.SetMeterProvider`
4. **Register the otelgin middleware**: `r.Use(otelgin.Middleware("markpost"))`
5. **Install the hand-written slog Handler** (injects trace_id), `slog.SetDefault`
6. **Graceful shutdown**: `Shutdown(ctx)` flushes the exporters + `Close()` on the three timberjack loggers

## Output Format

All three files are JSON Lines (JSONL), one JSON object per line, analyzable with `jq`:

```bash
# join the three files by trace_id
jq 'select(.trace_id=="a1b2c3d4...")' /app/data/logs/app-*.jsonl
jq 'select(.trace_id=="a1b2c3d4...")' /app/data/logs/traces-*.jsonl
jq 'select(.trace_id=="a1b2c3d4...")' /app/data/logs/metrics-*.jsonl
```

The stdout exporters emit JSON by default; stdoutmetric's output is verbose (one line per data point), so the metrics file is larger than traces. This is the accepted default format.
