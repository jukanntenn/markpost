#!/bin/sh
# Monthly restore drill (MRFC 2026-07-09-wal-archival-disaster-recovery):
# restore the newest backup set (implicit latest = base backup + full WAL
# replay, the same path a disaster recovery would take) into a throwaway data
# dir inside a one-off container, start postgres there read-write, and assert
# the restored posts row count sits within the RPO bound of the live count.
# A backup never exercised is a hope, not a backup.
set -eu
project_dir="${1:?usage: pgbackrest-drill.sh <compose-project-dir>}"
cd "$project_dir" || exit 1
[ -f backup.env ] && . ./backup.env
: "${B2_ACCESS_KEY_ID:?}" "${B2_SECRET_ACCESS_KEY:?}"

image="markpost-postgres-archival:17"
drill_dir="${project_dir:?}/drill"
live_count=$(docker compose exec -T postgres psql -U markpost -d markpost -tAc "SELECT count(*) FROM posts" | tr -d "[:space:]")

cleanup() {
    docker rm -f markpost-drill >/dev/null 2>&1 || true
    rm -rf "$drill_dir"
}
trap cleanup EXIT INT TERM
rm -rf "$drill_dir"
mkdir -m 700 -p "$drill_dir/data" "$drill_dir/spool"

# Env overrides point pgbackrest at the drill data dir and a private spool
# (sharing the production spool would collide on its status locks); the repo
# and stanza config are the production ones, bind-mounted read-only.
docker run --rm \
    -v "$project_dir/pgbackrest.conf:/etc/pgbackrest/pgbackrest.conf:ro" \
    -v "$drill_dir/data:/var/lib/postgresql/data" \
    -v "$drill_dir/spool:/var/spool/pgbackrest" \
    -e PGBACKREST_PG1_PATH=/var/lib/postgresql/data \
    -e PGBACKREST_SPOOL_PATH=/var/spool/pgbackrest \
    -e PGBACKREST_REPO1_S3_KEY="$B2_ACCESS_KEY_ID" \
    -e PGBACKREST_REPO1_S3_KEY_SECRET="$B2_SECRET_ACCESS_KEY" \
    --entrypoint pgbackrest "$image" \
    --stanza=markpost restore

# The restored cluster starts with the image default postgresql.conf — the
# archival GUCs were command-line flags on the production container, so they
# do not persist into a restored PGDATA and the drill cannot contaminate the
# archive. POSTGRES_PASSWORD is only read at initdb; the restored users exist.
docker run -d --name markpost-drill \
    -v "$drill_dir/data:/var/lib/postgresql/data" \
    -e POSTGRES_PASSWORD=drill \
    "$image" >/dev/null

i=0
until docker exec markpost-drill pg_isready -U markpost -d markpost >/dev/null 2>&1; do
    i=$((i + 1))
    [ "$i" -gt 60 ] && { echo "ERROR: drill postgres never became ready" >&2; exit 1; }
    sleep 2
done

drilled_count=$(docker exec markpost-drill psql -U markpost -d markpost -tAc "SELECT count(*) FROM posts" | tr -d "[:space:]")

# RPO bound: restore may lag live by at most the WAL tail (~5 min at
# ~0.12 writes/second ≈ 36 rows); it must never exceed live.
floor=$((live_count - 36))
if [ "$drilled_count" -lt "$floor" ] || [ "$drilled_count" -gt "$live_count" ]; then
    echo "ERROR: drill restored $drilled_count posts, live has $live_count (expected ${floor}..${live_count})" >&2
    exit 1
fi
echo "drill ok: restored $drilled_count posts against live $live_count"
