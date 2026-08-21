#!/usr/bin/env bash
#
# Slow-client probes (defensive posture documentation, not a pass/fail gate):
#
#  A. Slow upload: POST a ~32 KiB JSON body at ~1 KiB/s (~33s in flight) to a
#     valid post key. The server has only ReadHeaderTimeout (10s) today — no
#     ReadTimeout — so the expectation is that these complete successfully,
#     demonstrating that a handful of slow writers occupies a connection and
#     memory for tens of seconds without any timeout reclaiming it.
#  B. Slow download: GET a post page at ~1 KiB/s (~12s+ for the compressed
#     body), showing responses park in socket buffers while the client drips.
#
# Goroutine/thread counts of the Go process are sampled before/during/after
# so the per-slow-connection cost is visible.
#
# Usage:
#   bash scripts/loadtest/capacity/slow-client.sh [N_SLOW]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
OUT_DIR="$ROOT/scripts/loadtest/out/results/capacity"
BASE="https://localhost:2053"
APP="${APP_CONTAINER:-markpost-capacity-app-1}"
N="${1:-10}"

mkdir -p "$OUT_DIR"
KEY="$(head -1 "$ROOT/scripts/loadtest/out/write_keys.txt")"
QID="$(jq -r '.[0]' "$ROOT/scripts/loadtest/out/qids.json")"

BODY_FILE="$OUT_DIR/slow-body.json"
jq -n --arg t "Slow client probe" --arg b "$(head -c 32000 /dev/zero | tr '\0' 'x')" \
    '{title: $t, body: $b}' >"$BODY_FILE"

thread_count() {
    docker exec "$APP" sh -c "ls /proc/\$(pidof markpost)/task 2>/dev/null | wc -l" 2>/dev/null || echo "?"
}

echo "==> threads before: $(thread_count)"

echo "==> [A] ${N} slow uploads (~33s each, ~1KiB/s)"
pids=()
for i in $(seq 1 "$N"); do
    curl -sk -X POST -H 'Content-Type: application/json' \
        --data-binary "@$BODY_FILE" --limit-rate 1k \
        -o /dev/null -w "upload $i: %{http_code} in %{time_total}s\n" \
        "$BASE/$KEY" &
    pids+=($!)
done
mid_threads="$(thread_count)"
for p in "${pids[@]}"; do wait "$p"; done

echo "==> threads during uploads: ${mid_threads}; after: $(thread_count)"

echo "==> [B] ${N} slow downloads of /${QID} (~1KiB/s)"
pids=()
for i in $(seq 1 "$N"); do
    curl -sk --limit-rate 1k -H 'Accept-Encoding: gzip' \
        -o /dev/null -w "download $i: %{http_code} %{size_download}B in %{time_total}s\n" \
        "$BASE/$QID" &
    pids+=($!)
done
for p in "${pids[@]}"; do wait "$p"; done

echo "==> threads after downloads: $(thread_count)"
echo "note: with only ReadHeaderTimeout set, no server-side timeout reclaims
      these connections — see the capacity report's hardening section."
