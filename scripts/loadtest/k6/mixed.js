// Mixed business-profile staircase: models the traffic composition that
// actually REACHES the origin behind the CDN (the edge absorbs the rest), per
// the capacity plan's "回源构成" model. Weights are env-tunable so the
// business assumptions can be re-calibrated without touching the script.
//
// Default composition (percent of iterations):
//   W_READ=80   /p-* reads, split READ_INM=85% CDN revalidations (conditional
//               GET only — what Cloudflare sends on s-maxage expiry) and the
//               remainder cold misses (first touch of an edge / fresh post)
//   W_DASH=10   dashboard API polling (GET /api/v1/posts with a JWT) —
//               uncacheable JSON, 100% reaches the origin; models ~100
//               always-open author tabs (2 queries × ~0.7 req/s each)
//   W_STATIC=3  static HTML pages (/, /login) served by Caddy — these carry
//               no Cache-Control today, so every navigation re-originate
//   W_WRITE=5   POST /mpk-* creates (raised above the 0.12/s business mean
//               to stress the write + delivery-fanout path)
//   W_LOGIN=2   logins (bcrypt, the second-most-expensive CPU path)
//
// Per-stage aggregates use explicit custom metrics (this k6 version only
// materializes tag sub-metrics for threshold-referenced metrics); the per-kind
// split rides dedicated counters and a kind-tagged latency trend. Wire egress
// per stage is computed by analyze.py from data_received time buckets.
//
// Usage:
//   STAGES=10,15,20,25 STEP=120s k6 run scripts/loadtest/k6/mixed.js
//   STAGES=18 STEP=1800s k6 run ...          # sweet-spot hold at 18 it/s
//
// Requires ../out/qids.json and ../out/write_keys.txt (seed.sh + seed_write.sh),
// and a seeded login-capable user (MIX_USER/MIX_PASS, default loadtest_1).
import http from "k6/http";
import exec from "k6/execution";
import { Counter, Trend } from "k6/metrics";
import { SharedArray } from "k6/data";
import { originGet, tileBody, normalBodySize, baseURL, tlsOptions, summaryTrendStats, acceptEncodingHeaders } from "./lib.js";

const BASE_URL = baseURL();
const STAGES = (__ENV.STAGES || "10,15,20,25,30").split(",").map((s) => parseInt(s, 10));
const STEP = parseDur(__ENV.STEP || "120s");
const RAMP = parseDur(__ENV.RAMP || "5s");
const PREWARM = parseInt(__ENV.PREWARM || "100", 10);
const W_READ = parseFloat(__ENV.W_READ || "80");
const W_DASH = parseFloat(__ENV.W_DASH || "10");
const W_STATIC = parseFloat(__ENV.W_STATIC || "3");
const W_WRITE = parseFloat(__ENV.W_WRITE || "5");
const W_LOGIN = parseFloat(__ENV.W_LOGIN || "2");
const READ_INM = parseFloat(__ENV.READ_INM || "85");
const MIX_USER = __ENV.MIX_USER || "loadtest_1";
const MIX_PASS = __ENV.MIX_PASS || "loadtestpass";

function parseDur(s) {
  const m = /^(\d+)([sm])$/.exec(String(s));
  if (!m) throw new Error(`bad duration: ${s}`);
  return parseInt(m[1], 10) * (m[2] === "m" ? 60 : 1);
}

const qids = new SharedArray("qids", function () {
  return JSON.parse(open(__ENV.QIDS_FILE || "../out/qids.json"));
});
const keys = new SharedArray("keys", function () {
  return open(__ENV.KEYS_FILE || "../out/write_keys.txt")
    .trim()
    .split("\n")
    .filter((k) => k.length > 0);
});

const stageMetrics = {};
for (const rate of STAGES) {
  const s = `s${rate}`;
  stageMetrics[rate] = {
    dur: new Trend(`${s}_duration`),
    wait: new Trend(`${s}_waiting`),
    reqs: new Counter(`${s}_reqs`),
    err: new Counter(`${s}_err`),
  };
}

const kinds = ["read304", "readcold", "dash", "static", "write", "login"];
const kindCount = {};
for (const k of kinds) kindCount[k] = new Counter(`mixed_${k}`);
const kindDur = new Trend("mixed_kind_duration");

const k6Stages = [];
for (const rate of STAGES) {
  k6Stages.push({ duration: `${RAMP}s`, target: rate });
  k6Stages.push({ duration: `${STEP}s`, target: rate });
}
const maxRate = Math.max(...STAGES);

export const options = {
  scenarios: {
    mixed: {
      executor: "ramping-arrival-rate",
      startRate: 0,
      timeUnit: "1s",
      preAllocatedVUs: Math.max(20, Math.ceil(maxRate * 0.4)),
      maxVUs: Math.min(3000, Math.max(50, Math.ceil(maxRate * 2))),
      stages: k6Stages,
    },
  },
  // A scan, not a gate: degradation at the top stages is the data.
  ...tlsOptions,
  summaryTrendStats,
  noConnectionReuse: false,
};

export function setup() {
  const t0 = Date.now();
  // Login once for the dashboard-poll kind (authors log in rarely; the token
  // lives 24h). bcrypt cost is exercised by the dedicated login kind instead.
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ username: MIX_USER, password: MIX_PASS }),
    { headers: { "Content-Type": "application/json", ...acceptEncodingHeaders } }
  );
  const token = loginRes.status === 200 ? loginRes.json("token") : "";
  if (!token) {
    throw new Error(`setup login failed for ${MIX_USER} (HTTP ${loginRes.status})`);
  }
  const etags = [];
  for (let i = 0; i < Math.min(PREWARM, qids.length); i++) {
    const res = http.get(`${BASE_URL}/${qids[i]}`, {
      responseType: "text",
      headers: { ...acceptEncodingHeaders },
    });
    const etag = res.headers && res.headers.Etag ? res.headers.Etag : "";
    if (etag) etags.push({ qid: qids[i], etag });
  }
  return { t0, token, etags };
}

function stageRateFor(elapsedSec) {
  let start = 0;
  for (let i = 0; i < STAGES.length; i++) {
    const stageEnd = start + RAMP + STEP;
    if (elapsedSec < stageEnd) return STAGES[i];
    start = stageEnd;
  }
  return STAGES[STAGES.length - 1];
}

const jsonHeaders = { "Content-Type": "application/json", ...acceptEncodingHeaders };

function note(m, res) {
  m.reqs.add(1);
  m.dur.add(res.timings.duration);
  m.wait.add(res.timings.waiting);
  if (res.status >= 400) m.err.add(1);
}

export default function (data) {
  const it = exec.scenario.iterationInTest || 0;
  const elapsed = (Date.now() - data.t0) / 1000;
  const m = stageMetrics[stageRateFor(elapsed)];
  const tag = { stage: `r${stageRateFor(elapsed)}`, mech: "mixed" };

  const roll = Math.random() * 100;
  let acc = 0;
  if (roll < (acc += W_READ)) {
    if (Math.random() * 100 < READ_INM) {
      const e = data.etags[it % data.etags.length];
      const res = originGet(BASE_URL, e.qid, e.etag, tag);
      note(m, res);
      kindCount.read304.add(1);
      kindDur.add(res.timings.duration, { kind: "read304" });
    } else {
      // Fresh QID per iteration keeps every cold read a genuine miss.
      const idx = (PREWARM + it) % qids.length;
      const res = originGet(BASE_URL, qids[idx], undefined, tag);
      note(m, res);
      kindCount.readcold.add(1);
      kindDur.add(res.timings.duration, { kind: "readcold" });
    }
    return;
  }
  if (roll < (acc += W_DASH)) {
    const res = http.get(`${BASE_URL}/api/v1/posts?page=1&limit=20`, {
      headers: { Authorization: `Bearer ${data.token}`, ...acceptEncodingHeaders },
      tags: { name: "mixed", ...tag },
    });
    note(m, res);
    kindCount.dash.add(1);
    kindDur.add(res.timings.duration, { kind: "dash" });
    return;
  }
  if (roll < (acc += W_STATIC)) {
    const path = it % 2 === 0 ? "/" : "/login";
    const res = http.get(`${BASE_URL}${path}`, {
      headers: { ...acceptEncodingHeaders },
      tags: { name: "mixed", ...tag },
    });
    note(m, res);
    kindCount.static.add(1);
    kindDur.add(res.timings.duration, { kind: "static" });
    return;
  }
  if (roll < (acc += W_WRITE)) {
    // "Load" must appear in the title: seeded delivery channels filter on the
    // keyword, so without it the delivery fan-out path is silently skipped.
    const body = JSON.stringify({ title: `Load mixed post ${it}`, body: tileBody(normalBodySize()) });
    const res = http.post(`${BASE_URL}/${keys[it % keys.length]}`, body, {
      headers: jsonHeaders,
      tags: { name: "mixed", ...tag },
    });
    note(m, res);
    kindCount.write.add(1);
    kindDur.add(res.timings.duration, { kind: "write" });
    return;
  }
  {
    const res = http.post(
      `${BASE_URL}/api/v1/auth/login`,
      JSON.stringify({ username: MIX_USER, password: MIX_PASS }),
      { headers: jsonHeaders, tags: { name: "mixed", ...tag } }
    );
    note(m, res);
    kindCount.login.add(1);
    kindDur.add(res.timings.duration, { kind: "login" });
  }
}

export function handleSummary(data) {
  const fmt = (v) => (v != null ? Number(v).toFixed(1) : "-");
  const lines = [];
  for (const rate of STAGES) {
    const dur = data.metrics[`s${rate}_duration`];
    const wait = data.metrics[`s${rate}_waiting`];
    const reqs = data.metrics[`s${rate}_reqs`];
    const err = data.metrics[`s${rate}_err`];
    const n = reqs ? reqs.values.count : 0;
    const e = err ? err.values.count : 0;
    lines.push(
      `r${String(rate).padStart(5)} actual=${(n / STEP).toFixed(1).padStart(6)}/s ` +
        `p50=${fmt(dur && dur.values.med)}ms p95=${fmt(dur && dur.values["p(95)"])}ms ` +
        `wait_p95=${fmt(wait && wait.values["p(95)"])}ms err=${e}`
    );
  }
  const kindLines = kinds.map((k) => {
    const c = data.metrics[`mixed_${k}`];
    const t = data.metrics[`mixed_kind_duration{kind:${k}}`] || data.metrics.mixed_kind_duration;
    return `  ${k.padEnd(9)} n=${c ? c.values.count : 0} p95=${fmt(t && t.values["p(95)"])}ms`;
  });
  const body = `
==================== MIXED stages=${STAGES.join(",")} step=${STEP}s ====================
${lines.join("\n")}
kind split (whole run):
${kindLines.join("\n")}
=====================================================================================
`;
  return { stdout: body };
}
