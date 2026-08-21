// Soak (longevity) test: a mixed read + write load sustained for an extended
// period to surface slow failures that short runs miss — ristretto memory
// reaching steady state vs. unbounded growth, Postgres connection-pool leaks,
// and goroutine accumulation in the delivery dispatcher.
//
// Load profile (origin view, behind the CDN):
//   ~15 read req/s (mostly revalidation 304 + some cold miss)
//   ~2  write req/s
// held for HOLD (default 60m), with a 2m ramp up/down on either side so the
// origin isn't hit by a cold-start thundering herd. 60m deliberately exceeds
// both the Postgres ConnMaxLifetime (30m) and the CDN s-maxage (1h) so a full
// connection-recycle and cache-revalidation cycle occurs.
//
// How to read the result:
//   - process.runtime.go.mem.heap_alloc should plateau near the render-cache
//     MaxCost (128 MiB), not climb monotonically.
//   - process.runtime.go.goroutines should be stable (delivery worker pool +
//     http handlers), not grow without bound.
//   - markpost.delivery.pending should track write rate, not accumulate.
//   All three are read from metrics-*.jsonl after the run.
//
// Usage:
//   k6 run scripts/loadtest/k6/soak.js
//   HOLD=60m READ_RATE=15 WRITE_RATE=2 k6 run scripts/loadtest/k6/soak.js
//
// Requires both scripts/loadtest/out/qids.json and write_keys.txt.

import http from "k6/http";
import exec from "k6/execution";
import { check } from "k6";
import { SharedArray } from "k6/data";
import {
  originGet,
  bandwidthSummary,
  tileBody,
  normalBodySize,
  baseURL,
  tlsOptions,
  summaryTrendStats,
  acceptEncodingHeaders,
} from "./lib.js";

const BASE_URL = baseURL();
const READ_RATE = parseInt(__ENV.READ_RATE || "15", 10);
const WRITE_RATE = parseInt(__ENV.WRITE_RATE || "2", 10);
const HOLD = __ENV.HOLD || "60m";
const RAMP = __ENV.RAMP || "2m";

const qids = new SharedArray("qids", function () {
  return JSON.parse(open(__ENV.QIDS_FILE || "../out/qids.json"));
});
const keys = new SharedArray("keys", function () {
  const raw = open(__ENV.KEYS_FILE || "../out/write_keys.txt");
  return raw
    .trim()
    .split("\n")
    .filter((k) => k.length > 0);
});

export const options = {
  scenarios: {
    read: {
      executor: "ramping-arrival-rate",
      startRate: 0,
      timeUnit: "1s",
      preAllocatedVUs: Math.max(READ_RATE * 3, 50),
      maxVUs: Math.max(READ_RATE * 6, 100),
      stages: [
        { duration: RAMP, target: READ_RATE },
        { duration: HOLD, target: READ_RATE },
        { duration: RAMP, target: 0 },
      ],
    },
    write: {
      executor: "ramping-arrival-rate",
      startRate: 0,
      timeUnit: "1s",
      preAllocatedVUs: Math.max(WRITE_RATE * 5, 20),
      maxVUs: Math.max(WRITE_RATE * 10, 40),
      stages: [
        { duration: RAMP, target: WRITE_RATE },
        { duration: HOLD, target: WRITE_RATE },
        { duration: RAMP, target: 0 },
      ],
    },
  },
  thresholds: {
    "http_req_duration{scenario:read}": ["p(95)<200"],
    "http_req_duration{scenario:write}": ["p(95)<500"],
    http_req_failed: ["rate<0.02"],
  },
  ...tlsOptions,
  summaryTrendStats,
  noConnectionReuse: false,
};

const writeHeaders = { "Content-Type": "application/json" };

export default function () {
  const sc = exec.scenario.name;
  const it = exec.scenario.iterationInTest || 0;
  if (sc === "read") {
    // 70% revalidation (warm QID + If-None-Match), 30% cold miss (fresh QID).
    const warm = it % 10 < 7;
    const idx = warm ? it % Math.min(50, qids.length) : it % qids.length;
    const qid = qids[idx];
    if (warm) {
      const first = http.get(`${BASE_URL}/${qid}`, {
        responseType: "text",
        tags: { name: "read" },
        headers: { ...acceptEncodingHeaders },
      });
      const etag = first.headers && first.headers.Etag ? first.headers.Etag : "";
      if (etag) originGet(BASE_URL, qid, etag);
    } else {
      originGet(BASE_URL, qid);
    }
    return;
  }
  // write
  const key = keys[it % keys.length];
  const body = JSON.stringify({
    // "Load" must appear in the title: seeded delivery channels filter on the
    // keyword, so without it the delivery fan-out path is silently skipped.
    title: `Load soak post ${it}`,
    body: tileBody(normalBodySize()),
  });
  const res = http.post(`${BASE_URL}/${key}`, body, {
    headers: writeHeaders,
    tags: { name: "create" },
  });
  check(res, {
    "created 201": (r) => r.status === 201,
    "not rate-limited": (r) => r.status !== 429,
  });
}

export function handleSummary(data) {
  const b = bandwidthSummary(data);
  const body = `
==================== SOAK (HOLD=${HOLD}) ====================
latency  p50=${b.p50}ms  p95=${b.p95}ms  p99=${b.p99}ms
bandwidth recv=${b.recvMB}MB (${b.avgRecvMbps}Mbps avg, ${b.originUtilPct}% of 3Mbps)
duration ${b.durSec}s
CHECK (from metrics-*.jsonl after run):
  - heap_alloc plateaus ~128MiB (not monotonic rise)
  - goroutines stable (not climbing)
  - delivery.pending tracks write rate (no backlog)
============================================================
`;
  return { stdout: body };
}
