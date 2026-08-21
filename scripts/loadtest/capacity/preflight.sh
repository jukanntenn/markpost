#!/usr/bin/env bash
#
# Preflight: the light validation the capacity plan requires BEFORE any long
# run, so a mis-shaped stack or a broken scenario fails in minutes instead of
# after a 60-minute soak.
#
#   preflight.sh basic  stack health, socket DSN, zstd/gzip, shaping on/off
#                       timing, L1 relaxation, login, monitor self-test
#                       (no seed data needed beyond the admin user)
#   preflight.sh mini   30-60s mini-runs of every scenario family (cold /
#                       re304 / warm / mixed) with assertions on the summary
#                       exports (requires seeds: qids.json + write_keys.txt)
#   preflight.sh all    both
#
# Exits non-zero on the first failed assertion.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
K6_DIR="$ROOT/scripts/loadtest/k6"
BIN_DIR="$ROOT/scripts/loadtest/k6-bin"
OUT_DIR="$ROOT/scripts/loadtest/out/results/preflight"
BASE="https://localhost:2053"
APP="${APP_CONTAINER:-markpost-capacity-app-1}"
# The largest static chunk (~356 KB) doubles as the shaping probe payload.
BIG_ASSET="/_next/static/chunks/d689f63755e58483.js"

FAILS=0
ok() { echo "  PASS  $*"; }
bad() { echo "  FAIL  $*" >&2; FAILS=$((FAILS + 1)); }

curlk() { curl -sk "$@"; }

download_secs() {
    # One fetch of the big static asset, prints wall seconds (millisecond res).
    local t0 t1
    t0=$(date +%s.%N)
    curlk -o /dev/null "$BASE$BIG_ASSET"
    t1=$(date +%s.%N)
    echo "$t0 $t1" | awk '{printf "%.3f", $2 - $1}'
}

basic() {
    echo "== [basic] stack health"
    code="$(curlk -o /dev/null -w '%{http_code}' "$BASE/api/v1/health")"
    [[ "$code" == "200" ]] && ok "health 200 via Caddy" || bad "health returned $code"

    echo "== [basic] DB over Unix socket (production topology)"
    dsn="$(grep '^dsn' "$SCRIPT_DIR/config.toml")"
    [[ "$dsn" == *"/var/run/postgresql"* ]] && ok "config DSN is socket-form" || bad "config DSN not socket-form: $dsn"
    qid="$(jq -r '.[0]' "$ROOT/scripts/loadtest/out/qids.json" 2>/dev/null || true)"
    if [[ -n "${qid:-}" && "$qid" != "null" ]]; then
        code="$(curlk -o /dev/null -w '%{http_code}' "$BASE/$qid")"
        [[ "$code" == "200" ]] && ok "GET /$qid 200 (socket DSN works end-to-end)" || bad "GET /$qid returned $code"
    else
        echo "  SKIP  no seeded qids yet (run seed.sh before mini)"
    fi

    echo "== [basic] response headers + compression"
    if [[ -n "${qid:-}" && "$qid" != "null" ]]; then
        hdrs="$(curlk -D- -o /dev/null "$BASE/$qid")"
        echo "$hdrs" | grep -qi '^etag:' && ok "ETag present" || bad "ETag missing"
        echo "$hdrs" | grep -qi '^cache-control: public, max-age=300, s-maxage=3600' &&
            ok "Cache-Control public,max-age=300,s-maxage=3600" || bad "Cache-Control wrong/missing"
        echo "$hdrs" | grep -qi '^cache-tag: post-' && ok "Cache-Tag present" || bad "Cache-Tag missing"
        zstd_hdr="$(curlk -H 'Accept-Encoding: zstd' -D- -o /dev/null "$BASE/$qid" | grep -i '^content-encoding:')"
        [[ "$zstd_hdr" == *zstd* ]] && ok "zstd negotiation works (k6 fidelity fixed)" || bad "no Content-Encoding: zstd ($zstd_hdr)"
        gzip_hdr="$(curlk -H 'Accept-Encoding: gzip' -D- -o /dev/null "$BASE/$qid" | grep -i '^content-encoding:')"
        [[ "$gzip_hdr" == *gzip* ]] && ok "gzip fallback works" || bad "no Content-Encoding: gzip ($gzip_hdr)"
    fi

    echo "== [basic] unknown QID → 404 JSON"
    code="$(curlk -o /dev/null -w '%{http_code}' "$BASE/p-doesnotexist123")"
    [[ "$code" == "404" ]] && ok "404 on unknown QID" || bad "unknown QID returned $code"

    echo "== [basic] egress shaping"
    bash "$SCRIPT_DIR/shape.sh" clear
    unshaped="$(download_secs)"
    bash "$SCRIPT_DIR/shape.sh" apply >/dev/null
    shaped="$(download_secs)"
    ok "unshaped fetch of 356KB asset: ${unshaped}s"
    ok "shaped fetch of 356KB asset:   ${shaped}s"
    awk -v s="$shaped" -v u="$unshaped" 'BEGIN { exit !(s > u + 0.5 && s > 0.8) }' &&
        ok "shaping measurably active (3mbit + 30ms netem)" || bad "shaping not effective (unshaped=${unshaped}s shaped=${shaped}s)"

    echo "== [basic] L1 read limiter relaxed (origin-ceiling mode)"
    limited=0
    for i in $(seq 1 60); do
        c="$(curlk -o /dev/null -w '%{http_code}' "$BASE/p-probe-$i-$RANDOM")"
        [[ "$c" == "429" ]] && limited=1 && break
    done
    [[ $limited == 0 ]] && ok "60 rapid reads, no 429" || bad "got 429 — L1 not relaxed"

    echo "== [basic] login works (seeded loadtest_1)"
    login_body="$(curlk -X POST "$BASE/api/v1/auth/login" -H 'Content-Type: application/json' \
        -d '{"username":"loadtest_1","password":"loadtestpass"}')"
    echo "$login_body" | jq -e '.token' >/dev/null 2>&1 &&
        ok "login returns token" || bad "login failed: $(echo "$login_body" | head -c 200)"

    echo "== [basic] monitor self-test"
    rows="$(bash "$SCRIPT_DIR/monitor.sh" sample)"
    app_row="$(echo "$rows" | head -1)"
    echo "$app_row" | awk -F, 'NF==8 && $3!=$4' >/dev/null 2>&1 &&
        ok "monitor samples parse ($app_row)" || bad "monitor row malformed: $app_row"
}

mini() {
    mkdir -p "$OUT_DIR"
    [[ -x "$BIN_DIR/k6" ]] || { bash "$ROOT/scripts/loadtest/run.sh" </dev/null >/dev/null 2>&1 || true; }
    [[ -x "$BIN_DIR/k6" ]] || { echo "k6 binary missing; run scripts/loadtest/run.sh once" >&2; exit 1; }

    assert_summary() {
        local name="$1" need_metric="$2"
        local summary="$OUT_DIR/${name}-summary.json"
        [[ -f "$summary" ]] || { bad "$name: no summary export"; return; }
        jq -e '(.metrics.http_reqs.values.count // .metrics.http_reqs.count) > 0' "$summary" >/dev/null &&
            ok "$name: requests recorded" || bad "$name: zero requests"
        local failrate
        failrate="$(jq '(.metrics.http_req_failed.values.rate // .metrics.http_req_failed.value // 1)' "$summary")"
        awk -v f="$failrate" 'BEGIN { exit !(f < 0.05) }' &&
            ok "$name: fail rate $(awk "BEGIN{printf \"%.1f%%\", $failrate*100}")" ||
            bad "$name: fail rate too high ($failrate)"
        if [[ -n "$need_metric" ]]; then
            jq -e --arg m "$need_metric" '(.metrics[$m].values.count // .metrics[$m].count) > 0' "$summary" >/dev/null &&
                ok "$name: $need_metric > 0" || bad "$name: $need_metric == 0"
        fi
    }

    run_mini() {
        local name="$1" script="$2"
        shift 2
        # shellcheck disable=SC2086
        env "$@" taskset -c 2-11 "$BIN_DIR/k6" run "$script" \
            --summary-export="$OUT_DIR/${name}-summary.json" >/dev/null 2>&1 || true
    }

    echo "== [mini] cold (5/s × 30s)"
    run_mini cold "$K6_DIR/capacity.js" MECH=cold STAGES=5 STEP=30s
    assert_summary cold origin_cold_miss_200

    echo "== [mini] re304 (30/s × 30s)"
    run_mini re304 "$K6_DIR/capacity.js" MECH=re304 STAGES=30 STEP=30s PREWARM=20
    assert_summary re304 origin_revalidate_304

    echo "== [mini] warm (30/s × 30s)"
    run_mini warm "$K6_DIR/capacity.js" MECH=warm STAGES=30 STEP=30s
    assert_summary warm origin_cold_miss_200

    echo "== [mini] mixed (10/s × 45s)"
    run_mini mixed "$K6_DIR/mixed.js" STAGES=10 STEP=45s PREWARM=20
    assert_summary mixed mixed_read304
}

case "${1:-all}" in
basic) basic ;;
mini) mini ;;
all)
    basic
    mini
    ;;
*)
    echo "usage: $0 basic|mini|all" >&2
    exit 1
    ;;
esac

echo
if [[ $FAILS -eq 0 ]]; then
    echo "preflight: ALL PASS"
else
    echo "preflight: ${FAILS} FAILURE(S)" >&2
    exit 1
fi
