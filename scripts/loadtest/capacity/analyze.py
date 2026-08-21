#!/usr/bin/env python3
"""Capacity-run analyzer: joins k6 per-stage custom metrics with monitor
(cgroup/PSI) samples and wire-egress time buckets into the per-stage knee
table, buckets raw samples over time for the restart-storm timeline, and
extracts soak metrics from the backend's OTel JSONL files.

Note on formats: k6's --summary-export file FLATTENS metric values (no
.values wrapper) — e.g. metrics["s5_duration"]["p(95)"] — while the
in-script handleSummary data nests them under .values. This tool reads the
export file and handles the flattened shape (with a .values fallback for
older formats).

Subcommands:
  run NAME           per-stage table for a capacity.sh-produced run
                     (reads out/results/capacity/{NAME}-summary.json,
                     {NAME}.json raw samples, {NAME}-monitor.csv, manifest.csv)
  buckets RAW.json   5s buckets of rps/p95/fail% (restart-storm timeline)
  soak LOGDIR        heap/goroutine/delivery-pending series from metrics JSONL

All output is markdown-ready so it can be pasted into the report.
"""

import csv
import json
import os
import re
import sys

CAP_DIR = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "..", "out", "results", "capacity"
)
TWO_CORES = 2.0


def _load_json(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def _manifest_for(name):
    """Return the manifest row dict for a run name."""
    manifest = os.path.join(CAP_DIR, "manifest.csv")
    if not os.path.exists(manifest):
        return None
    with open(manifest, encoding="utf-8") as f:
        for row in csv.reader(f):
            if len(row) >= 7 and row[2] == name:
                envs = {}
                for kv in row[4].split():
                    if "=" in kv:
                        k, v = kv.split("=", 1)
                        envs[k] = v
                return {
                    "start": int(row[0]),
                    "end": int(row[1]),
                    "envs": envs,
                    "summary": row[5],
                    "monitor": row[6],
                }
    return None


def _iter_raw_points(path):
    """Yield (metric_name, time_s, value, tags) for Point lines in a k6 JSON export."""
    from datetime import datetime, timezone

    with open(path, encoding="utf-8") as f:
        for line in f:
            if '"Point"' not in line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                continue
            data = obj.get("data", {})
            t = data.get("time")
            if not isinstance(t, str):
                continue
            # RFC3339 with up-to-ns fraction: truncate to microseconds for
            # fromisoformat, keep the timezone offset.
            try:
                iso = re.sub(r"(\.\d{6})\d+", r"\1", t)
                ts = datetime.fromisoformat(iso).timestamp()
            except ValueError:
                continue
            yield obj.get("metric"), ts, data.get("value"), data.get("tags", {})


def _raw_anchor_and_egress(path):
    """First main-scenario request time (epoch s) and per-window wire bytes.

    data_received points carry only {group, scenario} tags in this k6 version,
    so per-stage egress is bucketed purely by timestamp.
    """
    first_req = None
    egress_points = []  # (time_s, bytes)
    for metric, t, value, tags in _iter_raw_points(path):
        if metric == "http_req_duration" and first_req is None:
            first_req = t
        elif metric == "data_received" and tags.get("group") != "::setup":
            egress_points.append((t, value))
    return first_req, egress_points


def _parse_psi(field):
    m = re.search(r"avg10=([0-9.]+)", field or "")
    return float(m.group(1)) if m else 0.0


def _load_monitor(path):
    rows = []
    with open(path, encoding="utf-8") as f:
        reader = csv.reader(f)
        next(reader, None)
        for r in reader:
            if len(r) != 8:
                continue
            try:
                rows.append(
                    {
                        "ts": float(r[0]),
                        "container": r[1],
                        "usage_usec": int(r[2]) if r[2] != "ERR" else None,
                        "psi_cpu": _parse_psi(r[5]) if r[5] != "ERR" else None,
                        "mem": int(r[6]) if r[6] != "ERR" else None,
                        "psi_mem": _parse_psi(r[7]) if r[7] != "ERR" else None,
                    }
                )
            except ValueError:
                continue
    return rows


def _monitor_window(rows, container, lo, hi):
    sel = [
        r
        for r in rows
        if r["container"].endswith(f"-{container}-1")
        and r["usage_usec"] is not None
        and lo <= r["ts"] <= hi
    ]
    if len(sel) < 2:
        return None
    wall = sel[-1]["ts"] - sel[0]["ts"]
    if wall <= 0:
        return None
    usage = (sel[-1]["usage_usec"] - sel[0]["usage_usec"]) / 1e6
    return {
        "cpu_pct": usage / wall / TWO_CORES * 100.0,
        "mem_mb_max": max(r["mem"] for r in sel) / 1048576,
        "psi_cpu_max": max(r["psi_cpu"] for r in sel),
        "psi_mem_max": max(r["psi_mem"] for r in sel),
    }


def _mval(summary, key, field):
    """Read a metric value from a flattened (or nested) summary export."""
    m = summary.get("metrics", {}).get(key)
    if not m:
        return None
    v = m.get(field)
    if v is None and "values" in m:
        v = m["values"].get(field)
    return v


def cmd_run(name):
    man = _manifest_for(name)
    if not man:
        sys.exit(f"no manifest row for {name}")
    summary = _load_json(man["summary"])
    stages = [int(s) for s in re.split(r"[;,]", man["envs"].get("STAGES", "")) if s]
    step = int(re.sub(r"[^0-9]", "", man["envs"].get("STEP", "120s")) or 120)
    ramp = int(re.sub(r"[^0-9]", "", man["envs"].get("RAMP", "5s")) or 5)

    # Anchor stage windows on the first actual request (setup/prewarm shifts
    # k6's t0 by seconds vs the manifest wallclock).
    raw = os.path.join(CAP_DIR, f"{name}.json")
    t0 = None
    egress_points = []
    if os.path.exists(raw):
        t0, egress_points = _raw_anchor_and_egress(raw)
    if t0 is None:
        t0 = man["start"]

    mon_rows = _load_monitor(man["monitor"]) if os.path.exists(man["monitor"]) else []

    def f(v):
        return "-" if v is None else f"{v:.1f}"

    print(f"### run `{name}` — stages={stages} step={step}s ramp={ramp}s\n")
    print(
        "| stage | target/s | actual/s | dur p50 | dur p95 | dur p99 | wait p95 | err% "
        "| wire Mbps (%3M) | app CPU% | app mem | PSI cpu | PSI mem | pg CPU% |"
    )
    print("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|")
    cursor = t0
    for rate in stages:
        hold_lo = cursor + ramp
        hold_hi = cursor + ramp + step
        n = _mval(summary, f"s{rate}_reqs", "count") or 0
        e = _mval(summary, f"s{rate}_err", "count") or 0
        wire = sum(v for t, v in egress_points if hold_lo <= t <= hold_hi)
        mbps = wire * 8 / step / 1e6
        app_m = _monitor_window(mon_rows, "app", hold_lo, hold_hi) or {}
        pg_m = _monitor_window(mon_rows, "postgres", hold_lo, hold_hi) or {}
        print(
            f"| r{rate} | {rate} | {n / step:.1f} "
            f"| {f(_mval(summary, f's{rate}_duration', 'med'))} "
            f"| {f(_mval(summary, f's{rate}_duration', 'p(95)'))} "
            f"| {f(_mval(summary, f's{rate}_duration', 'p(99)'))} "
            f"| {f(_mval(summary, f's{rate}_waiting', 'p(95)'))} "
            f"| {100 * e / n if n else 100:.2f} "
            f"| {mbps:.2f} ({mbps / 3 * 100:.0f}%) "
            f"| {app_m.get('cpu_pct', 0):.0f} | {app_m.get('mem_mb_max', 0):.0f}MB "
            f"| {app_m.get('psi_cpu_max', 0):.2f} | {app_m.get('psi_mem_max', 0):.2f} "
            f"| {pg_m.get('cpu_pct', 0):.0f} |"
        )
        cursor = hold_hi


def cmd_buckets(raw_path, bucket_sec=5.0):
    from collections import defaultdict

    buckets = defaultdict(lambda: {"dur": [], "fails": 0, "reqs": 0, "r304": 0, "c200": 0})
    for metric, t, value, tags in _iter_raw_points(raw_path):
        if t is None:
            continue
        b = int(t // bucket_sec)
        if metric == "http_req_duration":
            buckets[b]["dur"].append(value)
            buckets[b]["reqs"] += 1
        elif metric == "http_req_failed":
            buckets[b]["fails"] += 1
        elif metric == "origin_revalidate_304":
            buckets[b]["r304"] += 1
        elif metric == "origin_cold_miss_200":
            buckets[b]["c200"] += 1
    if not buckets:
        sys.exit("no points parsed")
    base = min(buckets)
    print("| t (s) | reqs | p95 ms | fails | 304 | 200 |")
    print("|---|---|---|---|---|---|")
    for b in range(base, max(buckets) + 1):
        d = buckets.get(b)
        if not d:
            continue
        durs = sorted(d["dur"])
        p95 = durs[max(0, int(len(durs) * 0.95) - 1)] if durs else 0
        print(
            f"| {int((b - base) * bucket_sec)} | {d['reqs']} | {p95:.0f} "
            f"| {d['fails']} | {d['r304']} | {d['c200']} |"
        )


def cmd_soak(logdir):
    """Extract runtime series from OTel stdoutmetric JSONL exports.

    Format: one ResourceMetrics JSON object per line, containing ScopeMetrics
    with many metrics each. Gauges carry single data points (asInt/asDouble +
    RFC3339 Time). markpost.delivery.* may be Sum (monotonic) — deltas are
    reported for those.
    """
    from collections import defaultdict
    from datetime import datetime

    wanted = re.compile(
        r"^(process\.runtime\.go\.(?:mem\.)?[a-z_]+|markpost\.delivery\.[a-z_]+)$"
    )
    series = defaultdict(list)
    for fname in sorted(os.listdir(logdir)):
        if not fname.startswith("metrics"):
            continue
        with open(os.path.join(logdir, fname), encoding="utf-8", errors="replace") as f:
            for line in f:
                if "process.runtime" not in line and "markpost.delivery" not in line:
                    continue
                try:
                    obj = json.loads(line)
                except json.JSONDecodeError:
                    continue
                for scope in obj.get("ScopeMetrics", []):
                    for m in scope.get("Metrics", []):
                        name = m.get("Name", "")
                        if not wanted.match(name):
                            continue
                        # stdoutmetric shape: {"Data": {"DataPoints": [{"Time": ..., "Value": n}]}}
                        pts = m.get("Data", {}).get("DataPoints", [])
                        for dp in pts:
                            v = dp.get("Value", dp.get("asInt", dp.get("asDouble")))
                            t = dp.get("Time")
                            if v is None or not t:
                                continue
                            try:
                                ts = datetime.fromisoformat(
                                    re.sub(r"(\.\d{6})\d+", r"\1", t.replace("Z", "+00:00"))
                                ).timestamp()
                            except ValueError:
                                continue
                            series[name].append((ts, float(v)))
                            break
    for name, pts in sorted(series.items()):
        if not pts:
            continue
        pts.sort()
        tail = [v for _, v in pts[-5:]]
        head = [v for _, v in pts[:5]]
        first_t, last_t = pts[0][0], pts[-1][0]
        extra = ""
        if "markpost.delivery" in name:
            extra = f" delta={pts[-1][1] - pts[0][1]:.3g}"
        print(
            f"- `{name}`: n={len(pts)} span={last_t - first_t:.0f}s "
            f"first={head[0]:.3g} last={tail[-1]:.3g} "
            f"min={min(v for _, v in pts):.3g} max={max(v for _, v in pts):.3g}{extra} "
            f"(head_avg={sum(head) / len(head):.3g}, tail_avg={sum(tail) / len(tail):.3g})"
        )


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)
    cmd = sys.argv[1]
    if cmd == "run":
        cmd_run(sys.argv[2])
    elif cmd == "buckets":
        cmd_buckets(sys.argv[2], float(sys.argv[3]) if len(sys.argv) > 3 else 5.0)
    elif cmd == "soak":
        cmd_soak(sys.argv[2])
    else:
        print(__doc__)
        sys.exit(1)


if __name__ == "__main__":
    main()
