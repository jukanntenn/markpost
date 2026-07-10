#!/usr/bin/env bash
#
# Runs the three vegeta load-test scenarios against a running markpost server.
#
# Scenarios (per performance-optimization.md P5 item 18):
#   hot      - a small set of QIDs hit round-robin; after the first request each
#              is a render-cache + CDN hit. Measures the warm serving path.
#   cold     - unique QIDs streamed one per request; each is a cache miss that
#              forces a full DB read + render. Requires seed COUNT >= total
#              requests (RATE x DURATION) for genuine all-miss behavior; below
#              that, QIDs repeat and later hits warm the cache.
#   all-cold - like cold but draws QIDs in a shuffled order from the full pool,
#              maximizing cache-miss spread when COUNT is large.
#
# Prerequisites:
#   - vegeta on PATH (go install github.com/tsenart/vegeta/v12@v12.13.0)
#   - jq on PATH
#   - server running and reachable at $HOST:$PORT
#   - scripts/loadtest/seed.sh already run (out/hot.txt + out/qids.json exist)
#
# Usage:
#   bash scripts/loadtest/run.sh                 # all scenarios, defaults
#   SCENARIOS="hot cold" bash scripts/loadtest/run.sh
#   HOST=localhost PORT=7330 RATE=200 DURATION=30s bash scripts/loadtest/run.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUT_DIR="$SCRIPT_DIR/out"

HOST="${HOST:-localhost}"
PORT="${PORT:-7330}"
DURATION="${DURATION:-30s}"
# Per-scenario rates; all-cold uses the cold rate unless overridden.
# HOT_RATE defaults to 100 to match the L1 read limiter (100/s, burst 200);
# raising it above the limiter yields 429s. To test higher origin load, raise
# MARKPOST_RATELIMIT__READ__PER_SECOND on the server in lockstep.
HOT_RATE="${HOT_RATE:-100}"
COLD_RATE="${COLD_RATE:-100}"
SCENARIOS="${SCENARIOS:-hot cold all-cold}"

BUCKETS='[0,5ms,10ms,25ms,50ms,100ms,250ms,500ms,1s,2s]'

require() {
    command -v "$1" >/dev/null 2>&1 || { echo "error: $1 not found on PATH" >&2; exit 1; }
}
require vegeta
require jq

if [[ ! -f "$OUT_DIR/hot.txt" || ! -f "$OUT_DIR/qids.json" ]]; then
    echo "error: target lists not found. Run bash scripts/loadtest/seed.sh first." >&2
    exit 1
fi

base_url="http://$HOST:$PORT"
mkdir -p "$OUT_DIR/results"

run_hot() {
    echo "==> [hot] $HOT_RATE/s for $DURATION (cache hits)"
    local targets="$OUT_DIR/results/hot_targets.txt"
    : > "$targets"
    while read -r qid; do
        echo "GET $base_url/$qid" >> "$targets"
    done < "$OUT_DIR/hot.txt"

    vegeta attack -rate="$HOT_RATE/s" -duration="$DURATION" \
        -targets="$targets" -name=hot \
        | tee "$OUT_DIR/results/hot.bin" \
        | vegeta report
    vegeta report -type=json -buckets="$BUCKETS" \
        "$OUT_DIR/results/hot.bin" > "$OUT_DIR/results/hot.json"
    echo "    report: $OUT_DIR/results/hot.json"
}

# emit_targets streams vegeta JSON-format targets from a QID array. The first
# argument selects jq's iteration order so cold walks the pool sequentially and
# all-cold reverses it for a different access pattern. The stream repeats the
# pool indefinitely so vegeta's -lazy reader never hits EOF before -duration
# elapses. When the pool is exhausted and restarts, repeated QIDs hit the render
# cache; a large COUNT (>= rate x duration) keeps the first pass fully cold.
emit_targets() {
    local order="$1"
    local arr
    arr=$(jq "$order" "$OUT_DIR/qids.json")
    jq -ncr --arg url "$base_url" --argjson arr "$arr" \
        'def emit: ($arr[] | {method:"GET", url:($url + "/" + .)}), emit; emit'
}

run_cold() {
    local name="$1" order="$2" rate="$3"
    echo "==> [$name] $rate/s for $DURATION (cache misses)"
    # emit_targets produces an unbounded stream; when vegeta's -duration
    # elapses it closes stdin, so the jq writer receives SIGPIPE. That is
    # expected — disable pipefail for this segment so it does not abort the
    # script.
    set +o pipefail
    emit_targets "$order" \
        | vegeta attack -rate="$rate/s" -duration="$DURATION" \
            -format=json -lazy -name="$name" \
        | tee "$OUT_DIR/results/$name.bin" \
        | vegeta report
    set -o pipefail
    vegeta report -type=json -buckets="$BUCKETS" \
        "$OUT_DIR/results/$name.bin" > "$OUT_DIR/results/$name.json"
    echo "    report: $OUT_DIR/results/$name.json"
}

for scenario in $SCENARIOS; do
    case "$scenario" in
        hot)      run_hot ;;
        cold)     run_cold cold    "."      "$COLD_RATE" ;;
        all-cold) run_cold all-cold "reverse" "$COLD_RATE" ;;
        *) echo "unknown scenario: $scenario (skipping)" >&2 ;;
    esac
done

echo "==> Done. Plots: vegeta plot $OUT_DIR/results/*.bin > out/results/plot.html"
