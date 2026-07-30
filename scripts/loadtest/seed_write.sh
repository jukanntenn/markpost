#!/usr/bin/env bash
#
# Seeds test users (with post_keys) and optional delivery channels for
# write-path load testing. The k6 write/soak scenarios generate request bodies
# in JS, so only the post_keys list is needed.
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
#   WEBHOOK_URL      webhook target for seeded channels (default dev no-op;
#                    set to http://webhook-mock:3002/webhook for the e2e stack)
#   COMPOSE_FILE     path to docker-compose.yml (default devops/docker-compose.yml)
#   SERVICE          compose service running the server (default backend)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

USERS="${USERS:-100}"
CHANNELS="${CHANNELS:-0}"
CHANNEL_KEYWORDS="${CHANNEL_KEYWORDS:-}"
PREFIX="${PREFIX:-loadtest}"
PASSWORD="${PASSWORD:-loadtestpass}"
COMPOSE_FILE="$PROJECT_ROOT/${COMPOSE_FILE:-devops/docker-compose.yml}"
SERVICE="${SERVICE:-backend}"
OUT_DIR="$SCRIPT_DIR/out"

mkdir -p "$OUT_DIR"

echo "==> Seeding $USERS users ($CHANNELS channels each) via seed-users CLI"
# seed-users prints post_keys to stdout, status to stderr. Dev image (Go
# present) runs `go run`; the production-shaped e2e/load image uses the
# compiled binary. The channels' webhook target defaults to a dev no-op sink;
# set WEBHOOK_URL (e.g. http://webhook-mock:3002/webhook in the e2e stack) so
# the dispatcher's sends actually land and the delivery metrics advance.
EXEC_ENV=()
if [[ -n "${WEBHOOK_URL:-}" ]]; then
    EXEC_ENV+=(-e "MP_SEED_WEBHOOK_URL=$WEBHOOK_URL")
fi
if docker compose -f "$COMPOSE_FILE" exec -T "${EXEC_ENV[@]}" "$SERVICE" which go >/dev/null 2>&1; then
    docker compose -f "$COMPOSE_FILE" exec -T "${EXEC_ENV[@]}" "$SERVICE" \
        go run ./cmd/server seed-users \
        --count="$USERS" \
        --prefix="$PREFIX" \
        --password="$PASSWORD" \
        --channels="$CHANNELS" \
        --channel-keywords="$CHANNEL_KEYWORDS" \
        > "$OUT_DIR/write_keys.txt"
else
    docker compose -f "$COMPOSE_FILE" exec -T "${EXEC_ENV[@]}" "$SERVICE" \
        markpost --config /app/config.toml seed-users \
        --count="$USERS" \
        --prefix="$PREFIX" \
        --password="$PASSWORD" \
        --channels="$CHANNELS" \
        --channel-keywords="$CHANNEL_KEYWORDS" \
        > "$OUT_DIR/write_keys.txt"
fi

echo "==> Captured $(wc -l < "$OUT_DIR/write_keys.txt") post_keys"

echo "==> Seed complete:"
echo "      users          : $USERS ($CHANNELS channels each)"
echo "      post_keys      : $OUT_DIR/write_keys.txt"
echo ""
echo "The e2e/load compose raises the L2 limit (100/s), so USERS=100 is ample."
echo "Next: SCENARIO=write bash scripts/loadtest/run.sh"
