# Deployment & Operations Guide

This document covers the full lifecycle of markpost across all environments:
local acceptance, dev, staging, and production. It replaces the earlier
single-container quick-start (which no longer matches the image, which ships a
Caddy + Go composite served by s6-overlay with no separate Node server).

## Architecture

Every environment shares one topology:

```
           ┌─────────────────────────────────────────────────────┐
           │ markpost container (s6-overlay)                     │
           │                                                     │
  :2053 ─▶ │  Caddy :2053  ──reverse_proxy──▶  Go backend :7330  │
           │                                                     │
           └──────────────────────────┬──────────────────────────┘
                                      │ Unix socket (/var/run/postgresql)
                                      ▼
           ┌─────────────────────────────────────────────────────┐
           │ postgres container (postgres:17-alpine)             │
           └─────────────────────────────────────────────────────┘
```

Caddy serves the Next.js static export (`/app/frontend`) directly and
reverse-proxies `/api/v1/*`, `/static/*`, `/swagger/*`, `/mpk-*`, `/p-*` to the
Go backend on `127.0.0.1:7330`. Postgres is reached over a shared
`postgres-socket` named volume (Unix socket).

### Internal ports (identical everywhere)

| Component  | Port   | Published?                                |
| ---------- | ------ | ----------------------------------------- |
| Go backend | `7330` | No (Caddy proxies to it)                  |
| Caddy      | `2053` | Yes (host port varies per env)            |
| Postgres   | `5432` | No (Unix socket, container-internal only) |

### Host ports (per environment)

The container always listens on 2053; the host port it is mapped to varies:

| Environment       | Host port → container | Notes                                                           |
| ----------------- | --------------------- | --------------------------------------------------------------- |
| e2e / acceptance  | `2053:2053`           | Local verification                                              |
| dev (fn)          | `8089:2053`           | Fixed for backward compat                                       |
| staging (oect)    | `8089:2053`           | Fixed for backward compat                                       |
| production (ttyo) | `2053:2053`           | Behind Cloudflare (edge port 443 → origin 2053 via Origin Rule) |

In production, visitors connect to Cloudflare on the standard HTTPS port 443
(`https://markpost.cc`). Cloudflare then reaches the origin on 2053 (the origin's
host port), set via an Origin Rule — see §4.1. The 443 port is never involved on
the origin side.

## TLS strategy

| Environment      | TLS profile | Mechanism                                               |
| ---------------- | ----------- | ------------------------------------------------------- |
| e2e / acceptance | `internal`  | Caddy `tls internal` (self-signed CA, baked into image) |
| dev / staging    | `http`      | `auto_https off`, plain HTTP                            |
| production       | `origin`    | Cloudflare Origin CA cert (`/app/certs/origin.pem`)     |

---

## 1. Local acceptance verification

The acceptance compose (`docker/docker-compose.yml`) mirrors the production
topology (Caddy + Go + Postgres over Unix socket) but builds from local source
and uses Caddy's internal CA for self-signed TLS. It stores **no persistent
data** — like the e2e suite, the app data dir and Postgres data live entirely
in the ephemeral containers.

```bash
# From the repo root:
docker compose -f docker/docker-compose.yml up -d --build

# Verify (self-signed cert, so -k):
curl -k https://localhost:2053/api/v1/health

# Tear down (no named data volumes to worry about):
docker compose -f docker/docker-compose.yml down
```

The config is read from `.local/config.acceptance.toml` (gitignored, per-developer).

## 2. Dev (fn @ 192.168.5.200)

Remote fast-iteration environment. Self-hosted, no Cloudflare, no HTTPS. Existing
data (489 posts, 1 user, 1 channel) is preserved in the named `pgdata` volume.

### Deploy / update

```bash
# Build & push the rolling main tag from the current workspace (default
# tag, default registry), then deploy:
python3 docker/build.py --push
ansible-playbook devops/ansible/deploy.yml          # dev (fn) is the default target
```

The playbook ends with a post-deploy verification (see §5): it polls
`/api/v1/health` through the public URL and checks `/api/v1/version` against
the deploying checkout's `git describe` — a mismatch means the container is
still running the old image.

### Schema self-healing

dev's schema is behind the current code (missing `delivery_attempts`,
`delivery_history` tables and `refresh_tokens.revoked` column as of this writing).
The new image's `AutoMigrate` + idempotent migration chain (see
`backend/internal/infra/db.go`) brings the schema up to date automatically on
boot — no manual migration step is needed. The 489 existing posts are untouched
(AutoMigrate is additive: it adds columns/tables/indexes, never drops or rewrites).

---

## 3. Staging (oect @ 192.168.5.50) — SQLite → PostgreSQL migration

staging currently runs SQLite (`./data/db.sqlite3`, 2103 posts, 1 user, 1
channel). This section covers the one-time migration to PostgreSQL, after which
staging's topology matches production.

### Prerequisites

1. Build and push the new image (which includes the `migrate-sqlite-to-postgres`
   subcommand):

   ```bash
   python3 docker/build.py --push
   ```

2. The staging vault (`group_vars/staging/vault.yml`) must define `db_password`
   for the new Postgres user. Add it as a single encrypted variable (plaintext
   via stdin, so it never lands in shell history):

   ```bash
   printf '%s' '<a-strong-password>' | ansible-vault encrypt_string \
       --vault-id markpost-staging@~/.local/bin/avpm-client \
       --stdin-name db_password >> devops/ansible/group_vars/staging/vault.yml
   ```

### Migration procedure

There is no rollback safety net and no downtime constraint for staging. If the
migration fails, diagnose and fix, then re-run.

**Step 1 — Stop the app and back up the SQLite file:**

```bash
ssh alice@192.168.5.50 << 'EOF'
cd /mnt/ssd/alice/docker/markpost
docker compose stop markpost
cp data/db.sqlite3 data/db.sqlite3.pre-migration.bak
EOF
```

**Step 2 — Deploy the new compose (adds the Postgres sibling, switches config to
`driver = "postgresql"`):**

```bash
ansible-playbook devops/ansible/deploy.yml -e target=staging
```

This renders the new `docker-compose.yml` (with the Postgres service) and the new
`config.toml` (Unix-socket Postgres DSN). The `docker_compose_v2` task starts the
Postgres container. The markpost container starts, runs `AutoMigrate` against the
**empty** Postgres (creating the full schema), and seeds a fresh admin — but the
2103 old posts are NOT there yet.

**Step 3 — Run the migration (dry-run first, then for real):**

```bash
ssh alice@192.168.5.50 << 'EOF'
cd /mnt/ssd/alice/docker/markpost

# Dry run: report source row counts and target state without writing.
docker compose run --rm \
    -v "$(pwd)/data/db.sqlite3.pre-migration.bak:/migration/source.db:ro" \
    markpost markpost -c /app/config.toml \
    migrate-sqlite-to-postgres --sqlite /migration/source.db --dry-run

# Real run: copy users → posts → refresh_tokens → token_blacklist → delivery_channels,
# preserve primary-key ids, and resync Postgres sequences.
docker compose run --rm \
    -v "$(pwd)/data/db.sqlite3.pre-migration.bak:/migration/source.db:ro" \
    markpost markpost -c /app/config.toml \
    migrate-sqlite-to-postgres --sqlite /migration/source.db
EOF
```

**Step 4 — Restart the app and verify:**

```bash
ssh alice@192.168.5.50 << 'EOF'
cd /mnt/ssd/alice/docker/markpost
docker compose up -d markpost
# Verify row counts match the source (2103 posts, 1 user, 1 channel):
docker compose exec postgres psql -U markpost -d markpost -c \
    "SELECT 'users' as t, count(*) FROM users UNION ALL
     SELECT 'posts', count(*) FROM posts UNION ALL
     SELECT 'delivery_channels', count(*) FROM delivery_channels;"
EOF
```

**Step 5 — Once verified, the old SQLite backup can be removed:**

```bash
ssh alice@192.168.5.50 'rm /mnt/ssd/alice/docker/markpost/data/db.sqlite3.pre-migration.bak'
```

---

## 4. Production (ttyo @ 43.133.160.29) — greenfield

Production is a fresh VPS behind Cloudflare. This section covers the one-time
setup (VPS prep, Cloudflare config, Origin CA cert) and the recurring deploy.

### 4.1 One-time Cloudflare setup

Perform these in the Cloudflare dashboard for the `markpost.cc` zone:

1. **DNS record:** Create an A record `markpost.cc` → `<VPS public IP>`,
   **Proxied** (orange cloud). This hides the origin IP and enables CDN/WAF/DDoS.

2. **SSL/TLS mode:** Set to **Full (strict)**. This requires an origin cert
   (next step) and validates it, preventing MITM on the CF↔origin leg.

3. **Origin CA certificate:** Go to SSL/TLS → Origin Server → Create Certificate.
   - Key type: RSA (2048) or ECC.
   - Hostnames: `markpost.cc, *.markpost.cc` (or just `markpost.cc`).
   - Validity: 15 years (or as desired).
   - Copy the **certificate** (save as `origin.pem`) and the **private key**
     (`origin.key`).

4. **Origin Rule (destination port rewrite):** Go to Rules → Origin Rules →
   Create rule. Cloudflare connects to the origin on port 443 by default, but the
   origin's Caddy listens on 2053 (the origin's host port). Add a rule to rewrite
   the destination port:
   - **If:** `Hostname` `equals` `markpost.cc`
   - **Then:** `Destination port` `Rewrite to` `2053`
   - This is available on the Free plan (10 rules).

> **Caching note:** Cloudflare caches based on the visitor-facing edge port (443,
> the standard HTTPS port users hit), not the origin port. `_next/static/*`
> assets are edge-cached by default and do not re-hit the origin on every request.
> The origin port (2053) is irrelevant to cache decisions because the cache lookup
> happens at the edge before the origin connect.

### 4.2 One-time VPS preparation

```bash
ssh alice@43.133.160.29 << 'EOF'
# Install Docker if not present (Docker Engine + Compose plugin).
# Create the certs directory and place the Origin CA cert + key:
mkdir -p ~/docker/markpost/certs
# (Transfer origin.pem and origin.key into ~/docker/markpost/certs/ — out of band,
#  e.g. scp from your workstation. The key must stay private.)
chmod 600 ~/docker/markpost/certs/origin.key
chmod 644 ~/docker/markpost/certs/origin.pem
EOF
```

**Firewall — restrict origin to Cloudflare IPs:** Allow inbound TCP 2053 only
from Cloudflare's published CIDR ranges. This prevents direct connections to the
origin if its IP leaks. Fetch the current ranges from
`https://www.cloudflare.com/ips/` (IPv4 + IPv6).

Example with ufw (adapt to your firewall):

```bash
# Reset and allow SSH + Cloudflare only on 2053.
ufw default deny incoming
ufw allow 22/tcp                          # SSH (consider restricting to your IP)
# IPv4
for ip in $(curl -s https://www.cloudflare.com/ips-v4); do ufw allow from "$ip" to any port 2053 proto tcp; done
# IPv6
for ip in $(curl -s https://www.cloudflare.com/ips-v6); do ufw allow from "$ip" to any port 2053 proto tcp; done
ufw enable
```

Also sync the Cloudflare CIDRs into `group_vars/production/vars.yml`'s
`cloudflare_cidrs` variable (space-separated) so Caddy's `trusted_proxies` only
trusts Cloudflare.

### 4.3 Production secrets

Secrets are per-variable `!vault` blocks in `group_vars/production/vault.yml`,
each encrypted with the avpm keyring identity `markpost-prod` (declared in the
root `ansible.cfg`). Add or rotate a value with (plaintext via stdin):

```bash
printf '%s' '<the-secret>' | ansible-vault encrypt_string \
    --vault-id markpost-prod@~/.local/bin/avpm-client \
    --stdin-name <name> >> devops/ansible/group_vars/production/vault.yml
```

Required keys: `jwt_access_signing_key`, `jwt_refresh_signing_key`,
`admin_password`, `db_password`. Optional: `github_client_id`,
`github_client_secret`, `cloudflare_api_token`.

### 4.4 Deploy to production

Production runs a **pinned Docker Hub release** published by the
`docker-publish.yml` workflow on `v*` git tags (see §7), not a locally built
image. Bump `markpost_version` in `group_vars/production/vars.yml` after staging has
validated the same version, then:

```bash
ansible-playbook devops/ansible/deploy.yml -e target=production
```

The playbook creates the directory structure, renders the three config files,
pulls + recreates the containers (postgres → migrate → app), and verifies the
deployment through Cloudflare (health + version).

### 4.5 Verify production

```bash
# Through Cloudflare (what users see):
curl -f https://markpost.cc/api/v1/health

# Direct to origin (should work only from a Cloudflare IP after firewall setup):
curl -f --resolve markpost.cc:2053:43.133.160.29 \
    --cacert <(echo) https://markpost.cc:2053/api/v1/health 2>/dev/null || true
```

---

## 5. Ansible playbook reference

Run from the repo root — the root `ansible.cfg` supplies the inventory and the
avpm vault identities, so no `-i` / `--vault-password-file` flags are needed.
The environment is chosen with `-e target=<env>`; **dev is the default**, so
the bare command deploys fn (the crashable dogfood environment).

### Directory layout (unified)

```
ansible.cfg                     # root: inventory + avpm vault_identity_list
devops/ansible/
  hosts.yml                    # dev/staging/production groups
  deploy.yml                   # single playbook (default target: dev)
  group_vars/
    all.yml                    # shared: app_name, ports, paths
    dev.yml                    # dev: rolling internal-registry main tag
    dev/vault.yml              # dev secrets (per-variable !vault, avpm markpost-dev)
    staging.yml                # staging: pinned Docker Hub release
    staging/vault.yml          # staging secrets (avpm markpost-staging)
    production.yml             # production: pinned release, origin TLS, CF CIDRs
    production/vault.yml       # production secrets (avpm markpost-prod)
  host_vars/
    fn.yml / oect.yml / ttyo.yml
  templates/
    docker-compose.yml.j2      # single, conditional certs mount + host port
    config.toml.j2             # single, full section set
    Caddyfile.dev              # static HTTP Caddyfile (dev)
    Caddyfile.staging          # static HTTP Caddyfile (staging)
    Caddyfile.production.j2    # Origin CA + cloudflare_cidrs (production)
```

### Commands

```bash
ansible-playbook devops/ansible/deploy.yml                    # dev (fn)
ansible-playbook devops/ansible/deploy.yml -e target=staging
ansible-playbook devops/ansible/deploy.yml -e target=production
```

### Environment variable matrix

| Variable           | dev                                      | staging                          | production                     |
| ------------------ | ---------------------------------------- | -------------------------------- | ------------------------------ |
| `image`            | `192.168.5.50:5000/markpost:main`        | `jukanntenn/markpost:<pinned>`   | `jukanntenn/markpost:<pinned>` |
| `expected_version` | `git describe` of the deploying checkout | `<pinned git tag>`               | `<pinned git tag>`             |
| `host_port`        | `8089`                                   | `8089`                           | `2053`                         |
| `tls_profile`      | `http`                                   | `http` (tunnel terminates HTTPS) | `origin` (CF Full strict)      |
| `public_url`       | `http://192.168.5.200:8089`              | `https://markpost.bytehome.fun`  | `https://markpost.cc`          |
| `debug`            | `true`                                   | `false`                          | `false`                        |
| `cloudflare_cidrs` | _(unset)_                                | _(unset)_                        | CF CIDR list                   |

staging and production pin the same version (`markpost_version` in their
group_vars): staging is the promotion gate, so the artifact validated there
must be byte-identical to what production runs.

### Post-deploy health check

The playbook's final task runs `scripts/check_deploy.py` from the controller:

1. Polls `{public_url}/api/v1/health` until `{"status": "ok"}` (5s interval,
   120s timeout) — through the path real visitors use (LAN for dev, tunnel for
   staging, Cloudflare edge for production), so a broken port/Caddy/edge path
   fails the deploy even though the container itself is healthy.
2. Compares `{public_url}/api/v1/version` against the expected version: the
   deploying checkout's `git describe` for dev (`main`-tag deploys), or the
   pinned git tag for staging/production. This catches a container that is up
   but still running the previous image.

On failure, check `docker compose logs markpost` on the host. There is no
automatic rollback — migrations have usually already run by the time the app
is up, so the recovery path is fix-forward and redeploy.

---

## 6. Operational tasks

### Reset a user's password

```bash
# On the target host, in the app dir:
docker compose exec markpost markpost -c /app/config.toml \
    reset-password -u <username>
```

### Prune expired posts (cron)

```bash
docker compose exec markpost markpost -c /app/config.toml \
    prune-expired-posts --dry-run   # check first, then omit --dry-run
```

### Prune old delivery history (cron)

```bash
docker compose exec markpost markpost -c /app/config.toml \
    prune-delivery-history
```

### Sync Cloudflare CIDRs

When Cloudflare updates their IP ranges, update two places:

1. The VPS firewall (re-run the ufw loop above).
2. `devops/ansible/group_vars/production/vars.yml` → `cloudflare_cidrs` (space-separated),
   then redeploy production.

### View logs

```bash
docker compose logs -f markpost    # Caddy + Go (s6 merges both to stdout)
```

---

## 7. Image tag semantics

| Tag          | Registry            | Points at                                              | Moved by                               |
| ------------ | ------------------- | ------------------------------------------------------ | -------------------------------------- |
| `main`       | `192.168.5.50:5000` | whatever `docker/build.py` last built from a workspace | local `python3 docker/build.py --push` |
| `X.Y.Z`      | Docker Hub          | git tag `vX.Y.Z` (stable release)                      | `docker-publish.yml` on tag push       |
| `X.Y.Z-rc.N` | Docker Hub          | git tag `vX.Y.Z-rc.N` (prerelease)                     | `docker-publish.yml` on tag push       |
| `latest`     | Docker Hub          | the newest **stable** release (never a prerelease)     | `docker-publish.yml`, stable only      |

- Git tags follow SemVer 2.0.0: `v0.1.3` stable, `v0.1.3-rc.1` prerelease
  (the hyphen is required — `v0.1.3rc1` is invalid semver and breaks every
  semver-aware tool). Docker tags strip the leading `v`.
- A release is stable iff the git tag matches `^v\d+\.\d+\.\d+$` exactly —
  one rule, shared verbatim by `docker-publish.yml` (latest gating) and
  `release.yml` (prerelease / make_latest flags).
- Docker Hub tags published before the format switch (`v0.1.0` …
  `v0.2.0-rc.3`, with the `v`) remain as-is; new releases use the no-`v` form.
- `main` never leaves the internal registry, and CI never builds it — it is
  purely the local-build rolling tag that dev (fn) tracks.
- Local multi-arch builds (`docker/build.py --push --all-platforms`) go through
  buildx + QEMU; the release workflow instead builds each platform natively
  (amd64 on x86 runners, arm64 on the GitHub arm runner) and merges manifests —
  no emulation, no registry cache.
