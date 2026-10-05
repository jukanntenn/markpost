# markpost architecture

English | [中文](architecture.zh.md)

Read this before changing the source tree. It is the ordered map of the codebase — components, their boundaries, and where new behavior goes; decision rationale lives in the linked RFCs.

## What this package is

markpost is a lightweight Markdown-to-HTML publishing service: an author posts Markdown over the HTTP API and gets a stable, pre-rendered HTML page backed by a render cache. It ships as one multi-arch Docker image (s6 supervises Caddy + Go), fronted by a Cloudflare CDN, and is consumed by the maintainer's own deployment plus a standalone CLI and an MCP server that drive the same API.

## Components

| Component   | Responsibility                                                                                       | Public surface                |
| ----------- | ---------------------------------------------------------------------------------------------------- | ----------------------------- |
| `backend/`  | The Go service: Gin HTTP handlers, GORM persistence, JWT auth, async delivery fan-out, OpenTelemetry | `/api/v1/*` REST              |
| `frontend/` | The admin/user console: Next.js 16 static export, React 19                                           | static bundle served by Caddy |
| `cli/`      | Standalone `markpost` client for scripting and agents                                                | `markpost <subcommand>`       |
| `mcp/`      | markpost-mcp MCP server exposing the API to models                                                   | MCP tools                     |
| `e2e/`      | Playwright chromium suite against the production-shaped compose                                      | `pnpm test`                   |
| `devops/`   | dev.py compose environment, Dockerfiles, ansible deploy                                              | `python3 devops/dev.py <cmd>` |
| `docker/`   | Production s6 image build                                                                            | `docker/build.py`             |
| `specs/`    | Current-state design reference                                                                       | —                             |
| `docs/`     | Operation guides and the documentation standard                                                      | —                             |
| `.agents/`  | Skills and RFCs driving the development loop                                                         | —                             |

## Where new behavior goes

Runtime behavior starts in `backend/internal/` — a new HTTP route adds a handler under `internal/web`, business rules land in `internal/service/<domain>`, and persistence contracts in `internal/repository` plus a migration. User-facing console work stays under `frontend/` within the static-export constraint. Cross-cutting contracts (API shapes, auth, delivery semantics) are specified in `specs/` and decided in RFCs under `.agents/rfcs/`; operational procedures live in `docs/`. The documentation rules themselves — the [documentation standard](AGENTS.md), the [bilingual documentation contract](i18n/README.md), and the [RFC rules](../.agents/rfcs/README.md) — own where prose goes.

The contributor entry points are in [development.md](development.md).
