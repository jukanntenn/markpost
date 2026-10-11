# RFC: Rebuilding markpost's Grafana tenant assets on the platform contract

Status: implemented

English | [中文](2026-10-11-grafana-tenant-rebuild.zh.md)

## Problem

The Grafana platform on the observability NAS — the store-and-query half of [the OTLP observability stack](2026-09-19-otlp-observability-stack.md) — was handed to the external devops ops repo, which rebuilds it from scratch: compose, `.env`, datasources, and policies are re-created, the data volume is wiped, and API keys are re-issued. Everything markpost had on the old instance — the handoff-material provisioning files under `~/docker/grafana/provisioning/` documented as an obligation of [the observability wiring overhaul](2026-10-04-observability-wiring-overhaul.md) — dies with the old volume. Tenants re-onboard under a new file-provisioning contract, and markpost needs its tenant side in-repo: dashboards and alerting written against the contract, an auditable push, and post-deploy proof that the platform actually loaded them. The rewrite also has to absorb two facts that postdate the old files: [the metrics wiring change](../architecture/2026-10-11-metrics-wiring-runtime-source.md) replaced the `process.runtime.go.*` family with the `go.*` conventions (any ported runtime panels would show nothing), and dev now ships telemetry, so the pipeline-down detector must cover it too.

## Decision

markpost's tenant assets live in this repo under `devops/ansible/files/grafana/` and are pushed by [`devops/ansible/grafana-tenant.yml`](../../../../devops/ansible/grafana-tenant.yml); the platform (compose, datasources, root policies, the Grafana data volume) stays owned by the devops repo. The runbook's [alert-ownership section](../../../../docs/monitoring.md) carries the current state.

### The tenant onboarding contract as markpost implements it

A four-file provisioning set under `devops/ansible/files/grafana/provisioning/`:

- **`dashboards/markpost.yml`** — the dashboard provider (`folder: markpost`, `allowUiUpdates: false`); unset fields take the official defaults, including the 10-second rescan that makes pushes hot.
- **`dashboards/markpost/*.json`** — four dashboards with uids `markpost-overview`, `markpost-http`, `markpost-runtime`, `markpost-delivery`. The `markpost-*` prefix is markpost's dashboard-uid namespace, never reused after a deletion. The `T-*` uid namespace named by the contract is read as platform-side tenant objects (folders, datasources, platform-owned policies) owned by the devops repo; markpost's own files carry only `markpost-*` uids — stated plainly here for review.
- **`alerting/markpost-rules.yaml`** — the alert rules (inventory below).
- **`alerting/markpost-contact-points.yaml`** — one contact point (`markpost-email`, type email, delivering through the mailrise SMTP gateway to markpost's Feishu channel) paired with a notification-policy entry routing `grafana_folder = markpost` alerts to it. Route and contact point ship in one file so the pair is reviewed together. Caveat verified against Grafana's source: provisioning a `policies` entry replaces the org's policy-tree root, so exactly one policies file can hold the root on the shared instance — the platform's policies file must embed this route for it to survive re-provisioning.

Datasource references are restricted to the contract uids: `victoriametrics` for every metrics panel and rule, `victorialogs` for the one ERROR-trend logs panel on the HTTP dashboard (the paused ERROR-log-rate rule's live stand-in); no panel references `jaeger`. Queries use the instrument and attribute names exactly as markpost emits them (`deployment.environment.name` is pinned by the compose template), which assumes the platform's collector forwards OTLP to VictoriaMetrics natively — a Prometheus-translation hop would underscore the dotted names.

Every alert rule links its owning dashboard: the `dashboardUid`/`panelId` rule fields plus a `dashboard` annotation with the dashboard URL, next to `summary` and `runbook` annotations. Grafana's dashboard best practices name "most dashboards should be linked to by alerts" as the directed-browsing goal, and the notification templates render the link (and can attach a panel screenshot) for triage speed.

### The push and its assertions

[`grafana-tenant.yml`](../../../../devops/ansible/grafana-tenant.yml) runs on the `observability` host and copies exactly the whitelisted tenant files — one task per file, never a directory-level copy over `provisioning/`, so other tenants' files are untouchable. Deletions ride as explicit `state: absent` tasks (a commented example shows the retire shape). Dashboard pushes take effect on the 10-second provisioning rescan without a restart; alerting provisioning is applied through the admin reload endpoint, which the playbook calls. Post-push assertions hit the Grafana HTTP API from the NAS host and fail the deploy on any miss: folder `markpost` exists and holds exactly the four dashboards, every dashboard uid resolves through the dashboards API, the ruler API shows all twelve rules in the folder, and every notification-policy receiver resolves to an existing contact point. The API token is a per-group vault variable (`grafana_api_token`, identity `markpost-observability`); the push runs token-free and the assertions are gated on the variable existing — the same setup-order contract as the otel-token sync.

### The rule inventory: eleven evaluating plus one paused

The [runbook's rule table](../../../../docs/monitoring.md) has six rows; they expand to eleven evaluating rules in three groups (`http`, `delivery`, `telemetry`, all 60s):

1. `markpost-http-5xx-warn` — 5xx share of request rate above 5% (5m window), pending 5m.
2. `markpost-http-5xx-crit` — 5xx share above 20% (2m window), pending 2m.
3. `markpost-http-p95` — p95 request duration above 1s, pending 10m.
4. `markpost-http-rate-collapse` — 10m request rate under 20% of the hour-ago baseline, pending 10m.
5. `markpost-delivery-backlog` — `markpost.delivery.pending` above 100, pending 10m.
6. `markpost-delivery-backlog-growth` — backlog up 50+ in 15m, pending 0m.
7. `markpost-delivery-failures` — delivery-failure rate above 0, by `error_category`, pending 15m.
8. `markpost-cdn-purge-failures` — CDN purge-failure rate above 0, pending 15m.
9. `markpost-telemetry-gap-dev` — no `go.goroutine.count` from `markpost-dev` for 10m, pending 0m.
10. `markpost-telemetry-gap-staging` — the same for `markpost-staging`.
11. `markpost-telemetry-gap-production` — the same for production.

The twelfth, `markpost-error-log-rate`, is provisioned paused with the documented reason (the logs datasource's integer frames are rejected by Grafana 13.2.1's alert expressions). The rebuild's count differs from the wiring-overhaul era's "ten of eleven evaluate" on two points, both deliberate: dev gains its own telemetry-gap rule because dev now ships telemetry (`otel_otlp_endpoint` is set in its vars), and the table's single "Delivery / CDN purge failures" row is provisioned as two rules so the delivery and CDN failure signals page with independent thresholds. Every rule's `noDataState` is OK — absence means "no traffic" or "pipeline down", and the pipeline-down case is exactly what the `absent_over_time` telemetry-gap rules fire on; availability itself stays kuma's job. Routing labels group by `alertname` and `env`: the per-env gap rules carry an `env` label because Alertmanager-format label names cannot carry the dotted `deployment.environment.name`.

### Ordering

The runtime dashboard and the telemetry-gap rules are written directly against the `go.*` family that [the metrics wiring change](../architecture/2026-10-11-metrics-wiring-runtime-source.md) ships; deploying this stack before that backend reaches an environment leaves its runtime dashboard empty and its gap rules firing — deploy after [#135](https://github.com/jukanntenn/markpost/issues/135)'s backend, not before. The playbook itself runs only after [#133](https://github.com/jukanntenn/markpost/issues/133)'s observability inventory lands (the `observability` host group is re-applied here byte-identically so the tree is runnable ahead of the merge).

## Alternatives considered

**Port the old NAS JSON files.** Cheapest in effort — but this is a zero-compatibility rebuild by contract: the old volume is wiped, so nothing on the old instance needs matching, and the old assets carry v1-era assumptions the rewrite exists to drop (`process.runtime.go.*` runtime panels whose instruments no longer exist, no dev telemetry gap, alert links absent). Porting would import dead weight and preserve the drift.

**API-driven provisioning (scripts pushing dashboards and rules through the HTTP API).** Avoids the provisioning-path assumption and allows per-tenant tokens with narrow scopes — but the platform chose file provisioning as the tenant contract precisely because it is declarative, reviewable in git, and hot-reloaded without credentials in the deploy path; an API script would re-invent state management (creation, update, deletion ordering) that the provisioner already owns, and the push would stop being auditable as plain file copies.

**Keep alert links out of the annotations.** Smaller rule files — but official guidance directs responders from an alert to its dashboard, and the link costs one annotation line; without it every page opens a hunt for the right dashboard in the triage table.

## Consequences

markpost's dashboards and alerting are reviewable artifacts in this repo, pushed by an auditable playbook and proven loaded by API assertions — the Grafana-side observability story no longer depends on handoff material on a server nobody's repo tracks. The costs: the rebuild starts from an empty data volume (no imported history — baselines for the "thresholds are starting points" tuning restart from the re-onboard date); the Grafana API key had to be re-issued and vaulted under a new identity (`markpost-observability`), and the old NAS provisioning files are gone with the old volume rather than archived; the platform's actual provisioning path, port, and the mailrise apprise address are flagged assumptions in the playbook and files, to correct at review or first run; and the policies-entry root-replacement semantics leave markpost's route surviving only as long as the platform's own policies file carries it — the pairing assertion catches its loss on the next push.
