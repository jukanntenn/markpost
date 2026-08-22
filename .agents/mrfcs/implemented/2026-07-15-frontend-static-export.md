# MRFC: Frontend is a static export with dev-only API rewrites

Status: implemented

## Problem

The frontend assumed a Node server at the edge: server-side proxying of `/api` to the backend, request-time i18n configuration, and server-rendered routes. That assumption forced a Node runtime into every deployment and fought the single-image deployment model, where Caddy — not Node — is the only process in front of the backend.

## Decision

`frontend/next.config.ts` pins `output: "export"` unconditionally: the build produces plain static files under `frontend/out/`, which Caddy serves. During development only (`NODE_ENV !== "production"`), a `rewrites()` block proxies `/api/v1` and `/swagger` to `BACKEND_URL` (default `http://127.0.0.1:7330`) so the dev server talks to a local backend; the block is attached only in dev because the mere presence of the `rewrites` key triggers a build warning under static export. The frontend never gains a `middleware.ts` or `proxy.ts` — a static export cannot run one. i18n is pure-client next-intl; OAuth completes as a same-page redirect; client-side route guards are UX only, with security enforced by the backend (see `specs/frontend/routes.md`).

## Alternatives considered

**Standalone Next.js server mode.** Keeps server features available, but adds a Node process per deployment and a second origin or proxy hop; no page in the product needs server rendering.

**`middleware.ts`/edge proxy for API calls.** The Next-native mechanism, but it requires a server runtime that a static export does not produce — dev-only `rewrites` deliver the same ergonomics where a server exists (the dev server) and Caddy covers production.

**Vercel-style managed hosting.** Moves the runtime question to a platform; incompatible with the self-hosted single-image goal and adds an external dependency to deploys.

## Consequences

The deployable frontend is a directory of static files — no Node runtime, no server-side anything, and the entire `out/` tree is cacheable at the CDN. All server interaction is direct client → backend (same-origin via Caddy in production, via dev rewrites locally). Anything requiring a server (middleware, request-time i18n, API routes) is out of bounds by construction, which is the point.
