// Read-path load test: simulates the origin traffic that survives behind a
// Cloudflare CDN (performance-optimization.md §Who handles the 304).
//
// Three scenarios, selected by SCENARIO env var:
//   cold-miss      — each QID requested once with a plain GET, forcing a full
//                    DB read + render (singleflight collapses concurrent same-
//                    QID misses). Simulates a new post being seen by an edge
//                    for the first time.
//   revalidate-304 — first GET warms the render cache, then the same QID is
//                    re-requested with If-None-Match to reproduce a CDN
//                    revalidation; the origin should answer 304 (bodyless).
//   warm-hit       — a small fixed QID set hit round-robin; all but the first
//                    per QID are render-cache hits (the hot post case).
//
// Usage:
//   SCENARIO=cold-miss      k6 run scripts/loadtest/k6/read.js
//   SCENARIO=revalidate-304 RATE=50 k6 run scripts/loadtest/k6/read.js
//   SCENARIO=warm-hit RATE=100 k6 run scripts/loadtest/k6/read.js
//
// Requires scripts/loadtest/out/qids.json (run seed.sh first).

import http from "k6/http";
import exec from "k6/execution";
import { check } from "k6";
import { SharedArray } from "k6/data";
import {
  originGet,
  bandwidthSummary,
  revalidate304,
  coldMiss200,
  baseURL,
  tlsOptions,
  summaryTrendStats,
  acceptEncodingHeaders,
} from "./lib.js";

const BASE_URL = baseURL();
const SCENARIO = __ENV.SCENARIO || "cold-miss";
const RATE = parseInt(__ENV.RATE || "20", 10);
const DURATION = __ENV.DURATION || "60s";

const qids = new SharedArray("qids", function () {
  return JSON.parse(open(__ENV.QIDS_FILE || "../out/qids.json"));
});

const warmCount = parseInt(__ENV.WARM_COUNT || "10", 10);

export const options = buildOptions();

function buildOptions() {
  // arrival-rate executors give a stable request rate (like vegeta -rate),
  // independent of per-VU think time. cold-miss caps at the origin's ~25 resp/s
  // envelope; revalidate/warm are cheap and tolerate higher rates.
  const rate = SCENARIO === "cold-miss" ? RATE : RATE;
  return {
    scenarios: {
      [SCENARIO]: {
        executor: "constant-arrival-rate",
        rate,
        timeUnit: "1s",
        duration: DURATION,
        preAllocatedVUs: Math.max(rate * 2, 20),
        maxVUs: Math.max(rate * 5, 50),
      },
    },
    thresholds: thresholdsFor(SCENARIO),
    ...tlsOptions,
    summaryTrendStats,
    noConnectionReuse: false,
  };
}

function thresholdsFor(s) {
  if (s === "cold-miss") {
    // Full render behind singleflight; allow generous tail.
    return {
      http_req_duration: ["p(95)<1500"],
      http_req_failed: ["rate<0.02"],
    };
  }
  if (s === "revalidate-304") {
    // 304 is bodyless + render-cache ETag lookup; expect sub-10ms p95 and a
    // high 304 ratio.
    return {
      http_req_duration: ["p(95)<50"],
      http_req_failed: ["rate<0.01"],
      origin_revalidate_304: ["count>0"],
    };
  }
  // warm-hit: render-cache hit, no DB/render.
  return { http_req_duration: ["p(95)<20"], http_req_failed: ["rate<0.01"] };
}

export default function () {
  if (SCENARIO === "cold-miss") {
    // Unique QID per iteration (round-robins when the pool is smaller than
    // rate×duration; raise COUNT in seed.sh for genuine all-miss).
    const i = exec.scenario.iterationInTest >= 0 ? exec.scenario.iterationInTest : 0;
    const idx = (exec.scenario.iterationInTest || 0) % qids.length;
    const res = originGet(BASE_URL, qids[idx]);
    check(res, { "200 or 404": (r) => r.status === 200 || r.status === 404 });
    return;
  }

  if (SCENARIO === "revalidate-304") {
    // First touch warms the cache; subsequent touches carry If-None-Match so
    // the origin serves a 304. Alternate QIDs so the pool isn't a single key.
    const idx = (exec.scenario.iterationInTest || 0) % qids.length;
    const qid = qids[idx];
    const first = http.get(`${BASE_URL}/${qid}`, {
      responseType: "text",
      tags: { name: "read" },
      headers: { ...acceptEncodingHeaders },
    });
    const etag = first.headers && first.headers.Etag ? first.headers.Etag : "";
    if (etag) {
      const reval = originGet(BASE_URL, qid, etag);
      check(reval, { "revalidate 304": (r) => r.status === 304 });
    }
    return;
  }

  if (SCENARIO === "warm-hit") {
    // Fixed small set, round-robin → render-cache hits after first per QID.
    const idx = (exec.scenario.iterationInTest || 0) % Math.min(warmCount, qids.length);
    const res = originGet(BASE_URL, qids[idx]);
    check(res, { 200: (r) => r.status === 200 });
    return;
  }
}

export function handleSummary(data) {
  const b = bandwidthSummary(data);
  const r304 =
    (data.metrics.origin_revalidate_304 && data.metrics.origin_revalidate_304.values.count) || 0;
  const c200 =
    (data.metrics.origin_cold_miss_200 && data.metrics.origin_cold_miss_200.values.count) || 0;
  const total = r304 + c200;
  const r304Pct = total > 0 ? ((r304 / total) * 100).toFixed(1) : "0";
  const body = `
==================== READ [${SCENARIO}] ====================
latency  p50=${b.p50}ms  p95=${b.p95}ms  p99=${b.p99}ms
bandwidth recv=${b.recvMB}MB (${b.avgRecvMbps}Mbps avg, ${b.originUtilPct}% of 3Mbps origin envelope)
origin   304=${r304} (${r304Pct}%)  cold-200=${c200}
duration ${b.durSec}s
============================================================
`;
  return { stdout: body };
}
