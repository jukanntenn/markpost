#!/usr/bin/env bash
#
# Origin egress shaping for the capacity stack: reproduces the VPS's 3 Mbps
# uplink + ~30ms RTT to the Cloudflare edge on the app container's eth0.
#
# Method: tc (tbf + netem) installed into the app container's network
# namespace from a privileged sidecar sharing that netns. No host root or
# host-side tc is needed (docker group membership suffices for --cap-add
# NET_ADMIN). Shaping applies to ALL app egress — HTTP responses AND the
# delivery-webhook fan-out to webhook-mock — which mirrors the real VPS where
# everything shares one uplink. Postgres rides the Unix-socket volume, not
# eth0, so DB traffic is unaffected (same as production).
#
# Usage:
#   bash scripts/loadtest/capacity/shape.sh apply   [RATE] [DELAY]   # default 3mbit 30ms
#   bash scripts/loadtest/capacity/shape.sh status
#   bash scripts/loadtest/capacity/shape.sh clear
#
# Environment:
#   APP_CONTAINER  default markpost-capacity-app-1
set -euo pipefail

APP_CONTAINER="${APP_CONTAINER:-markpost-capacity-app-1}"
RATE="${2:-3mbit}"
DELAY="${3:-30ms}"

# The sidecar needs iproute2; alpine installs it on the fly (image is cached
# after the first run). tbf first (bucket + drop policy), netem underneath as
# the shaping child so the added RTT applies per packet.
tc_cmd() {
    docker run --rm --network "container:${APP_CONTAINER}" --cap-add NET_ADMIN \
        alpine:3.21 sh -c "apk add -q --no-cache iproute2 >/dev/null 2>&1; $1"
}

case "${1:-}" in
apply)
    echo "==> shaping ${APP_CONTAINER} eth0 egress: rate=${RATE} delay=${DELAY} ±5ms"
    tc_cmd "tc qdisc replace dev eth0 root handle 1: tbf rate ${RATE} burst 32kb latency 400ms && \
             tc qdisc replace dev eth0 parent 1:1 handle 10: netem delay ${DELAY} 5ms"
    tc_cmd "tc -s qdisc show dev eth0"
    ;;
status)
    tc_cmd "tc -s qdisc show dev eth0"
    ;;
clear)
    echo "==> clearing qdiscs on ${APP_CONTAINER} eth0"
    tc_cmd "tc qdisc del dev eth0 root 2>/dev/null || true"
    echo "    done"
    ;;
*)
    echo "usage: $0 apply|status|clear [rate] [delay]" >&2
    exit 1
    ;;
esac
