#!/usr/bin/env python3
"""Monthly restore drill (MRFC 2026-07-09-wal-archival-disaster-recovery).

Restore the newest backup set (implicit latest = base backup + full WAL
replay, the same path a disaster recovery would take) into a throwaway data
dir inside a one-off container, start postgres there, and assert the restored
posts row count sits within the RPO bound of the live count. A backup never
exercised is a hope, not a backup.
"""

import argparse
import subprocess
import sys
import time
from pathlib import Path

from pgbackrest_check import load_env

IMAGE = "markpost-postgres-archival:17"
DRILL_CONTAINER = "markpost-drill"
# RPO bound: restore may lag live by at most the WAL tail (~5 min at
# ~0.12 writes/second ≈ 36 rows); it must never exceed live.
RPO_ROW_SLACK = 36


def psql_count(project_dir: str, *docker_args: str) -> int:
    proc = subprocess.run(
        list(docker_args)
        + ["psql", "-U", "markpost", "-d", "markpost", "-tAc",
           "SELECT count(*) FROM posts"],
        cwd=project_dir, capture_output=True, text=True,
    )
    if proc.returncode != 0:
        print(f"ERROR: psql failed: {proc.stderr.strip()}", file=sys.stderr)
        sys.exit(1)
    return int(proc.stdout.strip())


def cleanup(project_dir: str) -> None:
    subprocess.run(
        ["docker", "rm", "-f", DRILL_CONTAINER],
        capture_output=True,
    )
    subprocess.run(["rm", "-rf", str(Path(project_dir) / "drill")], check=False)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("project_dir", help="compose project directory")
    args = parser.parse_args()
    project = Path(args.project_dir)
    drill = project / "drill"

    env = load_env(args.project_dir)
    if not env.get("B2_ACCESS_KEY_ID") or not env.get("B2_SECRET_ACCESS_KEY"):
        print("ERROR: B2 credentials missing from backup.env", file=sys.stderr)
        return 1

    live_count = psql_count(
        args.project_dir, "docker", "compose", "exec", "-T", "postgres"
    )

    try:
        cleanup(args.project_dir)
        (drill / "data").mkdir(mode=0o700, parents=True)
        (drill / "spool").mkdir(mode=0o700, parents=True)

        # Env overrides point pgbackrest at the drill data dir and a private
        # spool (sharing the production spool would collide on its status
        # locks); the repo and stanza config are the production ones.
        restore = subprocess.run(
            ["docker", "run", "--rm",
             "-v", f"{project / 'pgbackrest.conf'}:/etc/pgbackrest/pgbackrest.conf:ro",
             "-v", f"{drill / 'data'}:/var/lib/postgresql/data",
             "-v", f"{drill / 'spool'}:/var/spool/pgbackrest",
             "-e", "PGBACKREST_PG1_PATH=/var/lib/postgresql/data",
             "-e", "PGBACKREST_SPOOL_PATH=/var/spool/pgbackrest",
             "-e", f"PGBACKREST_REPO1_S3_KEY={env['B2_ACCESS_KEY_ID']}",
             "-e", f"PGBACKREST_REPO1_S3_KEY_SECRET={env['B2_SECRET_ACCESS_KEY']}",
             "--entrypoint", "pgbackrest", IMAGE,
             "--stanza=markpost", "restore"],
            capture_output=True, text=True,
        )
        if restore.returncode != 0:
            print(f"ERROR: restore failed: {restore.stderr.strip()}", file=sys.stderr)
            return 1

        # The restored cluster starts with the image default postgresql.conf —
        # the archival GUCs were command-line flags on the production
        # container, so they do not persist into a restored PGDATA and the
        # drill cannot contaminate the archive. POSTGRES_PASSWORD is only read
        # at initdb; the restored users exist.
        subprocess.run(
            ["docker", "run", "-d", "--name", DRILL_CONTAINER,
             "-v", f"{drill / 'data'}:/var/lib/postgresql/data",
             "-e", "POSTGRES_PASSWORD=drill",
             IMAGE],
            capture_output=True, check=True,
        )

        for _ in range(60):
            ready = subprocess.run(
                ["docker", "exec", DRILL_CONTAINER, "pg_isready",
                 "-U", "markpost", "-d", "markpost"],
                capture_output=True,
            )
            if ready.returncode == 0:
                break
            time.sleep(2)
        else:
            print("ERROR: drill postgres never became ready", file=sys.stderr)
            return 1

        drilled_count = psql_count(
            args.project_dir, "docker", "exec", DRILL_CONTAINER
        )
        floor = live_count - RPO_ROW_SLACK
        if not floor <= drilled_count <= live_count:
            print(
                f"ERROR: drill restored {drilled_count} posts, live has "
                f"{live_count} (expected {floor}..{live_count})",
                file=sys.stderr,
            )
            return 1
        print(f"drill ok: restored {drilled_count} posts against live {live_count}")
        return 0
    finally:
        cleanup(args.project_dir)


if __name__ == "__main__":
    sys.exit(main())
