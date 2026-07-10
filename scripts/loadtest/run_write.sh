#!/usr/bin/env bash
#
# Runs the write-path load test against a running markpost server.
#
# Scenarios:
#   plain    - POST /:post_key with users that have NO delivery channels.
#              Measures baseline write throughput (PostKey lookup + post insert).
#   delivery - Same targets, but users were seeded with delivery channels
#              (seed_write.sh CHANNELS=N). Each create triggers inline channel
#              query + keyword filter + attempt insert. The latency delta vs
#              plain isolates the synchronous delivery-fanout cost.
#
# Prerequisites:
#   - vegeta on PATH (go install github.com/tsenart/vegeta/v12@v12.13.0)
#   - server running at $HOST:$PORT
#   - scripts/loadtest/seed_write.sh already run (out/write_targets.jsonl exists)
#
# Usage:
#   SCENARIO=plain bash scripts/loadtest/run_write.sh
#   SCENARIO=delivery RATE=50 DURATION=10s bash scripts/loadtest/run_write.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="$SCRIPT_DIR/out"

HOST="${HOST:-localhost}"
PORT="${PORT:-7330}"
RATE="${RATE:-50}"
DURATION="${DURATION:-10s}"
SCENARIO="${SCENARIO:-plain}"
BUCKETS='[0,5ms,10ms,25ms,50ms,100ms,250ms,500ms,1s,2s,5s]'

require() {
    command -v "$1" >/dev/null 2>&1 || { echo "error: $1 not found on PATH" >&2; exit 1; }
}
require vegeta

TARGETS_FILE="$OUT_DIR/write_targets.jsonl"
if [[ ! -f "$TARGETS_FILE" ]]; then
    echo "error: $TARGETS_FILE not found. Run bash scripts/loadtest/seed_write.sh first." >&2
    exit 1
fi

# L2 limit check: 10/min/user. Warn if the key pool is too small for the load.
KEYS_FILE="$OUT_DIR/write_keys.txt"
if [[ -f "$KEYS_FILE" ]]; then
    key_count=$(wc -l < "$KEYS_FILE")
    # Approx total requests; DURATION like "10s" → parse seconds.
    dur_secs=$(echo "$DURATION" | grep -oE '^[0-9]+')
    total=$((RATE * dur_secs))
    # Each user can serve ~10/min = 10/60 per second.
    needed=$((total * 6 / 10 + 1))
    if (( key_count < needed )); then
        echo "WARNING: $key_count users may not sustain ${RATE}/s × ${DURATION} (${total} reqs)" >&2
        echo "         under the 10/min/user L2 limit (need ~$needed users). Expect 429s," >&2
        echo "         or raise MARKPOST_RATELIMIT__PUBLIC_WRITE__PER_SECOND and restart." >&2
    fi
fi

mkdir -p "$OUT_DIR/results"

echo "==> [write-$SCENARIO] $RATE/s for $DURATION"
# -lazy streams targets; the static file is finite so vegeta stops at EOF or
# duration, whichever comes first. The file has TARGETS entries (default 1000),
# which should exceed RATE×DURATION for a full run.
set +o pipefail
vegeta attack -rate="$RATE/s" -duration="$DURATION" \
    -targets="$TARGETS_FILE" -format=json -lazy -name="write-$SCENARIO" \
    | tee "$OUT_DIR/results/write-$SCENARIO.bin" \
    | vegeta report
set -o pipefail

vegeta report -type=json -buckets="$BUCKETS" \
    "$OUT_DIR/results/write-$SCENARIO.bin" > "$OUT_DIR/results/write-$SCENARIO.json"
echo "    report: $OUT_DIR/results/write-$SCENARIO.json"
