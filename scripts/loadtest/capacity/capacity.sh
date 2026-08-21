#!/usr/bin/env bash
#
# Capacity-run driver: wraps every k6 invocation with the three things a
# capacity measurement needs that ad-hk6 runs lack —
#   1. CPU isolation: k6 is pinned to cores 2-11 (SUT owns 0-1 via cpuset),
#   2. correlated resource sampling: monitor.sh runs for exactly the duration
#      of the k6 run, into a per-run CSV,
#   3. a manifest line (start/end wallclock + files) so analyze.py can join
#      k6 stage windows with monitor windows after the fact.
#
# Usage:
#   capacity.sh scan cold [STAGES] [STEP]     # cold-miss staircase
#   capacity.sh scan re304 [STAGES] [STEP]    # revalidation-304 staircase
#   capacity.sh scan warm [STAGES] [STEP]     # warm-hit staircase (shaped)
#   capacity.sh scan warmcpu [STAGES] [STEP]  # warm-hit, shaping cleared first
#                                             #   (CPU-ceiling control run)
#   capacity.sh scan mixed [STAGES] [STEP]    # mixed business profile
#   capacity.sh hold RATE SECONDS             # mixed single-rate hold
#   capacity.sh spike [STAGES]                # viral single-QID ramp
#   capacity.sh wrap NAME SCRIPT "ENV=..."    # generic wrapper
#
# Results land in scripts/loadtest/out/results/capacity/:
#   NAME-summary.json (k6 summary export)  NAME.json (raw samples)
#   NAME-monitor.csv (cgroup/PSI samples)  manifest.csv (run index)
#
# Environment:
#   K6_CPUS   cores for the load generator (default 2-11)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
K6_DIR="$ROOT/scripts/loadtest/k6"
BIN_DIR="$ROOT/scripts/loadtest/k6-bin"
OUT_DIR="$ROOT/scripts/loadtest/out/results/capacity"
K6_CPUS="${K6_CPUS:-2-11}"

K6_VERSION="v2.1.0"
detect_platform() {
    case "$(uname -m)" in
    x86_64 | amd64) echo "linux-amd64" ;;
    aarch64 | arm64) echo "linux-arm64" ;;
    *) echo "linux-amd64" ;;
    esac
}

ensure_k6() {
    if [[ -x "$BIN_DIR/k6" ]]; then return; fi
    echo "==> Fetching k6 ${K6_VERSION}" >&2
    mkdir -p "$BIN_DIR"
    local tmpdir platform
    tmpdir="$(mktemp -d)"
    platform="$(detect_platform)"
    trap 'rm -rf "$tmpdir"' EXIT
    curl -sSL "https://github.com/grafana/k6/releases/download/${K6_VERSION}/k6-${K6_VERSION}-${platform}.tar.gz" |
        tar xz -C "$tmpdir"
    mv "$tmpdir/k6-${K6_VERSION}-${platform}/k6" "$BIN_DIR/k6"
    chmod +x "$BIN_DIR/k6"
}

# wrap NAME SCRIPT ENV_ASSIGNMENTS — the common path for every subcommand.
wrap() {
    local name="$1" script="$2" envs="${3:-}"
    mkdir -p "$OUT_DIR"
    ensure_k6
    local summary="$OUT_DIR/${name}-summary.json"
    local raw="$OUT_DIR/${name}.json"
    local mon="$OUT_DIR/${name}-monitor.csv"
    echo "==> [${name}] $script ${envs}"
    bash "$SCRIPT_DIR/monitor.sh" start "$mon" 2
    local start_ts end_ts rc=0
    start_ts="$(date +%s)"
    # shellcheck disable=SC2086
    env $envs taskset -c "$K6_CPUS" "$BIN_DIR/k6" run "$script" \
        --out "json=$raw" \
        --summary-export="$summary" || rc=$?
    end_ts="$(date +%s)"
    bash "$SCRIPT_DIR/monitor.sh" stop
    echo "${start_ts},${end_ts},${name},${script},${envs//,/;},${summary},${mon}" >>"$OUT_DIR/manifest.csv"
    echo "==> [${name}] done rc=${rc} ($((end_ts - start_ts))s) -> results in $OUT_DIR/${name}-*"
    if [[ $rc -ne 0 ]]; then
        echo "    note: k6 exited ${rc} (threshold/setup). Summary files are still written; check output." >&2
    fi
}

scan_defaults() {
    case "$1" in
    cold) echo "5,10,15,20,25,30 120" ;;
    re304) echo "100,200,300,500,700,1000 60" ;;
    warm) echo "50,100,150,200,300,400 60" ;;
    warmcpu) echo "50,100,150,200,300,400 60" ;;
    mixed) echo "10,15,20,25,30 120" ;;
    *) echo "" ;;
    esac
}

cmd="${1:-}"
shift || true
case "$cmd" in
scan)
    mech="$1"
    defaults="$(scan_defaults "$mech")"
    stages="${2:-$(echo "$defaults" | awk '{print $1}')}"
    step="${3:-$(echo "$defaults" | awk '{print $2}')}"
    extra=""
    if [[ "$mech" == "warmcpu" ]]; then
        echo "==> clearing egress shaping for the unshaped CPU-ceiling control run"
        bash "$SCRIPT_DIR/shape.sh" clear
        extra=" (UNSHAPED)"
    fi
    ts="$(date +%H%M%S)"
    if [[ "$mech" == "mixed" ]]; then
        # mixed is its own script (weighted business profile), not a MECH of
        # capacity.js — running capacity.js with MECH=mixed throws every
        # iteration (observed: 0.1ms iterations, no HTTP).
        wrap "scan-${mech}-${ts}" "$K6_DIR/mixed.js" "STAGES=${stages} STEP=${step}s"
    else
        wrap "scan-${mech}-${ts}${extra:+-unshaped}" "$K6_DIR/capacity.js" \
            "MECH=${mech/warmcpu/warm} STAGES=${stages} STEP=${step}s"
    fi
    if [[ "$mech" == "warmcpu" ]]; then
        bash "$SCRIPT_DIR/shape.sh" apply
    fi
    ;;
hold)
    rate="$1"
    secs="$2"
    wrap "hold-${rate}-$(date +%H%M%S)" "$K6_DIR/mixed.js" "STAGES=${rate} STEP=${secs}s"
    ;;
spike)
    stages="${1:-100,300,600,1000,1500,2000}"
    wrap "spike-$(date +%H%M%S)" "$K6_DIR/spike.js" "STAGES=${stages}"
    ;;
wrap)
    wrap "$1" "$2" "${3:-}"
    ;;
*)
    echo "usage: $0 scan cold|re304|warm|warmcpu|mixed [STAGES] [STEP]" >&2
    echo "       $0 hold RATE SECONDS | spike [STAGES] | wrap NAME SCRIPT ENVS" >&2
    exit 1
    ;;
esac
