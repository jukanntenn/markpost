# Load Testing (vegeta)

End-to-end HTTP load tests for markpost's read path, per
`specs/backend/performance-optimization.md` P5 item 18. Three scenarios drive
the same `GET /:qid` endpoint under different cache states.

## Prerequisites

1. **vegeta v12.13.0**

   ```bash
   go install github.com/tsenart/vegeta/v12@v12.13.0
   ```

   Verify: `vegeta -version`.

2. **jq** — for streaming unique QID targets in the cold scenarios.

3. **A running server.** For the dev environment:

   ```bash
   python3 devops/dev.py start
   ```

   The server must be reachable at `localhost:7330` (override with `HOST`/`PORT`).

## Quick start

```bash
# 1. Seed 1000 posts (32 KB bodies) and extract vegeta target lists.
bash scripts/loadtest/seed.sh

# 2. Run all three scenarios at the default rates.
bash scripts/loadtest/run.sh
```

Results land in `scripts/loadtest/out/results/` (`*.bin` raw, `*.json` report).

## Seeding

`seed.sh` generates fake posts and imports them:

| Step | Where | What |
|------|-------|------|
| Generate | host | `go run ./tools/` → `backend/fake.json` |
| Import | server container | `docker compose exec backend go run ./cmd/server import-fake-posts --file /app/fake.json` |
| Extract | host | `jq` → `out/hot.txt` (few QIDs) + `out/qids.json` (all QIDs) |

The import runs inside the container so it can reach Postgres on the container
network without exposing the DB port. `backend/` is bind-mounted at `/app`, so
`fake.json` is visible on both sides.

Tunable env vars:

| Var | Default | Notes |
|-----|---------|-------|
| `COUNT` | `1000` | Number of posts. For a fully all-cold run, set this ≥ `RATE × DURATION` so no QID repeats. |
| `BODY_BYTES` | `32768` | Target body size; matches the spec's 32 KB average. |
| `SEED` | `1` | Fixed RNG seed → reproducible QIDs and bodies. |
| `HOT_COUNT` | `10` | How many QIDs go into `hot.txt`. |

## Scenarios

All three hit `GET http://$HOST:$PORT/:qid`. They differ only in which QIDs they
send and whether the render cache can absorb them.

| Scenario | Rate | Targets | Cache behavior |
|----------|------|---------|----------------|
| **hot** | 100/s | 10 QIDs round-robin | First request per QID misses; the rest are render-cache + (if present) CDN hits. Measures the warm path. |
| **cold** | 100/s | QIDs streamed in order, one per request | Each QID is a miss on first touch. If `COUNT < RATE × DURATION`, later wraps become hits. |
| **all-cold** | 100/s | Same pool, reversed order | Same as cold but with a different access pattern to spread misses. Set `COUNT` large for genuine zero-repeat. |

The hot rate defaults to 100/s to match the L1 read limiter (`100/s`, burst 200);
exceeding it triggers 429s. To push origin throughput harder, raise
`MARKPOST_RATELIMIT__READ__PER_SECOND` on the server in lockstep with `HOT_RATE`.

Defaults: `DURATION=30s`. Override per-scenario rate with `HOT_RATE` / `COLD_RATE`.
Run a subset with `SCENARIOS="hot cold" bash scripts/loadtest/run.sh`.

### Reading the report

vegeta prints a text table to stdout and writes a JSON report to
`out/results/<scenario>.json`. Key columns:

- **Latencies (p50/p90/p95/p99)** — tail latency under load. Compare hot vs cold
  to see the render-cache's effect (hot p99 should be far lower).
- **Success** — fraction of responses with status in `[200,400)`. A 304 (revalidation
  hit) counts as success; a 404/500 counts as failure. If cold shows failures, the
  QID pool may be exhausted (raise `COUNT`).
- **Bytes In/Out** — per-request and total. Confirms the gzip/zstd + externalized-CSS
  savings from P0 items 1–3.

### Plotting

```bash
vegeta plot scripts/loadtest/out/results/hot.bin scripts/loadtest/out/results/cold.bin \
    > scripts/loadtest/out/results/plot.html
```

## Micro-benchmarks (P5 item 17)

The Go-level render benchmarks live in
`backend/internal/service/post/render_bench_test.go` and run independently of a
server:

```bash
cd backend
go test -bench=. -benchmem -run=^$ ./internal/service/post/
```

These isolate the render pipeline (goldmark + bluemonday + minify), the cache-hit
path, singleflight collapse, and the ETag/minify costs — the components whose
end-to-end effect the vegeta scenarios above measure over HTTP.

## Write-path load testing (POST /:post_key)

The write path creates a post via `POST /:post_key` (public, no JWT). It is
governed by the **L2 rate limiter** (10/min, burst 20, plus 1000/day per user),
keyed on `user_id`. A single post_key can sustain only ~10 requests/minute, so
write load tests seed many users and round-robin their keys.

### L2 rate-limit constraint

The default L2 limit is **10/min per user**. At 50/s × 10s = 500 requests, you
need ≥ 84 users (500 ÷ 6) for every user to stay under 10/min. Two options:

- **Scale users** (default): seed enough users so each gets < 10 requests/min.
  `seed_write.sh` warns if the pool is too small.
- **Raise the limit**: set `MARKPOST_RATELIMIT__PUBLIC_WRITE__PER_SECOND` to the
  target rate in `devops/docker-compose.yml` and restart (`python3 devops/dev.py
  stop && python3 devops/dev.py start`). This tests raw server throughput
  without the production throttle.

### Seeding write targets

```bash
# 100 users, no delivery channels (plain scenario)
bash scripts/loadtest/seed_write.sh

# 200 users, 3 Feishu delivery channels each (delivery scenario)
USERS=200 CHANNELS=3 bash scripts/loadtest/seed_write.sh
```

`seed_write.sh` runs the `seed-users` CLI inside the container (creates users
via the production user-repo path with unique `mpk-` keys), captures the keys to
`out/write_keys.txt`, then generates `out/write_targets.jsonl` — vegeta JSON
targets whose post bodies follow a **normal distribution** (mean 32 KiB, σ 8 KiB,
clamped to [1 KiB, 32 KiB]) matching the spec's post-size assumption.

### Scenarios

| Scenario | What it measures | How to run |
|----------|-----------------|------------|
| **plain** | Baseline write throughput: PostKey DB lookup + post insert. Users have no delivery channels. | `SCENARIO=plain bash scripts/loadtest/run_write.sh` |
| **delivery** | Write with synchronous delivery fan-out: channel query + keyword filter compile/match + attempt insert (inline in the create request). The latency delta vs `plain` is the delivery cost. | `SCENARIO=delivery bash scripts/loadtest/run_write.sh` (requires `CHANNELS>0` seed) |

Both use the same `write_targets.jsonl` file; the difference is whether the
seeded users have delivery channels. The webhook HTTP send itself is asynchronous
(scheduler + pond pool) and does **not** block the write response.

Defaults: `RATE=50`, `DURATION=10s`. Override with env vars:
`RATE=100 DURATION=30s SCENARIO=plain bash scripts/loadtest/run_write.sh`.

### Reading the write-path report

Compare `plain` vs `delivery`:
- **p50/p99 latency** — `delivery` should be higher; the delta is the inline
  fan-out cost (channel query + filter + attempt insert).
- **Success ratio** — 100% means no 429s (user pool large enough). 429s indicate
  the L2 limit was hit; seed more users or raise the limit.
- **Status codes** — `201` = created, `400` = body validation (should not appear
  if targets are ≤ 32 KiB), `429` = rate-limited, `403` = invalid post_key.
