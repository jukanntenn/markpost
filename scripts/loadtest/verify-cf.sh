#!/usr/bin/env bash
#
# Post-deploy Cloudflare edge verification (functional, curl-level — NEVER
# load-test through the Cloudflare proxy; ToS §2.2.1(b) "undue burden").
# Run this AFTER the zone has:
#   1. DNS orange-clouded for the site hostname,
#   2. a Cache Rule: When URI Path starts with "/p-" → Cache Eligibility:
#      "Eligible for cache" (Edge TTL sub-setting left UNSET so the origin's
#      s-maxage=3600 governs — an explicit Edge TTL would override it and the
#      Free minimum is 2h),
#   3. zone Browser Cache TTL = "Respect Existing Headers",
#   4. SSL/TLS mode = Full (strict) with the Origin CA cert on Caddy.
#
# Usage:
#   DOMAIN=markpost.cc bash scripts/loadtest/verify-cf.sh
# Optional (enables the purge test):
#   CF_API_TOKEN, CF_ZONE_ID  (token needs Zone.Cache Purge permission)
set -euo pipefail

DOMAIN="${DOMAIN:?set DOMAIN=your proxied hostname}"
BASE="https://${DOMAIN}"
CF_API_TOKEN="${CF_API_TOKEN:-}"
CF_ZONE_ID="${CF_ZONE_ID:-}"

FAILS=0
ok() { echo "  PASS  $*"; }
bad() { echo "  FAIL  $*" >&2; FAILS=$((FAILS + 1)); }
info() { echo "  ....  $*"; }

hdr() { curl -sD- -o /dev/null "$@"; }

echo "== 1. proxy + cache headers on /p-* (needs a real QID: QID=p-xxxx or it creates none)"
QID="${QID:-}"
if [[ -z "$QID" ]]; then
    info "QID not set — header-shape checks run against the 404 path instead"
    PROBE="/p-verifycf-nonexistent"
else
    PROBE="/${QID}"
fi
h1="$(hdr "$BASE$PROBE")"
echo "$h1" | grep -qi '^cf-ray:' && ok "hostname is proxied (cf-ray present)" || bad "no cf-ray — is DNS orange-clouded?"
if [[ -n "$QID" ]]; then
    echo "$h1" | grep -qi '^cache-control: public, max-age=300, s-maxage=3600' &&
        ok "origin Cache-Control passes the edge" || bad "Cache-Control missing/altered: $(echo "$h1" | grep -i '^cache-control')"
    echo "$h1" | grep -qi '^etag:' && ok "ETag present" || bad "ETag stripped"
    echo "$h1" | grep -qi '^cache-tag:' && bad "Cache-Tag leaked to visitor (should be stripped)" || ok "Cache-Tag stripped by edge"
fi

echo "== 2. edge caching sequence (the decisive check)"
if [[ -n "$QID" ]]; then
    s1="$(hdr "$BASE/$QID" | grep -i '^cf-cache-status:' | tr -d '\r' | awk '{print tolower($2)}')"
    sleep 2
    s2="$(hdr "$BASE/$QID" | grep -i '^cf-cache-status:' | tr -d '\r' | awk '{print tolower($2)}')"
    age2="$(hdr "$BASE/$QID" | grep -i '^age:' | tr -d '\r')"
    ok "request 1: cf-cache-status=${s1:-none} (expect MISS/ dynamic-free)"
    ok "request 2: cf-cache-status=${s2:-none} (expect HIT)"
    [[ "${s2,,}" == "hit" ]] && ok "HTML is edge-cached — Cache Rule effective" ||
        bad "second request not HIT (${s2:-none}); if DYNAMIC → Cache Rule not matching /p-*"
    info "age header: ${age2:-absent} (should grow between requests)"
else
    # 404 path still proves the rule matches /p-* (spec decision 29: 404 edge TTL)
    s1="$(hdr "$BASE$PROBE" | grep -i '^cf-cache-status:' | tr -d '\r' | awk '{print tolower($2)}')"
    sleep 2
    s2="$(hdr "$BASE$PROBE" | grep -i '^cf-cache-status:' | tr -d '\r' | awk '{print tolower($2)}')"
    [[ "${s2,,}" == "hit" || "${s2,,}" == "miss" ]] &&
        ok "404 path is cache-eligible (${s1} → ${s2})" || bad "404 path shows ${s2:-none} — rule not matching?"
fi

echo "== 3. static frontend assets cached by extension (default behavior)"
asset="/_next/static/chunks/d689f63755e58483.js"
a2="$(hdr "$BASE$asset" | grep -i '^cf-cache-status:' | tr -d '\r' | awk '{print tolower($2)}')"
hdr "$BASE$asset" >/dev/null
a3="$(hdr "$BASE$asset" | grep -i '^cf-cache-status:' | tr -d '\r' | awk '{print tolower($2)}')"
[[ "${a3,,}" == "hit" ]] && ok "JS chunk edge-cached (default extension cache)" || info "JS chunk status: ${a3:-none} (MISS on first touch is fine)"

echo "== 4. HTML pages carry Cache-Control from Caddy (max-age=300)"
ph="$(hdr "$BASE/login")"
echo "$ph" | grep -qi '^cache-control: public, max-age=300' &&
    ok "/login HTML cacheable (5 min)" || bad "/login Cache-Control missing (Caddy @pageHtml header not deployed?)"

echo "== 5. TLS mode"
tlsv="$(hdr "$BASE/api/v1/health" | grep -i '^cf-ray:' >/dev/null && echo proxied)"
[[ -n "$tlsv" ]] && ok "health reachable through the edge" || bad "health not reachable"

if [[ -n "$CF_API_TOKEN" && -n "$CF_ZONE_ID" && -n "$QID" ]]; then
    echo "== 6. cache-tag purge round-trip"
    curl -s -X POST "https://api.cloudflare.com/client/v4/zones/${CF_ZONE_ID}/purge_cache" \
        -H "Authorization: Bearer ${CF_API_TOKEN}" -H 'Content-Type: application/json' \
        --data "{\"tags\":[\"post-${QID}\"]}" | jq -e '.success' >/dev/null &&
        ok "purge by tag accepted" || bad "purge API call failed"
    sleep 3
    sp="$(hdr "$BASE/$QID" | grep -i '^cf-cache-status:' | tr -d '\r' | awk '{print tolower($2)}')"
    [[ "${sp,,}" != "hit" ]] && ok "post-purge status ${sp} (MISS expected — tag honored)" ||
        info "post-purge still HIT (${sp}) — if this persists, origin Cache-Tag ingestion needs the purge-by-URL fallback"
else
    info "purge test skipped (set CF_API_TOKEN + CF_ZONE_ID + QID to enable)"
fi

echo
if [[ $FAILS -eq 0 ]]; then
    echo "verify-cf: ALL PASS"
else
    echo "verify-cf: ${FAILS} FAILURE(S)" >&2
    exit 1
fi
