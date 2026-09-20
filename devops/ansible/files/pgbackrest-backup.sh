#!/bin/sh
# Scheduled pgBackRest backups (MRFC 2026-07-09-wal-archival-disaster-recovery):
# a full base backup on the first of each month, an incremental page scan every
# other day — both run inside the postgres container, which carries the
# pgbackrest client. Static on purpose (the heartbeat.py pattern): the compose
# project dir arrives as $1 so the ansible cron entry owns every
# deployment-specific value; cron redirects output into the log file so runs
# and failures are observable instead of vanishing into cron mail.
set -u
project_dir="${1:?usage: pgbackrest-backup.sh <compose-project-dir>}"
cd "$project_dir" || exit 1

# Day 1 takes the full; every other run is an incremental against the latest
# full — transfer scales with writes (~0.12/s here), not with stored data.
type=incr
[ "$(date -u +%d)" = "01" ] && type=full

exec docker compose exec -T postgres \
    pgbackrest --stanza=markpost backup --type="$type"
