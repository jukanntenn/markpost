# MRFC: Beszel agent authenticates to the hub with a vaulted WebSocket token

Status: implemented

English | [中文](2026-10-02-beszel-agent-websocket-token.zh.md)

## Problem

The hub is now self-hosted on the observability server (192.168.5.57, `beszel.bytehome.fun` at the edge) instead of an externally operated box, in the WebSocket-only topology (`DISABLE_SSH=true`): agents dial the hub outbound, the production firewall stays closed. The repo's agent template rendered only `KEY` + `HUB_URL` — a half-state from beszel 0.18, whose SSH-era assumptions predate token-based WebSocket auth: without a `TOKEN` the agent cannot authenticate to the hub, so any deploy rendering the template would silently break the host-metrics line. The agent image pin was also stale (0.18.8 vs 0.20.0), and 0.19 made agents verify the hub's TLS certificate (compatible with the real-cert edge). Separately, the hub embeds PocketBase and has no external-database support — the "PostgreSQL/MySQL/MariaDB" expectation for the hub was unverifiable upstream (no driver, no env in `go.mod`), so the hub runs SQLite on a bind mount with PocketBase backups as the durability story.

## Decision

**The agent authenticates with a per-system token vaulted as `beszel_agent_token`.** `beszel-agent-compose.yml.j2` renders `TOKEN: "{{ beszel_agent_token }}"` alongside `HUB_URL`/`KEY`; deploy.yml's four agent tasks require BOTH `beszel_hub_url` and `beszel_agent_token`, so the agent never renders without its complete credential pair (the setup-order contract now names both). `group_vars/production/vars.yml` sets `beszel_hub_url: https://beszel.bytehome.fun`, the hub's public key (`beszel_agent_key`, not a secret), and bumps the pin to `0.20.0`. The hub itself stays outside the repo (own compose on the observability server): SQLite on a bind-mounted `beszel_data`, Feishu alerts via shoutrrr `lark://`, thresholds CPU 95 / memory 92 / disk 90 (2 min) and agent-down 1 min — one threshold per metric, so the critical tier is used.

## Alternatives considered

**Keep SSH mode (hub dials agents).** It lost: the hub sits on the LAN and cannot reach the VPS; opening 45876 inbound contradicts the "firewall opens nothing" rule the topology MRFC established, and `DISABLE_SSH` on the hub already excludes it.

**Point the hub at an external database.** It lost: no such feature exists upstream (embedded PocketBase, SQLite only) — persistence is the bind mount plus hub-side backups; revisiting means changing product, not config.

**Fill the group_vars without touching the template.** It lost: rendering `HUB_URL` without `TOKEN` produces an agent that dials, fails auth, and falls back to a dead SSH listener — the silent-breakage mode this change exists to remove; the joint guard makes the vars atomic.

## Consequences

A production deploy now (re)takes over the agent compose on ttyo — the interim hand-written compose there must be treated as superseded at the next deploy, which is safe because it carries the identical env set. The hub URL, public key, and pin are plain vars; the token is the only secret added. Agents on beszel 0.20 verify the hub certificate, so the edge must keep serving a real cert on `beszel.bytehome.fun`. Rotation of a leaked token is hub-side (regenerate in the Add-System dialog) plus one vault edit.
