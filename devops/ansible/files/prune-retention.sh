#!/bin/sh
# Daily retention sweep (MRFC 2026-08-31-per-user-history-retention-policy).
# Runs both prune subcommands in a throwaway container against the live compose
# project — per-user retention_days and the global config both apply inside it.
# Static on purpose: the compose project dir arrives as $1 so the ansible cron
# entry owns every deployment-specific value (the heartbeat.py pattern). Output
# goes to stdout/stderr; the cron entry redirects into a log file so sweeps and
# failures are observable instead of vanishing into cron mail.
set -u
project_dir="${1:?usage: prune-retention.sh <compose-project-dir>}"
cd "$project_dir" || exit 1

rc=0
for cmd in prune-expired-posts prune-delivery-history; do
    echo "--- $(date -Is) ${cmd}"
    if ! docker compose run --rm --no-deps --entrypoint markpost markpost \
        -c /app/config.toml "$cmd" --batch-size 1000; then
        echo "ERROR: ${cmd} failed" >&2
        rc=1
    fi
done
exit "$rc"
