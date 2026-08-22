# Disaster Recovery

markpost's resilience posture: a single instance whose read path survives origin death on the CDN edge, whose data is 7-day-retention ephemeral content, and whose backup design is WAL archival to object storage. The decided-but-undeployed backup architecture and its alternatives (live replica, R2, wal-g) are recorded in [the disaster-recovery MRFC](../../.agents/mrfcs/proposed/2026-07-09-wal-archival-disaster-recovery.md); the single-instance decision itself (no Redis, no replica, no second VPS) is part of [the performance-pass MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.md).

## Current posture

- **Single instance.** One VPS runs the markpost container (Caddy + Go + Next.js) and a sibling Postgres container ([`postgres-tuning.md`](./postgres-tuning.md)). There is no replica and no shared cache; in-memory state (render cache, rate-limit buckets) is process-local and rebuilt on restart.
- **The deploy pipeline configures no automated backup.** `devops/ansible/` provisions the instance but schedules no `pg_dump`/WAL upload; backing up is an operator action following the decided design in the MRFC above.
- **The read path degrades gracefully without the origin.** Posts already cached at the CDN edge stay readable for up to their one-hour TTL while the origin is down ([`caching.md`](./caching.md)); only the write path and uncached reads wait for recovery. The data's own value decays on the same scale — it is 7-day-retention ephemeral content.

## Recovery matrix

| Failure                        | Impact                      | Recovery                                                                                 |
| ------------------------------ | --------------------------- | ---------------------------------------------------------------------------------------- |
| VPS crash (restartable)        | service down                | restart; RPO = 0                                                                         |
| VPS destroyed / host data loss | all data gone               | new VPS, restore from object storage; RPO ~seconds (WAL) or ≤1 h (dump); RTO ~30 min     |
| Postgres data-file corruption  | partial data                | PITR to the instant before corruption                                                    |
| Accidental `DELETE`            | rows lost                   | PITR to before the deletion                                                              |
| Cloudflare outage              | new posts cannot be created | existing posts still readable from CDN edge; write path returns when Cloudflare recovers |
| Object-store outage (rare)     | new backups stall           | existing backups intact; B2 is itself multi-replica                                      |

## Cost

| Item                                                         | Cost                                      |
| ------------------------------------------------------------ | ----------------------------------------- |
| Cloudflare free tier                                         | $0/month (unlimited bandwidth, free DDoS) |
| Backblaze B2 backup (~40 GB)                                 | $0.20/month                               |
| `ristretto`, `singleflight`, `pgBackRest`, `vegeta`, `pprof` | $0 (open source)                          |
| **Total marginal**                                           | **~$0.20/month**                          |
