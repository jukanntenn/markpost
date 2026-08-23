# Cloudflare Production Deployment

English | [中文](deployment-cloudflare.zh.md)

Operating guide for the SaaS production instance (`markpost.cc`): a VPS origin behind Cloudflare (Free plan), deployed with the Ansible playbook. Design decisions — SSL mode, Origin CA, ports, client-IP relay, cache and purge contracts — live in [the Cloudflare spec](../specs/backend/cloudflare.md); the frozen multi-environment reference (acceptance, dev, staging, operational tasks) is [`deployment.md`](deployment.md).

## Topology

```
Visitor ──HTTPS:443──> Cloudflare edge (proxied, Full strict) ──HTTPS:443──> host Caddy gateway (Origin CA) ──HTTP──> 127.0.0.1:8080 → container Caddy :2053 ──> Go :7330 / static export
```

| Plane  | Value                                                                                                                                                             |
| ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Edge   | port 443, TLS terminated by Cloudflare                                                                                                                            |
| Origin | host port 443 → the host's systemd Caddy gateway (shared across the VPS's services), TLS with the Origin CA certificate; proxies `markpost.cc` → `127.0.0.1:8080` |
| App    | container Caddy plain HTTP `:2053`, published loopback-only; Go `127.0.0.1:7330`, static export from `/app/frontend`, Postgres over the shared Unix-socket volume |

443 is the only proxied HTTPS port with edge caching enabled — every other proxied HTTPS port (2053, 8443, …) is cache-disabled, and re-enabling is Enterprise-only. Services that need no edge cache can stay on those ports.

## One-time: Cloudflare dashboard

1. **DNS** — DNS → Records: add an A record `markpost.cc` → origin IP, set to **Proxied** (orange cloud).
2. **SSL mode** — SSL/TLS → Overview: select **Full (strict)**.
3. **Origin CA certificate** — SSL/TLS → Origin Server → Create Certificate: key type RSA (2048), hostnames `markpost.cc, *.markpost.cc`, validity 15 years. Copy the certificate to `origin.pem` and the private key to `origin.key` (shown once).
4. **Cache Rule for post pages** — Rules → Overview → Create rule → Cache Rule: rule name `cache-post-pages`; When incoming requests match: `URI Path` `starts with` `/p-`; Then → Cache eligibility → **Eligible for cache**; leave the rule's **Edge TTL** unset so the origin's `s-maxage` governs (an explicit Edge TTL overrides it, and the Free minimum is 2 h); **Deploy**. Without this rule post pages stay `DYNAMIC` — HTML is not default-cached.
5. **Browser Cache TTL** — Caching → Configuration: select **Respect Existing Headers**, so the origin's `max-age` reaches browsers unchanged.

## One-time: origin VPS

1. **Certificate** — place `origin.pem` and `origin.key` under `~/docker/markpost/certs/` (out-of-band transfer; the key readable by its owner only). The deploy playbook copies them to `/etc/caddy/certs/markpost/` for the gateway.
2. **Host Caddy gateway** — the host runs a systemd Caddy binding 443 with `import /etc/caddy/conf.d/*.caddy`; the deploy playbook templates the markpost site block (Origin CA TLS → reverse proxy to the loopback port) and reloads it on change. Other services join the same gateway by dropping their own site block there.
3. **Firewall** — default-deny inbound; keep SSH reachable for administration; allow TCP 443 only from Cloudflare's published ranges, so the origin is reachable only through the edge.
4. **`cloudflare_cidrs`** — put the same ranges, space-separated, into `devops/ansible/group_vars/production/vars.yml`. Fetch the current list with:

```bash
{ curl -s https://www.cloudflare.com/ips-v4; echo; curl -s https://www.cloudflare.com/ips-v6; } | tr '\n' ' '
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
2. From the repo root (the gateway tasks on the VPS need root — supply the sudo password):

```bash
ansible-playbook devops/ansible/deploy.yml -e target=production --ask-become-pass
```

The playbook renders compose + `config.toml` + Caddyfile, stages the host Caddy gateway site block (reloading it on change), pulls the pinned image, starts postgres → migrate → app, and verifies health + version through the edge.

## Verify

```bash
curl -f https://markpost.cc/api/v1/health
curl -f https://markpost.cc/api/v1/version        # == expected_version
curl -sD- -o /dev/null https://markpost.cc/<qid> | grep -i cf-cache-status   # MISS, then HIT on repeat (GET — HEAD is never cached)
```

`CF-Cache-Status` values: [spec reference](../specs/backend/cloudflare.md#cf-cache-status-reference). If everything shows `DYNAMIC` — including `/_next/static/*` assets, which are default-cached without any rule — the zone suspended caching: check **Development Mode** (Caching → Configuration; auto-expires after 3 h) and that the Cache Rule is **Deployed**, not a draft. On failure: `docker compose logs markpost` on the host; recovery is fix-forward — migrations have usually run by the time the app is up.

## Recurring: Cloudflare CIDR resync

When https://www.cloudflare.com/ips/ changes, refresh the host firewall's Cloudflare rules on the VPS (the only enforcement point), and re-run the fetch command above into `cloudflare_cidrs` in `devops/ansible/group_vars/production/vars.yml` — the documented home of the list.
