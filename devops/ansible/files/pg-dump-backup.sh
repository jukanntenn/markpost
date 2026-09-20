#!/bin/sh
# Daily format-diversity dump (MRFC 2026-07-09-wal-archival-disaster-recovery):
# a logical pg_dump streamed through zstd into the B2 dumps/ prefix via rclone.
# Logical dumps survive base-image or page-level corruption that would break
# the PITR chain. rclone is rate-limited so the 3 Mbps uplink keeps headroom
# for origin traffic. Credentials come from backup.env (0600, vaulted) — the
# same no-delete application key the pgBackRest repo uses.
set -u
project_dir="${1:?usage: pg-dump-backup.sh <compose-project-dir>}"
cd "$project_dir" || exit 1
[ -f backup.env ] && . ./backup.env
: "${B2_S3_ENDPOINT:?}" "${B2_S3_REGION:?}" "${B2_ACCESS_KEY_ID:?}" "${B2_SECRET_ACCESS_KEY:?}"

stamp=$(date -u +%Y%m%dT%H%M)

docker compose exec -T postgres pg_dump -U markpost markpost \
    | zstd -19 -T2 \
    | rclone rcat ":s3,provider=Other,endpoint=${B2_S3_ENDPOINT}:${B2_BUCKET:-markpost-backups}/dumps/markpost-${stamp}.sql.zst" \
        --s3-access-key-id "$B2_ACCESS_KEY_ID" \
        --s3-secret-access-key "$B2_SECRET_ACCESS_KEY" \
        --s3-region "$B2_S3_REGION" \
        --s3-no-check-bucket \
        --bwlimit 2M
