# AGENTS.md — mcp

`markpost-mcp`, the standalone MCP server wrapping the markpost REST API for AI agents. Design: [specs/mcp/mcp-server.md](../specs/mcp/mcp-server.md); golden reference `github/github-mcp-server` (clone under `.local/contexts`). Repo-wide orders live in the [root AGENTS.md](../AGENTS.md); this file adds this tree's own.

- Independent Go module (`github.com/jukanntenn/markpost/mcp`): run Go commands from `mcp/`; it never imports `backend/` — the REST API is the only contract, mirrored in `internal/markpost`
- `go test ./... && go test --tags e2e ./e2e` — unit + e2e (e2e needs docker: postgres testcontainer, builds the real backend from `../backend`; `TESTCONTAINERS_SKIP=1` to skip)
- `go test ./internal/tools -run TestToolSnapshot -update` — regenerate the tool-surface snapshot after an intentional tool change
- New tool ⇒ update `internal/tools/testdata/tools.json` via the snapshot regen, plus success + error tests in the toolset's `_test.go`; DTO drift is caught by the e2e suite, which CI re-runs on `backend/**` changes
- golangci-lint v2 (fmt + run, config in `.golangci.yml`); go-sdk is the official `modelcontextprotocol/go-sdk` — check `.local/contexts/go-sdk` examples for idioms before inventing API usage
