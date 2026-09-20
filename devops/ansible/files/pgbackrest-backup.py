#!/usr/bin/env python3
"""Scheduled pgBackRest backups (MRFC 2026-07-09-wal-archival-disaster-recovery).

A full base backup on the first of each month, an incremental page scan every
other day — both run inside the postgres container, which carries the
pgbackrest client. The compose project dir arrives as the sole argument so the
ansible cron entry owns every deployment-specific value (the heartbeat.py
pattern); cron redirects output into the log file so runs and failures are
observable instead of vanishing into cron mail.
"""

import argparse
import datetime
import subprocess
import sys


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("project_dir", help="compose project directory")
    args = parser.parse_args()

    # Day 1 takes the full; every other run is an incremental against the
    # latest full — transfer scales with writes (~0.12/s here), not with
    # stored data.
    backup_type = "full" if datetime.datetime.now(datetime.timezone.utc).day == 1 else "incr"

    return subprocess.run(
        [
            "docker",
            "compose",
            "exec",
            "-T",
            "postgres",
            "pgbackrest",
            "--stanza=markpost",
            "backup",
            f"--type={backup_type}",
        ],
        cwd=args.project_dir,
    ).returncode


if __name__ == "__main__":
    sys.exit(main())
