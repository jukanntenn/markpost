# MRFC: WAL-archival disaster recovery to B2

Status: proposed

English | [中文](2026-07-09-wal-archival-disaster-recovery.zh.md)

## Problem

markpost runs as a single instance: one VPS, one Postgres container, no replica. If the server dies or the host loses its data, everything is gone — the deploy pipeline schedules no backup at all. The data is ephemeral content at ~0.12 writes/second under per-user retention (global default 7 days; a VIP subset is retained indefinitely per [the retention-policy MRFC](../implemented/2026-08-31-per-user-history-retention-policy.md)), and the VPS uplink is ~3 Mbps — outbound bytes are the binding constraint the [caching spec](../../../specs/backend/caching.md) already designs around. The recovery design must be proportionate: minimal loss at minimal cost, with backup traffic sized by *new writes* rather than by total stored data, and without replica-operator complexity.

## Proposal

Adopt **WAL archival to object storage** as the DR architecture, deployed at the **pgBackRest** tier from day one: monthly full base backups plus daily incrementals to **Backblaze B2**, continuous WAL archival (`archive_timeout=300`, RPO ≤ 5 minutes), PITR restore. At the realistic ~1 GB database this moves ~0.1–0.2 GB/day — under 1% of the 3 Mbps uplink — because transfer scales with writes; any repeated full dump instead re-sends the whole corpus on every run.

Three properties round the tier out:

- **Format diversity.** A daily compressed `pg_dump` (off-peak, rate-limited, 14-day retention) survives base-image or page-level corruption that would break the PITR chain.
- **Immutability.** The B2 bucket is versioned, the upload key holds no delete capability, and expiry runs server-side via lifecycle rules — a compromised host can add backups but not remove them.
- **Observable, drilled.** Daily `pgbackrest check` plus a no-fresh-backup lag probe push to the existing uptime-kuma monitor; `pg_wal` disk-headroom alerting (see Risks); a monthly automated restore drill in a throwaway container asserts row counts and samples.

Provisioning is operator work on the existing Ansible-managed instance; nothing lands in application code.

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

**`wal-g` instead of pgBackRest.** Both are mainstream and speak the S3 API B2 implements; pgBackRest is Postgres-specific with page-level incrementals, retention management, and `check` round-trip verification. The 2026-04 maintainer turnover (see Risks) re-weighted the usual "stronger community" argument; wal-g remains the drop-in fallback if the sponsor coalition stalls.

## Acceptance criteria

- pgBackRest — monthly full, daily incremental, continuous WAL archival to B2 with `archive_timeout=300` — runs unattended on the production instance; failure is observable (daily `pgbackrest check` + lag probe → uptime-kuma; `pg_wal` disk-headroom alerting).
- Backups are immutable: versioned bucket, upload key without delete capability, expiry via server-side lifecycle rules.
- The daily off-peak, rate-limited logical dump (14-day retention) runs as the format-diversity layer.
- A documented restore procedure exists — stating the 3 Mbps restore-download premise behind the ~30–45 min RTO — and a monthly automated drill in a throwaway container exercises it with row-count and sample assertions.
- Enabling follows the safe order: archiving GUCs → one planned Postgres restart → immediate first full backup (WAL accumulated without a base backup is unusable).

## Risks

Archive stalls fill `pg_wal`: Postgres cannot recycle un-archived segments, so a long B2 or credential outage grows the disk until the write path dies. `archive-push-queue-max` backpressure, the check/lag alerts, and disk-headroom monitoring on the 40 GB disk give days of reaction room. pgBackRest itself passed through maintainer turnover — archived by its sole maintainer on 2026-04-27, continued under a paid sponsor coalition (Percona, AWS, Supabase, and others) from 2026-05-19, v2.59.1 (2026-08-17) current — and the repo format is MIT with objects self-held in B2, so even a stalled project leaves existing backups restorable by any old binary. B2 remains an external dependency with its own outage profile (rare; multi-replica). The WAL tail bounds loss at ≤ 5 minutes of writes (~36 commits, plus the ~600 ms `synchronous_commit=off` window already accepted in [the tuning spec](../../../specs/backend/postgres-tuning.md)); and until this proposal is implemented, the instance runs with no automated backup at all — the current-state posture is documented in [`specs/backend/disaster-recovery.md`](../../../specs/backend/disaster-recovery.md).
