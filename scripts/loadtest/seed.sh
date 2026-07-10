#!/usr/bin/env bash
#
# Seeds the database with fake posts for load testing.
#
# Pipeline: generate_fake_data (host) -> fake.json -> import-fake-posts (in the
# running server container) -> extract hot.txt (a few QIDs) and qids.json (all
# QIDs) for vegeta targets.
#
# The generate step runs on the host because the seed tool lives in
# backend/tools. The import step runs inside the dev server container so it can
# reach Postgres on the container network without exposing the DB port. The
# fake.json file is placed under backend/ which is bind-mounted into the
# container at /app, so both sides see the same file.
#
# Usage:
#   bash scripts/loadtest/seed.sh                  # defaults: 1000 posts, 32 KB bodies
#   COUNT=5000 BODY_BYTES=32768 bash scripts/loadtest/seed.sh
#
# Env vars:
#   COUNT        number of posts to generate (default 1000)
#   BODY_BYTES   target body size in bytes (default 32768, the spec average)
#   SEED         RNG seed for reproducible QIDs/bodies (default 1)
#   COMPOSE_FILE path to docker-compose.yml (default devops/docker-compose.yml)
#   SERVICE      compose service name running the server (default backend)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BACKEND_DIR="$PROJECT_ROOT/backend"

COUNT="${COUNT:-1000}"
BODY_BYTES="${BODY_BYTES:-32768}"
SEED="${SEED:-1}"
COMPOSE_FILE="$PROJECT_ROOT/${COMPOSE_FILE:-devops/docker-compose.yml}"
SERVICE="${SERVICE:-backend}"

FAKE_JSON="$BACKEND_DIR/fake.json"
OUT_DIR="$SCRIPT_DIR/out"
HOT_COUNT="${HOT_COUNT:-10}"

echo "==> Generating $COUNT fake posts ($BODY_BYTES-byte bodies, seed=$SEED)"
cd "$BACKEND_DIR"
go run ./tools/ \
    -count="$COUNT" \
    -body-bytes="$BODY_BYTES" \
    -seed="$SEED" \
    -output="$FAKE_JSON"

echo "==> Importing fake.json into the database (inside $SERVICE container)"
# fake.json lives under backend/ which maps to /app in the container.
docker compose -f "$COMPOSE_FILE" exec -T "$SERVICE" \
    go run ./cmd/server import-fake-posts --file /app/fake.json

echo "==> Extracting vegeta target lists"
mkdir -p "$OUT_DIR"
# hot.txt: a small set of QIDs (round-robined by vegeta) for the hot-cache
# scenario. Each line is one QID; the run.sh wraps it as an http-format target.
jq -r '.[0:'"$HOT_COUNT"'] | .[].qid' "$FAKE_JSON" > "$OUT_DIR/hot.txt"
# qids.json: the full QID list as a JSON array, consumed by the cold/all-cold
# scenario drivers to emit unique (or near-unique) per-request URLs.
jq -c '[.[].qid]' "$FAKE_JSON" > "$OUT_DIR/qids.json"

echo "==> Seed complete:"
echo "      posts imported : $COUNT"
echo "      hot QIDs       : $OUT_DIR/hot.txt ($(wc -l < "$OUT_DIR/hot.txt") entries)"
echo "      all QIDs       : $OUT_DIR/qids.json"
echo "      Next: bash scripts/loadtest/run.sh"
