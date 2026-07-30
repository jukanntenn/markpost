// Write-path load test: POST /:post_key, measuring create latency and the
// asynchronous delivery fan-out. Each create enqueues a delivery job (the post
// service's Enqueue is non-blocking); the dispatcher drains pending attempts on
// a background ticker. This scenario verifies writes don't block on delivery
// and that the delivery counters (markpost.delivery.dispatched_total /
// pending) advance without backlog — read those from metrics-*.jsonl after the
// run.
//
// The L2 rate limit is 10/min per user (keyed on post_key). To stay under it,
// seed many users (seed_write.sh USERS≥100) and round-robin their keys; the
// default 10 req/s × 60s = 600 requests needs ≥100 users (600/6).
//
// Usage:
//   k6 run scripts/loadtest/k6/write.js
//   RATE=10 DURATION=60s k6 run scripts/loadtest/k6/write.js
//
// Requires scripts/loadtest/out/write_keys.txt (run seed_write.sh first).

import http from 'k6/http';
import exec from 'k6/execution';
import { check } from 'k6';
import { SharedArray } from 'k6/data';
import { bandwidthSummary, tileBody, normalBodySize, baseURL, tlsOptions, summaryTrendStats } from './lib.js';

const BASE_URL = baseURL();
const RATE = parseInt(__ENV.RATE || '10', 10);
const DURATION = __ENV.DURATION || '60s';

const keys = new SharedArray('keys', function () {
  const raw = open(__ENV.KEYS_FILE || '../out/write_keys.txt');
  return raw.trim().split('\n').filter((k) => k.length > 0);
});

export const options = {
  scenarios: {
    write: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: Math.max(RATE * 2, 20),
      maxVUs: Math.max(RATE * 5, 50),
    },
  },
  thresholds: {
    // Create is a DB insert + non-blocking Enqueue; should stay well under 200ms.
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.02'],
  },
  ...tlsOptions,
  summaryTrendStats,
  noConnectionReuse: false,
};

const headers = { 'Content-Type': 'application/json' };

export default function () {
  const i = exec.scenario.iterationInTest || 0;
  const key = keys[i % keys.length];
  // Body drawn from a normal distribution around 32 KiB (spec workload).
  const body = JSON.stringify({
    title: `Load test post ${i}`,
    body: tileBody(normalBodySize()),
  });
  const res = http.post(`${BASE_URL}/${key}`, body, { headers, tags: { name: 'create' } });
  check(res, {
    'created 201': (r) => r.status === 201,
    'not rate-limited': (r) => r.status !== 429,
  });
}

export function handleSummary(data) {
  const b = bandwidthSummary(data);
  const reqs = data.metrics.http_reqs && data.metrics.http_reqs.values.count;
  const failRate = data.metrics.http_req_failed && data.metrics.http_req_failed.values.rate;
  const body = `
==================== WRITE ====================
latency  p50=${b.p50}ms  p95=${b.p95}ms  p99=${b.p99}ms
requests ${reqs}  failed=${(failRate * 100).toFixed(1)}%
bandwidth sent=${b.sentMB}MB  recv=${b.recvMB}MB
duration ${b.durSec}s
note     verify markpost.delivery.dispatched_total ≈ created count
         in metrics-*.jsonl (async fan-out should not backlog)
============================================================
`;
  return { stdout: body };
}
