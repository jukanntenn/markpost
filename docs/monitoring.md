# Availability Monitoring

English | [中文](monitoring.zh.md)

markpost's availability is monitored by a self-hosted [uptime-kuma](https://github.com/louislam/uptime-kuma) instance probing production and staging from outside, plus a reverse heartbeat from each app host; a Beszel agent on the origin reports host and container metrics to an off-site hub; Grafana on the observability server owns application, business, and telemetry-pipeline alerts. Alerts go to Feishu (primary) and email via the mailrise gateway (fallback). This runbook owns the monitor inventory, naming standard, notification setup, alert ownership boundaries, and the heartbeat's deploy/remove procedures.

<a id="probe-model"></a>

## Probe model

A single URL cannot watch a CDN-fronted origin: edge-cached pages stay green while the origin is down, and an origin-only probe cannot see the edge path users actually traverse. The monitor set therefore spans three vantage points:

| Vantage                                 | What it sees                                                                                                                      | Carried by                             |
| --------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| Edge path (kuma → public URL)           | The full user path: DNS, Cloudflare, gateway, container, static export                                                            | Homepage monitors                      |
| Origin probe (kuma → uncached endpoint) | The Go process and, via `/api/v1/ready`, the database — `/api/v1/*` carries `no-store`, so these requests always reach the origin | Readiness monitors                     |
| Reverse heartbeat (VPS → kuma)          | The app host's own local verdict, bypassing Cloudflare entirely; silence means host death                                         | Push monitor + supervisor loop per env |

Endpoint semantics (`/health` liveness vs `/ready` readiness) live in [`api-schema.md`](../specs/backend/api-schema.md); both are exempt from rate limiting ([`rate-limiting.md`](../specs/backend/rate-limiting.md)). The CDN behavior that motivates the layering is specified in [`caching.md`](../specs/backend/caching.md) and [`cloudflare.md`](../specs/backend/cloudflare.md).

<a id="monitors"></a>

## Monitors

**Naming standard** (MRFC 2026-10-04-observability-wiring-overhaul): monitors are grouped under two kuma group monitors, `markpost · production` and `markpost · staging`, so staging and production carry the identical shape and a promotion rehearses everything. Leaf monitors follow `markpost · <env> · <object> (<layer>)` and carry three tags: `env:production` / `env:staging`, `service:markpost`, and `layer:edge|origin|push`. The sidebar's Tags filter and the per-group layout both key off this.

Common settings for every monitor: Heartbeat Interval `60`, Retries `3`, Retry Interval `60`, both notification channels attached, no maintenance windows (the retry threshold absorbs deploy restarts; a window would silence real faults). Push monitors set retries explicitly: heartbeats `2`, daily backup checks `1` (a single missed daily push must not page).

| Group                 | Monitor                                 | Type                 | Target / key fields                                                                                                   |
| --------------------- | --------------------------------------- | -------------------- | --------------------------------------------------------------------------------------------------------------------- |
| markpost · production | markpost · prod · homepage (edge)       | HTTP(s)              | URL `https://markpost.cc/`, accepted status 200; enable certificate- and domain-expiry notification for `markpost.cc` |
| markpost · production | markpost · prod · origin readiness      | HTTP(s) - Json Query | URL `https://markpost.cc/api/v1/ready`; Json Query `status`, operator `==`, expected value `ready`                    |
| markpost · production | markpost · prod · host heartbeat (push) | Push                 | Interval `120`, Retries `2`; push URL is a secret held in the ansible vault (see [Heartbeat](#heartbeat))             |
| markpost · production | markpost · prod · db backup (push)      | Push                 | Interval `86400`, Retries `1`; fed by the daily backup check (see [docs/backup.md](backup.md))                        |
| markpost · staging    | markpost · stg · homepage (edge)        | HTTP(s)              | URL `https://markpost.bytehome.fun/`, accepted status 200                                                             |
| markpost · staging    | markpost · stg · origin readiness       | HTTP(s) - Json Query | URL `http://192.168.5.50:8089/api/v1/ready`; Json Query `status` == `ready`                                           |
| markpost · staging    | markpost · stg · host heartbeat (push)  | Push                 | Interval `120`, Retries `2` — symmetric with production                                                               |
| markpost · staging    | markpost · stg · db backup (push)       | Push                 | Interval `86400`, Retries `1`                                                                                         |

**Push URL contract**: vaulted push URLs are the **bare** endpoint with no query suffix, in the form each producer can reach: production (on vps1) pushes through the public route (`https://uptime-kuma.bytehome.fun/api/push/<token>`), while the staging host pushes to kuma over the LAN (`http://192.168.5.50:3001/api/push/<token>`) — its egress to vps2 hangs (the home IP is not accepted there), so the public form would silently time out. The script-side contract is identical everywhere: `heartbeat.py` appends `?status=…`; `pgbackrest-check.py` appends `?status=…` too. A second `?status=…` suffix (the form kuma's UI shows for copy-paste) produces duplicate parameters and kuma reads the status as "not up" — every genuine push would mark the monitor down.

Certificate- and domain-expiry notifications fire at kuma's global thresholds (default remaining days 7/14/21). The stg · origin readiness monitor probes the LAN address directly, so an entry failure and an instance failure are distinguishable.

<a id="notification-channels"></a>

## Notification channels

| Channel          | kuma setup                                                                                                                                            |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Feishu (primary) | Notification Type `Feishu`; Webhook URL = a Feishu group-bot webhook                                                                                  |
| Email (fallback) | Notification Type `Email (SMTP)`; host `mailrise.bytehome.fun`, port `465`, TLS on, user per vault/mailrise config, To = the mailrise apprise address |

Add both, then set them as default notifications (Settings → Notifications → apply as default) so every monitor alerts through both channels. The mailrise gateway converts each mail into an apprise notification — its recipient address encodes the target; verify it after changing mailrise's mapping.

<a id="alert-ownership"></a>

## Alert ownership

One fault, one owner — the three systems must not double-page:

| Layer                      | Owner   | Signals                                                          |
| -------------------------- | ------- | ---------------------------------------------------------------- |
| Edge / origin availability | kuma    | Homepage, origin readiness, heartbeat, certificate/domain expiry |
| Host resources             | beszel  | Disk, memory, CPU, agent down                                    |
| App / business / pipeline  | Grafana | RED signals, delivery backlog, error-log rate, telemetry no-data |

Grafana-managed alerting (contact points, notification policies, rules) is provisioned as code on the observability server (`~/docker/grafana/provisioning/alerting/`), routes by `alertname` + `deployment.environment.name`, and covers staging and production. Rule inventory (thresholds are starting points; tune after two weeks of baselines):

| Rule                                             | Condition                                               | For    | Severity                     |
| ------------------------------------------------ | ------------------------------------------------------- | ------ | ---------------------------- |
| HTTP 5xx ratio above 5% / 20%                    | 5xx share of request rate (5m)                          | 5m/2m  | warning/critical             |
| HTTP p95 latency above 1s                        | `histogram_quantile(0.95, …)`                           | 10m    | warning                      |
| Request rate collapsed                           | 10m rate < 20% of the hour-ago baseline                 | 10m    | critical                     |
| Delivery backlog above 100 / growing 50+ per 15m | `markpost.delivery.pending`                             | 10m/0m | warning (leading indicators) |
| Delivery / CDN purge failures                    | failure-counter rate > 0                                | 15m    | warning                      |
| Telemetry gap (per env)                          | no runtime gauges for 10m — the pipeline itself is down | 0m     | critical                     |

The ERROR-log-rate rule is provisioned **paused**: the logs datasource emits integer frames that Grafana 13.2.1's alert expressions reject; re-enable it once the plugin ships numeric frames (the Runtime dashboard keeps ERROR rates visible meanwhile).

<a id="alert-policy"></a>

## Alert policy (kuma)

- A monitor pages only after ~4 minutes of consecutive failure (interval 60 s × retries 3); transient edge jitter and deploy-time container swaps stay silent.
- Recovery notifications are on: the down→up transition always notifies.
- Repeat reminders are off (single-operator service; the recovery notice closes the loop).
- Certificate and domain expiry notify at 7/14/21 remaining days.

<a id="heartbeat"></a>

## Heartbeat (staging + production)

On each app host, a supervisor program `markpost-heartbeat` runs the static [`heartbeat.py`](../devops/ansible/files/heartbeat.py) installed at `<app_path>/heartbeat.py`: every 60 s it probes `http://127.0.0.1:<host_port>/api/v1/ready` and pushes the verdict to kuma's push endpoint. The probe URL and interval ride the command line; the secret push URL reaches the script through the supervisor program's `environment=` (from the vault variable — hence the conf's 0600 mode). kuma marks the monitor down when a `down` verdict arrives (app-level failure, including database trouble) or when pushes stop (host death). The log is `<app_path>/data/heartbeat.log`. The wiring is symmetric by design — staging rehearses exactly what production runs (MRFC 2026-10-04-observability-wiring-overhaul).

The deploy tasks in [`deploy.yml`](../devops/ansible/deploy.yml) install the script and program only when the env's vault variable `kuma_heartbeat_url` is defined, so the setup order is:

1. In kuma, add the env's heartbeat monitor (Push, interval 120, retries 2) under the env's group and copy its bare push URL — anyone holding it can forge up-beats, so treat it as a secret.
2. Vault it: `python3 scripts/vault.py set <env> kuma_heartbeat_url` (paste the push URL at the hidden prompt)
3. Deploy: `ansible-playbook devops/ansible/deploy.yml -e target=<env> --ask-become-pass` — the handler runs `supervisorctl reread && update` and starts the program.
4. Verify: `sudo supervisorctl status markpost-heartbeat` shows RUNNING and kuma receives beats.

Removal is manual (the deploy never uninstalls): delete `/etc/supervisor/conf.d/markpost-heartbeat.conf`, then `sudo supervisorctl reread && sudo supervisorctl update`, and delete the vault variable.

<a id="host-metrics"></a>

## Host metrics (Beszel)

The monitors above answer _whether_; the Beszel agent answers _why_ — host and per-container resource history with threshold alerts, reported to a self-hosted hub on the observability server. Design record: [host-metrics MRFC](../.agents/rfcs/implemented/process/2026-08-31-host-metrics-monitoring-beszel.md), [topology MRFC](../.agents/rfcs/implemented/process/2026-08-31-beszel-deployment-topology.md), and the WebSocket wiring in [the agent-token MRFC](../.agents/rfcs/implemented/process/2026-10-02-beszel-agent-websocket-token.md).

**Hub (own lifecycle, on the observability server).** Deployed at `~/docker/beszel` on 192.168.5.57 (`beszel.bytehome.fun` at the edge): pinned `henrygd/beszel`, bind-mounted `beszel_data` (upstream embeds PocketBase and has no external-database support), `DISABLE_SSH=true` — pure WebSocket topology where agents dial the hub and it never dials them. Back up the whole `beszel_data` dir (it also holds the hub keypair the agents' `KEY` verifies).

**Agent (repo-automated, production only).** A one-service compose project at `~/docker/beszel-agent`, rendered from [`beszel-agent-compose.yml.j2`](../devops/ansible/templates/beszel-agent-compose.yml.j2): pinned `henrygd/beszel-agent`, host networking, read-only `docker.sock`. It collects host CPU/memory/disk/load/network plus per-container stats for `markpost` and `markpost-postgres`, and reaches the hub by outbound WebSocket (`HUB_URL` + vaulted `TOKEN`; `KEY` verifies the hub) — the firewall opens nothing.

**Alerts.** Thresholds on the hub; suggested starting points, tuned after a week of curves (beszel holds one threshold per metric, so the critical tier is used):

| Metric                     | threshold (sustained) |
| -------------------------- | --------------------- |
| Disk usage                 | 90% for 2 min         |
| Memory                     | 92% for 2 min         |
| CPU                        | 95% for 2 min         |
| System status (agent down) | down for 1 min        |

Notifications go to the same Feishu channel as kuma (shoutrrr `lark://` URL configured on the hub).

**Setup order.**

1. Hub: add a system for the host, and take the public key and the per-system token its Add-System dialog shows.
2. Set `beszel_hub_url` and `beszel_agent_key` (the public key — not a secret) in `devops/ansible/group_vars/production/vars.yml`, and vault the token as `beszel_agent_token` in that env's `vault.yml`.
3. `ansible-playbook devops/ansible/deploy.yml -e target=production` — the deploy installs the agent only once both vars exist.
4. Verify: `docker compose -f ~/docker/beszel-agent/docker-compose.yml ps` shows the agent running, and the hub's system page shows live data.

Removal is manual (the deploy never uninstalls): `docker compose -f ~/docker/beszel-agent/docker-compose.yml down`, delete `~/docker/beszel-agent`, delete the vars.

<a id="alert-triage"></a>

## Alert triage

| Red monitors                           | Meaning                                          | First action                                                                           |
| -------------------------------------- | ------------------------------------------------ | -------------------------------------------------------------------------------------- |
| user path + readiness + heartbeat      | Total origin outage                              | SSH to the app host; `docker compose ps`, container logs                               |
| user path + readiness, heartbeat green | Cloudflare / edge path failure; origin alive     | Cloudflare dashboard; origin is fine                                                   |
| user path only                         | Static export or CDN cache issue                 | Compare `/` vs `/api/v1/ready` by curl; check the release                              |
| readiness (503) + heartbeat `down`     | Database failure                                 | `docker compose logs postgres`; disk space                                             |
| heartbeat only                         | Heartbeat loop, supervisor, or kuma reachability | `sudo supervisorctl status markpost-heartbeat`; heartbeat log                          |
| telemetry gap (Grafana, per env)       | App alive but telemetry path broken              | Collector reachability from the env; `docker logs otelcol` on the observability server |
| Beszel agent offline (via Feishu)      | ttyo alive but agent/route/hub trouble           | `docker compose -f ~/docker/beszel-agent/docker-compose.yml ps`; hub reachability      |
