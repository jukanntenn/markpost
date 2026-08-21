# Database DSN Specification

This document defines markpost's PostgreSQL connection DSN formats, sslmode choices, timezone handling, and password injection. The database schema is owned by [database-schema.md](./database-schema.md); config loading and the env override mechanism are owned by [configuration.md](./configuration.md).

PostgreSQL is the only supported database. The `db.driver` config key accepts exactly `"postgresql"` (validated by `oneof=postgresql`, defaulted by Viper), and `internal/infra/db.go` opens the connection with GORM's postgres driver on pgx v5 — never lib/pq.

## DSN formats

pgx accepts two DSN styles; both work anywhere a DSN is accepted (`config.toml` `[db] dsn`, `MARKPOST_DB__DSN`, the dev compose environment).

**Keyword format** (readable; a password containing special characters needs no escaping):

```
host=localhost port=5432 user=markpost password=CHANGE_ME dbname=markpost sslmode=verify-full
```

**URL format** (a password containing `@:/` and similar characters must be percent-encoded):

```
postgres://markpost:CHANGE_ME@localhost:5432/markpost?sslmode=verify-full
```

**Unix domain socket** (same-host deployment; no TCP overhead — see the tuning notes in [postgres-tuning.md](./postgres-tuning.md)):

```
host=/var/run/postgresql user=markpost password=CHANGE_ME dbname=markpost sslmode=disable
```

## sslmode

The value is the deployer's topology choice, not something this spec forces:

| Value         | When                                                                          |
| ------------- | ----------------------------------------------------------------------------- |
| `disable`     | Private network / Unix socket (no TLS)                                        |
| `require`     | Force TLS, no certificate validation                                          |
| `verify-full` | Force TLS + certificate validation (recommended for cross-network production) |

With the Cloudflare Full-strict setup in [cloudflare.md](./cloudflare.md), `verify-full` closes the end-to-end encryption loop between CDN and origin.

## Timezone handling

`db.timezone` (IANA name, default `UTC`) pins three things to one zone so writes, reads, and `time.Now()` all agree regardless of the process `TZ` or server default (`internal/infra/db.go`):

- `time.Local` is set to the configured zone, so pgx's timestamptz decode and every `time.Now()` caller land in it
- a `timezone=<zone>` parameter is injected into the DSN (unless already present), which the pgx-backed driver applies as the session timezone on every pooled connection
- GORM's `NowFunc` stamps `autoCreateTime`/`autoUpdateTime` columns in the same zone

The zone is validated through `time.LoadLocation` at startup, so an invalid name fails fast.

## Connection pool

`infra.New` configures the pool: `MaxOpenConns(25)`, `MaxIdleConns(10)`, `ConnMaxLifetime(30m)`. The dev compose raises no per-connection Postgres limits beyond `max_connections=50` on the server side.

## Password handling

The password is part of the DSN string. Injection uses the standard config precedence (see [configuration.md](./configuration.md)): environment variable > TOML file > built-in default. The production pattern is a placeholder DSN in `config.toml` with the real value supplied by environment:

```bash
export MARKPOST_DB__DSN="host=db user=markpost password=real_secret dbname=markpost sslmode=verify-full"
```

## References

- [database-schema.md](./database-schema.md) — schema design
- [configuration.md](./configuration.md) — config loading, env override mechanism
- [cloudflare.md](./cloudflare.md) — deployment modes, CDN↔origin TLS
