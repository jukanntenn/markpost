# Disaster Recovery

English | [中文](disaster-recovery.zh.md)

markpost's resilience posture: a single instance whose read path survives origin death on the CDN edge, whose data is retention-managed ephemeral content, and whose backup tier is pgBackRest WAL archival to object storage with a daily logical dump for format diversity. The architecture and its alternatives (hourly-dump start, live replica, R2, wal-g) are recorded in [the WAL-archival MRFC](../../.agents/mrfcs/implemented/2026-07-09-wal-archival-disaster-recovery.md); the single-instance decision itself (no Redis, no replica, no second VPS) is part of [the performance-pass MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.md). Operating procedures — provisioning, activation, restore, drills — live in [`docs/backup.md`](../../docs/backup.md).

## Current posture

- **Single instance.** One VPS runs the markpost container (Caddy + Go + Next.js) and a sibling Postgres container ([`postgres-tuning.md`](./postgres-tuning.md)). There is no replica and no shared cache; in-memory state (render cache, rate-limit buckets) is process-local and rebuilt on restart.
- **The deploy pipeline provisions the archival tier, activated per environment by the vault.** With `b2_repo_key_id` vaulted (staging and production each, against their own bucket), `devops/ansible/` swaps postgres to a locally-built image carrying the pgBackRest client, applies the three archival GUCs, templates the B2 repo config, and schedules the backup/observability/drill crons; without the vault var the deploy is byte-identical to the pre-backup pipeline (the setup-order contract shared with the heartbeat and Beszel agent).
- **Backups are immutable and observed.** The B2 bucket is versioned and the upload key holds no delete capability; a daily check pushes the archive round-trip, backup freshness, and `pg_wal` growth verdicts to uptime-kuma; a monthly cron drill restores into a throwaway container and asserts row counts within the RPO bound.
- **The read path degrades gracefully without the origin.** Posts already cached at the CDN edge stay readable for up to their one-hour TTL while the origin is down ([`caching.md`](./caching.md)); only the write path and uncached reads wait for recovery. The data's own value decays on the same scale — retention-managed ephemeral content (7-day default, per-user overrides, VIP indefinite).

## Recovery matrix

| Failure                        | Impact                      | Recovery                                                                                   |
| ------------------------------ | --------------------------- | ------------------------------------------------------------------------------------------ |
| VPS crash (restartable)        | service down                | restart; RPO = 0                                                                           |
| VPS destroyed / host data loss | all data gone               | new VPS, restore from object storage; RPO ≤ 5 min (WAL tail); RTO ~30–45 min at 3 Mbps     |
| Postgres data-file corruption  | partial data                | PITR to the instant before corruption; the daily logical dump covers base-image corruption |
| Accidental `DELETE`            | rows lost                   | PITR to before the deletion; retention-pruned rows return and the sweep re-prunes them     |
| Cloudflare outage              | new posts cannot be created | existing posts still readable from CDN edge; write path returns when Cloudflare recovers   |
| Object-store outage (rare)     | new backups stall           | existing backups intact; B2 is itself multi-replica; `pg_wal` growth tripwire alerts       |

## Cost

| Item                                                         | Cost                                           |
| ------------------------------------------------------------ | ---------------------------------------------- |
| Cloudflare free tier                                         | $0/month (unlimited bandwidth, free DDoS)      |
| Backblaze B2 backup (~2–3 GB at realistic volume)            | ~$0–0.10/month, bounded by the 10 GB free tier |
| `ristretto`, `singleflight`, `pgBackRest`, `vegeta`, `pprof` | $0 (open source)                               |
| **Total marginal**                                           | **≤ $0.10/month**                              |
