// Viral-post spike: one QID ramped from 0 to a high rate. Expected shape if
// singleflight + the render cache hold: the FIRST request renders, everything
// else is a cache hit, and p95 latency stays flat across stages until a wall
// (bandwidth on the shaped link, CPU on the unshaped one) — instead of the
// p95 explosion fifty concurrent cold renders would cause.
//
// Per-stage metrics are explicit custom metrics (see capacity.js for why).
//
// Usage:
//   QID=p-xxxx STAGES=100,300,600,1000,1500,2000 STEP=10s k6 run scripts/loadtest/k6/spike.js
//   (QID defaults to the first entry of qids.json)
import exec from "k6/execution";
import { Counter, Trend } from "k6/metrics";
import { SharedArray } from "k6/data";
import { originGet, baseURL, tlsOptions, summaryTrendStats } from "./lib.js";

const BASE_URL = baseURL();
const STAGES = (__ENV.STAGES || "100,300,600,1000,1500,2000")
  .split(",")
  .map((s) => parseInt(s, 10));
const STEP = parseInt(__ENV.STEP || "10", 10);
const RAMP = parseInt(__ENV.RAMP || "10", 10);

const qids = new SharedArray("qids", function () {
  return JSON.parse(open(__ENV.QIDS_FILE || "../out/qids.json"));
});
const QID = __ENV.QID || qids[0];

const stageMetrics = {};
for (const rate of STAGES) {
  const s = `s${rate}`;
  stageMetrics[rate] = {
    dur: new Trend(`${s}_duration`),
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
    spike: {
      executor: "ramping-arrival-rate",
      startRate: 0,
      timeUnit: "1s",
      preAllocatedVUs: Math.max(20, Math.ceil(maxRate * 0.2)),
      maxVUs: Math.min(3000, Math.max(50, Math.ceil(maxRate))),
      stages: k6Stages,
    },
  },
  ...tlsOptions,
  summaryTrendStats,
  noConnectionReuse: false,
};

export function setup() {
  return { t0: Date.now() };
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

export default function (data) {
  const elapsed = (Date.now() - data.t0) / 1000;
  const rate = stageRateFor(elapsed);
  const m = stageMetrics[rate];
  const res = originGet(BASE_URL, QID, undefined, { stage: `r${rate}`, mech: "spike" });
  m.reqs.add(1);
  m.dur.add(res.timings.duration);
  if (res.status !== 200) m.err.add(1);
}

export function handleSummary(data) {
  const fmt = (v) => (v != null ? Number(v).toFixed(1) : "-");
  const lines = [];
  for (const rate of STAGES) {
    const dur = data.metrics[`s${rate}_duration`];
    const reqs = data.metrics[`s${rate}_reqs`];
    const err = data.metrics[`s${rate}_err`];
    const n = reqs ? reqs.values.count : 0;
    lines.push(
      `r${String(rate).padStart(5)} actual=${(n / STEP).toFixed(1).padStart(6)}/s ` +
        `p95=${fmt(dur && dur.values["p(95)"])}ms err=${err ? err.values.count : 0}`
    );
  }
  const body = `
==================== SPIKE qid=${QID} stages=${STAGES.join(",")} step=${STEP}s ====================
${lines.join("\n")}
=================================================================================================
`;
  return { stdout: body };
}
