#!/usr/bin/env python3
"""Post-deploy verification for markpost (run on the controller by deploy.yml).

Three phases:
  1. Poll GET {base}/api/v1/health until it answers {"status": "ok"}
     (interval/timeout configurable). Container-level healthchecks only prove
     the process is up; this exercises the published port through Caddy — and
     for staging/production, through the tunnel / Cloudflare edge the real
     visitors use.
  2. GET {base}/ must answer 200 with the landing page's build-time title.
     The API being healthy says nothing about the static export: a broken
     export or Caddy rewrite would still pass phase 1. The prerendered body is
     only a hydration skeleton, so the <title> is the one stable server-side
     marker that the landing route is what's deployed (it differs from the
     404/login titles).
  3. When --version is given, compare GET {base}/api/v1/version against the
     expected build version. This is the actual deploy verification: a
     container can be healthy while still running the previous image.

Expected version comes from the playbook: `git describe` of the deploying
checkout for `main`-tag (dev) deploys, or the pinned git tag for releases.
"""
from __future__ import annotations

import argparse
import json
import sys
import time
import urllib.error
import urllib.request


# The landing route's build-time title, duplicated from
# frontend/src/lib/metadata.ts (PAGE_TITLES.landing) — the static export bakes
# it into out/index.html, where it survives even though the body is a
# client-hydration skeleton. Update both sides together.
LANDING_TITLE = "Markpost — Self-hosted Markdown Publishing"


def fetch_json(url: str, timeout: float) -> dict | None:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as resp:
            if resp.status != 200:
                return None
            return json.loads(resp.read().decode())
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError, OSError):
        return None


def wait_ready(base_url: str, interval: float, timeout: float) -> bool:
    url = f"{base_url}/api/v1/health"
    deadline = time.monotonic() + timeout
    attempt = 0
    while time.monotonic() < deadline:
        attempt += 1
        body = fetch_json(url, timeout=interval)
        if body and body.get("status") == "ok":
            print(f"health: ok ({url}, {attempt} attempt(s))")
            return True
        print(f"health: not ready yet ({url}, attempt {attempt})", flush=True)
        time.sleep(interval)
    return False


def check_landing(base_url: str, timeout: float) -> bool:
    url = f"{base_url}/"
    try:
        with urllib.request.urlopen(url, timeout=timeout) as resp:
            status = resp.status
            body = resp.read().decode("utf-8", errors="replace")
    except (urllib.error.URLError, TimeoutError, OSError) as e:
        print(f"landing: fetch failed ({url}): {e}")
        return False
    if status != 200:
        print(f"landing: GET / returned {status} (expected 200)")
        return False
    if LANDING_TITLE not in body:
        print(f"landing: title {LANDING_TITLE!r} not found in GET / body")
        print("hint: the static export is missing or stale, or the Caddy "
              "rewrite serves the wrong file; if the landing title changed "
              "in frontend/src/lib/metadata.ts, update LANDING_TITLE here")
        return False
    print(f"landing: ok ({url})")
    return True


def check_version(base_url: str, expected: str, timeout: float) -> bool:
    url = f"{base_url}/api/v1/version"
    body = fetch_json(url, timeout=timeout)
    if body is None:
        print(f"version: endpoint unreachable ({url})")
        return False
    actual = body.get("version", "")
    if actual != expected:
        print(f"version: MISMATCH — expected {expected!r}, running {actual!r}")
        print("hint: the container is serving an old image; check the pulled tag")
        return False
    print(f"version: {actual} (matches expected)")
    return True


def main() -> int:
    parser = argparse.ArgumentParser(description="Verify a markpost deployment")
    parser.add_argument("--base-url", required=True,
                        help="Public base URL, e.g. https://markpost.cc or http://192.168.5.200:8089")
    parser.add_argument("--version", default=None,
                        help="Expected build version (VERSION build-arg); omit to skip the version check")
    parser.add_argument("--interval", type=float, default=5.0,
                        help="Seconds between health polls (default: 5)")
    parser.add_argument("--timeout", type=float, default=120.0,
                        help="Seconds to wait for health before failing (default: 120)")
    args = parser.parse_args()

    base = args.base_url.rstrip("/")
    if not wait_ready(base, args.interval, args.timeout):
        print(f"FAILED: {base}/api/v1/health not ready within {args.timeout:.0f}s")
        return 1

    if not check_landing(base, timeout=args.interval):
        print(f"FAILED: {base}/ did not serve the landing page")
        return 1

    if args.version is None:
        return 0
    if not check_version(base, args.version, timeout=args.interval):
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
