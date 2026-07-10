#!/usr/bin/env bash
#
# Seeds test users (with post_keys) and optional delivery channels for write-path
# load testing, then generates vegeta targets.
#
# Pipeline:
#   seed-users CLI (in container) -> post_keys to stdout -> out/write_keys.txt
#   generate_write_targets (host) -> out/write_targets.jsonl
#
# Usage:
#   bash scripts/loadtest/seed_write.sh                       # 100 users, no channels
#   USERS=200 CHANNELS=3 bash scripts/loadtest/seed_write.sh  # 200 users, 3 channels each
#
# Env vars:
#   USERS            number of users to seed (default 100)
#   CHANNELS         Feishu delivery channels per user (default 0)
#   CHANNEL_KEYWORDS keyword filter for each channel (default "")
#   PREFIX           username prefix (default loadtest)
#   PASSWORD         password for seeded users (default loadtestpass)
#   TARGETS          number of vegeta targets to generate (default 1000)
#   HOST             target host for generated URLs (default localhost)
#   PORT             target port (default 7330)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BACKEND_DIR="$PROJECT_ROOT/backend"

USERS="${USERS:-100}"
CHANNELS="${CHANNELS:-0}"
CHANNEL_KEYWORDS="${CHANNEL_KEYWORDS:-}"
PREFIX="${PREFIX:-loadtest}"
PASSWORD="${PASSWORD:-loadtestpass}"
TARGETS="${TARGETS:-1000}"
HOST="${HOST:-localhost}"
PORT="${PORT:-7330}"
COMPOSE_FILE="$PROJECT_ROOT/devops/docker-compose.yml"
SERVICE="${SERVICE:-backend}"
OUT_DIR="$SCRIPT_DIR/out"

mkdir -p "$OUT_DIR"

echo "==> Seeding $USERS users ($CHANNELS channels each) via seed-users CLI"
# seed-users prints post_keys to stdout, status to stderr.
docker compose -f "$COMPOSE_FILE" exec -T "$SERVICE" \
    go run ./cmd/server seed-users \
    --count="$USERS" \
    --prefix="$PREFIX" \
    --password="$PASSWORD" \
    --channels="$CHANNELS" \
    --channel-keywords="$CHANNEL_KEYWORDS" \
    > "$OUT_DIR/write_keys.txt"

echo "==> Captured $(wc -l < "$OUT_DIR/write_keys.txt") post_keys"

echo "==> Generating $TARGETS vegeta write targets (normal body distribution)"
cd "$BACKEND_DIR"
go run ./tools/write_targets/ \
    --count="$TARGETS" \
    --keys-file="$OUT_DIR/write_keys.txt" \
    --host="$HOST" \
    --port="$PORT" \
    > "$OUT_DIR/write_targets.jsonl"

# Quick body-size distribution check (decoded JSON payload byte size).
echo "==> Body size distribution (decoded JSON payload, bytes):"
if command -v jq >/dev/null 2>&1; then
    jq -r '.body | @base64d | utf8bytelength' "$OUT_DIR/write_targets.jsonl" \
        | awk '{sum+=$1; n++; a[n]=$1} END {
            asort(a);
            printf "    count=%d  min=%d  p50=%d  mean=%d  max=%d\n", n, a[1], a[int(n/2)], sum/n, a[n]
        }'
fi

echo "==> Seed complete:"
echo "      users          : $USERS ($CHANNELS channels each)"
echo "      post_keys      : $OUT_DIR/write_keys.txt"
echo "      targets        : $OUT_DIR/write_targets.jsonl"
echo ""
echo "L2 rate limit (10/min/user): ensure USERS >= RATE * DURATION / 6"
echo "or raise MARKPOST_RATELIMIT__PUBLIC_WRITE__PER_SECOND and restart."
echo "Next: SCENARIO=plain bash scripts/loadtest/run_write.sh"
