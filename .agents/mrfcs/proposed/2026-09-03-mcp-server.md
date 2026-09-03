# MRFC: MCP server for AI agent integration

Status: proposed

English | [中文](2026-09-03-mcp-server.zh.md)

## Problem

markpost's product contract is an HTTP API, and the clients it serves are humans (dashboard) and scripts (curl). AI agents are now first-class operators: an agent that can _publish_, _read back_, _manage delivery_, and _administer_ markpost turns the instance into an agent-reachable publishing endpoint — the integration surface the product wants. But agents do not speak REST natively; they speak MCP (tools with schemas, listed and invoked over stdio or streamable-http). Nothing in the repo serves that: the only MCP mention is third-party coding-agent tooling in `.mcp.json`. An MCP server must be built, and three of its shape decisions are load-bearing: whether it couples to the backend or wraps its public contract; how it authenticates given markpost has no personal access tokens and 24h access-token expiry; and how a 47-tool surface (the admin mirror is inherently destructive) is gated for safety. The reference the industry converged on — github/github-mcp-server — answers the same three questions for api.github.com and is the alignment target for architecture and test shape alike.

## Proposal

`mcp/` will be a standalone Go module (`github.com/jukanntenn/markpost/mcp`) that wraps the REST API with a thin typed HTTP client and imports nothing from the backend — the github-mcp-server relationship to api.github.com, enabling independent build and release. The MCP layer will be the official `modelcontextprotocol/go-sdk` v1.7.0 (the golden reference's pin); tools are typed handlers whose JSON Schemas derive from argument structs, and their output is the backend's REST JSON verbatim. Authentication uses username/password credentials from `MARKPOST_MCP_*` environment (never flags) with fully automatic session maintenance: eager login at startup, refresh-token rotation tracked by the client, single-flight 401 recovery (refresh, then re-login) under the client mutex because concurrent refreshes would trip the backend's token-theft detection; `change_my_password` adopts the fresh pair the backend returns. The tool surface is four toolsets — `posts`, `delivery`, `account` default-on; `admin` (28 tools mirroring `/api/v1/admin`) opt-in — plus a `--read-only` mode that removes every write tool at registration time (server-side guarantee). Transports: `stdio` for local hosts, `http` (stateless streamable-http, optional constant-time bearer guard) for remote serving; CLI is urfave/cli v2 like the backend. Testing mirrors the golden reference: httptest contract tests per tool, in-memory-session tool tests, a tool-schema snapshot locking the 47-tool surface, and a `--tags e2e` suite that starts a postgres testcontainer, builds the real backend, and drives the real binary over stdio. Distribution rides the product's release tag: cross-compiled binaries on the GitHub release, `jukanntenn/markpost-mcp` multi-arch image, and `go install` from source. Full design: [specs/mcp/mcp-server.md](../../../specs/mcp/mcp-server.md); operation guide: [docs/mcp.md](../../../docs/mcp.md).

## Alternatives considered

**Embed in the backend, share the service layer.** No duplicated DTOs and no HTTP hop — but the server could not be distributed or versioned independently of the backend, could not point at a remote instance, and every backend refactor would ripple into the agent surface. Rejected: standalone distribution is the requirement, and the REST API is already the stable contract.

**mark3labs/mcp-go instead of the official SDK.** The community SDK predates the official one; github-mcp-server itself migrated from it to `modelcontextprotocol/go-sdk`. Rejected for a greenfield v1: the golden reference and the protocol's own examples are written against the official SDK, and alignment with them is the project's stated bar.

**Static access token in the environment.** Matches the golden reference's PAT-in-env shape most literally — and breaks every 24h by default, making the tool unusable day-to-day. Credential login plus rotation tracking is the honest equivalent of a long-lived PAT for a token system that has none. Adding a real PAT concept to the backend was rejected as scope creep for v1 (a candidate future MRFC).

**Ship markpost-mcp inside the main image as an s6 service.** Zero extra deployment for self-hosters, but it couples the agent surface's release to the product image and forces an in-image auth story. Deferred to a future MRFC; v1 ships the standalone image and binaries.

## Acceptance criteria

- A configured MCP host lists the default toolsets' tools and can `create_post` → `get_post` → `delete_post` against a live instance; tool outputs are the backend's REST JSON verbatim and failures carry the backend's error code, message, and HTTP status.
- The session survives a full day unattended: with the access token expired server-side, the next tool call transparently refreshes and succeeds; with the refresh token also revoked, it re-logs-in and succeeds.
- `--toolsets admin` registers the 28 admin tools; without it (and under `--read-only`) no write tool is listed — verified by ListTools assertions, not client behavior.
- `internal/tools/testdata/tools.json` matches the shipped tool surface; regenerating it without an intentional change fails the snapshot test.
- `go test ./... && go test --tags e2e ./e2e` passes from `mcp/` — the e2e run starts postgres, builds the backend, and drives the real binary over stdio; CI runs both on `mcp/**` and `backend/**` path filters.
- A release tag attaches `markpost-mcp-<os>-<arch>` archives to the GitHub release and publishes the multi-arch `jukanntenn/markpost-mcp` image.

## Risks

A hand-maintained REST client must track backend DTO changes — the e2e path filter on `backend/**` is the tripwire, re-running the suite against the real server when the contract moves; a subtle contract drift that the fakes mirror incorrectly could pass both sides. Passwords live in agent-host environments with the same trust model the golden reference grants a PAT — a compromised host environment leaks a reusable credential, not a scoped one. OAuth-only users cannot use the server until MCP-native OAuth or a backend PAT concept lands. The admin toolset, even opt-in, gives agents destructive reach (delete users/posts/channels, bulk retention) that deployers must consciously enable; a misconfigured `--toolsets all` on an admin credential is a big blast radius. Known follow-ups deliberately out of scope: MCP-native OAuth for the HTTP transport, a backend PAT concept, in-image s6 bundling.
