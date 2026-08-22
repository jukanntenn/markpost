# PostgreSQL Tuning

English | [中文](postgres-tuning.zh.md)

Connection, storage, and topology tuning for the single Postgres instance markpost runs against: connection-pool bounds in the Go process, server GUCs and TOAST compression applied by the deploy templates and migrations, and the sibling-container + Unix-socket topology. The decision record (GUC values, lz4 over application-level compression, Docker over bare metal) lives in [the performance-pass MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.md); DSN formats are specified in [`dsn.md`](./dsn.md).

## Connection pool

`internal/infra/db.go` (`New`) bounds the GORM-backed pool:

```go
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(10)
sqlDB.SetConnMaxLifetime(30 * time.Minute)
```

Unbounded pools under concurrent reads exhaust Postgres connections; 25 open / 10 idle with a 30-minute recycle is sized for the 2-core envelope and the ~0.12 writes/second mean load.

## Server GUCs

Five GUCs are applied as `-c` flags on the postgres service command (layering overrides on the image's initdb-generated `postgresql.conf`), in the production Ansible template (`devops/ansible/templates/docker-compose.yml.j2`) and — via `command: postgres -c ...` — in the dev compose (`devops/docker-compose.yml`):

| GUC                    | Value  | Why                                                                                                                                   |
| ---------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| `shared_buffers`       | 256 MB | The box is not a dedicated DB server (shares 2 GB with Caddy + Go + Next.js); 256 MB leaves OS cache room.                            |
| `effective_cache_size` | 1 GB   | Planner hint for total cache available (shared_buffers + OS cache).                                                                   |
| `maintenance_work_mem` | 128 MB | Vacuum/index maintenance workspace.                                                                                                   |
| `max_connections`      | 50     | Headroom over the pool's 25 open connections.                                                                                         |
| `synchronous_commit`   | off    | Write rate is ~0.12/s of 7-day-retention ephemeral content; the crash-window loss is acceptable and `off` cannot cause inconsistency. |

`shared_buffers` and `max_connections` are postmaster-context (restart required); the rest are reloadable. These are not Go code and cannot be unit-tested — after restart, `SHOW` confirms the values and `pg_settings.source` reports `command line` for the overrides.

## TOAST and lz4

Postgres TOAST automatically compresses and stores out-of-line any `text` value over ~2 KB (on by default, transparent to SQL) — for a 32 KB post body it stores ~10–12 KB after compression. The `posts.body` column uses the **lz4** TOAST compressor instead of the default pglz:

```sql
-- internal/infra/migrations/000001_init.up.sql
ALTER TABLE posts ALTER COLUMN body SET COMPRESSION lz4;
```

lz4 decompresses ~3× faster than pglz at comparable ratios; on a 32 KB body the decompression cost is tens of microseconds per read, further amortized by the render cache. The `ALTER` lives in the versioned migration — golang-migrate runs each version exactly once per database (`schema_migrations` gating), so the `AccessExclusiveLock` that `SET COMPRESSION` takes is acquired exactly once. `SET COMPRESSION` is metadata-only (idempotent in effect; it does not rewrite rows): existing rows keep their old compression until rewritten, so a one-time `VACUUM FULL posts` in a maintenance window retrofits them — deliberately not automated because it holds `AccessExclusiveLock` for the duration of the rewrite.

Application-level gzip-into-`BYTEA` (~4× compression vs ~3× for TOAST) is the documented escalation step if disk pressure ever demands it; it moves decompression into the Go hot path on every read and changes the column type, so TOAST-lz4 is the default (see the MRFC).

## Topology: sibling container, Unix socket, Docker

Postgres runs in a **sibling container** (not inside the markpost container), with data on a named volume (`pgdata`) that bypasses the container's overlay2 writable layer, and the Go process connects over a **Unix domain socket** through a shared `/var/run/postgresql` volume — eliminating the TCP/NAT overhead a cross-container TCP connection would add. The socket path matches the postgres image's default `unix_socket_directories`; the DSN uses `host=/var/run/postgresql ... sslmode=disable` (see [`dsn.md`](./dsn.md)).

Whether Docker's overhead compromises the workload — it does not, for topology-specific reasons:

| Overhead category  | Impact here                    | Why                                                                                                                                                                                                            |
| ------------------ | ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CPU virtualization | **none**                       | Linux containers use cgroups + namespaces; no instruction translation. The Go binary in the container is the same binary on bare metal, scheduled by the same kernel.                                          |
| Storage I/O        | **negligible**                 | `./data`, `pgdata`, and the Postgres socket dir are bind mounts / named volumes — they bypass overlay2 and hit the host filesystem directly. Only read-only image content lives on overlay2, off the hot path. |
| Network            | **one NAT hop, ~microseconds** | The host→container port hop is the only cross-namespace traversal; the three in-container services talk over loopback, and Go↔Postgres uses the Unix socket, not the docker0 bridge.                           |

The topology lands in Docker's sweet spot — it sidesteps the two well-known container performance traps (cross-container networking and overlay2 data writes). Bare metal would free single-digit microseconds per request while leaving the binding constraint (outbound bytes on the 3 Mbps link, see [`caching.md`](./caching.md)) untouched, and would abandon the declarative reproducible environment, the one-command self-hosted install (`docker compose up`), and the Ansible/CI pipeline that builds consistently across amd64/arm64.

## Storage estimation

"Up to 1000 posts/day/user" is a hard cap, not the expected mean; a notification/temporary-share tool's real volume is single-digit posts per user per day. Using a conservative mean of μ = 10 posts/user/day:

```
10 000 users × 10 posts/day × 32 KB × 7 days = 22.4 GB (raw)
× 1.3 (Postgres row/index overhead) = 29 GB
× TOAST compression (32 KB → ~12 KB) ≈ 11 GB on disk
```

11 GB fits comfortably in the 40 GB disk; even doubling the mean yields ~22 GB, still within budget. If actual growth exceeds the estimate, the decided escalation ladder (free lz4 switch → application-level gzip-into-BYTEA → cold-tier offload to object storage) is recorded in the MRFC.

Delivery-queue tables add ~4.2 GB (see [`delivery-queue.md`](./delivery-queue.md) _Storage_).
