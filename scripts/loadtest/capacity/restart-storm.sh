#!/usr/bin/env bash
#
# Release-window storm test (performance-optimization.md P5 item 20): hold a
# steady CDN-revalidation load (conditional GETs) against the app, restart the
# app container mid-run (a release: new binary, empty render cache), and
# measure what the revalidation stream experiences — connection-refused window
# during the restart, then post-restart requests whose render cache entries
# are gone (200 + fresh render via singleflight instead of 304).
#
# Output: the k6 raw JSON (for 5s-bucket analysis) + a restart timeline
# (health-poll log) + the container-restart wallclock boundaries recorded in
# the manifest.
#
# Usage:
#   bash scripts/loadtest/capacity/restart-storm.sh [RATE] [DURATION] [RESTART_AT]
#   defaults: 60 it/s, 240s, restart at T+60s
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
BIN_DIR="$ROOT/scripts/loadtest/k6-bin"
OUT_DIR="$ROOT/scripts/loadtest/out/results/capacity"
APP="${APP_CONTAINER:-markpost-capacity-app-1}"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yml"

RATE="${1:-60}"
DURATION="${2:-240s}"
RESTART_AT="${3:-60}"
# k6 duration strings need a unit; accept bare numbers as seconds.
[[ "$DURATION" =~ ^[0-9]+$ ]] && DURATION="${DURATION}s"

mkdir -p "$OUT_DIR"
[[ -x "$BIN_DIR/k6" ]] || { echo "k6 binary missing; run scripts/loadtest/run.sh once" >&2; exit 1; }

NAME="restart-storm-$(date +%H%M%S)"
RAW="$OUT_DIR/${NAME}.json"
SUMMARY="$OUT_DIR/${NAME}-summary.json"
TIMELINE="$OUT_DIR/${NAME}-timeline.log"

echo "==> [${NAME}] re304 load at ${RATE}/s for ${DURATION}, app restarts at T+${RESTART_AT}s"
bash "$SCRIPT_DIR/monitor.sh" start "$OUT_DIR/${NAME}-monitor.csv" 2

# Health poller: records when the app stops answering and when it returns.
(
    while :; do
        ts="$(date +%s)"
        code="$(curl -sk -o /dev/null -w '%{http_code}' --max-time 2 https://localhost:2053/api/v1/health || echo 000)"
        echo "${ts} ${code}" >>"$TIMELINE"
        sleep 0.5
    done
) &
POLL_PID=$!
trap 'kill $POLL_PID 2>/dev/null || true; bash "$SCRIPT_DIR/monitor.sh" stop' EXIT

MECH=re304 STAGES="${RATE}" STEP="${DURATION}" \
    taskset -c 2-11 "$BIN_DIR/k6" run "$ROOT/scripts/loadtest/k6/capacity.js" \
    --out "json=$RAW" --summary-export="$SUMMARY" >"$OUT_DIR/${NAME}-k6.log" 2>&1 &

K6_PID=$!
sleep "$RESTART_AT"

echo "==> restarting app container (simulated release)"
RESTART_AT_TS="$(date +%s)"
echo "RESTART ${RESTART_AT_TS}" >>"$TIMELINE"
docker restart "$APP" >/dev/null
# docker restart recreates the netns, dropping the tc qdiscs — re-apply the
# 3 Mbps shaping before the post-restart traffic resumes.
bash "$SCRIPT_DIR/shape.sh" apply >/dev/null

wait $K6_PID || true
kill $POLL_PID 2>/dev/null || true
bash "$SCRIPT_DIR/monitor.sh" stop || true
trap - EXIT

# down/up must be scoped to AFTER the restart, not the first line of the file.
restart_ts="$(sed -n 's/^RESTART //p' "$TIMELINE" | head -1)"
down="$(awk -v r="$restart_ts" '$1+0 >= r+0 && $2!="200" {print; exit}' "$TIMELINE")"
up="$(awk -v d="$(echo "$down" | awk '{print $1}')" 'd && $1+0 >= d+0 && $2=="200" {print; exit}' "$TIMELINE")"
echo "==> first non-200: ${down:-none}   first 200 again: ${up:-?}"
echo "==> raw samples: $RAW"
echo "==> analyze with: python3 scripts/loadtest/capacity/analyze.py buckets $RAW"
