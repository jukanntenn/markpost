# MRFC: One multi-arch Docker image, Caddy as the edge

Status: implemented

## Problem

Shipping the backend and the frontend as separate artifacts meant per-environment assembly: a Node process or static host for the frontend, a reverse proxy picked and configured per deployment, and version skew between the two halves at every deploy. Every environment (staging, production, homelab) re-solved the same plumbing with different answers.

## Decision

markpost ships as one versioned multi-arch Docker image built by `docker/build.py` from `docker/Dockerfile`. Inside the container, s6-overlay supervises the Go backend and Caddy; the frontend is baked in as its static export. Caddy serves the exported frontend and reverse-proxies `/api/v1` and `/swagger` to the backend, so the public surface is one origin with no cross-origin setup. Environment differences live in `devops/ansible/` group_vars and Caddyfile templates, not in the image.

## Alternatives considered

**Separate frontend and backend images.** Independent versioning per half, but every deploy must pin a compatible pair and each environment still needs a proxy and TLS story; version skew becomes an operator problem.

**A Node server for the frontend (SSR/standalone).** Keeps server-side rendering available, but pays a runtime the product does not use — the app is fully client-rendered — and adds a second process to supervise for zero rendering benefit.

**nginx instead of Caddy.** Equally capable as a reverse proxy, but Caddy's automatic TLS issuance and markedly smaller configuration fit the self-hosted targets better.

## Consequences

One artifact, one version number, one deploy unit: what e2e tests (`dagger call -m e2e all`) is exactly what runs in production. Container builds are multi-arch and therefore slower. Caddy is now a load-bearing component — its per-environment templates are part of the deploy surface and are syntax-checked in prek.
