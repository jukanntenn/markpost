# MRFC: Availability monitoring via uptime-kuma

Status: implemented

English | [中文](2026-08-30-availability-monitoring.zh.md)

## Problem

markpost had no external failure detection. Docker healthchecks only influence local restarts; the CDN keeps serving cached pages while the origin is down; a single Cloudflare vantage point cannot distinguish an edge failure from an origin failure; and the liveness-only `/api/v1/health` stays green while Postgres is down — the write path would fail for users before anything visible to the operator turned red. Outages were discovered manually.

## Decision

Availability monitoring is layered across three vantage points, operated by a self-hosted uptime-kuma instance probing production and staging:

- `GET /api/v1/ready` is a readiness endpoint: a driver-level database round trip answering `200 {"status":"ready"}` or `503 {"status":"unavailable"}`, registered outside every rate limiter next to `/health`. Liveness `/health` is unchanged and remains what the Docker healthcheck polls — a dead database must mark the service unready, not kill the container.
- The monitor inventory, notification channels (Feishu primary, SMTP fallback), alert policy (60 s interval, 3 retries, ~4 minutes to page, recovery notices on, repeat reminders off, certificate/domain expiry at 7/14/21 days), and the triage table live in [`docs/monitoring.md`](../../../docs/monitoring.md).
- The production VPS runs a supervisor program `markpost-heartbeat`: a loop probing `http://127.0.0.1:8080/api/v1/ready` and pushing the verdict to kuma's push endpoint, so kuma sees the origin's own view past Cloudflare, and push silence covers host death. The push URL is a vault secret (`kuma_heartbeat_url`); the deploy installs the program only when that variable exists.

## Alternatives considered

- **Black-box monitoring only, no app changes.** Rejected because `/health` is liveness-only: with Postgres down every external probe stays green while the write path fails. The readiness endpoint is the root fix; without it the monitoring stack cannot see the database at all.
- **Scripted kuma configuration via the socket.io API** (e.g. the third-party `uptime-kuma-api` Python library). Rejected: a new dependency plus 2.x compatibility risk to configure six monitors once; the runbook carries exact field values instead, and kuma-side changes are rare.
- **A systemd timer for the heartbeat.** Rejected in favor of supervisor: the production host already runs supervisor for several services, one supervision system per host is enough, and supervisor's `autorestart` recovers a crashed loop without extra timer unit semantics.
- **Direct database and container monitors** (kuma's `postgres`/`docker` types). Rejected: production Postgres is reachable only over its Unix-socket volume and Docker is not exposed off-host, so the external vantage cannot reach either; readiness covers database fitness without widening the attack surface.
- **Maintenance windows covering deploys.** Rejected: the ~4-minute retry threshold already absorbs deploy-time container swaps, and a window can silence a genuine fault that lands during one.

## Consequences

Bought: origin, database, edge-path, and whole-host failures each surface within ~4 minutes, and the triage table tells them apart from which monitors are red. Cost: one more supervisor program on the VPS, a vault secret that must be rotated if leaked (its holder can forge up-beats), and kuma itself becomes load-bearing — its outage means silence, not false alarms. Verification: the readiness contract is covered by `ready_test.go` (both verdicts); after the first production deploy with the vault variable set, `sudo supervisorctl status markpost-heartbeat` must show RUNNING and kuma must receive beats. The kuma-side setup (monitors, channels, vaulting the push URL) is operational follow-up sequenced by the runbook's setup order.
