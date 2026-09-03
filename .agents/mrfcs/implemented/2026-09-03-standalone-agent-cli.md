# MRFC: Standalone agent-first markpost CLI

English | [中文](2026-09-03-standalone-agent-cli.zh.md)

Status: implemented

## Problem

markpost's only clients were the web dashboard and raw curl. Scripts — and increasingly AI agents — need a stable, scriptable client: session management (JWT login, refresh, logout), publishing via post key, list/view/delete, and predictable machine-readable behavior. Every curl consumer re-implements token refresh, the paginated `{items, total, page, limit, total_pages}` envelope, and the `{code, message, errors}` error decoding; a hung interactive prompt or an unusable exit code is fatal to automation. markpost specifically wants AI agents as first-class API consumers, so the client must be agent-friendly by design, not by accident.

## Decision

`cli/` is a standalone Go module (`markpost/cli`) producing the `markpost` binary, distributed independently of the server image. It is built on `urfave/cli/v2` v2.27.4 — the CLI framework the backend already depends on — with gh (cli/cli, cloned under `.local/contexts/cli` as the golden reference) supplying the architecture patterns rather than the framework:

- **Factory + lazy closures** (`internal/cliapp`): `Config`, `SaveConfig`, `Client` are memoized function fields so `markpost version` touches neither disk nor network, and tests swap one closure.
- **IOStreams** (`internal/iostreams`): buffers + TTY facts (stdout/stdin) in one injectable object; prompts require both streams to be terminals, so non-interactive callers get flag errors instead of a hang. All v1 output is plain text; TTY decides compact vs indented `--json`.
- **Typed REST client** (`internal/api`): one `do`/`send` core injects the bearer token, retries exactly once after a 401 via `/api/v1/auth/refresh`, fires a `TokensChanged` callback that persists the new pair, and decodes the server error envelope. Wire types are re-declared here (mirroring the server DTOs) instead of importing the server module.
- **Config** (`internal/config`): single `config.toml` at `$MARKPOST_CONFIG_DIR` > `$XDG_CONFIG_HOME/markpost` > `~/.config/markpost`, written atomically 0600 in a 0700 directory. Environment overrides: `MARKPOST_SERVER`, `MARKPOST_TOKEN` (no refresh; a 401 is terminal), `MARKPOST_POST_KEY`. A stored session is only offered to the server that issued it.
- **Commands**: `auth login|status|token|logout`, `posts create|list|view|delete`, `post-key show|rotate`, `api` (gh-style passthrough; relative paths resolve under `/api/v1`), `status`, `config get|set`, `version`. Read commands accept `--json`.
- **Agent friendliness**: `internal/agentenv` detects the driving agent by gh's env conventions (`AI_AGENT` first) → full help instead of a terse usage line on errors, `Agent/<name>` User-Agent suffix, never a blocking prompt. Exit codes mirror gh: 0 ok, 1 error, 2 cancel, 4 auth (with a login hint).
- **urfave semantics adopted as-is**: flags must precede positional arguments (no interspersed parsing in v2); `App.OnUsageError` is applied to every subcommand explicitly because urfave copies it only onto the root.
- **Testing mirrors the reference's tiers**: package unit tests with exact-output assertions (testify `require`/`assert`) against a shared httptest fake backend (`internal/testserver`); command tests run the entire app in-process; acceptance tests (build tag `acceptance`) exec the compiled binary against a real server via `MARKPOST_E2E_*` env, skipping cleanly when unset.

## Alternatives considered

- **cobra (gh's framework)**: rejected — the backend already standardizes on urfave/cli/v2, and the task fixed the framework choice. gh's patterns (factory, iostreams, error taxonomy, agent detection) were ported; its framework was not.
- **`cli/` inside the backend Go module**: rejected — standalone distribution wants an isolated dependency graph; sharing the module would pull testcontainers and postgres into CLI builds and couple a distributed client to server internals. Re-declaring wire types is the same trade-off gh makes against the GitHub API.
- **httpmock-style transport registry for API tests**: rejected — an httptest.Server exercises real HTTP (URL building, headers, body encoding) with zero new dependencies, matching this repo's real-postgres testing ethos.
- **`--jq`/`--template` output (gh's json_flags)**: deferred — a jq engine is a new heavyweight dependency for little v1 value; `--json` piped into standard tooling covers agents today.
- **YAML config (gh's choice)**: rejected — the project's configuration dialect is TOML (backend viper config, devops compose env); consistency within markpost beat mimicking the reference.
- **Patching urfave for interspersed flags**: rejected — fighting the framework's parse loop buys cosmetic argument freedom at real maintenance cost; the flags-first convention is documented in every UsageText.

## Consequences

The CLI tracks the server's REST v1 contract by hand (`internal/api/types.go`); API changes need a matching CLI update, mitigated by the `api` passthrough covering new endpoints until typed commands exist. Refresh tokens sit plaintext in `config.toml` (like gh's hosts.yml fallback) with 0600/0700 permissions — no OS keyring in v1. Sessions deliberately do not leak across servers. The acceptance work surfaced a server bug — same-second logout/re-login mints a byte-identical, already-blacklisted JWT ([#84](https://github.com/jukanntenn/markpost/issues/84)) — the suite carries a documented sleep workaround until the server mints unique tokens. Verified: `go test ./...` and `-race` green, golangci-lint clean, acceptance green against the dev stack.
