# MRFC: Beszel deployment topology on ttyo

Status: proposed

English | [中文](2026-08-31-beszel-deployment-topology.zh.md)

## Problem

Beszel's agent needs a transport, and the repository needs an automation boundary. ttyo runs default-deny inbound (SSH, and 443 from Cloudflare CIDRs only), so the agent's connection must stay outbound. The hub's home, the agent's route to it, and how much of this the repository automates must be decided without new inbound ports or weakening that posture.

## Proposal

The [host-metrics MRFC](./2026-08-31-host-metrics-monitoring-beszel.md) selected the component; this layer fixes its topology and the automation boundary.

- The hub lives on a separate, operator-managed server — never on the monitored host. Its deployment, exposure (panel URL, TLS, any fronting), credentials, notification channels, and upgrades are manual ops work outside this repository; the runbook carries an ops checklist handoff, not automation.
- The repo automates only the agent on ttyo: its own compose project, separate from the app's, so monitoring and app deploys stay independent. The deploy playbook renders the compose from a template with a pinned image, host networking, and the read-only `docker.sock` mount, and points the agent at the operator-provisioned `HUB_URL` — an outbound WebSocket (default agent port 45876), so the firewall opens nothing new.
- The agent's `KEY` is the hub's public key — not a secret — and lives in the template. The repo carries no hub secrets: hub admin credentials and notification tokens live with the ops-managed hub, outside the repository's vault.
- Removal of the automated side is documented in the runbook like the heartbeat's removal row: deploy templates never uninstall. Hub removal is ordinary ops work on the hub host.

## Alternatives considered

- **Co-locating the hub on ttyo behind the shared gateway** (`beszel.markpost.cc` + wildcard Origin CA + Cloudflare, hub on loopback). Rejected on review: hub and host die together, so the resource layer goes dark with the machine it watches; it also wired repo automation around a panel that belongs to the ops side.
- **Repo-automated hub deployment** (ansible-templated hub compose, hub credentials and notification token in the repo vault). Rejected on review: the hub is ops infrastructure — its host and lifecycle belong to the operator, and repo templating would bake in topology choices (loopback port, gateway site block) the ops host may not share.
- **A socket-proxy container in front of the agent's `docker.sock`.** One more moving part now; the direct read-only socket is proportionate. Escalate if the risk ages.
- **Folding the agent into markpost's compose file.** Rejected: the app compose stays app-shaped (app + db + migrate); monitoring must be deployable and removable independently.
- **An overlay network (Tailscale/WireGuard) for the agent→hub route.** A new network subsystem for one outbound WebSocket; `HUB_URL` over ordinary HTTPS with key-pair auth is sufficient, and an overlay remains the operator's optional choice on the ops side.

## Acceptance criteria

- `ansible-playbook devops/ansible/deploy.yml -e target=production` renders the agent compose idempotently; the agent connects to the operator-provisioned `HUB_URL` and reports in.
- No hub automation, hub secrets, or hub topology in the repo; the runbook's setup order splits the agent (automated) from the hub (ops checklist).
- `ufw status` on ttyo is identical before and after — the agent's connection is outbound.
- `docs/monitoring.md` and its zh pair carry this layer's setup order and removal rows.

## Risks

- The hub host and the ttyo→hub route join the dependency chain (recorded in the selecting layer): a hub outage silences threshold alerting while the availability layer stays independent; a route outage surfaces as agent-offline alerts.
- If the route rides the ops network's tunnel, tunnel maintenance silently pauses metric flow — agent-offline alerting is the backstop.
- Beszel image pins drift silently without a renovator; the runbook's upgrade row owns them.
