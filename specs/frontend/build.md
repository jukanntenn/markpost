# Frontend Build Specification

English | [中文](build.zh.md)

This document defines the frontend build configuration: pure static export (`output: "export"`), the Turbopack bundler, the server capability boundary, the API address strategy, and the artifact layout. The frontend architecture lives in [architecture.md](./architecture.md).

## 1. Build configuration

### 1.1 Pure static export

`next.config.ts`:

```ts
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export",
};

export default nextConfig;
```

`output: "export"` produces a purely static `out/` directory (HTML/CSS/JS) with **no Node.js runtime**.

> Source basis (Next.js v16.1.6): the `output` option accepts `'standalone' | 'export'` (`config-shared.ts:1258-1266`); `'export'` emits the `out` directory (`build/index.ts:950`, default directory name `'out'`).

### 1.2 Why pure static export

All markpost business logic lives in the Go backend; the frontend needs no API Routes, SSR proxy, or Server Actions. The advantages of pure static export:

| Advantage            | Detail                                                                                                                    |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Zero Node memory     | No resident Node process (saves 50-150MB+); memory goes to the Go backend + PostgreSQL (fits the 2C/2G hardware envelope) |
| Trivial deployment   | Static files drop onto any static server / CDN — `nginx root /var/www/out;` suffices                                      |
| Zero cold-start cost | The CDN edge answers directly; no Node cold start                                                                         |
| Image reuse          | The same `out/` deploys to any same-origin topology (consistent with the image reuse in the docker build spec)            |

## 2. Turbopack

### 2.1 The default bundler (Next.js 16)

From Next.js 16 on, **Turbopack is stable and enabled by default**; both `next dev` and `next build` use it. No `--turbopack` flag is needed.

> Source basis (Next.js v16 docs, `version-16.mdx:92`): "Starting with Next.js 16, Turbopack is stable and used by default with `next dev` and `next build`."

### 2.2 Configuration rules

- **No `webpack` field in `next.config.ts`** — under Next.js 16, configuring a webpack field makes the Turbopack build fail outright (`version-16.mdx:108`, "the build will fail to prevent misconfiguration issues")
- **No `turbopack` configuration block needed** — zero-config out of the box (JS/TS via SWC, CSS via Lightning CSS, CommonJS/ESM)
- A custom loader for a custom file type, if ever needed, goes through `turbopack.rules` (native Turbopack) rather than a webpack loader
- Falling back to Webpack requires an explicit `--webpack` opt-out (this project does not use it)

### 2.3 Turbopack works with pure static export

Turbopack is fully compatible with `output: "export"`. The Turbopack documentation does not list static export as a limitation.

## 3. Server capability boundary

A pure static export runs no server runtime, so the following Next.js server capabilities are unavailable across the board (build-time error or missing from the artifacts):

| Capability                                                      | Why unavailable                           | Alternative                                                                                                         |
| --------------------------------------------------------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| middleware / proxy (`NextResponse.rewrite` forwarding `/api/*`) | static export does not support middleware | `next.config.ts` rewrites in development; Caddy reverse proxy in production                                         |
| API Routes (Route Handlers, e.g. a health route)                | static export emits no route handlers     | the backend `/api/v1/health` endpoint                                                                               |
| Server-side i18n assembly (`getRequestConfig` + `cookies()`)    | a server API; static export runs none     | pure client-side assembly (`NextIntlClientProvider` receives locale + messages directly) — see [i18n.md](./i18n.md) |

## 4. API address strategy

### 4.1 Relative paths plus a reverse proxy

The frontend API client sends **relative paths** (`/api/v1/...`), and the reverse proxy (Nginx/Caddy) forwards them to the Go backend in deployment.

```
browser → Nginx → static files (out/) + /api/* forwarded to the Go backend
```

The build artifact `out/` is **fully decoupled** from the backend address — the same image deploys to any same-origin topology.

### 4.2 No NEXT_PUBLIC_ environment variables

The Next.js documentation is explicit (`environment-variables.mdx:152`): `NEXT_PUBLIC_` variables are inlined into the JS bundle at `next build` time and **frozen at build time, immutable at runtime**. Changing the backend address would force a rebuild, which conflicts with Docker image reuse — hence no `NEXT_PUBLIC_API_BASE_URL`.

The relative-path scheme leaves the backend address to the reverse-proxy configuration at deploy time, decoupled from the build artifact.

## 5. Build artifacts

### 5.1 Artifact directory

`out/` (the Next.js default, `build/index.ts:950`).

### 5.2 Routing: fully static prerendering

Every route prerenders to one HTML file (`/login` → `out/login.html`, `/dashboard` → `out/dashboard.html`). Each route has its own HTML, so the first paint is fast. All pages are `'use client'` interactive components.

### 5.3 Fonts: next/font self-hosting

The built-in self-hosting of `next/font/google` **works fully** under pure static export — font files download at build time and embed into `out/`; the runtime sends no Google Fonts request.

> Source basis (Next.js docs, `fonts.mdx`): "built-in self-hosting for any font file... removes external network requests for improved privacy and performance."

## 6. package.json scripts

```json
{
  "scripts": {
    "dev": "next dev --port ${FRONTEND_PORT:-3034}",
    "build": "next build",
    "serve": "npx serve out -p ${FRONTEND_PORT:-3034}",
    "lint": "eslint .",
    "typecheck": "tsc --noEmit -p tsconfig.check.json",
    "test": "vitest",
    "test:ui": "vitest --ui",
    "test:run": "vitest run",
    "format": "prettier --write .",
    "format:check": "prettier --check ."
  }
}
```

| Script                    | Purpose                               |
| ------------------------- | ------------------------------------- |
| `dev`                     | dev server (Turbopack by default)     |
| `build`                   | production build (emits `out/`)       |
| `serve`                   | local preview of the static artifacts |
| `lint`                    | ESLint                                |
| `typecheck`               | `tsc --noEmit` type check             |
| `test` / `test:run`       | Vitest (watch mode / single run)      |
| `test:ui`                 | Vitest UI                             |
| `format` / `format:check` | Prettier (write / check)              |

The scripts object carries no `start` entry: a static export has no Node runtime for `next start` to serve.

## 7. Capability boundary

Feature availability under pure static export:

| Feature                                     | Available | Note                                               |
| ------------------------------------------- | --------- | -------------------------------------------------- |
| Static HTML/CSS/JS                          | ✅        | `out/` fully static                                |
| `next/font` self-hosted fonts               | ✅        | embedded at build time                             |
| Client-side routing (`next/link`, `router`) | ✅        | SPA-style navigation                               |
| Client-side data fetching (fetch)           | ✅        | direct backend API (relative path + reverse proxy) |
| Turbopack bundling                          | ✅        | the default bundler                                |
| middleware / Proxy                          | ❌        | unsupported (pure static export)                   |
| API Routes (Route Handlers)                 | ❌        | unsupported                                        |
| Server Actions                              | ❌        | unused                                             |
| Server Components (data fetching)           | ❌        | static rendering only, no runtime                  |
| ISR (incremental static regeneration)       | ❌        | fully static                                       |
| `NEXT_PUBLIC_` runtime variables            | ⚠️        | frozen at build time; unused                       |

## 8. Deployment shape

Static files from `out/` plus a reverse proxy (Nginx/Caddy/CDN):

- The reverse proxy / CDN serves the static files directly
- The reverse proxy forwards `/api/*` to the Go backend
- **No Node.js runtime** — memory saved (fits the 2C/2G hardware envelope of [caching.md](../backend/caching.md))

See [cloudflare.md](../backend/cloudflare.md) for the three deployment modes (SaaS / self-hosted / homelab) and [docker/build-specification.md](../docker/build-specification.md).

## References

- [architecture.md](./architecture.md) — frontend architecture, provider stack, API client
- [i18n.md](./i18n.md) — pure client-side i18n assembly
- [routes.md](./routes.md) — route structure
