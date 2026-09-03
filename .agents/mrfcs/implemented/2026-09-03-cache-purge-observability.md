# MRFC: Cache and purge observability metrics

Status: implemented

English | [中文](2026-09-03-cache-purge-observability.zh.md)

## Problem

The read path's two caching mechanisms — the in-process render cache (ristretto, `backend/internal/service/post/cache.go`) and the best-effort Cloudflare cache-tag purge (`backend/internal/service/post/purger.go`) — emitted no metrics. When a reader reported stale content or origin load looked wrong, the only way to answer "is the cache actually working?" was to infer from response headers per request or to query the database; nothing in the observability files (`metrics-*.jsonl`, `app-*.jsonl`) recorded a cache hit, a miss, or a purge attempt, let alone its outcome (success, failure, skipped). The render cache can also be disabled by config, and nothing distinguished "disabled" from "enabled but ineffective". The observability spec's metric inventory had no row for either mechanism, so operators had no documented way to reason about cache effectiveness — including how origin-side signals relate to Cloudflare's visitor-facing `CF-Cache-Status` header.

## Decision

Five business counters, attribute-free, one per outcome, live in the OTel metric channel (meter `markpost`, exported to `metrics-*.jsonl` every 60 s) alongside the shipped naming and structure — dot-separated semconv-style names (`backend/internal/observability/metrics.go`):

| Metric                            | Type    | Counts                                                                        |
| --------------------------------- | ------- | ----------------------------------------------------------------------------- |
| `markpost.render_cache.hit_total` | counter | Render requests served from the render cache                                   |
| `markpost.render_cache.miss_total` | counter | Render requests that missed and entered the render/singleflight path          |
| `markpost.cdn.purge_success_total` | counter | Cache-tag purge requests completed with HTTP < 300                            |
| `markpost.cdn.purge_failure_total` | counter | Purge attempts that failed (marshal, request build, transport, or HTTP ≥ 300) |
| `markpost.cdn.purge_skipped_total` | counter | Purges not attempted (no-op purger or unconfigured credentials)               |

Placement:

- **Hit/miss is counted at the request decision point** — the fast-path `cache.Get` in `RenderPostHTML` and `GetPostMarkdown` (`backend/internal/service/post/post.go`), routed through a nil-safe `recorder()` accessor after the `logger()` precedent — not inside `ristrettoCache.Get`. The double-check inside `singleflight.Do` performs a second `Get` on every cold miss; counting there would inflate misses ~2× on cold traffic. A miss means "the request entered the singleflight path"; the rare race where the double-check finds a concurrent leader's just-filled entry is accepted as an undercounted hit. Both the HTML and raw variants share the counters (no variant attribute).
- **Cache disabled (`[render] enabled = false`) still counts misses**: `noopCache` always misses, so the hit rate truthfully reads 0% — the metric answers "is the cache effective" including the disabled case.
- **Purge outcome classification** (`backend/internal/service/post/purger.go`) follows the control flow: `skipped` for the no-op path, `failure` for marshal/build/transport errors and HTTP ≥ 300, `success` otherwise. A purge attempt is derivable as success + failure; no separate initiation counter. Purge logging uses `slog` with structured fields (`qid`, HTTP status or error) instead of `log.Printf`.
- Instruments are delivered through the service-local `Metrics` interface (`post.Service`'s `WithMetrics` injection + `noopMetrics` fallback), whose purger subset is the `PurgeMetrics` interface; `cmd/server/metrics_adapters.go` adapts `*observability.Metrics`, and `NewService` builds the purger after options have injected the recorder.

Docs: the five rows are in the metric inventory in [`specs/backend/observability.md`](../../../specs/backend/observability.md), which also corrects the drifted rows to the shipped reality (`markpost.auth.login_total` → separate `login_success_total`/`login_failure_total` counters; `markpost.delivery.failed_total` label `reason` → the actual `error_category` attribute). [`specs/backend/caching.md`](../../../specs/backend/caching.md) carries the subsection reading origin cache metrics against `CF-Cache-Status` (edge HIT/MISS/EXPIRED vs origin hit rate; the edge absorbs most reads, so low origin traffic with a high hit rate is the healthy steady state, not a fault). Both specs update their bilingual twins in the same change.

## Alternatives considered

**A metered `renderCache` wrapper counting inside `Get`.** Mechanically the cleanest (one wrapper covers both implementations) but it counts the singleflight double-check `Get` too — every cold miss books two misses — and excluding the re-check would need a flag or context that reinvents the call-site knowledge the wrapper was meant to hide. Counting at the request decision point is one line per outcome and exact.

**One counter with an outcome attribute (e.g. `markpost.cdn.purge_total` with `outcome=success|failure|skipped`).** Fewer instruments and the shape the spec's login row implied, but every shipped business metric is a separate counter per outcome (`login_success_total`, `delivery.failed_total`, …); introducing a second style for two adjacent mechanisms makes jq queries inconsistent across the inventory. The spec row that suggested the labeled style was itself drifted from the shipped code.

**Exporting ristretto's built-in `Metrics` (hit ratio, internal counters).** Free detail, but the ratios are per-`Get` internal accounting, not request-level effectiveness — they cannot answer "what fraction of render requests avoided a render" without the same call-site attribution, and they couple operators to ristretto's internal vocabulary.

**A separate purge-initiation counter.** Redundant: an attempt is exactly success + failure; skipped is defined as not attempting. Aggregate queries stay simple.

## Consequences

Answering "is the render cache working" and "did the purge fire and succeed" requires only the observability files and `jq` — no database access. The obligations accepted: the hit/miss accounting is tied to the fast-path `Get` location, so a refactor moving it must carry the instrumentation along (the cold-then-warm unit test in `post_metrics_test.go` pins one miss then one hit, and misses-only for a disabled cache); `observability/metrics_test.go` pins the metric names and the attribute-free decision, and `purger_test.go` pins the outcome classification; a purge attempt is a derived quantity (success + failure), not a first-class series. Self-hosted instances without Cloudflare see only `purge_skipped_total` climb — that is the expected steady state, and the caching spec's reading guide says so explicitly so it is not mistaken for a fault. Cardinality stays flat: five counters, no attributes, one series each per 60 s export. The spec corrections beyond the issue's strict scope (the login and `error_category` drift fixes) were called out in the delivery PR for reviewer veto.
