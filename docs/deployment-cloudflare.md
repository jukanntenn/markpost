# Cloudflare Production Deployment

English | [中文](deployment-cloudflare.zh.md)

Operating guide for the SaaS production instance (`markpost.cc`): a VPS origin behind Cloudflare (Free plan), deployed with the Ansible playbook. Design decisions — SSL mode, Origin CA, ports, client-IP relay, cache and purge contracts — live in [the Cloudflare spec](../specs/backend/cloudflare.md); the frozen multi-environment reference (acceptance, dev, staging, operational tasks) is [`deployment.md`](deployment.md).

## Topology

```
Visitor ──HTTPS:443──> Cloudflare edge (proxied, Full strict) ──HTTPS──> VPS :2053 ──> Caddy ──> Go :7330 / static export
```

| Plane  | Value                                                                                                                          |
| ------ | ------------------------------------------------------------------------------------------------------------------------------ |
| Edge   | port 443, TLS terminated by Cloudflare                                                                                         |
| Origin | host port 2053 → container 2053, Caddy with the Origin CA certificate; an Origin Rule rewrites the destination port 443 → 2053 |
| App    | Go `127.0.0.1:7330`, static export from `/app/frontend`, Postgres over the shared Unix-socket volume                           |

## One-time: Cloudflare dashboard

1. **DNS** — DNS → Records: add an A record `markpost.cc` → origin IP, set to **Proxied** (orange cloud).
2. **SSL mode** — SSL/TLS → Overview: select **Full (strict)**.
3. **Origin CA certificate** — SSL/TLS → Origin Server → Create Certificate: key type RSA (2048), hostnames `markpost.cc, *.markpost.cc`, validity 15 years. Copy the certificate to `origin.pem` and the private key to `origin.key` (shown once).
4. **Origin Rule** — Rules → Overview → Create rule → Origin Rule: rule name `origin-port-2053`; When incoming requests match: `Hostname` `equals` `markpost.cc`; Set origin parameters → **Destination port** → `Rewrite to` `2053`; **Deploy**.

## One-time: origin VPS

1. **Certificate** — place `origin.pem` and `origin.key` under `~/docker/markpost/certs/` (out-of-band transfer; the key readable by its owner only).
2. **Firewall** — default-deny inbound; keep SSH reachable for administration; allow TCP 2053 only from Cloudflare's published ranges, so the origin is reachable only through the edge.
3. **`cloudflare_cidrs`** — put the same ranges, space-separated, into `devops/ansible/group_vars/production/vars.yml`. Fetch the current list with:

```bash
curl -s https://www.cloudflare.com/ips | tr '\n' ' '
```

## Secrets

Per-variable `!vault` entries in `devops/ansible/group_vars/production/vault.yml` (avpm identity `markpost-prod` from the root `ansible.cfg`; keyring unlocked via `avpm unlock`). Required keys: `jwt_access_signing_key`, `jwt_refresh_signing_key`, `admin_password`, `db_password`, `github_client_id`, `github_client_secret`, `cloudflare_api_token`. Generate each with:

```bash
printf '%s' '<secret>' | ansible-vault encrypt_string \
    --vault-id markpost-prod@~/.local/bin/avpm-client \
    --stdin-name <name> >> devops/ansible/group_vars/production/vault.yml
```

**Check an existing value** (from the repo root; `ansible.cfg` supplies the vault identity):

```bash
ansible localhost -m debug -a "var=<name>" \
    -e @devops/ansible/group_vars/production/vault.yml
```

**Change a value** — delete the variable's `!vault` block in `vault.yml`, then re-run the `encrypt_string` command above with the new value.

**GitHub OAuth app** → `github_client_id` / `github_client_secret`: GitHub → Settings → Developer settings → OAuth Apps → **New OAuth App**. Homepage `https://markpost.cc`, callback URL `https://markpost.cc/auth/callback`. Copy the Client ID and generate a Client Secret.

**Cloudflare API token** → `cloudflare_api_token`: dashboard → My Profile → API Tokens → **Create Token** → template **Zone Cache Purge**, zone scope `markpost.cc` (drives cache purge on post deletion; the zone id sits in `vars.yml`).

## Deploy

Production runs a pinned Docker Hub release (`docker-publish.yml` on `v*` tags). Promote after staging has validated the same version:

1. In `devops/ansible/group_vars/production/vars.yml`, set `markpost_version` to the Docker Hub tag (no leading `v`) and `expected_version` to the matching `v`-prefixed git tag.
2. From the repo root:

```bash
ansible-playbook devops/ansible/deploy.yml -e target=production
```

The playbook renders compose + `config.toml` + Caddyfile, pulls the pinned image, starts postgres → migrate → app, and verifies health + version through the edge.

## Verify

```bash
curl -f https://markpost.cc/api/v1/health
curl -f https://markpost.cc/api/v1/version        # == expected_version
curl -sI https://markpost.cc/<qid> | grep -i cf-cache-status   # MISS, then HIT on repeat
```

`CF-Cache-Status` values: [spec reference](../specs/backend/cloudflare.md#cf-cache-status-reference). On failure: `docker compose logs markpost` on the host; recovery is fix-forward — migrations have usually run by the time the app is up.

## Recurring: Cloudflare CIDR resync

When https://www.cloudflare.com/ips/ changes, update both homes of the list and redeploy: refresh the host firewall's Cloudflare rules on the VPS, and re-run the fetch command above into `cloudflare_cidrs` in `devops/ansible/group_vars/production/vars.yml`.
