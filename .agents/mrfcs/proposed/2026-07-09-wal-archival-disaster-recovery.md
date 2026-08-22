# MRFC: WAL-archival disaster recovery to B2

Status: proposed

English | [中文](2026-07-09-wal-archival-disaster-recovery.zh.md)

## Problem

markpost runs as a single instance: one VPS, one Postgres container, no replica. If the server dies or the host loses its data, everything is gone — the deploy pipeline schedules no backup at all. The data is 7-day-retention ephemeral content at ~0.12 writes/second, so the recovery design must be proportionate: minimal loss at minimal cost, without replica-operator complexity.

## Proposal

Adopt **WAL archival to object storage** as the DR architecture: continuous upload of WAL segments plus periodic full base backups to **Backblaze B2**, restored by replaying WAL forward from a base backup. Start with the simplest tier — hourly `pg_dump` uploaded to B2 by cron (RPO ≤ 1 hour, RTO ~10 minutes) — and upgrade to **pgBackRest** continuous WAL archival (RPO in seconds with PITR) when the hourly RPO proves insufficient. Provisioning is operator work on the existing Ansible-managed instance; nothing lands in application code.

| Property             | WAL archival (chosen)                                     | Live streaming replica (rejected)                                   |
| -------------------- | --------------------------------------------------------- | ------------------------------------------------------------------- |
| RPO (data loss)      | seconds (WAL) or ≤1 h (dump)                              | ~0 (synchronous) or seconds (asynchronous)                          |
| RTO (downtime)       | ~30 minutes (provision VPS, pull base backup, replay WAL) | seconds to minutes (automatic failover)                             |
| Extra infrastructure | none — object storage only, ~$0.20/month for 40 GB        | a second always-on VPS (~$5/month)                                  |
| Operational cost     | low — pgBackRest configured once, runs unattended         | high — replication-lag monitoring, failover automation, split-brain |

## Alternatives considered

**A live streaming replica with automatic failover.** RPO/RTO gains do not justify 25× the cost and replication-operator complexity: the write rate is ~0.12/s, the data decays on a 7-day horizon, and during an outage the read path stays alive on the CDN edge while only writes wait. (Single-instance resilience — no Redis, no second VPS — is decided in [the performance-pass MRFC](../implemented/2026-07-09-read-path-performance-pass.md); this record covers the backup tier that decision leaves open.)

**Cloudflare R2 instead of B2.** Backup is write-heavy, read-rarely: B2 storage is 3× cheaper ($0.005 vs $0.015/GB/month) and the one-time restore egress (~$0.40 for 40 GB) is negligible. B2 also keeps backups outside the Cloudflare umbrella, so one compromised Cloudflare account cannot delete both the live path and the backups.

**`wal-g` instead of pgBackRest.** Both are mainstream and speak the S3 API B2 implements; pgBackRest is Postgres-specific with incremental backups, parallel processing, and integrity checks, and has stronger community documentation. Either works; pgBackRest is the recommendation.

## Acceptance criteria

- A cron-driven `pg_dump`-to-B2 (or pgBackRest full backup) runs unattended on the production instance and its failure is observable.
- A documented restore procedure exists and has been executed once against a scratch VPS, restoring to a consistent database within the stated RTO.
- The escalation to continuous WAL archival is a configuration change, not a redesign.

## Risks

B2 is an external dependency with its own outage profile (rare; multi-replica); the hourly `pg_dump` tier loses up to an hour of writes; and until this proposal is implemented, the instance runs with no automated backup at all — the current-state posture is documented in [`specs/backend/disaster-recovery.md`](../../../specs/backend/disaster-recovery.md).
