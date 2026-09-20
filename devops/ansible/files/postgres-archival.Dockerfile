# postgres:17-alpine plus the pgBackRest client, so archive_command can run
# pgbackrest inside the DB container itself (MRFC
# 2026-07-09-wal-archival-disaster-recovery). Compose builds this image locally
# from the app dir and it is never pushed or registry-pulled. A derived image is
# required because the official pgbackrest/pgbackrest image is glibc/Ubuntu and
# cannot run inside musl-based postgres:17-alpine; Alpine's community repo
# ships the same tool. Keeping the base identical to the stock postgres service
# means the only delta is the backup client.
FROM postgres:17-alpine
RUN apk add --no-cache pgbackrest
