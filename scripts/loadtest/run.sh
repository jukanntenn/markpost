#!/usr/bin/env bash
#
# k6 load-test runner for markpost.
#
# Scenarios (see scripts/loadtest/k6/*.js and README.md for full design):
#   read-cold-miss      origin renders each QID once (singleflight collapse)
#   read-revalidate-304 CDN revalidation: GET + If-None-Match → 304
#   read-warm-hit       hot QIDs, render-cache hits
#   write               POST /:post_key + async delivery fan-out
#   soak                mixed read+write held 60m (memory/conn/goroutine steady-state)
#
# The k6 binary is fetched on first run into scripts/loadtest/k6-bin/ (pinned
# version below); it is NOT committed. Requires jq + curl.
#
# Usage:
#   bash scripts/loadtest/run.sh                       # all short scenarios
#   SCENARIO=read-revalidate-304 RATE=50 bash scripts/loadtest/run.sh
#   SCENARIO=soak bash scripts/loadtest/run.sh
#   SCENARIO=write RATE=10 bash scripts/loadtest/run.sh
#
# Env vars:
#   SCENARIO  run a single scenario (default: all except soak)
#   SCHEME/HOST/PORT  target (default https://localhost:2053, the e2e compose)
#   RATE      req/s for read/write scenarios (per-scenario defaults in the JS)
#   DURATION  scenario duration (default per JS)
#   HOLD/RAMP soak hold/ramp duration (default 60m/2m)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
K6_DIR="$SCRIPT_DIR/k6"
BIN_DIR="$SCRIPT_DIR/k6-bin"
OUT_DIR="$SCRIPT_DIR/out/results"

K6_VERSION="v2.1.0"

# Detect platform (linux/amd64 or linux/arm64); k6 ships tarballs for both.
detect_platform() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64) echo "linux-amd64" ;;
        aarch64|arm64) echo "linux-arm64" ;;
        *) echo "linux-amd64" ;;
    esac
}
K6_PLATFORM="$(detect_platform)"

require() {
    command -v "$1" >/dev/null 2>&1 || { echo "error: $1 not found on PATH" >&2; exit 1; }
}
require curl
require jq

# ── Ensure k6 binary (curl release tarball, pinned version) ────────────────
ensure_k6() {
    if [[ -x "$BIN_DIR/k6" ]]; then
        return
    fi
    echo "==> Fetching k6 ${K6_VERSION} (${K6_PLATFORM})"
    mkdir -p "$BIN_DIR"
    local tmpdir
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT
    curl -sSL "https://github.com/grafana/k6/releases/download/${K6_VERSION}/k6-${K6_VERSION}-${K6_PLATFORM}.tar.gz" \
        | tar xz -C "$tmpdir"
    mv "$tmpdir/k6-${K6_VERSION}-${K6_PLATFORM}/k6" "$BIN_DIR/k6"
    chmod +x "$BIN_DIR/k6"
}

# ── Scenario runner ────────────────────────────────────────────────────────
# Maps the SCENARIO alias to a JS file + env exports.
run_scenario() {
    local name="$1"
    local script
    local k6_env=()
    case "$name" in
        read-cold-miss)
            script="$K6_DIR/read.js"
            k6_env+=(SCENARIO=cold-miss);;
        read-revalidate-304)
            script="$K6_DIR/read.js"
            k6_env+=(SCENARIO=revalidate-304);;
        read-warm-hit)
            script="$K6_DIR/read.js"
            k6_env+=(SCENARIO=warm-hit);;
        write)
            script="$K6_DIR/write.js";;
        soak)
            script="$K6_DIR/soak.js";;
        *)
            echo "error: unknown scenario '$name'" >&2
            echo "valid: read-cold-miss read-revalidate-304 read-warm-hit write soak" >&2
            exit 1;;
    esac

    echo "==> [${name}] $(basename "$script")"
    mkdir -p "$OUT_DIR"
    env "${k6_env[@]}" "$BIN_DIR/k6" run "$script" \
        --out "json=$OUT_DIR/${name}.json" \
        --summary-export="$OUT_DIR/${name}-summary.json"
    echo "    results: $OUT_DIR/${name}.json"
}

ensure_k6

if [[ -n "${SCENARIO:-}" ]]; then
    run_scenario "$SCENARIO"
else
    for s in read-cold-miss read-revalidate-304 read-warm-hit write; do
        run_scenario "$s"
    done
    echo "==> soak omitted by default (long-running). Run explicitly:"
    echo "    SCENARIO=soak bash scripts/loadtest/run.sh"
fi

echo "==> Done."
