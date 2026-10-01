#!/usr/bin/env python3
# markpost availability heartbeat: probe the loopback-published readiness
# endpoint, then push the verdict to the uptime-kuma push monitor.
#
# kuma flags the monitor down in two independent ways: this script pushes
# status=down (app-level failure — covers the Cloudflare blind spot, since the
# push path bypasses the edge) or the pushes stop entirely (host death, this
# script crashed). The push URL is secret: anyone holding it can forge "up"
# beats and mask an outage. It therefore arrives via the KUMA_HEARTBEAT_URL
# environment variable (set by the supervisor program from the ansible vault),
# never baked into this file; the non-secret knobs arrive as command-line
# arguments.
#
# Standard library only, per house rule for script-type programs. The loop
# never exits on purpose; supervisor's autorestart is the recovery path if it
# dies anyway.
import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request


def probe_ready(ready_url: str) -> bool:
    # Body must say "ready": a gateway misroute can answer 200 with an error
    # page, and a false "up" here would mask a real outage.
    try:
        with urllib.request.urlopen(ready_url, timeout=5) as resp:
            return json.load(resp).get("status") == "ready"
    except (urllib.error.URLError, OSError, ValueError):
        return False


def push(push_url: str, status: str, msg: str) -> None:
    query = urllib.parse.urlencode({"status": status, "msg": msg})
    try:
        # A failed push is not fatal: kuma marks the monitor down only after
        # its own retry window without a beat, so one transient blip on the
        # kuma side does not immediately page.
        urllib.request.urlopen(f"{push_url}?{query}", timeout=10).close()
    except (urllib.error.URLError, OSError, ValueError):
        pass


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="markpost availability heartbeat")
    parser.add_argument("--ready-url", required=True, help="readiness endpoint to probe")
    parser.add_argument("--interval", type=int, default=60, help="seconds between beats")
    args = parser.parse_args(argv)

    push_url = os.environ.get("KUMA_HEARTBEAT_URL", "")
    if not push_url:
        print("KUMA_HEARTBEAT_URL is not set", file=sys.stderr)
        return 2

    while True:
        if probe_ready(args.ready_url):
            push(push_url, "up", "ready")
        else:
            push(push_url, "down", "local readiness probe failed")
        time.sleep(args.interval)


if __name__ == "__main__":
    raise SystemExit(main())
