#!/usr/bin/env bash
#
# Batched import-fake-posts for memory-capped containers.
#
# The seed CLI loads the whole fake.json into memory (ReadFile + Unmarshal +
# struct copies ≈ 3x the file size), which OOM-kills inside the capacity
# stack's 1280m app container for large seeds (observed: 16k×32KB = 552MB
# JSON → exit 137). This wrapper slices the file on the host into BATCH-size
# chunks and imports each — peak memory stays ~3x the chunk, not the corpus.
#
# Usage:
#   bash scripts/loadtest/capacity/import-batches.sh [FAKE_JSON] [BATCH] [OUT_PREFIX]
#   defaults: backend/fake.json, 2000, scripts/loadtest/out
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
FAKE_JSON="${1:-$ROOT/backend/fake.json}"
BATCH="${2:-2000}"
OUT_PREFIX="${3:-$ROOT/scripts/loadtest/out/batch}"
COMPOSE_FILE="${COMPOSE_FILE:-$SCRIPT_DIR/docker-compose.yml}"
SERVICE="${SERVICE:-app}"

total="$(jq 'length' "$FAKE_JSON")"
chunks=$(( (total + BATCH - 1) / BATCH ))
echo "==> importing $total posts in $chunks batches of $BATCH"

for ((i = 0; i < chunks; i++)); do
    lo=$((i * BATCH))
    hi=$(( (i + 1) * BATCH ))
    [[ $hi -gt $total ]] && hi=$total
    chunk_file="${OUT_PREFIX}-${i}.json"
    jq -c ".[${lo}:${hi}]" "$FAKE_JSON" >"$chunk_file"
    docker compose -f "$COMPOSE_FILE" cp "$chunk_file" "$SERVICE:/tmp/fake-batch.json" >/dev/null
    docker compose -f "$COMPOSE_FILE" exec -T "$SERVICE" \
        markpost --config /app/config.toml import-fake-posts --file /tmp/fake-batch.json
    rm -f "$chunk_file"
    echo "    batch $((i + 1))/$chunks: $((hi - lo)) posts"
done
echo "==> batch import complete"
