# MRFC: Beszel deployment topology on ttyo

Status: proposed

English | [中文](2026-08-31-beszel-deployment-topology.zh.md)

## Problem

Beszel's hub needs operator access and its agent needs a transport. ttyo runs default-deny inbound (SSH, and 443 from Cloudflare CIDRs only) behind a shared host Caddy gateway that imports `/etc/caddy/conf.d/*.caddy`, holds a wildcard Origin CA certificate (`markpost.cc, *.markpost.cc`), and is driven by an ansible deploy that already templates gateway site blocks. Where the hub lives, how it is reached, and how its secrets travel must be decided without new inbound ports or weakening that posture.

## Proposal

The [host-metrics MRFC](./2026-08-31-host-metrics-monitoring-beszel.md) selected the component; this layer fixes its topology.

- Hub and agent both run on ttyo as their own compose project, separate from the app's — monitoring and app deploys stay independent. The hub binds to loopback only; the deploy playbook renders the compose file from a template with pinned versions from `group_vars`.
- Operator access reuses the shared-gateway machinery ([origin-port-443 MRFC](../implemented/2026-08-23-origin-port-443-shared-gateway.md)): a proxied `beszel.markpost.cc` DNS record plus one gateway site block reverse-proxying to the hub's loopback port — covered by the existing wildcard Origin CA, zero firewall change, Cloudflare in front. The hub's own login is the application-layer gate; Cloudflare Access is the recorded escalation.
- The agent uses host networking plus the read-only `docker.sock` mount. Its `KEY` is the hub's public key — not a secret — and lives in the template; hub admin credentials and the notification token are per-variable vault entries in `group_vars/production/vault.yml` (avpm), like `kuma_heartbeat_url`.
- Removal is documented in the runbook like the heartbeat's removal row: deploy templates never uninstall.

## Alternatives considered

- **Hub at home (oect/fn), agent-only on ttyo.** Survives ttyo death and preserves history, but couples production observability to a home box and its tunnel. Host death is already the kuma heartbeat's signal, so the marginal benefit buys a new dependency. Revisit if history retention across outages starts to matter.
- **SSH-tunnel-only access, no public hostname.** Smallest surface, but daily friction for a solo operator; the gateway + Cloudflare path reuses existing machinery, opens no ports, and keeps the hub behind login.
- **A socket-proxy container in front of `docker.sock`.** One more moving part now; the direct read-only socket behind an origin-gated hub is proportionate. Escalate if the risk ages.
- **Folding Beszel into markpost's compose file.** Rejected: the app compose stays app-shaped (app + db + migrate); monitoring must be deployable and removable independently.
- **An overlay network (Tailscale/WireGuard) for access.** A new network subsystem for one panel; a Cloudflare-fronted login is enough for a solo operator.

## Acceptance criteria

- `ansible-playbook devops/ansible/deploy.yml -e target=production` renders the monitoring compose and gateway site block idempotently; a changed site block reloads the gateway.
- `https://beszel.markpost.cc` reaches the hub login through Cloudflare; `ufw status` is identical before and after.
- No secret values in templates or git: hub credentials and the notification token are vault variables; the agent `KEY` in the template is the public key.
- `docs/monitoring.md` and its zh pair carry the setup order and removal rows for this layer.

## Risks

- A public hub URL attracts scanner traffic; Cloudflare plus hub auth absorb it, and Cloudflare Access is the named escalation.
- Beszel image pins drift silently without a renovator; the runbook's upgrade row owns them.
- If the shared-gateway convention changes, this site block must move with it — the same coupling the heartbeat program already accepted.
