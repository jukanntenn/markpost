# MRFC: Cache and purge observability metrics

Status: proposed

English | [中文](2026-09-03-cache-purge-observability.zh.md)

## Problem

The read path's two caching mechanisms — the in-process render cache (ristretto, `backend/internal/service/post/cache.go`) and the best-effort Cloudflare cache-tag purge (`backend/internal/service/post/purger.go`) — emit no metrics. When a reader reports stale content or origin load looks wrong, the only way to answer "is the cache actually working?" is to infer from response headers per request or to query the database; nothing in the observability files (`metrics-*.jsonl`, `app-*.jsonl`) records a cache hit, a miss, or a purge attempt, let alone its outcome (success, failure, skipped). The render cache can also be disabled by config, and nothing distinguishes "disabled" from "enabled but ineffective". The observability spec's metric inventory has no row for either mechanism, so operators have no documented way to reason about cache effectiveness — including how origin-side signals relate to Cloudflare's visitor-facing `CF-Cache-Status` header.

## Proposal

Add five business counters to the existing OTel metric channel (meter `markpost`, exported to `metrics-*.jsonl` every 60 s), following the shipped naming and structure — dot-separated semconv-style names, one counter per outcome, no attributes:

| Metric                            | Type    | Emitted when                                                                 |
| --------------------------------- | ------- | ---------------------------------------------------------------------------- |
| `markpost.render_cache.hit_total` | counter | A render request is served from the render cache                              |
| `markpost.render_cache.miss_total` | counter | A render request misses and enters the render/singleflight path              |
| `markpost.cdn.purge_success_total` | counter | A cache-tag purge request completes with HTTP < 300                          |
| `markpost.cdn.purge_failure_total` | counter | A purge attempt fails (marshal, request build, transport, or HTTP ≥ 300)     |
| `markpost.cdn.purge_skipped_total` | counter | Purge is not attempted (no-op purger or unconfigured credentials)            |

Placement decisions:

- **Hit/miss is counted at the request decision point** — the fast-path `cache.Get` in `RenderPostHTML` — not inside `ristrettoCache.Get`. The double-check inside `singleflight.Do` performs a second `Get` on every cold miss; counting there would inflate misses ~2× on cold traffic. A miss means "the request entered the singleflight path"; the rare race where the double-check finds a concurrent leader's just-filled entry is accepted as an undercounted hit. Both the HTML and raw variants share the counters (no variant attribute).
- **Cache disabled (`[render] enabled = false`) still counts misses**: `noopCache` always misses, so the hit rate truthfully reads 0% — the metric answers "is the cache effective" including the disabled case.
- **Purge outcome classification** follows the existing control flow: `skipped` for the no-op path, `failure` for marshal/build/transport errors and HTTP ≥ 300, `success` otherwise. A purge attempt is derivable as success + failure; no separate initiation counter.
- **Purge logging migrates `log.Printf` → `slog`** with structured fields (`qid`, HTTP status or error), aligning the purger with the observability spec's logging rules.
- Instruments are delivered through the established service-local `Metrics` interface (`post.Service`'s `WithMetrics` injection + `noopMetrics` fallback), extended with the new methods; `*observability.Metrics` implements them.

Docs: the five rows enter the metric inventory in [`specs/backend/observability.md`](../../../specs/backend/observability.md), which also corrects the drifted `markpost.auth.login_total` row to the shipped separate-counter reality (`login_success_total`/`login_failure_total`); [`specs/backend/caching.md`](../../../specs/backend/caching.md) gains a short subsection reading origin cache metrics against `CF-Cache-Status` (edge HIT/MISS/EXPIRED vs origin hit rate; the edge absorbs most reads, so low origin traffic with a high hit rate is the healthy steady state, not a fault). Both specs update their bilingual twins in the same change.

## Alternatives considered

**A metered `renderCache` wrapper counting inside `Get`.** Mechanically the cleanest (one wrapper covers both implementations) but it counts the singleflight double-check `Get` too — every cold miss books two misses — and excluding the re-check would need a flag or context that reinvents the call-site knowledge the wrapper was meant to hide. Counting at the request decision point is one line per outcome and exact.

**One counter with an outcome attribute (e.g. `markpost.cdn.purge_total` with `outcome=success|failure|skipped`).** Fewer instruments and the shape the spec's login row implied, but every shipped business metric is a separate counter per outcome (`login_success_total`, `delivery.failed_total`, …); introducing a second style for two adjacent mechanisms makes jq queries inconsistent across the inventory. The spec row that suggested the labeled style was itself drifted from the shipped code.

**Exporting ristretto's built-in `Metrics` (hit ratio, internal counters).** Free detail, but the ratios are per-`Get` internal accounting, not request-level effectiveness — they cannot answer "what fraction of render requests avoided a render" without the same call-site attribution, and they couple operators to ristretto's internal vocabulary.

**A separate purge-initiation counter.** Redundant: an attempt is exactly success + failure; skipped is defined as not attempting. Aggregate queries stay simple.

## Acceptance criteria

- The five counters appear in `metrics-*.jsonl` after rendering and after a post deletion.
- A cold-then-warm render of one post yields exactly one miss then one hit, and a disabled cache yields misses only.
- Purge outcomes classify as specified: success on HTTP < 300; failure on each error branch; skipped for the no-op/unconfigured path.
- Purger log lines are slog-structured with `qid` (and status/error) fields.
- Unit tests cover the instrumentation: counters increment through a real meter SDK reader, and the `noopMetrics` fallback keeps tests without metrics working.
- `specs/backend/observability.md` carries the five rows (and the corrected login rows); `specs/backend/caching.md` carries the `CF-Cache-Status` reading guide; both bilingual twins updated in the same change.
- Answering "is the render cache working" and "did the purge fire and succeed" requires only the observability files and `jq` — no database access.

## Risks

- **Instrumentation placement regressions** (e.g. a future refactor moving the fast-path `Get`) would silently distort hit/miss accounting; the cold-then-warm unit test pins the exact semantics.
- **Cardinality and volume**: five counters with no attributes — one series each, exported every 60 s; negligible growth of the metrics file.
- **Self-hosted instances without Cloudflare** emit only `purge_skipped_total`; that is the expected steady state and must not be read as a fault (covered by the caching spec's reading guide).
- **Spec correction beyond the issue's strict scope** (the login-row drift fix) touches a row this issue did not ask for; it is called out in the PR body so reviewers can veto it.
