# Availability Monitoring

English | [中文](monitoring.zh.md)

markpost's availability is monitored by a self-hosted [uptime-kuma](https://github.com/louislam/uptime-kuma) instance probing production and staging from outside, plus a reverse heartbeat from the production host; a Beszel agent on the origin reports host and container metrics to an off-site hub. Alerts go to Feishu (primary) and email (fallback). This runbook owns the monitor inventory, notification setup, alert policy, and the heartbeat's deploy/remove procedures.

<a id="probe-model"></a>

## Probe model

A single URL cannot watch a CDN-fronted origin: edge-cached pages stay green while the origin is down, and an origin-only probe cannot see the edge path users actually traverse. The monitor set therefore spans three vantage points:

| Vantage                                 | What it sees                                                                                                                      | Carried by                             |
| --------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| Edge path (kuma → public URL)           | The full user path: DNS, Cloudflare, gateway, container, static export                                                            | Homepage monitors                      |
| Origin probe (kuma → uncached endpoint) | The Go process and, via `/api/v1/ready`, the database — `/api/v1/*` carries `no-store`, so these requests always reach the origin | Readiness monitors                     |
| Reverse heartbeat (VPS → kuma)          | The origin's own local verdict, bypassing Cloudflare entirely; silence means host death                                           | Push monitor + supervisor loop on ttyo |

Endpoint semantics (`/health` liveness vs `/ready` readiness) live in [`api-schema.md`](../specs/backend/api-schema.md); both are exempt from rate limiting ([`rate-limiting.md`](../specs/backend/rate-limiting.md)). The CDN behavior that motivates the layering is specified in [`caching.md`](../specs/backend/caching.md) and [`cloudflare.md`](../specs/backend/cloudflare.md).

<a id="monitors"></a>

## Monitors

Common settings for every monitor: Heartbeat Interval `60`, Retries `3`, Retry Interval `60`, both notification channels attached, no maintenance windows (the retry threshold absorbs deploy restarts; a window would silence real faults).

| Monitor                 | Type                 | Target / key fields                                                                                       |
| ----------------------- | -------------------- | --------------------------------------------------------------------------------------------------------- |
| prod · user path        | HTTP(s)              | URL `https://markpost.cc/`, accepted status 200; enable certificate-expiry notification                   |
| prod · origin readiness | HTTP(s) - Json Query | URL `https://markpost.cc/api/v1/ready`; Json Query `status`, operator `==`, expected value `ready`        |
| prod · heartbeat        | Push                 | Interval `120`, Retries `2`; push URL is a secret held in the ansible vault (see [Heartbeat](#heartbeat)) |
| prod · domain           | toggle               | On the prod · user path monitor, enable domain-expiry notification for `markpost.cc`                      |
| stg · user path         | HTTP(s)              | URL `https://markpost.bytehome.fun/`, accepted status 200                                                 |
| stg · origin readiness  | HTTP(s) - Json Query | URL `http://192.168.5.50:8089/api/v1/ready`; Json Query `status` == `ready`                               |

Certificate- and domain-expiry notifications fire at kuma's global thresholds (default remaining days 7/14/21). The stg · origin readiness monitor probes the LAN address directly, so an entry failure and an instance failure are distinguishable.

<a id="notification-channels"></a>

## Notification channels

| Channel          | kuma setup                                                                      |
| ---------------- | ------------------------------------------------------------------------------- |
| Feishu (primary) | Notification Type `Feishu`; Webhook URL = a Feishu group-bot webhook            |
| Email (fallback) | Notification Type `Email (SMTP)`; SMTP host/port/credentials, sender, recipient |

Add both, then set them as default notifications (Settings → Notifications → apply as default) so every monitor alerts through both channels.

<a id="alert-policy"></a>

## Alert policy

- A monitor pages only after ~4 minutes of consecutive failure (interval 60 s × retries 3); transient edge jitter and deploy-time container swaps stay silent.
- Recovery notifications are on: the down→up transition always notifies.
- Repeat reminders are off (single-operator service; the recovery notice closes the loop).
- Certificate and domain expiry notify at 7/14/21 remaining days.

<a id="heartbeat"></a>

## Heartbeat (production)

On the production VPS, a supervisor program `markpost-heartbeat` runs the static [`heartbeat.py`](../devops/ansible/files/heartbeat.py) installed at `~/docker/markpost/heartbeat.py`: every 60 s it probes `http://127.0.0.1:8080/api/v1/ready` and pushes the verdict to kuma's push endpoint. The probe URL and interval ride the command line; the secret push URL reaches the script through the supervisor program's `environment=` (from the vault variable — hence the conf's 0600 mode). kuma marks the monitor down when a `down` verdict arrives (app-level failure, including database trouble) or when pushes stop (host death). The log is `~/docker/markpost/data/heartbeat.log`.

The deploy tasks in [`deploy.yml`](../devops/ansible/deploy.yml) install the script and program only when the vault variable `kuma_heartbeat_url` is defined, so the setup order is:

1. In kuma, add the prod · heartbeat monitor (Push, interval 120, retries 2) and copy its push URL — anyone holding it can forge up-beats, so treat it as a secret.
2. Vault it: `ansible-vault encrypt_string '<push-url>' --name kuma_heartbeat_url >> devops/ansible/group_vars/production/vault.yml`
3. Deploy: `ansible-playbook devops/ansible/deploy.yml -e target=production` — the handler runs `supervisorctl reread && update` and starts the program.
4. Verify: `sudo supervisorctl status markpost-heartbeat` shows RUNNING and kuma receives beats.

Removal is manual (the deploy never uninstalls): delete `/etc/supervisor/conf.d/markpost-heartbeat.conf`, then `sudo supervisorctl reread && sudo supervisorctl update`, and delete the vault variable.

<a id="host-metrics"></a>

## Host metrics (Beszel)

The monitors above answer _whether_; the Beszel agent answers _why_ — host and per-container resource history with threshold alerts, reported to a hub on a separate ops-managed server. Design record: [host-metrics MRFC](../.agents/mrfcs/implemented/2026-08-31-host-metrics-monitoring-beszel.md) and [topology MRFC](../.agents/mrfcs/implemented/2026-08-31-beszel-deployment-topology.md).

**Agent (repo-automated, production only).** A one-service compose project at `~/docker/beszel-agent`, rendered from [`beszel-agent-compose.yml.j2`](../devops/ansible/templates/beszel-agent-compose.yml.j2): pinned `henrygd/beszel-agent`, host networking, read-only `docker.sock`. It collects host CPU/memory/disk/load/network plus per-container stats for `markpost` and `markpost-postgres`, and reaches the hub by outbound WebSocket only — the firewall opens nothing.

**Hub (ops-owned, outside this repo).** Deploy per [beszel.dev](https://beszel.dev) (docker compose, embedded SQLite) on its own server behind TLS. Add a system for ttyo there and copy the public key it shows.

**Alerts.** Dual warning/critical thresholds on the hub; suggested starting points, tuned after a week of curves:

| Metric                     | warning / critical |
| -------------------------- | ------------------ |
| Disk usage                 | 80% / 90%          |
| Memory                     | 80% / 92%          |
| CPU                        | 85% / 95%          |
| System status (agent down) | — / any            |

Notifications go to the same Feishu channel as kuma (shoutrrr `lark://` URL configured on the hub).

**Setup order.**

1. Ops: bring the hub up on its server, add a system, copy the public key.
2. Set `beszel_hub_url` (the hub's `/beszel/agent` WebSocket URL) and `beszel_agent_key` (the public key — not a secret) in `devops/ansible/group_vars/production/vars.yml`.
3. `ansible-playbook devops/ansible/deploy.yml -e target=production` — the deploy installs the agent only once both vars exist.
4. Verify: `docker compose -f ~/docker/beszel-agent/docker-compose.yml ps` shows the agent running, and the hub's system page shows live data.

Removal is manual (the deploy never uninstalls): `docker compose -f ~/docker/beszel-agent/docker-compose.yml down`, delete `~/docker/beszel-agent`, delete the two vars.

<a id="alert-triage"></a>

## Alert triage

| Red monitors                           | Meaning                                          | First action                                                                      |
| -------------------------------------- | ------------------------------------------------ | --------------------------------------------------------------------------------- |
| user path + readiness + heartbeat      | Total origin outage                              | SSH to ttyo; `docker compose ps`, container logs                                  |
| user path + readiness, heartbeat green | Cloudflare / edge path failure; origin alive     | Cloudflare dashboard; origin is fine                                              |
| user path only                         | Static export or CDN cache issue                 | Compare `/` vs `/api/v1/ready` by curl; check the release                         |
| readiness (503) + heartbeat `down`     | Database failure                                 | `docker compose logs postgres`; disk space                                        |
| heartbeat only                         | Heartbeat loop, supervisor, or kuma reachability | `sudo supervisorctl status markpost-heartbeat`; heartbeat log                     |
| Beszel agent offline (via Feishu)      | ttyo alive but agent/route/hub trouble           | `docker compose -f ~/docker/beszel-agent/docker-compose.yml ps`; hub reachability |
