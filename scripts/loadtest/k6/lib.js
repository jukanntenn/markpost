// Shared helpers for the k6 load-test scenarios.
//
// design: performance-optimization.md models the origin behind a Cloudflare
// CDN. The origin almost never sees a plain repeat GET (the edge absorbs it);
// it sees (a) cold misses — first origin render for a QID — and (b) CDN
// revalidations — a conditional GET carrying the last ETag once s-maxage (1h)
// elapses, to which the origin replies 304 (bodyless) from the render cache.
// These helpers let each scenario reproduce that request shape without a real
// CDN by toggling the If-None-Match header.

import http from "k6/http";
import { Counter, Trend } from "k6/metrics";

// ── Base URL + TLS ────────────────────────────────────────────────────────
// The e2e compose (the load-test target) fronts the app with Caddy using a
// self-signed internal CA on https://localhost:2053. SCHEME/HOST/PORT let a
// plain-HTTP dev server be targeted too.
export function baseURL() {
  const scheme = __ENV.SCHEME || "https";
  const host = __ENV.HOST || "localhost";
  const port = __ENV.PORT || "2053";
  return `${scheme}://${host}:${port}`;
}

// tlsOptions holds the insecureSkipTLSVerify flag for self-signed e2e Caddy.
// Spread into each scenario's options.
export const tlsOptions = {
  insecureSkipTLSVerify: true,
};

// summaryTrendStats adds p(99) to the default trend stats so the reports and
// handleSummary can read a 99th percentile (k6 defaults to med/p(90)/p(95)).
export const summaryTrendStats = ["avg", "min", "med", "max", "p(90)", "p(95)", "p(99)"];

// ── Custom metrics ────────────────────────────────────────────────────────
// revalidate_304 / cold_200 split the origin's read work into its two real
// modes (per the spec's "Who handles the 304" table); bytes_received tracks
// origin egress against the 3 Mbps / 1 TB-month envelope.
export const revalidate304 = new Counter("origin_revalidate_304");
export const coldMiss200 = new Counter("origin_cold_miss_200");
export const bytesPerReq = new Trend("origin_bytes_per_request");

// ── Markdown body generation (mirrors backend/tools/write_targets) ────────
const markdownBlocks = [
  "## 深入分析\n\n技术选型对项目成功至关重要。本文分析各种方案的优缺点。\n\n" +
    "- 第一要点\n- 第二要点\n- 第三要点\n\n" +
    "| 维度 | A | B |\n|------|---|---|\n| 延迟 | 低 | 中 |\n\n",
  "### 代码\n\n```go\nfunc f(x int) int { return x * 2 }\n```\n\n" +
    "参考 [文档](https://example.com) 和 `内联代码`。\n\n",
  "#### 权衡\n\n**吞吐量** 与 **延迟** 存在权衡。~~过度优化~~ 提前优化引入复杂度。\n\n> 简单是终极的复杂。\n\n",
  "##### 列表\n\n1. CAP 定理\n2. 缓存失效\n3. 恰好一次语义\n\n",
];

// utf8ByteLength returns the UTF-8 byte length of s (k6 has no TextEncoder).
// Code points < 0x80 are 1 byte; < 0x800 are 2; otherwise 3 (the fixture has
// no astral-plane chars).
function utf8ByteLength(s) {
  let n = 0;
  for (let i = 0; i < s.length; i++) {
    const c = s.charCodeAt(i);
    n += c < 0x80 ? 1 : c < 0x800 ? 2 : 3;
  }
  return n;
}

// tileBody builds a markdown body of approximately targetBytes (UTF-8 bytes,
// not chars) by tiling the block set, matching the spec's 32 KiB average post
// size. The fixture is CJK-heavy (3 bytes/char), so byte-length is what the
// backend's body_max_bytes validator counts.
export function tileBody(targetBytes) {
  let body = "";
  let i = 0;
  while (utf8ByteLength(body) < targetBytes) {
    body += markdownBlocks[i % markdownBlocks.length];
    i++;
  }
  return body;
}

// normalBodySize draws from a normal distribution for the *byte* length of the
// body, clamped under the backend's 32 KiB body_max_bytes limit. The validator
// counts UTF-8 bytes (len([]byte)), and the markdown fixture is CJK-heavy
// (3 bytes/char), so we cap the target well below the limit and tileBody
// measures its output in bytes.
export function normalBodySize(mean = 18000, stddev = 4000) {
  // Box-Muller transform for a standard-normal sample.
  const u = 1 - Math.random();
  const v = Math.random();
  const z = Math.sqrt(-2 * Math.log(u)) * Math.cos(2 * Math.PI * v);
  let size = Math.round(mean + z * stddev);
  const min = 1024;
  const max = 30000; // < 32768 body_max_bytes; CJK bytes counted, not chars
  if (size < min) size = min;
  if (size > max) size = max;
  return size;
}

// ── Read helpers ──────────────────────────────────────────────────────────

// originGet issues a GET for a QID, optionally simulating a CDN revalidation
// (If-None-Match). It records the 304/200 split and per-request bytes so the
// report can distinguish cold-miss cost from cheap revalidation.
export function originGet(baseUrl, qid, etag) {
  const params = { tags: { name: "read" }, responseType: "text" };
  if (etag) {
    params.headers = { "If-None-Match": etag };
  }
  const res = http.get(`${baseUrl}/${qid}`, params);
  const received = (res.body && res.body.length) || 0;
  bytesPerReq.add(received);
  if (res.status === 304) {
    revalidate304.add(1);
  } else if (res.status === 200) {
    coldMiss200.add(1);
  }
  return res;
}

// ── Bandwidth + latency summary (shared by all scenarios) ─────────────────
export function bandwidthSummary(data) {
  const durMs = data.state && data.state.testRunDurationMs ? data.state.testRunDurationMs : 1;
  const durSec = durMs / 1000;
  const recv = data.metrics.data_received && data.metrics.data_received.values;
  const sent = data.metrics.data_sent && data.metrics.data_sent.values;
  const totalRecvMB = recv ? recv.count / (1024 * 1024) : 0;
  const totalSentMB = sent ? sent.count / (1024 * 1024) : 0;
  const avgRecvMbps = ((totalRecvMB * 8) / durSec).toFixed(2);
  const originUtilPct = ((avgRecvMbps / 3) * 100).toFixed(1); // vs 3 Mbps envelope
  const durVals = data.metrics.http_req_duration && data.metrics.http_req_duration.values;
  const fmt = (v) => (v ? Number(v).toFixed(0) : "0");
  return {
    recvMB: totalRecvMB.toFixed(2),
    sentMB: totalSentMB.toFixed(2),
    avgRecvMbps,
    originUtilPct,
    // k6's default percentiles are med (≈p50), p(90), p(95); p(99) requires a
    // custom summaryTrendStats. Use med for p50, and p(99) only if present.
    p50: fmt(durVals && (durVals["p(50)"] != null ? durVals["p(50)"] : durVals.med)),
    p95: fmt(durVals && durVals["p(95)"]),
    p99: fmt(durVals && durVals["p(99)"]),
    durSec: durSec.toFixed(0),
  };
}
