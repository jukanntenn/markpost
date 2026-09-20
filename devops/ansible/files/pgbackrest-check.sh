#!/bin/sh
# Daily backup observability (MRFC 2026-07-09-wal-archival-disaster-recovery).
# Three probes, one verdict pushed to the uptime-kuma monitor so failure shows
# up as an explicit down push (or, if even this script dies, as silence):
#   1. pgbackrest check — verifies the WAL archive round-trip (push + get).
#   2. freshness — the newest backup older than 26h means silent backup death.
#   3. pg_wal growth — un-archived segments piling up is the disk-full failure
#      mode that kills the write path; >128 segments (2 GB) is the tripwire.
# KUMA_BACKUP_URL is optional: sourced from backup.env (0600, vaulted) when the
# monitor exists; without it the script still fails loudly via its exit code
# and the log file.
set -u
project_dir="${1:?usage: pgbackrest-check.sh <compose-project-dir>}"
cd "$project_dir" || exit 1
[ -f backup.env ] && . ./backup.env

rc=0
msg=""

if ! docker compose exec -T postgres pgbackrest --stanza=markpost check; then
    rc=1
    msg="archive round-trip check failed"
fi

age_s=$(docker compose exec -T postgres pgbackrest --stanza=markpost info --output=json | python3 -c '
import json, sys, datetime
repos = json.load(sys.stdin)
stops = [b["stop"] for r in repos for b in r.get("backup", [])]
if not stops:
    print(-1)
else:
    newest = max(datetime.datetime.fromisoformat(s) for s in stops)
    print(int((datetime.datetime.now(datetime.timezone.utc) - newest.astimezone(datetime.timezone.utc)).total_seconds()))
')
if [ "${age_s:--1}" -lt 0 ] || [ "${age_s:-0}" -gt $((26 * 3600)) ]; then
    rc=1
    msg="${msg:+$msg; }no backup newer than 26h (age=${age_s}s)"
fi

wal=$(docker compose exec -T postgres psql -U markpost -d markpost -tAc "SELECT count(*) FROM pg_ls_waldir()" 2>/dev/null | tr -d "[:space:]")
if [ "${wal:-0}" -gt 128 ]; then
    rc=1
    msg="${msg:+$msg; }pg_wal holds $wal segments — archive stall filling the disk"
fi

if [ -n "${KUMA_BACKUP_URL:-}" ]; then
    status=up
    [ "$rc" -eq 0 ] || status=down
    curl -fsS -o /dev/null --max-time 10 \
        "${KUMA_BACKUP_URL}&status=${status}&msg=$(printf "%s" "${msg:-ok}" | tr " " "+")" || true
fi

[ "$rc" -eq 0 ] || echo "ERROR: ${msg}" >&2
exit "$rc"
