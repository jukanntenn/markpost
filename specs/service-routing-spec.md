# Service Routing & Deployment Specification

This document defines the architectural rules for projects built with a Go
backend and a Next.js frontend that run as **separate services**. It covers
routing principles, environment topology, Docker Compose structure, and the
integration contract between frontend and backend.

Any project following this spec reproduces the same mechanisms with
project-specific details (ports, domain names, image names, route definitions)
while adhering to the constraints below.

---

## 1. Core Principle

**The frontend is the network entry point.** All client traffic — browser
requests, API calls, public resource access — arrives at the Next.js server
first. The backend is never directly exposed to external traffic; it is
reachable only through the frontend's rewrite proxy or the internal container
network.

The backend must not serve frontend assets, render HTML pages, or act as a
reverse proxy. It is a pure API server.

---

## 2. Routing Architecture

### 2.1 No Root-Level Catch-All Routes on the Backend

The backend **must not** register any root-level wildcard route (e.g.
`GET /:id`, `POST /:slug`). Every backend route must be identifiable by an
explicit prefix or pattern.

There are two acceptable patterns for root-level routes:

1. **Path prefix**: routes under a fixed prefix (e.g. `/api/v1/*`).
2. **Value-prefix**: single-segment routes where the database value itself
   begins with a known prefix, and the route pattern mirrors that prefix
   (e.g. `GET /item-:id` where the database stores `item-abc123`).

The key rule: if a route is registered at the root level (single path segment),
the matched parameter **must** carry a prefix that is part of the stored
database value — never synthesized at routing time.

### 2.2 Frontend Rewrites

The frontend configures Next.js `rewrites` to proxy backend-bound requests.
All rewrites read the backend address from a single environment variable with
a code-level default matching the backend's default port:

```
API_PROXY_TARGET=http://backend:7330   # Production (Docker internal)
API_PROXY_TARGET=http://localhost:7330  # Default in code
```

Rewrites are unconditional — they apply in both development and production
(`NODE_ENV=development` and `NODE_ENV=production`).

General pattern:

```ts
async rewrites() {
  const target = process.env.API_PROXY_TARGET || "http://127.0.0.1:7330";
  return [
    // API routes — always present
    { source: "/api/:path*", destination: `${target}/api/:path*` },
    // Project-specific value-prefixed routes
    // { source: "/<prefix>-:param", destination: `${target}/<prefix>-:param` },
  ];
}
```

The code default (`http://127.0.0.1:<port>`) matches the backend's code-level
port default. This enables zero-config local development — no `.env` file is
required. Production deployments override `API_PROXY_TARGET` to point to the
backend container via Docker internal DNS.

### 2.3 Frontend Pages

Frontend pages live at the root level without any `basePath` configuration.
Standard Next.js App Router conventions apply. Frontend route guards (public,
protected, role-restricted) are defined per project.

### 2.4 Health Check Endpoints

Both services expose a health check at different paths to avoid collision:

| Service | Path | Scope |
|---|---|---|
| Frontend | `/health` | Root level — handled by Next.js directly |
| Backend | Under the API prefix (e.g. `/api/v1/health`) | Proxied via frontend rewrite |

Docker health checks use the container-internal address (`127.0.0.1:port`).

---

## 3. Environment Models

### 3.1 Overview

Every project defines three environment models. They differ in how services are
started but share the **same routing architecture** (frontend entry point,
rewrites to backend).

### 3.2 Local Development (No Docker)

Used for day-to-day coding with the fastest feedback loop.

- Backend: run directly (`go run ./cmd/server`).
- Frontend: run directly (`pnpm dev`).
- Database: lightweight embedded database (e.g. SQLite).
- Integration: Next.js rewrites proxy `API_PROXY_TARGET=http://localhost:<port>`.

No containers are involved. IDE task runners (VS Code tasks, Make targets) are
the typical launch mechanism.

### 3.3 Development Docker Compose

Used when a production-like database or infrastructure is needed locally.

- Backend: Docker container with hot-reload tooling (e.g. Air).
- Frontend: **runs on the host** (`pnpm dev`) — not containerized — to preserve
  HMR and fast refresh.
- Database: production-grade database (e.g. PostgreSQL) in a container.
- Integration: Next.js rewrites proxy `API_PROXY_TARGET=http://localhost:<port>`.

The frontend is not containerized in this model because Docker volume mounts
add latency to file watching, degrading the development experience.

### 3.4 Production Docker Compose

Used for all remote deployments (staging, production, on-premise).

- Backend: pre-built Docker image (compiled binary).
- Frontend: pre-built Docker image (Next.js standalone `node server.js`).
- Database: production-grade database in a container or managed service.
- Integration: Next.js rewrites proxy `API_PROXY_TARGET=http://backend:<port>`
  via Docker internal DNS.

All three services run in the same Docker Compose project.

---

## 4. Docker Compose Topology

### 4.1 Service Definitions

```yaml
services:
  frontend:
    image: <project>-web:<tag>
    ports:
      - "${FRONTEND_PORT:-3000}:3000"
    environment:
      - API_PROXY_TARGET=http://backend:${BACKEND_PORT:-7330}
    depends_on:
      backend:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://127.0.0.1:3000/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  backend:
    image: <project>:<tag>
    # No ports mapped to the host — internal only
    environment:
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=${BACKEND_PORT:-7330}
      - DB_DRIVER=postgresql
      - DB_DSN=postgres://user:pass@postgres:5432/dbname
    depends_on:
      postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://127.0.0.1:${BACKEND_PORT:-7330}/api/v1/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  postgres:
    image: postgres:17-alpine
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    environment:
      - POSTGRES_DB=${POSTGRES_DB}
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

### 4.2 Port Exposure Rules

| Service | Exposed to Host | Rationale |
|---|---|---|
| Frontend | Yes | Only entry point for all external traffic |
| Backend | No | Accessible only via Docker internal network |
| Database | Yes | Allows external tooling (GUI clients, backups, debugging) |

### 4.3 Startup Order

```
postgres (healthy) → backend (healthy) → frontend
```

Frontend rewrites will fail gracefully if the backend is unreachable (HTTP 502),
but correct startup ordering prevents this during a cold start.

---

## 5. Image Build

### 5.1 Separate Dockerfiles

Two independent Dockerfiles, each producing one image:

| File | Image | Contents |
|---|---|---|
| `docker/backend.Dockerfile` | `<project>` | Go binary, templates, locales |
| `docker/frontend.Dockerfile` | `<project>-web` | Next.js standalone server |

### 5.2 Build Script

A single build script (`docker/build.py` or equivalent) builds **both** images
in one invocation. It supports:

- **Load mode** (default): builds for the host platform, loads into local
  Docker.
- **Push mode**: builds for multiple platforms, pushes to a registry.

### 5.3 Multi-Database Support

The backend Dockerfile **must not** compile out database drivers. The binary
ships with support for all supported databases (e.g. SQLite, PostgreSQL, MySQL).
Users select the database at runtime via configuration, not at build time.

This requires `CGO_ENABLED=1` and the corresponding C libraries (e.g.
`gcc`, `musl-dev`, `sqlite-dev`) in the build stage.

---

## 6. Backend Constraints

### 6.1 No Frontend Integration Code

The backend **must not** contain:

- Reverse proxy middleware that forwards requests to a frontend server.
- Static file serving for frontend assets.
- A `NoRoute` handler that proxies to a frontend server.
- A `frontend_url` configuration field.

### 6.2 Swagger Documentation

API documentation (Swagger/OpenAPI) is registered **only when debug mode is
enabled**. In production (`debug = false`), the route group is not registered.

### 6.3 CORS

In production, `allow_origins` **must not** be `["*"]`. It should be restricted
to the frontend's origin(s). Development environments may use `["*"]` for
convenience.

### 6.4 Trusted Proxies

When the frontend proxies requests to the backend, the backend receives
requests from the frontend container's IP. The `trusted_proxies` configuration
must include the Docker network subnet or the frontend container to correctly
resolve client IPs from `X-Forwarded-For` headers.

### 6.5 Database Auto-Migration

The backend runs schema migrations automatically on startup. No separate
migration command is required for normal operations. Value migrations (e.g.
adding a prefix to existing identifiers) are also performed at startup as
one-time transformations.

---

## 7. Frontend Constraints

### 7.1 Next.js Configuration

```ts
const nextConfig: NextConfig = {
  output: "standalone",

  async rewrites() {
    const target = process.env.API_PROXY_TARGET || "http://127.0.0.1:7330";
    return [
      { source: "/api/:path*", destination: `${target}/api/:path*` },
      // Project-specific value-prefixed routes
    ];
  },
};
```

- `output: "standalone"` is **required** for Docker deployment.
- No `basePath` — frontend routes live at the root.
- Rewrites use a code default for the proxy target, enabling zero-config local
  development. Production overrides `API_PROXY_TARGET` via environment variable.

### 7.2 Health Check Route

A minimal API route must exist for Docker health checks:

```ts
// src/app/health/route.ts
export function GET() {
  return new Response("ok", { status: 200 });
}
```

### 7.3 Frontend Dockerfile Pattern

```dockerfile
FROM node:<version>-alpine AS builder
RUN npm install -g pnpm
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN pnpm install
COPY . .
RUN pnpm build
RUN cp -r .next/standalone /app/dist && \
    cp -r .next/static /app/dist/.next/static && \
    cp -r public /app/dist/public

FROM node:<version>-alpine
WORKDIR /app
COPY --from=builder /app/dist ./
ENV HOSTNAME=0.0.0.0
EXPOSE 3000
CMD ["node", "server.js"]
```

---

## 8. Value-Prefix Convention

When the backend exposes routes that accept a single path-segment parameter at
the root level (not under `/api`), the parameter value **must** carry a
built-in prefix stored in the database.

### Rules

1. The prefix is **part of the stored value**, not added by the route handler.
   For example, when the database stores `item-abc123`, the route
   `GET /item-:id` matches `/item-abc123` and the handler receives the full
   value `item-abc123`.

2. No prefix stripping or synthesis at routing time. The handler queries the
   database with the exact value from the URL.

3. URL generation uses the stored value directly:
   `{public_url}/{stored_value}` — no prefix concatenation.

4. Migrating existing data: if the prefix is introduced after initial launch,
   a startup migration adds the prefix to all existing rows that lack it.

### Why

This convention eliminates ambiguity between API routes, frontend routes, and
resource routes — all without requiring a separate routing layer (nginx,
Traefik) or a `basePath` configuration on the frontend.

---

## 9. Ansible Deployment

### 9.1 Unified Templates

All environments (dev, staging, production) share the **same** Docker Compose
and config templates. Environment-specific values come exclusively from:

- **Host variables** (`host_vars/<host>.yml`): non-sensitive per-host settings
  (user, home path).
- **Vault files** (`vars/<env>/vault.yml`): secrets (signing keys, database
  passwords).

The templates themselves must not contain environment-specific logic
(`if env == "staging"`). If environments genuinely diverge, duplicate the
template files.

### 9.2 Playbook Pattern

Each environment owns one playbook. The playbook:

1. Creates directories on the target host.
2. Renders `docker-compose.yml` and application config from templates.
3. Stops existing containers.
4. Pulls images from the registry.
5. Starts containers.

No custom deployment logic beyond these steps.

---

## 10. Checklist — Applying This Spec to a New Project

1. Define the backend route table. Identify any root-level parameterized routes
   and assign value-prefixes to the corresponding database fields.
2. Create `docker/backend.Dockerfile` and `docker/frontend.Dockerfile`.
3. Create or update `docker/build.py` to build both images.
4. Set up `devops/docker-compose.yml` with backend + database (no frontend
   container — frontend runs locally via `pnpm dev`).
5. Update `devops/dev.py` to use the new compose file.
6. Configure `API_PROXY_TARGET` default in `next.config.ts` rewrites.
7. Update `next.config.ts` with rewrites driven by `API_PROXY_TARGET`.
8. Add `/health` API route in Next.js.
9. Place backend health check under the API prefix.
10. Remove all frontend integration code from the backend (proxy middleware,
    static file serving, `NoRoute` handler, `frontend_url` config).
11. Add Swagger debug-only guard.
12. Restrict CORS in production templates.
13. Create Ansible environment templates with the three-service compose file.
14. Add database variables to vault files.
