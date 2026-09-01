# MRFC: Beszel deployment topology on ttyo

Status: implemented

English | [中文](2026-08-31-beszel-deployment-topology.zh.md)

## Problem

Beszel's agent needs a transport, and the repository needs an automation boundary. ttyo runs default-deny inbound (SSH, and 443 from Cloudflare CIDRs only), so the agent's connection must stay outbound. The hub's home, the agent's route to it, and how much of this the repository automates must be decided without new inbound ports or weakening that posture.

## Decision

The [host-metrics MRFC](./2026-08-31-host-metrics-monitoring-beszel.md) selected the component; this record fixes its topology and the automation boundary.

- The hub lives on a separate, operator-managed server — never on the monitored host. Its deployment, exposure, credentials, notification channels, and upgrades are manual ops work outside this repository; [`docs/monitoring.md`](../../../docs/monitoring.md) carries the ops checklist, not automation.
- The repo automates only the agent on ttyo: its own compose project at `~/docker/beszel-agent` (`beszel_agent_path` in `group_vars/all.yml`), separate from the app's. [`deploy.yml`](../../../devops/ansible/deploy.yml) renders [`beszel-agent-compose.yml.j2`](../../../devops/ansible/templates/beszel-agent-compose.yml.j2) with the pinned image, host networking, and the read-only `docker.sock` mount, and points the agent at the operator-provisioned `HUB_URL` — an outbound WebSocket, so the firewall opens nothing new. The tasks run only when `beszel_hub_url` is defined.
- The agent's `KEY` is the hub's public key — not a secret — and lives in the template. The repo carries no hub secrets: hub admin credentials and notification tokens live with the ops-managed hub, outside the repository's vault.
- Removal of the automated side is documented in the runbook like the heartbeat's removal row: deploy templates never uninstall. Hub removal is ordinary ops work on the hub host.

## Alternatives considered

- **Co-locating the hub on ttyo behind the shared gateway** (`beszel.markpost.cc` + wildcard Origin CA + Cloudflare, hub on loopback). Rejected on review: hub and host die together, so the resource layer goes dark with the machine it watches; it also wired repo automation around a panel that belongs to the ops side.
- **Repo-automated hub deployment** (ansible-templated hub compose, hub credentials and notification token in the repo vault). Rejected on review: the hub is ops infrastructure — its host and lifecycle belong to the operator, and repo templating would bake in topology choices the ops host may not share.
- **A socket-proxy container in front of the agent's `docker.sock`.** One more moving part; the direct read-only socket is proportionate. Escalate if the risk ages.
- **Folding the agent into markpost's compose file.** Rejected: the app compose stays app-shaped (app + db + migrate); monitoring is deployable and removable independently.
- **An overlay network (Tailscale/WireGuard) for the agent→hub route.** A new network subsystem for one outbound WebSocket; `HUB_URL` over ordinary HTTPS with key-pair auth is sufficient, and an overlay remains the operator's optional choice on the ops side.

## Consequences

Bought: one automation boundary — the repo owns and tests exactly the agent half, while the hub's host, exposure, and secrets stay with ops; no hub topology leaks into templates, and ttyo's firewall posture is untouched. Cost: the hub host and the ttyo→hub route join the dependency chain (recorded in the selecting layer); if the route rides the ops network's tunnel, tunnel maintenance silently pauses metric flow — agent-offline alerting is the backstop; Beszel image pins drift without a renovator, so the runbook's upgrade row owns them; activation waits on the operator setting `beszel_hub_url` and `beszel_agent_key`. Verification: the prek gates run `ansible-playbook --syntax-check` and template render+parse on every change to the automation; the runbook's setup order and `ufw status` before/after cover the deploy-time checks.
