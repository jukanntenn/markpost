# markpost-mcp: AI Agent Integration

English | [中文](mcp.zh.md)

`markpost-mcp` is the standalone MCP server that lets AI agents (Claude Desktop, Cursor, ZCode, VS Code, …) operate a markpost instance: publish and read posts, manage delivery channels, and — for admins — run the full admin surface. The design reference is [specs/mcp/mcp-server.md](../specs/mcp/mcp-server.md).

## 1. Install

Pick one:

- **Release binary** — download `markpost-mcp-<os>-<arch>.tar.gz` from the [releases page](https://github.com/jukanntenn/markpost/releases) and put the binary on `PATH`.
- **Go install** — `go install github.com/jukanntenn/markpost/mcp/cmd/markpost-mcp@latest`.
- **Docker** — `docker run -e MARKPOST_MCP_URL=... -e MARKPOST_MCP_USERNAME=... -e MARKPOST_MCP_PASSWORD=... -p 127.0.0.1:8973:8973 jukanntenn/markpost-mcp` (the image serves the HTTP transport).

## 2. Configure an MCP host

The server authenticates as one markpost user with credentials from the environment (they are deliberately never flags, so they stay out of shell history). A stdio entry for `.mcp.json` / host config:

```json
{
  "mcpServers": {
    "markpost": {
      "type": "stdio",
      "command": "markpost-mcp",
      "args": ["stdio", "--url", "https://markpost.example.com"],
      "env": {
        "MARKPOST_MCP_USERNAME": "alice",
        "MARKPOST_MCP_PASSWORD": "…"
      }
    }
  }
}
```

Authentication is automatic: the server logs in at startup (bad credentials fail fast) and keeps its session alive — markpost rotates refresh tokens and the server tracks the rotation, refreshing on 401 and re-logging-in when the refresh is rejected. Users created via GitHub OAuth without a local password are not supported.

## 3. Toolsets

Default-enabled toolsets cover the publishing workflow: `posts` (create/list/get/delete), `delivery` (channels CRUD + test, history, stats), `account` (retention, sessions, post key, password). The `admin` toolset (28 tools mirroring `/api/v1/admin`) is opt-in:

```json
"args": ["stdio", "--url", "https://markpost.example.com", "--toolsets", "all"]
```

`--read-only` removes every write tool server-side — a safe hand-off for agents that should only read. Tool schemas are locked by snapshot tests, so tool surfaces change only deliberately.

## 4. Remote serving (HTTP transport)

`markpost-mcp http` serves the streamable-http transport at `127.0.0.1:8973/mcp` by default:

```bash
markpost-mcp http --url https://markpost.example.com --toolsets all \
  --addr 0.0.0.0:8973 --http-token "$(openssl rand -hex 32)"
```

Clients connect with `{"type": "http", "url": "https://mcp.example.com/mcp", "headers": {"Authorization": "Bearer …"}}`. Set `--http-token` whenever the listener leaves loopback; without it the endpoint is unauthenticated. MCP-native OAuth is deferred; a static bearer is the v1 protection.

## 5. Verify

```bash
MARKPOST_MCP_USERNAME=… MARKPOST_MCP_PASSWORD=… \
  markpost-mcp stdio --url https://markpost.example.com --toolsets all
```

Then ask the agent to list tools, or from another terminal send `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` to the running server's stdin. Every tool returns the backend's REST JSON verbatim; failures carry the backend's error code, message, and HTTP status.
