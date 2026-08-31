# Availability Monitoring

English | [中文](monitoring.zh.md)

markpost's availability is monitored by a self-hosted [uptime-kuma](https://github.com/louislam/uptime-kuma) instance probing production and staging from outside, plus a reverse heartbeat from the production host. Alerts go to Feishu (primary) and email (fallback). This runbook owns the monitor inventory, notification setup, alert policy, and the heartbeat's deploy/remove procedures.

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

On the production VPS, a supervisor program `markpost-heartbeat` runs [`heartbeat.py.j2`](../devops/ansible/templates/heartbeat.py.j2) (rendered to `~/docker/markpost/heartbeat.py`): every 60 s it probes `http://127.0.0.1:8080/api/v1/ready` and pushes the verdict to kuma's push endpoint. kuma marks the monitor down when a `down` verdict arrives (app-level failure, including database trouble) or when pushes stop (host death). The log is `~/docker/markpost/data/heartbeat.log`.

The deploy tasks in [`deploy.yml`](../devops/ansible/deploy.yml) install the script and program only when the vault variable `kuma_heartbeat_url` is defined, so the setup order is:

1. In kuma, add the prod · heartbeat monitor (Push, interval 120, retries 2) and copy its push URL — anyone holding it can forge up-beats, so treat it as a secret.
2. Vault it: `ansible-vault encrypt_string '<push-url>' --name kuma_heartbeat_url >> devops/ansible/group_vars/production/vault.yml`
3. Deploy: `ansible-playbook devops/ansible/deploy.yml -e target=production` — the handler runs `supervisorctl reread && update` and starts the program.
4. Verify: `sudo supervisorctl status markpost-heartbeat` shows RUNNING and kuma receives beats.

Removal is manual (the deploy never uninstalls): delete `/etc/supervisor/conf.d/markpost-heartbeat.conf`, then `sudo supervisorctl reread && sudo supervisorctl update`, and delete the vault variable.

<a id="alert-triage"></a>

## Alert triage

| Red monitors                           | Meaning                                          | First action                                                  |
| -------------------------------------- | ------------------------------------------------ | ------------------------------------------------------------- |
| user path + readiness + heartbeat      | Total origin outage                              | SSH to ttyo; `docker compose ps`, container logs              |
| user path + readiness, heartbeat green | Cloudflare / edge path failure; origin alive     | Cloudflare dashboard; origin is fine                          |
| user path only                         | Static export or CDN cache issue                 | Compare `/` vs `/api/v1/ready` by curl; check the release     |
| readiness (503) + heartbeat `down`     | Database failure                                 | `docker compose logs postgres`; disk space                    |
| heartbeat only                         | Heartbeat loop, supervisor, or kuma reachability | `sudo supervisorctl status markpost-heartbeat`; heartbeat log |
