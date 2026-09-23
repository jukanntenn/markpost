# MRFC: Transport and exposure architecture for OTLP telemetry over frp

English | [中文](2026-09-19-otlp-transport-frp-exposure.zh.md)

Status: implemented

## Problem

Telemetry producers and the observability stack live on opposite sides of a NAT. markpost runs on a public VPS (vps1); the stack runs on a home/office NAS with no public IP and no inbound ports; the only path back is an frp chain — frps on a second VPS (vps2, 3Mbps, fronted by caddy for TLS termination), frpc on the NAS dialing out. Because the producer (vps1) and the frp entrypoint (vps2) are different machines, the OTLP ingestion endpoint is necessarily reachable from the public internet through vps2's caddy: there is no localhost shortcut. Transport security, authentication, and bandwidth protection are therefore first-class requirements of this design, not hardening applied later.

## Decision

The whole external chain is operated by the external operators from handoff material; this repo owns exactly one touchpoint: markpost's OTLP environment (`OTEL_EXPORTER_OTLP_ENDPOINT=https://otlp.bytehome.fun` for staging and production, LAN-direct for `markpost-dev`, bearer token per env vaulted as `otel_otlp_token`).

```
markpost@vps1 ──HTTPS(OTLP+gzip+Bearer)──► caddy@vps2:443 (otlp vhost, IP-allowlisted)
                                              │ 127.0.0.1 (loopback only)
                                           frps@vps2 ──encrypted tunnel──► frpc@NAS
                                                                           │ docker network
                                                                   otelcol-contrib ─► stores
```

- **Public surface** (all on vps2, all externally configured): caddy `:80`/`:443` plus the frps control port. frps proxy ports stay publicly bound (other tenants need direct reach, e.g. SMTP), so the loopback-only idea from the proposal era gave way to compensating controls: collector token auth (fail-closed) on every path, and the otlp vhost's `remote_ip` allowlist holding the production server's IP.
- **Two caddy vhosts** (spec handed off): `grafana.bytehome.fun` for operators (Grafana's own accounts, anonymous off) and `otlp.bytehome.fun` for producers, both reverse-proxying frps proxy ports on loopback.
- **Collector authentication**: `bearertokenauth` guards the OTLP receiver with one token per environment (`markpost-dev` / `markpost-staging` / `markpost`); the collector requires a valid token on every ingress (no token → 401, wrong token → 401, valid token → accepted — verified through both the LAN and the public path during rollout).
- **Bandwidth protection**: SDK-side gzip and batching in the repo; the frp per-proxy `transport.bandwidthLimit` on the otlp proxy is requested from the external operators so telemetry can never starve the 3Mbps relay.
- **Degradation**: when vps2 or the tunnel is down, the OTel SDK retries then drops; the dual-written `app-*.jsonl` on the producer host retains the incident evidence.

## Alternatives considered

**Same-host loopback binding** (producer → `127.0.0.1:<frps proxy port>`): the cheapest possible exposure — ingestion never touches the public internet at all. Lost on the real topology: it only works when a producer shares the machine with frps, and markpost (vps1) does not. Kept as the required pattern for any future producer that does co-locate with frps.

**Direct public bind of the frps proxy port for OTLP (bypassing caddy)**: one less hop. Lost: plaintext on the public segment with no TLS termination, no IP-allowlist layer, and a second public port to defend — while caddy already exists on vps2 and terminates TLS for everything else. The loopback bind plus caddy costs one reverse-proxy stanza and closes all three gaps.

**WireGuard site-to-site instead of publishing the OTLP endpoint**: the strongest network-level isolation — no public application ports at all. Lost for now: it adds a kernel network dependency and a second trust root spanning three machines run as different concerns (public VPSes vs the NAS), while frps+caddy were already deployed and working. Reconsider when multiple producers make per-host tunnel management routine.

**frp stcp (secret tunnel) visitor mode**: token-protected private exposure without any public port. Lost: every producer must run an frpc visitor — reasonable for an administrator's laptop, wrong as an onboarding requirement for every business system.

## Consequences

The public chain is verified end-to-end: `https://grafana.bytehome.fun` serves Grafana through the full chain; the otlp vhost answers 403 off-allowlist, 401 without or with a wrong token, and accepts authenticated pushes from the production server. Staging rides the same public ingress deliberately — it validates the production path before production depends on it.

Operational truths that surfaced during rollout and now stand as obligations:

- **`docker compose restart` does not reload `.env`** — an env change requires `up -d` (recreate). A stale-env collector silently rejected every push for days because SDK-side failures are quiet; the ingest-absence alert is tracked as follow-up work alongside the [observability stack MRFC](./2026-09-19-otlp-observability-stack.md).
- **Interface drift is the standing risk** of the externally operated chain: vhosts, proxy ports, limits, and network attachment route through the external operators. The handoff document states the required properties, and the black-box checks (403/401/accepted) verify the externally observable ones regardless of who configures them.
- **Token rotation is two-sided**: the collector's token list and every producer's env move together, or ingestion breaks quietly; rotation runs collector-side first (new tokens added), then producer-side.
