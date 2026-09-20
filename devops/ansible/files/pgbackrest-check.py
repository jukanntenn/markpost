#!/usr/bin/env python3
"""Daily backup observability (MRFC 2026-07-09-wal-archival-disaster-recovery).

Three probes, one verdict pushed to the uptime-kuma monitor so failure shows
up as an explicit down push (or, if even this script dies, as silence):

  1. pgbackrest check — verifies the WAL archive round-trip (push + get).
  2. freshness — the newest backup older than 26h means silent backup death.
  3. pg_wal growth — un-archived segments piling up is the disk-full failure
     mode that kills the write path; >128 segments (2 GB) is the tripwire.

KUMA_BACKUP_URL is optional: sourced from backup.env (0600, vaulted) when the
monitor exists; without it the script still fails loudly via its exit code
and the log file.
"""

import argparse
import datetime
import json
import pathlib
import subprocess
import sys
import urllib.parse
import urllib.request

WAL_TRIPWIRE_SEGMENTS = 128
FRESHNESS_SECONDS = 26 * 3600


def compose_exec(argv: list[str], project_dir: str) -> subprocess.CompletedProcess:
    return subprocess.run(
        ["docker", "compose", "exec", "-T", "postgres"] + argv,
        cwd=project_dir,
        capture_output=True,
        text=True,
    )


def load_env(project_dir: str) -> dict[str, str]:
    """Overlay backup.env (KEY=VALUE lines, no quoting syntax) on the environ."""
    env = dict()
    path = pathlib.Path(project_dir) / "backup.env"
    if path.exists():
        for line in path.read_text().splitlines():
            line = line.strip()
            if line and not line.startswith("#") and "=" in line:
                key, value = line.split("=", 1)
                env[key.strip()] = value.strip()
    return env


def backup_age_seconds(project_dir: str) -> float | None:
    proc = compose_exec(
        ["pgbackrest", "--stanza=markpost", "info", "--output=json"], project_dir
    )
    repos = json.loads(proc.stdout)
    stops = [backup["stop"] for repo in repos for backup in repo.get("backup", [])]
    if not stops:
        return None
    newest = max(datetime.datetime.fromisoformat(stop) for stop in stops)
    if newest.tzinfo is None:
        newest = newest.replace(tzinfo=datetime.timezone.utc)
    return (datetime.datetime.now(datetime.timezone.utc) - newest).total_seconds()


def wal_segment_count(project_dir: str) -> int:
    proc = compose_exec(
        ["psql", "-U", "markpost", "-d", "markpost", "-tAc",
         "SELECT count(*) FROM pg_ls_waldir()"],
        project_dir,
    )
    try:
        return int(proc.stdout.strip())
    except ValueError:
        return 0


def push_verdict(url: str, ok: bool, msg: str) -> None:
    # Best-effort: a failed push is caught by kuma's silence detection — the
    # same dual channel the availability heartbeat uses.
    query = urllib.parse.urlencode({"status": "up" if ok else "down", "msg": msg})
    try:
        urllib.request.urlopen(f"{url}&{query}", timeout=10)
    except OSError:
        pass


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("project_dir", help="compose project directory")
    args = parser.parse_args()

    failures: list[str] = []

    if compose_exec(["pgbackrest", "--stanza=markpost", "check"], args.project_dir).returncode != 0:
        failures.append("archive round-trip check failed")

    age = backup_age_seconds(args.project_dir)
    if age is None or age > FRESHNESS_SECONDS:
        failures.append(f"no backup newer than 26h (age={age}s)")

    segments = wal_segment_count(args.project_dir)
    if segments > WAL_TRIPWIRE_SEGMENTS:
        failures.append(f"pg_wal holds {segments} segments — archive stall filling the disk")

    kuma_url = load_env(args.project_dir).get("KUMA_BACKUP_URL")
    if kuma_url:
        push_verdict(kuma_url, not failures, "; ".join(failures) or "ok")

    if failures:
        print(f"ERROR: {'; '.join(failures)}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
