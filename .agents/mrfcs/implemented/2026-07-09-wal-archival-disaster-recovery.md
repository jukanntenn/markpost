# MRFC: WAL-archival disaster recovery to B2

Status: implemented

English | [中文](2026-07-09-wal-archival-disaster-recovery.zh.md)

## Problem

markpost runs as a single instance: one VPS, one Postgres container, no replica. If the server dies or the host loses its data, everything is gone — the deploy pipeline schedules no backup at all. The data is ephemeral content at ~0.12 writes/second under per-user retention (global default 7 days; a VIP subset is retained indefinitely per [the retention-policy MRFC](../implemented/2026-08-31-per-user-history-retention-policy.md)), and the VPS uplink is ~3 Mbps — outbound bytes are the binding constraint the [caching spec](../../../specs/backend/caching.md) already designs around. The recovery design must be proportionate: minimal loss at minimal cost, with backup traffic sized by *new writes* rather than by total stored data, and without replica-operator complexity.

## Decision

The DR tier is **pgBackRest WAL archival to Backblaze B2**, provisioned by the deploy pipeline and activated by the vault (`b2_repo_key_id` — the setup-order contract shared with the heartbeat and Beszel agent; procedures in [`docs/backup.md`](../../../docs/backup.md)):

- Monthly full base backups plus daily incrementals, both run inside the postgres container — its image builds locally from `postgres-archival.Dockerfile` (postgres:17-alpine + the Alpine `pgbackrest` package, because the official pgbackrest image is glibc and cannot enter musl). Continuous WAL archival rides `archive_mode=on` / `archive_command` / `archive_timeout=300`, bounding RPO at ≤ 5 minutes of writes.
- A daily logical `pg_dump` (03:30 UTC, zstd, rclone rate-limited to 2 MB/s, 14-day expiry via B2 lifecycle) provides format diversity against base-image or page-level corruption that would break the PITR chain.
- Backups are immutable: the bucket is versioned, the upload key holds no delete capability, and expiry runs server-side — a compromised host can add backups but not remove them.
- Failure is observed: a daily `pgbackrest check` plus a 26-hour freshness probe plus a `pg_wal`-growth tripwire (128 segments / 2 GB) push verdicts to uptime-kuma; a monthly cron drill restores into a throwaway container and asserts row counts within the RPO bound.

| Property             | pgBackRest WAL archival (chosen)                     | Hourly pg_dump start (rejected)                     | Live streaming replica (rejected)                   |
| -------------------- | ---------------------------------------------------- | --------------------------------------------------- | --------------------------------------------------- |
| RPO (data loss)      | ≤ 5 min (WAL tail)                                   | ≤ 1 h                                               | ~0 (synchronous) or seconds (asynchronous)          |
| RTO (downtime)       | ~30–45 min; premise: 3 Mbps restore download         | ~10 min at today's volume, grows linearly           | seconds to minutes (automatic failover)             |
| Daily upload (~1 GB) | ~0.1–0.2 GB                                          | ~1.2 GB compressed; ~13 min saturation/hour raw    | WAL stream + a second always-on VPS (~$5/month)     |
| Extra infrastructure | none — object storage, ≤ $0.10/month                 | none                                                | second VPS + failover tooling                       |
| Operational cost     | low — configured once; archive stalls need alerting  | low but silent-failure-prone                        | high — lag monitoring, failover automation, split-brain |

## Alternatives considered

**Hourly `pg_dump` as the starting tier.** Full logical dumps are O(total data) per run: an uncompressed hourly dump of the ~1 GB database saturates the 3 Mbps uplink ~13 minutes every hour and re-sends the whole corpus 24×/day (~1.2 GB/day compressed); at the design-max ~15 GB database the run outlasts the hour and the tier dies. RPO, bandwidth, and silent-failure surface are all dominated by WAL + incrementals at this write rate, so "simplest tier first" deferred the right architecture instead of earning it.

**A live streaming replica with automatic failover.** RPO/RTO gains do not justify 25× the cost and replication-operator complexity: the write rate is ~0.12/s, most data decays on a 7-day horizon, and during an outage the read path stays alive on the CDN edge while only writes wait. (Single-instance resilience — no Redis, no second VPS — is decided in [the performance-pass MRFC](../implemented/2026-07-09-read-path-performance-pass.md); this record covers the backup tier that decision leaves open.)

**Cloudflare R2 instead of B2.** Backup is write-heavy, read-rarely: B2 storage is 3× cheaper ($0.005 vs $0.015/GB/month) and the one-time restore egress is negligible. B2 also keeps backups outside the Cloudflare umbrella, so one compromised Cloudflare account cannot delete both the live path and the backups. A second *copy* on R2's free tier was floated as optional account-diversity hardening and left undecided.

**`wal-g` instead of pgBackRest.** Both are mainstream and speak the S3 API B2 implements; pgBackRest is Postgres-specific with page-level incrementals, retention management, and `check` round-trip verification. The 2026-04 maintainer turnover (see Consequences) re-weighted the usual "stronger community" argument; wal-g remains the drop-in fallback if the sponsor coalition stalls.

## Consequences

What the trade bought: RPO ≤ 5 minutes at under 1% of the 3 Mbps uplink and ≤ $0.10/month, with transfer scaling by writes — at the design-max ~15 GB database the same design still fits (monthly fulls grow, the daily stream stays ~MBs), whereas any repeated full dump would not. What it costs: activation rides one planned postgres restart (`archive_mode` is postmaster-context); the locally-built derived image is ours to maintain (Alpine package updates arrive with base-image rebuilds); and an archive stall fills `pg_wal` until the write path dies — mitigated, in order, by `archive-push-queue-max=1GiB` backpressure, the daily check/freshness alerts, the `pg_wal` tripwire, and days of disk headroom on 40 GB. The restore path is exercised monthly by the drill, not assumed. pgBackRest itself passed through maintainer turnover — archived by its sole maintainer on 2026-04-27, continued under a paid sponsor coalition (Percona, AWS, Supabase, and others) from 2026-05-19 — and the repo format is MIT with objects self-held in B2, so even a stalled project leaves existing backups restorable by any old binary. Loss bounds: the WAL tail plus the ~600 ms `synchronous_commit=off` window already accepted in [the tuning spec](../../../specs/backend/postgres-tuning.md). Runtime activation — bucket, no-delete key, lifecycle rules, vaulted key pair, first stanza-create + full — is operator work recorded in [`docs/backup.md`](../../../docs/backup.md); current posture lives in [`specs/backend/disaster-recovery.md`](../../../specs/backend/disaster-recovery.md).
