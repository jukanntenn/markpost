# Capacity / sweet-spot test environment

Answers two questions for the 2c/2g/3Mbps-behind-Cloudflare-free production
target: **where is the sweet spot** (sustained rate with SLOs and headroom
intact) and **where is the hard limit** (which resource wall each mechanism
hits). Full methodology and results: `docs` report referenced from the repo
root load-test README; the design discussion lives in
`specs/backend/performance-optimization.md`.

## Layout

| File                 | Role                                                                                                                                                                                            |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `docker-compose.yml` | SUT stack: app+postgres pinned to `cpuset 0,1` with memory caps, PG over Unix socket with production GUCs, e2e mocks; rate limits relaxed (L1 10000/s) so staircases measure the origin ceiling |
| `config.toml`        | app config for the stack (socket DSN, relaxed limiters, 256 KiB body cap)                                                                                                                       |
| `shape.sh`           | 3 mbit tbf + 30 ms netem on the app's egress (via a NET_ADMIN sidecar in the app netns)                                                                                                         |
| `monitor.sh`         | per-container cgroup v2 sampler (CPU/memory/PSI) → CSV                                                                                                                                          |
| `capacity.sh`        | run driver: pins k6 to cores 2-11, wraps runs with the monitor + manifest                                                                                                                       |
| `preflight.sh`       | light validation BEFORE long runs (stack, compression, shaping, limiters, 30-60s mini-runs)                                                                                                     |
| `analyze.py`         | per-stage knee table / restart-storm buckets / soak metric extraction                                                                                                                           |
| `restart-storm.sh`   | release-window test: revalidation load + mid-run app restart                                                                                                                                    |
| `slow-client.sh`     | slow-upload/slow-download probes (timeout posture documentation)                                                                                                                                |
| `../verify-cf.sh`    | post-deploy Cloudflare edge verification checklist (curl-level only)                                                                                                                            |

k6 scenarios live in `../k6/`: `capacity.js` (staircase, MECH=cold/re304/warm),
`mixed.js` (business-profile staircase/hold), `spike.js` (viral single QID).

## Quick start

```bash
# 1. Stack up (first build takes a few minutes)
docker compose -f scripts/loadtest/capacity/docker-compose.yml up -d --build

# 2. Shape egress to the VPS envelope
bash scripts/loadtest/capacity/shape.sh apply

# 3. Seeds (16k×32KB cold pool + 60×256KB worst-case + 100 write users)
COUNT=16000 SEED=1 SERVICE=app COMPOSE_FILE=scripts/loadtest/capacity/docker-compose.yml \
  bash scripts/loadtest/seed.sh
COUNT=60 BODY_BYTES=262144 SEED=2 SERVICE=app COMPOSE_FILE=scripts/loadtest/capacity/docker-compose.yml \
  bash scripts/loadtest/seed.sh && mv scripts/loadtest/out/qids.json scripts/loadtest/out/qids_256k.json
WEBHOOK_URL="http://webhook-mock:3002/webhook" USERS=100 CHANNELS=1 CHANNEL_KEYWORDS="Load" \
  SERVICE=app COMPOSE_FILE=scripts/loadtest/capacity/docker-compose.yml \
  bash scripts/loadtest/seed_write.sh

# 4. Light validation BEFORE anything long
bash scripts/loadtest/capacity/preflight.sh all

# 5. Staircases → sweet-spot hold → soak (see capacity.sh usage)
bash scripts/loadtest/capacity/capacity.sh scan cold
bash scripts/loadtest/capacity/capacity.sh scan re304
bash scripts/loadtest/capacity/capacity.sh scan warm
bash scripts/loadtest/capacity/capacity.sh scan warmcpu     # unshaped CPU-ceiling control
bash scripts/loadtest/capacity/capacity.sh scan mixed
bash scripts/loadtest/capacity/capacity.sh hold <RATE> 1800  # sweet-spot confirmation

python3 scripts/loadtest/capacity/analyze.py run scan-cold-<ts>
```

## Judgement criteria (from the reviewed plan)

- **Sweet spot**: sustained rate where p95 TTFB cold ≤ 300 ms / 304 ≤ 30 ms /
  write ≤ 200 ms, egress ≤ 70% of 3 Mbps, CPU PSI some avg10 < 10%, error
  rate < 0.1%, memory flat over the hold window.
- **Limit**: p95 duration > 2× the sweet-spot value, errors > 1%, or egress
  ≥ 95% — recorded per mechanism together with the binding resource
  (bandwidth / CPU / memory / limiter).

## Deviations from the real VPS (documented)

1. Memory is split as two container caps (app 1280m / postgres 768m) instead
   of one kernel budget; container OOM-kill ≠ host OOM-killer.
2. The load generator runs on the same host (cores 2-11), sharing the Docker
   daemon and kernel with the SUT — final numbers should be re-confirmed once
   on a real 2c/2g VPS with k6 from a second machine.
3. netem delay models one RTT class (30 ms ± 5 ms) to a Cloudflare PoP.
