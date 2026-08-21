// Staircase capacity scan for a single origin mechanism, one k6
// ramping-arrival-rate run over rising rates with per-stage metrics as
// EXPLICIT custom metrics (this k6 version only materializes tag sub-metrics
// when a threshold references them, so per-stage Trends/Counters are created
// up front instead). analyze.py joins the per-stage latency table with the
// monitor CSV and per-stage data_received (wire bytes, bucketed by time) to
// produce the rate-vs-latency knee table.
//
// Why one ramping run instead of one k6 invocation per rate: iterationInTest
// is continuous, so the cold mechanism walks the QID pool monotonically —
// every request a genuine cache miss — with no cross-step cache pollution and
// no per-step startup cost. Each rate becomes two stages: a RAMP-second ramp
// to the rate, then a STEP-second hold at r; ramp samples land in the stage
// they ramp INTO (<5% of samples — documented noise).
//
// Mechanisms (MECH env):
//   cold  — unique QID per iteration, plain GET: full DB read + render per
//           request (the first-touch-of-an-edge shape). Requires the seeded
//           QID pool ≥ Σ(rate)×STEP iterations; preflight asserts this.
//   re304 — setup() prewarms PREWARM QIDs and stores their ETags; iterations
//           send ONLY the conditional GET (If-None-Match), which is what the
//           CDN actually sends on s-maxage expiry. Every response should be a
//           bodyless 304.
//   warm  — WARM_COUNT hot QIDs round-robin, plain GETs: render-cache hits
//           with full bodies (the no-CDN self-hosted shape, and the CPU
//           ceiling probe when run unshaped).
//
// Usage:
//   MECH=cold STAGES=5,10,15,20,25,30 STEP=120s k6 run scripts/loadtest/k6/capacity.js
//   MECH=re304 STAGES=100,200,...,1000 STEP=60s k6 run ...
//
// Requires scripts/loadtest/out/qids.json (seed.sh). No thresholds on
// purpose: breaches at the top stages ARE the data — the runner tolerates
// degraded responses and reads the per-stage table instead.
import http from "k6/http";
import exec from "k6/execution";
import { Counter, Trend } from "k6/metrics";
import { SharedArray } from "k6/data";
import { originGet, baseURL, tlsOptions, summaryTrendStats, acceptEncodingHeaders } from "./lib.js";

const BASE_URL = baseURL();
const MECH = __ENV.MECH || "cold";
const STAGES = (__ENV.STAGES || "5,10,15,20,25,30").split(",").map((s) => parseInt(s, 10));
const STEP = parseDur(__ENV.STEP || "120s");
const RAMP = parseDur(__ENV.RAMP || "5s");
const PREWARM = parseInt(__ENV.PREWARM || "100", 10);
const WARM_COUNT = parseInt(__ENV.WARM_COUNT || "10", 10);

function parseDur(s) {
  const m = /^(\d+)([sm])$/.exec(String(s));
  if (!m) throw new Error(`bad duration: ${s}`);
  return parseInt(m[1], 10) * (m[2] === "m" ? 60 : 1);
}

const qids = new SharedArray("qids", function () {
  return JSON.parse(open(__ENV.QIDS_FILE || "../out/qids.json"));
});

// Per-stage metrics: s<rate>_duration / _waiting (Trend, ms), s<rate>_reqs /
// _err (Counter). Aligned with analyze.py's naming.
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

const k6Stages = [];
for (const rate of STAGES) {
  k6Stages.push({ duration: `${RAMP}s`, target: rate });
  k6Stages.push({ duration: `${STEP}s`, target: rate });
}
const maxRate = Math.max(...STAGES);

export const options = {
  scenarios: {
    staircase: {
      executor: "ramping-arrival-rate",
      startRate: 0,
      timeUnit: "1s",
      preAllocatedVUs: Math.max(20, Math.ceil(maxRate * 0.25)),
      maxVUs: Math.min(3000, Math.max(50, Math.ceil(maxRate * 1.5))),
      stages: k6Stages,
    },
  },
  ...tlsOptions,
  summaryTrendStats,
  noConnectionReuse: false,
};

// Prewarm etags for the re304 mechanism. Runs once before the VUs start, over
// the shaped link (PREWARM × ~12 KB ≈ seconds at 3 Mbps).
export function setup() {
  const t0 = Date.now();
  if (MECH !== "re304") {
    return { t0, etags: [] };
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
  return { t0, etags };
}

// stageFor maps wallclock elapsed to the stage being held (ramp segments
// count toward the stage they ramp into).
function stageFor(elapsedSec) {
  let start = 0;
  for (let i = 0; i < STAGES.length; i++) {
    const stageEnd = start + RAMP + STEP;
    if (elapsedSec < stageEnd) {
      return STAGES[i];
    }
    start = stageEnd;
  }
  return STAGES[STAGES.length - 1];
}

export default function (data) {
  const it = exec.scenario.iterationInTest || 0;
  const elapsed = (Date.now() - data.t0) / 1000;
  const rate = stageFor(elapsed);
  const m = stageMetrics[rate];

  let res;
  if (MECH === "cold") {
    res = originGet(BASE_URL, qids[it % qids.length], undefined, { stage: `r${rate}`, mech: MECH });
  } else if (MECH === "re304") {
    const e = data.etags[it % data.etags.length];
    res = originGet(BASE_URL, e.qid, e.etag, { stage: `r${rate}`, mech: MECH });
  } else if (MECH === "warm") {
    res = originGet(BASE_URL, qids[it % Math.min(WARM_COUNT, qids.length)], undefined, {
      stage: `r${rate}`,
      mech: MECH,
    });
  } else {
    throw new Error(`unknown MECH: ${MECH}`);
  }
  m.reqs.add(1);
  m.dur.add(res.timings.duration);
  m.wait.add(res.timings.waiting);
  if (res.status !== 200 && res.status !== 304) {
    m.err.add(1);
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
        `p99=${fmt(dur && dur.values["p(99)"])}ms wait_p95=${fmt(wait && wait.values["p(95)"])}ms ` +
        `err=${e}`
    );
  }
  const recv = data.metrics.data_received && data.metrics.data_received.values;
  const durVals = data.metrics.http_req_duration && data.metrics.http_req_duration.values;
  const dropped = data.metrics.dropped_iterations && data.metrics.dropped_iterations.values.count;
  const body = `
==================== CAPACITY [${MECH}] stages=${STAGES.join(",")} step=${STEP}s ramp=${RAMP}s ====================
${lines.join("\n")}
overall  p95=${fmt(durVals && durVals["p(95)"])}ms  wire_recv=${((recv ? recv.count : 0) / 1048576).toFixed(1)}MB  dropped_iterations=${dropped || 0}
(wire egress per stage is computed by analyze.py from data_received time buckets)
========================================================================================================================
`;
  return { stdout: body };
}
