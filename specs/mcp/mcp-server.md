# MCP Server Specification

English | [中文](mcp-server.zh.md)

This document specifies `markpost-mcp`: the standalone MCP (Model Context Protocol) server that exposes a running markpost instance to AI agents. It is a terminal design reference; the decision rationale lives in [the MCP server MRFC](../../.agents/mrfcs/implemented/2026-09-03-mcp-server.md).

## 1. Architecture

`markpost-mcp` is an independent Go module at `mcp/` (`github.com/jukanntenn/markpost/mcp`, its own `go.mod`) that talks to a markpost instance exclusively through its public contract — the REST API at `/api/v1` plus the two public post endpoints. It imports nothing from the backend module and links none of its packages, which is what allows independent build, independent release, and pointing one binary at any instance, local or remote. The architecture mirrors the golden reference [github/github-mcp-server](https://github.com/github/github-mcp-server): a thin typed API client (the analog of go-github) wrapped by MCP tool handlers that return the backend's JSON verbatim as text content.

The MCP layer is built on the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) v1.7.0 (protocol revisions through 2025-06-18/2026-07-28), the same SDK and pin the golden reference uses. Tools are declared with typed handler signatures; the SDK derives each tool's JSON Schema from the handler's arguments struct, with `jsonschema` struct tags carrying parameter descriptions.

## 2. Toolsets

Tools are grouped into four toolsets, each mirroring one area of the REST surface. Registration validates every requested name before registering anything, so a typo fails at startup without leaving a partially populated server. `--toolsets` (or `MARKPOST_MCP_TOOLSETS`) selects the enabled sets; `all` expands to every set.

| Toolset    | Default | Tools                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| ---------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `posts`    | on      | `create_post`, `list_posts`, `get_post`, `delete_post`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `delivery` | on      | `list_channels`, `create_channel`, `update_channel`, `delete_channel`, `test_channel`, `list_delivery_history`, `list_latest_deliveries`, `get_delivery_stats`, `list_pending_deliveries`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `account`  | on      | `get_my_retention`, `list_my_sessions`, `revoke_my_session`, `revoke_my_other_sessions`, `rotate_post_key`, `change_my_password`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `admin`    | off     | `admin_list_users`, `admin_get_user`, `admin_create_user`, `admin_delete_user`, `admin_set_user_role`, `admin_reset_user_password`, `admin_set_user_active`, `admin_set_user_vip`, `admin_set_user_retention`, `admin_bulk_set_retention`, `admin_retention_impact`, `admin_get_retention_defaults`, `admin_list_user_sessions`, `admin_revoke_user_sessions`, `admin_revoke_session`, `admin_list_posts`, `admin_delete_post`, `admin_list_channels`, `admin_create_channel`, `admin_set_channel_enabled`, `admin_delete_channel`, `admin_list_delivery_history`, `admin_get_delivery_stats`, `admin_list_locked_channels`, `admin_list_audit_logs`, `admin_get_stats`, `admin_get_settings`, `admin_set_setting` |

Forty-seven tools in total. Naming follows the golden reference's `verb_noun` convention; admin tools carry the `admin_` prefix matching the REST `/admin` namespace. The admin toolset is opt-in because most credentials are not admin and its tools are the most destructive surface; the three default toolsets cover the publishing workflow an agent needs.

`--read-only` (or `MARKPOST_MCP_READ_ONLY`) removes every write tool at registration time — a server-side guarantee, not a client-side promise. Read tools carry `ReadOnlyHint: true`; destructive tools carry `DestructiveHint: true`.

`create_post` resolves the caller's post key via the authenticated `GET /api/v1/post-key`, then submits `{title, body}` to the public `POST /{post_key}` endpoint (markpost's only creation path) and returns `{id, url}` with the render URL composed from the instance base URL. `get_post` fetches `GET /{qid}?format=raw`, returning the markdown source (`# title` + body) rather than rendered HTML.

## 3. Authentication

markpost has no personal access tokens, and access tokens expire after 24h by default, so the MCP server authenticates with username/password credentials from the environment (`MARKPOST_MCP_USERNAME` / `MARKPOST_MCP_PASSWORD` — environment-only by design, never flags, keeping credentials out of shell history) and keeps its session alive automatically:

- On startup the client logs in eagerly, so a bad URL or credential fails fast instead of at the first tool call.
- markpost rotates refresh tokens; the client persists each new pair returned by `/auth/refresh`.
- A 401 from any authenticated call triggers one recovery pass under the client mutex: refresh with the current refresh token; if the refresh is itself rejected, a full re-login. The single-flight comparison (on the failed access token) prevents concurrent refreshes — repeated use of one refresh token trips the backend's token-theft reuse detection.
- `change_my_password` adopts the fresh token pair the backend returns, so the MCP session survives its own password change.

Known limitation: users created via GitHub OAuth without a local password cannot authenticate; MCP-native OAuth for the HTTP transport is deferred to a future MRFC.

## 4. Transports and configuration

The CLI is `urfave/cli/v2` (the backend's framework), with two subcommands. Flags repeat on each subcommand (urfave v2 does not propagate app-level flags) and bind to the `MARKPOST_MCP_*` environment:

| Flag                  | Environment               | Default                  | Meaning                                      |
| --------------------- | ------------------------- | ------------------------ | -------------------------------------------- |
| `--url`               | `MARKPOST_MCP_URL`        | — (required)             | Instance base URL                            |
| `--toolsets`          | `MARKPOST_MCP_TOOLSETS`   | `posts,delivery,account` | Enabled toolsets (comma-separated, or `all`) |
| `--read-only`         | `MARKPOST_MCP_READ_ONLY`  | false                    | Drop every write tool                        |
| `--addr` (http)       | `MARKPOST_MCP_HTTP_ADDR`  | `127.0.0.1:8973`         | HTTP listen address                          |
| `--path` (http)       | `MARKPOST_MCP_HTTP_PATH`  | `/mcp`                   | HTTP endpoint path                           |
| `--http-token` (http) | `MARKPOST_MCP_HTTP_TOKEN` | —                        | Bearer token MCP clients must present        |

`stdio` runs the server over stdin/stdout (the transport for local MCP hosts); it logs only to stderr. `http` serves the streamable-http transport in stateless mode (`mcp.NewStreamableHTTPHandler`, the golden reference's remote-server mode), optionally guarded by a constant-time bearer check; with no token configured the listener stays on loopback by default and exposure is the publisher's choice.

Tool output is the backend's REST JSON verbatim (indented) as text content; tool errors carry the backend's own error code and message plus the HTTP status, so agents see markpost's semantics, not client-side paraphrase.

## 5. Testing

Three layers, aligned with the golden reference:

- **Unit** — the REST client is contract-tested against an `httptest` fake that records requests (login/refresh/retry sequences, query building, error mapping); every tool is exercised through an in-memory MCP session (`mcp.NewInMemoryTransports`) with success and error paths.
- **Tool snapshot** — `internal/tools/testdata/tools.json` locks the full tool surface (names, descriptions, annotations, input schemas); any change is a deliberate, reviewed diff (regenerate with `-update`).
- **E2E** — `mcp/e2e` behind the `--tags e2e` build flag (mirroring the golden reference's gate): starts a postgres testcontainer, builds and runs the real backend from `backend/` (migrations, seeded admin), launches the real `markpost-mcp` binary over stdio, and drives every toolset through an SDK client.

## 6. Distribution

One product version and release cadence: release tags attach cross-compiled binaries (linux/darwin × amd64/arm64, windows amd64; CGO disabled) to the GitHub release and publish the standalone `jukanntenn/markpost-mcp` Docker image (multi-arch, `http` transport as entrypoint). `go install github.com/jukanntenn/markpost/mcp/cmd/markpost-mcp@latest` installs from source. Operation guides: [docs/mcp.md](../../docs/mcp.md).
