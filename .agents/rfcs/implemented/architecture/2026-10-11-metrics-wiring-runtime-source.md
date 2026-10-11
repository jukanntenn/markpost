# RFC: Metrics wiring completion, contrib runtime source, and a generated metrics inventory

Status: implemented

English | [中文](2026-10-11-metrics-wiring-runtime-source.zh.md)

## Problem

The metrics surface had a declared-but-unwired hole and a hand-rolled legacy:

- `markpost.auth.login_success_total` / `login_failure_total` / `token_refresh_total` were declared in `internal/observability/metrics.go` but no code ever incremented them — the auth service had no metrics port at all.
- Go runtime metrics came from a hand-rolled block (`registerRuntimeMetrics`): four `process.runtime.go.*` gauges sampled by a `runtime.ReadMemStats` callback. ReadMemStats is a stop-the-world-adjacent call the OTel project deliberately avoids; the instrument names followed no convention; and the sampling missed everything the modern `runtime/metrics` API exposes (memory limit, GC goal, processor limit).
- Delivery latency was unobservable: the queue is enqueue-then-scan (attempt rows claimed by a ticker), so no request duration span ever covers the enqueue→dispatch gap, and retries stretch that gap to 36 minutes.
- The metric inventory was a hand-maintained table in [observability.md](../../../../specs/backend/observability.md) with no mechanism tying it to the code — the three dead auth counters proved it could not flag drift.
- `Dispatcher.Enqueue` fabricated `context.Background()`, dropping the creating request's trace context on the enqueue reads/writes — a standing violation of the ctx-form rule.

## Decision

Everything below ships as of this record; paths and names are kept current with the code.

### Runtime source migration

`go.opentelemetry.io/contrib/instrumentation/runtime` **v0.71.0** — the contrib release whose go.mod requires exactly the `otel v1.46.0` family markpost already pins, so no core dependency moved — is started by `observability.Init` (`internal/observability/otel.go`, `startRuntimeMetrics`) immediately after `otel.SetMeterProvider`, in both OTLP and file mode; the file-mode loadtest/capacity pipeline is unchanged apart from the richer instrument set. `Start` resolves its meter from the global provider at call time, which is why the call must follow the provider installation. The hand-rolled `registerRuntimeMetrics` block and all `process.runtime.go.*` instruments are deleted. A start failure is logged (`slog.Error`) and swallowed: exporter-construction failures in `Init` remain fatal per the existing convention, but a runtime-metrics registration failure must not kill the app — the process serves fine without gauges.

`OTEL_GO_X_DEPRECATED_RUNTIME_METRICS` is never set anywhere, so the deprecated `runtime.go.*` family is not produced and the deleted `process.runtime.go.*` names simply disappear.

**The GC-observability tradeoff.** The new-gen `go.*` family exposes heap pressure directly (`go.memory.used` with `memory.type`, `go.memory.limit`, `go.memory.allocated`, `go.memory.allocations`, `go.memory.gc.goal`) plus `go.goroutine.count`, `go.processor.limit`, and `go.config.gogc`. Direct GC frequency and pause reads (`NumGC`, pause ns) are given up: the new conventions carry no GC-count/pause instruments, and the upstream migration guidance is that GC behavior is inferred from heap-pressure signals (goal vs allocated). Accepted: markpost has no GC-tuning workflow, and heap pressure is the actionable signal for a request-serving Go process. The escalation path for a future need is NOT the deprecated env var — it is a bespoke observable instrument over `runtime/metrics` (`/gc/cycles:total`, `/sync/mutex/wait:seconds`), added deliberately.

### Auth counters

The auth service gains the standard metrics port (local `Metrics` interface + `noopMetrics` + `WithMetrics` option, mirroring the post and delivery services), adapted by `authMetrics` in `cmd/server/metrics_adapters.go` and wired in `main.go`. Wiring points:

- `LoginWithEmail`: every error return counts one failure — invalid credentials **and** the account-locked return (a locked rejection is a failed login from the caller's perspective, even with a correct password); success counts one success.
- `LoginWithGitHub`: every error return counts one failure, success one success. Early malformed-request rejections (missing state/code) count too — "failure on every failed return" is the rule that can't drift from the branch list.
- `RefreshToken`: `token_refresh_total` counts **completed refreshes only**. Rejected replays, theft detection, and expired/invalid tokens count nothing: those are security signals with their own log lines, not refreshes; the counter measures rotation volume.

Increments live at the three public entry points, never inside `completeLogin`, so a token refresh never inflates the login counters. All calls use the flow's ctx (the ctx-form rule).

### `markpost.delivery.duration`

A `Float64Histogram`, unit **`s`** (the OTel semantic-conventions duration unit, and what otelgin's `http.server.request.duration` uses), observed at the top of `Dispatcher.execute` — the dispatch point — as `now − Attempt.CreatedAt`. `CreatedAt` is the **persisted enqueue timestamp** set when the attempt row is created, so the measurement survives restarts and needs no in-memory map. **Every dispatch attempt observes, retries included**: a retried delivery legitimately lands 6/16/36 minutes out (the backoff sequence is 1m/5m/10m/20m), and that visibility — how far retries drift, whether the queue is backing up past its walls — is the point of the instrument. The explicit bucket set `0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300, 600, 1200, 2400, 3600` covers the fast path (sub-tick to a few seconds) through the full retry ladder, topped above the 40-minute expiry wall; the SDK default buckets stop at 10s and would push every retried delivery into +Inf. Tests inject the clock through the dispatcher's `now` field.

### Generated metrics inventory

[`docs/metrics.md`](../../../../docs/metrics.md) (and its zh pair) is generated by `backend/tools/metricsdoc`, wired via `//go:generate` in `internal/observability/gen.go` so `go generate ./...` regenerates it. The tool parses `internal/observability/metrics.go` and `internal/middleware/active_requests.go` with go/ast for instrument declarations (name, kind, unit, description, buckets), `cmd/server/metrics_adapters.go` for the method→instrument mapping and attribute keys, and the packages under `internal/` (golang.org/x/tools/go/packages) for service-layer call sites — producing a table of name, type, unit, description, attributes, adapter method, and call-site file:line. **Instruments with zero call sites are marked dead** — the visibility that motivated the generated doc. The otelgin and contrib-runtime sections are **version-anchored**: the tool reads the pinned versions from go.mod and refuses to run against a version missing from its verified tables, so a dependency bump without re-verification fails `go generate` instead of documenting stale facts. Output is deterministic and network-free; the optional `--live <vm-base-url>` flag adds a supplementary presence column reconciled against VictoriaMetrics `/api/v1/metadata` and is never used by gates.

The drift gate lives in the existing `backend-generate-check` pre-commit hook (backend/prek.toml): its `go generate ./... && git diff --exit-code` scope now also covers the repo-root `docs/metrics.*` files, and its `files` trigger gained `go.mod`/`go.sum` so version bumps re-run it. One home was picked deliberately: the backend hook already owns the Go toolchain invocation and the generator is a backend module tool; a root-level hook (mirroring the ansible-templates check) would duplicate that machinery for files the same command produces. CI runs `uvx prek run --all-files`, so drift fails CI through this hook. The pair is registered in `.hdsh/pairing.manifest.json`'s `generated` list (English source exempt from the language-switcher line so it stays byte-identical to generator output; the zh counterpart still links back), and the tool writes the `.i18n.yaml` record itself with computed git blob hashes — byte-identical to `hdsh pairing record` output.

The hand-maintained table in [observability.md](../../../../specs/backend/observability.md) is replaced by a pointer to the generated doc (one fact, one home); the spec keeps the rules that are rules — naming style, attribute-free cache/purge counters, the first-data-point caveat — and its runtime row now points at the generated doc and this record.

### Enqueue ctx threading

`domainpost.DeliveryEnqueuer.Enqueue` now takes `ctx`, and `Dispatcher.Enqueue` uses it instead of fabricating `context.Background()`; `post.CreatePost` passes its request ctx through. The job still outlives the request — only the enqueue-time repo reads and the pending-counter increment ride the request's trace context.

## Alternatives considered

**Keep the hand-rolled runtime metrics.** Zero new dependency and total control of names — but ReadMemStats sampling is the exact anti-pattern the contrib package exists to replace, the four gauges covered a fraction of `runtime/metrics`, and the names followed no convention the stores recognize.

**Set `OTEL_GO_X_DEPRECATED_RUNTIME_METRICS=true` for the `runtime.go.*` family.** Would have restored GC-count/pause names close to the old ones — but it re-imports the deprecated conventions the migration exists to retire, doubles the emitted series, and is explicitly documented as transitional upstream. Rejected outright; see the GC tradeoff above for the sanctioned escalation path.

**Observe delivery duration with an in-memory enqueue timestamp.** Avoids depending on `CreatedAt`'s meaning — but loses the measurement across restarts (the queue's whole point is surviving them), needs a map with eviction, and the persisted timestamp is already there and already drives the expiry wall.

**Keep the hand-maintained inventory table.** Cheaper than a generator — and provably unable to flag drift: three dead auth counters sat in a "current" table for weeks. The AST scan is ~600 lines of tooling that turns the inventory into a compilation artifact.

**A Prometheus-client-based metrics stack.** The ecosystem default for RED/USE dashboards — but markpost is already on the OTel SDK with OTLP and file exporters in both modes; a second metrics stack would fork the pipeline, the semantic conventions, and the loadtest tooling for no capability gained.

## Consequences

The auth funnel, delivery latency distribution, and the full `go.*` runtime family are now queryable in both pipelines, and the inventory can no longer lie about what is wired — a dead instrument is a visible table cell and a gate-checked regeneration away from being caught. The costs: two new direct dependencies (`go.opentelemetry.io/contrib/instrumentation/runtime`, plus promoting the already-indirect `golang.org/x/tools`); one more generated artifact to keep in the commit discipline; dashboards referencing the deleted `process.runtime.go.*` names break (none exist — the loadtest analyzer does not read them); and GC frequency/pause are no longer directly readable, an accepted trade recorded above. The enqueue ctx threading touches a domain interface, so future implementers of `DeliveryEnqueuer` inherit the ctx contract — which is the rule, not a cost.
