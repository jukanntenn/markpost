# Load Testing (k6)

End-to-end HTTP load tests for markpost. For the 2c/2g/3Mbps capacity study
(sweet spot / hard limits) see [CAPACITY_REPORT.md](CAPACITY_REPORT.md) and
[`capacity/`](capacity/README.md); the scenarios below are the per-mechanism
regression suite. The design targets the real
production architecture from
[`specs/backend/performance-optimization.md`](../../specs/backend/performance-optimization.md):
the origin sits behind a **Cloudflare CDN**, so it almost never sees a plain
repeat GET (the edge absorbs those). What reaches the origin is (a) **cold
misses** — the first render of a QID for an edge node — and (b) **CDN
revalidations** — a conditional GET carrying the last ETag once `s-maxage`
(1h) elapses, to which the origin answers `304` (bodyless) from the render
cache.

These scenarios reproduce that request shape **without a real CDN** by toggling
the `If-None-Match` header, so the measured latency reflects what the 2-core /
3 Mbps origin must actually sustain.

## Prerequisites

1. **A running server.** Use the **e2e compose** — it is the closest to
   production: the single-container image (Caddy + Go via s6), self-signed
   HTTPS on `https://localhost:2053`, and the mock services the write/delivery
   paths need (an OAuth mock and a Feishu webhook mock). It also raises the
   rate limits so write load tests aren't L2-throttled.

   ```bash
   docker compose -f e2e/docker-compose.yml up -d --build
   curl -k https://localhost:2053/api/v1/health
   ```

   The k6 scripts target `https://localhost:2053` with
   `insecureSkipTLSVerify` by default. Override with `SCHEME`/`HOST`/`PORT`
   (e.g. to point at a plain-HTTP dev server: `SCHEME=http PORT=7330`).

2. **jq + curl** — `run.sh` fetches the pinned k6 binary on first run.

3. **Seed data** — read scenarios need `out/qids.json`, write/soak need
   `out/write_keys.txt` (see Seeding below). The seed CLIs run against the e2e
   app container: `SERVICE=app COMPOSE_FILE=e2e/docker-compose.yml`.

## Quick start

```bash
# 0. Start the e2e stack (production-shaped, self-signed HTTPS, mocks)
docker compose -f e2e/docker-compose.yml up -d --build

# 1. Seed posts + write targets (runs the seed CLIs inside the e2e app container)
SERVICE=app COMPOSE_FILE=e2e/docker-compose.yml bash scripts/loadtest/seed.sh
SERVICE=app COMPOSE_FILE=e2e/docker-compose.yml bash scripts/loadtest/seed_write.sh

# 2. Run all short scenarios (cold-miss, revalidate-304, warm-hit, write)
bash scripts/loadtest/run.sh

# 3. Soak (1h) — run explicitly
SCENARIO=soak bash scripts/loadtest/run.sh
```

The k6 binary is fetched into `scripts/loadtest/k6-bin/` (gitignored) on first
run. Results land in `scripts/loadtest/out/results/` (`*.json` raw,
`*-summary.json` exported).

## Scenarios

| Scenario              | Simulates                                                                                                            | Rate         | Default duration             |
| --------------------- | -------------------------------------------------------------------------------------------------------------------- | ------------ | ---------------------------- |
| `read-cold-miss`      | A QID seen by an edge for the first time — full DB read + render, singleflight collapses concurrent same-QID misses. | 20 req/s     | 60s                          |
| `read-revalidate-304` | CDN revalidation after `s-maxage`: GET warms the cache, then `If-None-Match` triggers a bodyless `304`.              | 50 req/s     | 60s                          |
| `read-warm-hit`       | A viral post: a small fixed QID set hit round-robin; all but the first per QID are render-cache hits.                | 100 req/s    | 60s                          |
| `write`               | `POST /:post_key` (async delivery `Enqueue`) + L2 rate-limit distribution across seeded users.                       | 10 req/s     | 60s                          |
| `soak`                | Mixed read (15/s) + write (2/s) held 60m to surface memory/connection/goroutine leaks.                               | 15 + 2 req/s | ramp 2m + hold 60m + ramp 2m |

Rates are calibrated to the origin's ~25 resp/s physical envelope
(`performance-optimization.md`: 375 KB/s ÷ ~15 KB/page ≈ 25 origin responses/s).
They model the **回源** (origin-facing) load after the CDN absorbs the bulk of
user traffic, not the total user concurrency (which the edge handles).

```bash
SCENARIO=read-revalidate-304 RATE=50 bash scripts/loadtest/run.sh
SCENARIO=write RATE=10 DURATION=60s bash scripts/loadtest/run.sh
SCENARIO=soak HOLD=60m READ_RATE=15 WRITE_RATE=2 bash scripts/loadtest/run.sh
```

## What each scenario measures

- **Latency** p50/p95/p99 (`http_req_duration`).
- **Origin work split** — custom counters `origin_revalidate_304` vs
  `origin_cold_miss_200` separate the cheap revalidation path from expensive
  cold renders (the decisive distinction behind a CDN).
- **Bandwidth** — `data_received` vs the 3 Mbps origin envelope; the summary
  reports average Mbps and utilization % so a scenario that would saturate the
  link is obvious.
- **Failure rate** (`http_req_failed`); thresholds fail the run if breached.

### Write: verifying the delivery fan-out

The write scenario's `POST /:post_key` triggers an **asynchronous** delivery
fan-out: `CreatePost` enqueues a `DeliveryJob`, and the dispatcher claims the
pending attempt on its ticker and sends it to the author's Feishu webhook. The
HTTP response returns before the send lands, so verifying delivery takes a few
extra steps (the e2e stack's `webhook-mock` is the sink):

1. Seed users **with channels pointed at the mock** and a keyword that matches
   the generated title (`Load`):
   ```bash
   WEBHOOK_URL="http://webhook-mock:3002/webhook" USERS=100 CHANNELS=1 \
     CHANNEL_KEYWORDS="Load" SERVICE=app COMPOSE_FILE=e2e/docker-compose.yml \
     bash scripts/loadtest/seed_write.sh
   ```
2. Run the write scenario.
3. Check the mock received one webhook per created post:
   ```bash
   docker exec e2e-app-1 wget -qO- http://webhook-mock:3002/webhooks | jq length
   ```
4. In `metrics-*.jsonl`, `markpost.delivery.dispatched_total` should track the
   created count and `markpost.delivery.pending` should stay near zero (the
   dispatcher drains faster than writes arrive).

Note: successful attempts are archived to `delivery_history` and removed from
`delivery_attempts`, so `delivery_attempts` being empty after a run is **normal**
(it only holds in-flight / retrying rows).

### Soak: what to check after the run

The soak summary prints the k6-side numbers, but the slow-failure signals live
in the backend's `metrics-*.jsonl` (ensure dev/prod mounts it — see
`devops/dev.py` / the compose files):

- `process.runtime.go.mem.heap_alloc` — should plateau near the render-cache
  `MaxCost` (128 MiB), not climb monotonically (ristretto TinyLFU steady state).
- `process.runtime.go.goroutines` — should be stable (delivery worker pool +
  http handlers), not grow without bound.
- `markpost.delivery.pending` — should track the write rate, not accumulate.

A 60-minute hold deliberately exceeds both the Postgres `ConnMaxLifetime`
(30m) and the CDN `s-maxage` (1h) so a full connection-recycle and a cache
revalidation cycle occur during the test.

## Seeding

`seed.sh` generates fake posts and imports them via the `import-fake-posts` CLI
(production user-repo path, no DB port exposure):

| Var          | Default | Notes                                                 |
| ------------ | ------- | ----------------------------------------------------- |
| `COUNT`      | `1000`  | Posts. For genuine all-cold, set ≥ `RATE × DURATION`. |
| `BODY_BYTES` | `32768` | Body size; matches the spec's 32 KB average.          |
| `SEED`       | `1`     | Fixed RNG seed → reproducible QIDs/bodies.            |
| `HOT_COUNT`  | `10`    | QIDs reserved for the warm-hit pool.                  |

`seed_write.sh` seeds users (with `mpk-` post keys) and optional delivery
channels, capturing keys to `out/write_keys.txt`. The L2 limit is 10/min/user;
at `RATE=10 × 60s = 600` requests you need ≥100 users (`USERS=100`).

```bash
bash scripts/loadtest/seed.sh
USERS=100 CHANNELS=3 bash scripts/loadtest/seed_write.sh
```

## Micro-benchmarks

Go-level render/dispatch benchmarks run independently of a server and pinpoint
which stage dominates (goldmark vs bluemonday vs minify vs delivery filter):

```bash
cd backend
go test -bench=. -benchmem -run=^$ ./internal/service/post/
go test -bench=. -benchmem -run=^$ ./internal/service/delivery/filter/
```
