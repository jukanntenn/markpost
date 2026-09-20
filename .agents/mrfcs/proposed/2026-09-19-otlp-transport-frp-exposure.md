# MRFC: Transport and exposure architecture for OTLP telemetry over frp

English | [中文](2026-09-19-otlp-transport-frp-exposure.zh.md)

Status: proposed

## Problem

Telemetry producers and the observability stack live on opposite sides of a NAT. markpost runs on a public VPS (vps1); the stack runs on a home/office NAS with no public IP and no inbound ports; the only path back is an already-deployed frp chain — frps on a second VPS (vps2, 3Mbps, fronted by caddy for TLS termination), frpc on the NAS dialing out. Because the producer (vps1) and the frp entrypoint (vps2) are different machines, the OTLP ingestion endpoint is necessarily reachable from the public internet through vps2's caddy: there is no localhost shortcut. Transport security, authentication, and bandwidth protection are therefore first-class requirements of this design, not hardening applied later.

## Proposal

Scope first: **frp is an external service under external operations control — this project deploys no frp component and writes no frp config** (review scope ruling on this layer). The repo-owned deliverables are the producer side (markpost's OTLP env), the caddy vhost routing (this repo manages Caddyfiles via ansible), and the interface requirements handed to the external relay operators; every frp-side property below is something we require and verify, not something we configure.

The chain, end to end:

```
markpost@vps1 ──HTTPS(OTLP+gzip+Bearer)──► caddy@vps2:443 (otlp vhost, IP-allowlisted)
                                              │ 127.0.0.1 (loopback only)
                                           frps@vps2 ──encrypted tunnel──► frpc@NAS
                                                                           │ docker network
                                                                   otelcol-contrib ─► stores
```

- **Public surface** (all on vps2): caddy `:80`/`:443` are repo-configured; the frps control port `:7000` (token auth, `transport.tls.force = true`) and the loopback binding of frps proxy ports (`proxyBindAddr = "127.0.0.1"`) are interface requirements on the external relay — required and black-box verified from this side, configured by the external operators. The NAS accepts zero inbound connections — frpc only dials out.
- **Two caddy vhosts**: `grafana.<domain>` for operators (Grafana's own accounts, anonymous access off) and `otlp.<domain>` for producers, with a `remote_ip` allowlist holding producer IPs (vps1 today; one line per future business system).
- **Collector authentication**: otelcol-contrib's `bearertokenauth` extension guards the OTLP receiver; every business system gets its own static token, and the collector stamps `service.name` from the token's identity, so a producer cannot spoof another producer's telemetry. Auth is enforced from day one — the endpoint is publicly reachable, so this is the real gate, not defense in depth.
- **Bandwidth protection**: SDK-side gzip and batching are repo-owned; the frp per-proxy `transport.bandwidthLimit` on the otlp proxy (512KB/s initial) is requested from the external relay operators. markpost's single-service telemetry is a few MB/day compressed — far below the 3Mbps relay — but the cap guarantees telemetry can never starve Grafana or anything else sharing vps2.
- **Degradation**: when vps2 or the tunnel is down, the OTel SDK batches and retries, then drops. The local `app-*.jsonl` dual-write (owned by the [observability stack MRFC](./2026-09-19-otlp-observability-stack.md)) retains the incident evidence on vps1; visualization is blind, the service is not.
- Configuration lands with the implementation for the repo-owned touchpoints only: markpost's env vars and the caddy vhosts. The frpc proxy block and the frps settings become interface requirements in the handoff document for the external relay operators — including the requirement that their frpc container joins the shared external docker network so `localIP = "collector"` resolves to the collector by name.

## Alternatives considered

**Same-host loopback binding** (producer → `127.0.0.1:<frps proxy port>`): the cheapest possible exposure — ingestion never touches the public internet at all. Lost on the real topology: it only works when a producer shares the machine with frps, and markpost (vps1) does not. Kept as the required pattern for any future producer that does co-locate with frps.

**Direct public bind of the frps proxy port for OTLP (bypassing caddy)**: one less hop. Lost: plaintext on the public segment with no TLS termination, no IP-allowlist layer, and a second public port to defend — while caddy already exists on vps2 and terminates TLS for everything else. The loopback bind plus caddy costs one reverse-proxy stanza and closes all three gaps.

**WireGuard site-to-site instead of publishing the OTLP endpoint**: the strongest network-level isolation — no public application ports at all. Lost for now: it adds a kernel network dependency and a second trust root spanning three machines run as different concerns (public VPSes vs the NAS), while frps+caddy are already deployed and working. Reconsider when multiple producers make per-host tunnel management routine instead of one allowlist line each.

**frp stcp (secret tunnel) visitor mode**: token-protected private exposure without any public port. Lost: every producer must run an frpc visitor — reasonable for an administrator's laptop, wrong as an onboarding requirement for every business system.

## Acceptance criteria

- Relay interface requirements documented in the handoff (frps control token + `transport.tls.force`, proxy ports loopback-bound, otlp proxy bandwidth-limited, frpc attached to the shared docker network) and verified black-box where observable: the `otlp` vhost rejects non-allowlisted IPs; the collector rejects unauthenticated OTLP. The NAS accepts no inbound connections.
- The `otlp` vhost rejects non-allowlisted IPs; the collector rejects unauthenticated OTLP; the token's identity maps to `service.name` in stored telemetry.
- No plaintext on any public segment: browser→caddy and markpost→caddy over TLS; frps→frpc over `useEncryption`.
- The otlp proxy is bandwidth-limited (interface requirement agreed with external ops); Grafana stays usable under full telemetry flow.
- A tunnel outage degrades gracefully: SDK retries then drops; incident evidence remains in vps1-local `app-*.jsonl`.

## Risks

- **Token lifecycle**: static bearer tokens need a rotation procedure (documented in the ops runbook with the implementation); a compromised token's blast radius is one producer's telemetry stream.
- **Domain and DNS**: the two vhosts need DNS names and certificates (caddy ACME); a domain change touches every producer's endpoint config.
- **caddy misconfiguration is the new single point of exposure**: the design fails closed — collector auth still applies if the allowlist is dropped — and the acceptance criteria verify both layers independently.
- **vps2 is a shared single point**: its outage blinds visualization but never the service; local crash-channel logging covers incident forensics in that window.
- **External change control on the relay**: caddy routing is repo-owned, but frps/frpc are not — interface changes (proxy ports, limits, network attachment) route through the external operators, and documented config can drift from actual. Mitigation: the handoff document states the required properties, and the acceptance checks verify the externally observable ones (403/401 behavior) regardless of who configures them.
