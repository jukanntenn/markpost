#!/usr/bin/env bash
#
# SUT resource sampler for the capacity stack: per-container CPU usage, memory
# and PSI (pressure stall information) from the containers' own cgroup v2
# files, written as CSV rows while a load run is in flight.
#
# PSI note: cpu.pressure/memory.pressure "some avg10" is the modern
# saturation indicator — sustained some-avg10 > 10% means tasks are
# stalling for CPU/memory. CPU% is computed by the analyzer as
# Δusage_usec / Δwalltime / 2 cores (the cpuset gives each container 2 cores).
#
# Usage:
#   bash scripts/loadtest/capacity/monitor.sh start OUT.csv [interval_sec]   # backgrounds itself
#   bash scripts/loadtest/capacity/monitor.sh stop                            # kills the sampler
#
# CSV columns:
#   ts,container,cpu_usage_usec,cpu_nr_periods,cpu_nr_throttled,
#   cpu_psi_some_avg10,mem_current,mem_psi_some_avg10
# A row of "ts,container,ERR,..." marks a sample the container could not
# serve (e.g. mid-restart during restart-storm).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_FILE="$SCRIPT_DIR/monitor.pid"
APP="${APP_CONTAINER:-markpost-capacity-app-1}"
PG="${PG_CONTAINER:-markpost-capacity-postgres-1}"

sample_one() {
    local ts container
    ts="$(date +%s.%N)"
    container="$1"
    local out
    if ! out="$(docker exec "$container" sh -c '
        cat /sys/fs/cgroup/cpu.stat /sys/fs/cgroup/cpu.pressure \
            /sys/fs/cgroup/memory.current /sys/fs/cgroup/memory.pressure 2>/dev/null' 2>/dev/null)"; then
        echo "${ts},${container},ERR,ERR,ERR,ERR,ERR,ERR"
        return
    fi
    # cpu.stat: usage_usec nr_periods nr_throttled throttled_usec ...
    # *.pressure: two lines "some avg10=.. avg60=.. avg300=.. total=.." / "full ..."
    local usage periods throttled cpu_psi mem_current mem_psi
    usage="$(echo "$out" | awk '/^usage_usec/ {print $2; exit}')"
    periods="$(echo "$out" | awk '/^nr_periods/ {print $2; exit}')"
    throttled="$(echo "$out" | awk '/^nr_throttled/ {print $2; exit}')"
    cpu_psi="$(echo "$out" | awk '/^some/ && !seen {print $2; seen=1}')"
    mem_current="$(echo "$out" | awk '/^[0-9]+$/ {print $1; exit}')"
    mem_psi="$(echo "$out" | awk '/^some/ && c==1 {print $2; exit} /^full/ {c=1}')"
    echo "${ts},${container},${usage:-0},${periods:-0},${throttled:-0},${cpu_psi:-some=avg10=0.00},${mem_current:-0},${mem_psi:-some=avg10=0.00}"
}

sampler() {
    local out_csv interval
    out_csv="$1"
    interval="${2:-2}"
    echo "ts,container,cpu_usage_usec,cpu_nr_periods,cpu_nr_throttled,cpu_psi_some,mem_current,mem_psi_some" >>"$out_csv"
    while :; do
        sample_one "$APP" >>"$out_csv"
        sample_one "$PG" >>"$out_csv"
        sleep "$interval"
    done
}

case "${1:-}" in
start)
    OUT_CSV="$2"
    INTERVAL="${3:-2}"
    if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
        echo "monitor already running (pid $(cat "$PID_FILE"))" >&2
        exit 1
    fi
    sampler "$OUT_CSV" "$INTERVAL" &
    echo $! >"$PID_FILE"
    echo "==> monitor sampling to $OUT_CSV every ${INTERVAL}s (pid $(cat "$PID_FILE"))"
    ;;
stop)
    if [[ -f "$PID_FILE" ]]; then
        kill "$(cat "$PID_FILE")" 2>/dev/null || true
        rm -f "$PID_FILE"
        echo "==> monitor stopped"
    else
        echo "no monitor running" >&2
    fi
    ;;
sample)
    # one-shot: print a single pair of rows to stdout (for preflight checks)
    sample_one "$APP"
    sample_one "$PG"
    ;;
*)
    echo "usage: $0 start OUT.csv [interval] | stop | sample" >&2
    exit 1
    ;;
esac
