# MRFC: Observability wiring overhaul (three-env telemetry, kuma rework, Grafana alerting)

English | [中文](2026-10-04-observability-wiring-overhaul.zh.md)

Status: implemented

## Problem

Three findings broke the observability story the stack was designed around:

1. **Staging telemetry never arrived.** Staging was wired to the public OTLP ingress (`otlp.bytehome.fun`), but from the staging host that name resolves to the LAN reverse proxy, which has no otlp vhost (TLS fails), and the vps2 caddy `remote_ip` allowlist holds only production's IP anyway. Metrics, logs, and traces for `markpost-staging` were absent from all three stores since the stack was stood up — "staging as the promotion gate" never validated the telemetry path.
2. **The kuma push chains delivered wrong verdicts.** The vaulted push URLs carried kuma's copy-paste query suffix (`?status=up&msg=OK&ping=`) and `pgbackrest-check.py` appended a second `status`/`msg` pair with `&`. Kuma parsed the duplicate params into arrays, read the status as "not up", and recorded every genuine backup-check push as a down beat with a mangled `[object Object]` message — the backup monitors showed 0% uptime while the backup chains were healthy. Separately, the production heartbeat supervisor conf had been installed without `supervisorctl reread && update`, so the program never started and the heartbeat monitor never received a beat.
3. **No alerting beyond availability, and no environment identity.** Grafana had zero alert rules and zero contact points; the log pipeline shipped production DEBUG records (the otelslog bridge has no level filter, unlike the file handler); telemetry carried no deployment attribute — environment separation rode the `service.name` suffix, which cannot drive dashboards or alert routing cleanly.

## Decision

1. **Standard deployment attribute.** Every wired environment exports `OTEL_RESOURCE_ATTRIBUTES=deployment.environment.name=<env>` (compose template). Dashboards and alert rules slice on this label instead of parsing `service.name`.
2. **Staging goes LAN-direct.** `otel_otlp_endpoint` for staging is the collector over the LAN (`http://192.168.5.57:4318`) — the same path dev uses; production keeps the public ingress. This supersedes the staging leg of [the frp transport MRFC](2026-09-19-otlp-transport-frp-exposure.md).
3. **Bare push URL contract.** Vaulted kuma URLs are the bare push endpoint with no query suffix, in the form each producer can reach — production pushes through the public route, staging over the LAN (its egress to vps2 hangs; a public-form URL would time out silently). `heartbeat.py` appends `?query` (unchanged); `pgbackrest-check.py` now appends `?query` too (was `&`, which only worked with the suffix form and produced the duplicate-param bug).
4. **Symmetric heartbeat.** The deploy installs the supervisor program on staging and production, guarded per env on the vaulted `kuma_heartbeat_url`; staging has its own push monitor and vault variable. Both envs rehearse the identical shape.
5. **Kuma inventory standard.** Two group monitors (`markpost · production`, `markpost · staging`) parent the leaf monitors; every monitor carries self-describing tags (`env:production` / `env:staging`, `service:markpost`, `layer:edge|origin|push`); names follow `markpost · <env> · <object> (<layer>)`; push monitors set retries explicitly (heartbeat 2, daily backup checks 1).
6. **Two notification channels.** Feishu (primary) and email via the mailrise SMTP gateway (fallback), both default-enabled and applied to all monitors, matching the runbook's promise.
7. **Alert ownership boundary.** kuma owns edge/origin availability plus certificate and domain expiry; beszel owns host resources; Grafana owns application, business, and telemetry-pipeline signals — including the no-data "telemetry gap" alerts that watch the pipeline itself. Grafana-managed alerting (contact points, notification policies, rules) is provisioned as code on the observability server; the rule inventory lives in the runbook. Grafana 13.2.1 OSS ships no Feishu receiver integration, so Grafana delivers through the mailrise SMTP gateway (email contact point → apprise → the same Feishu channel); ten of the eleven rules evaluate, and the ERROR-log-rate rule is provisioned paused until the logs datasource emits SSE-compatible numeric frames.

## Alternatives considered

- **Repair staging's public path** (hosts/DNS override on the staging host plus adding the home egress IP to the vps2 caddy allowlist). Lost: two operator-owned moving parts — LAN split-horizon DNS and a dynamic home IP — would both have to stay correct forever, for a path whose producer-side code is byte-identical to LAN-direct; production alone keeps exercising the public transport around the clock.
- **Inject the deployment attribute at the collector** instead of the producer. Lost: the collector is operator-owned handoff material, and a deployment's environment is the producer's own identity — the repo owns exactly that touchpoint.
- **Recreate kuma monitors to rename them.** Lost: new push tokens orphan the vaulted URLs and force re-vaulting three secrets (plus a fourth for the new staging heartbeat). Renames plus group re-parenting keep tokens stable.
- **One dashboard copy per environment.** Lost: three copies drift silently; a single `deployment.environment.name`-driven dashboard set with multi-dimensional alert rules yields per-env series and per-env alert instances natively.
- **Alerting with vmalert inside VictoriaMetrics.** Lost: another operator-owned component to provision; Grafana-managed alerting already queries all three datasources with provisioning-file support and unified notification routing.

## Consequences

Bought: staging telemetry lands with producer code identical to production's and becomes visible in all three stores; dashboards and alerts route on a standard semconv attribute; both kuma push chains deliver real verdicts and the inventory scales by tags and groups; production DEBUG records stop leaking to the collector while the local JSONL files remain the crash channel; alerting finally exists at the application/business layer with the pipeline watching itself.

Cost: four vault push URLs must be rotated if leaked (bare form, same secrecy, simpler contract); staging no longer rehearses the public transport — the trade is deliberate, since that transport is operator infrastructure and production exercises it continuously; staging gains one supervisor program; the Grafana provisioning files on the observability server are handoff material that must stay in sync with the runbook's rule inventory; Grafana alerting reaches Feishu only through mailrise (one indirection, and it stays silent until the SMTP password is filled in), and the ERROR-log-rate rule waits paused on an upstream type-compatibility fix.

Verification: `markpost-staging` appears in VictoriaMetrics, VictoriaLogs, and Jaeger with `deployment.environment.name=staging`; kuma shows the two groups with green children (heartbeat beats within minutes of the deploy, backup verdicts at the daily check); dropping one environment's telemetry for ten minutes fires the telemetry-gap alert to Feishu; production logs in VictoriaLogs contain no DEBUG records.
