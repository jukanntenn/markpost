#!/usr/bin/env python3
"""Daily format-diversity dump (MRFC 2026-07-09-wal-archival-disaster-recovery).

A logical pg_dump streamed through zstd into the B2 dumps/ prefix via rclone.
Logical dumps survive base-image or page-level corruption that would break the
PITR chain. rclone is rate-limited so the 3 Mbps uplink keeps headroom for
origin traffic. Credentials come from backup.env (0600, vaulted) — the same
no-delete application key the pgBackRest repo uses.
"""

import argparse
import datetime
import subprocess
import sys

from pgbackrest_check import load_env


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("project_dir", help="compose project directory")
    args = parser.parse_args()

    env = {**load_env(args.project_dir)}
    for required in ("B2_S3_ENDPOINT", "B2_S3_REGION", "B2_ACCESS_KEY_ID", "B2_SECRET_ACCESS_KEY"):
        if not env.get(required):
            print(f"ERROR: {required} missing from backup.env", file=sys.stderr)
            return 1
    bucket = env.get("B2_BUCKET", "markpost-backups")
    stamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M")
    remote = f":s3,provider=Other,endpoint={env['B2_S3_ENDPOINT']}:{bucket}/dumps/markpost-{stamp}.sql.zst"

    dump = subprocess.Popen(
        ["docker", "compose", "exec", "-T", "postgres",
         "pg_dump", "-U", "markpost", "markpost"],
        cwd=args.project_dir, stdout=subprocess.PIPE,
    )
    zstd = subprocess.Popen(
        ["zstd", "-19", "-T2"], stdin=dump.stdout, stdout=subprocess.PIPE,
    )
    rclone = subprocess.Popen(
        ["rclone", "rcat", remote,
         "--s3-access-key-id", env["B2_ACCESS_KEY_ID"],
         "--s3-secret-access-key", env["B2_SECRET_ACCESS_KEY"],
         "--s3-region", env["B2_S3_REGION"],
         "--s3-no-check-bucket",
         "--bwlimit", "2M"],
        stdin=zstd.stdout,
    )
    assert dump.stdout and zstd.stdout
    dump.stdout.close()
    zstd.stdout.close()
    codes = [proc.wait() for proc in (dump, zstd, rclone)]
    if any(code != 0 for code in codes):
        print(f"ERROR: dump pipeline stages exited {codes}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
