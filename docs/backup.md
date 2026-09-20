# Backup and restore

[English](backup.md) | [中文](backup.zh.md)

Operating guide for markpost's disaster-recovery backups: pgBackRest WAL archival plus a daily logical dump to Backblaze B2 (design and rationale: [the WAL-archival MRFC](../.agents/mrfcs/implemented/2026-07-09-wal-archival-disaster-recovery.md); current posture: [`specs/backend/disaster-recovery.md`](../specs/backend/disaster-recovery.md)). The whole tier is production-only and activates when the vault defines `b2_repo_key_id` — until then the deploy stays byte-identical to the pre-backup pipeline.

<a id="b2-provisioning"></a>

## B2 provisioning (once, before first activation)

1. Create the bucket `markpost-backups` with **versioning enabled** — a compromised host can then only add objects, never overwrite history.
2. Create a bucket-restricted application key with `writeFiles`, `readFiles`, `listFiles` capabilities and **no `deleteFiles`** — expiry is executed by B2's server-side lifecycle, which does not go through this key.
3. Add lifecycle rules: expire noncurrent versions of `dumps/*` after 14 days; keep the `markpost/` (pgBackRest repo) prefix to the repo's own `repo1-retention-full=2` management, with noncurrent expiry after 30 days as a backstop.
4. Vault the key pair (see the [vault section](#vault-variables)) and deploy.

<a id="vault-variables"></a>

## Vault variables

```sh
ansible-vault encrypt_string --vault-id markpost-prod@avpm-client --stdin-name b2_repo_key_id \
    >> devops/ansible/group_vars/production/vault.yml
ansible-vault encrypt_string --vault-id markpost-prod@avpm-client --stdin-name b2_repo_app_key \
    >> devops/ansible/group_vars/production/vault.yml
```

`kuma_backup_url` (optional, same push-monitor pattern as the availability heartbeat in [monitoring](monitoring.md)) turns the daily check into an explicit up/down push; without it the check still fails loudly via cron logs.

<a id="activation"></a>

## First activation (safe order)

The deploy applies the archival GUCs (`archive_mode=on` needs the postgres restart the recreate already is). After it, initialize the stanza and take the first full immediately — WAL accumulated without a base backup is unusable:

```sh
docker compose exec postgres pgbackrest --stanza=markpost stanza-create
docker compose exec postgres pgbackrest --stanza=markpost backup --type=full
docker compose exec postgres pgbackrest --stanza=markpost check
```

<a id="schedule"></a>

## Standing schedule (cron, off-peak)

| Time (UTC)         | Entry                  | What                                                                      |
| ------------------ | ---------------------- | ------------------------------------------------------------------------- |
| daily 03:30        | `pg-dump-backup.sh`    | logical dump → zstd → rclone to B2 `dumps/`, rate-limited to 2 MB/s       |
| daily 04:45        | `pgbackrest-backup.sh` | incremental; full on the 1st of the month                                 |
| daily 05:30        | `pgbackrest-check.sh`  | archive round-trip check + 26 h freshness + pg_wal growth tripwire → kuma |
| monthly, 2nd 05:00 | `pgbackrest-drill.sh`  | restore drill into a throwaway container with row-count assertions        |

RPO is bounded by the WAL tail (`archive_timeout=300` → ≤ 5 minutes of writes); RTO is ~30–45 minutes with the 3 Mbps restore download as the premise.

<a id="disaster-recovery"></a>

## Disaster recovery runbook

1. Provision a replacement VPS and run the usual deploy (the vault carries the B2 credentials; the archival tier comes up with it).
2. Stop the postgres service before touching the data dir: `docker compose stop postgres`.
3. Restore the latest backup set: `docker compose run --rm --entrypoint pgbackrest postgres --stanza=markpost restore` (empty `pgdata` volume; add `--type=time --target="…"` for PITR to an earlier moment — deleted-then-pruned rows come back for the next retention sweep to re-prune).
4. Start postgres: `docker compose up -d postgres` — WAL replays automatically to the archive end.
5. Verify (`/api/v1/ready`, spot-check row counts), repoint DNS/CDN if the IP changed.
6. Take a fresh full backup immediately — this resets the RPO clock for any follow-on incident.

<a id="failure-modes"></a>

## Failure modes

- **`pgbackrest-check.sh` fails on round-trip** — B2 unreachable or credentials broken; watch `pg_wal` growth (the check trips at 128 segments / 2 GB) and fix before the 40 GB disk fills: un-archived segments cannot be recycled and a full disk stops the write path.
- **Freshness probe fails (no backup in 26 h)** — the backup cron died silently; inspect `pgbackrest-backup.log`.
- **Drill count assertion fails** — the restore chain is broken; treat as an incident and re-run the drill after fixing before trusting the repo.
