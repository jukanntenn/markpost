# AGENTS.md — cli

Standalone Go module (`markpost/cli`) producing the `markpost` client: session handling, publishing, and an `api` passthrough, designed for humans and AI agents. Repo-wide orders live in the [root AGENTS.md](../AGENTS.md); rationale in the [CLI MRFC](../.agents/mrfcs/implemented/2026-09-03-standalone-agent-cli.md), current state in [specs/cli.md](../specs/cli.md). urfave/cli/v2 is the fixed framework; flags precede positional arguments.

## Commands (run in `cli/`)

- `go test ./...` — unit tests (no Docker; `internal/testserver` is an httptest fake backend)
- `go test -race ./...` — with the race detector
- `golangci-lint run ./...` — lint; `golangci-lint fmt` — format (no standalone gofmt/goimports)
- `make build` — binary with injected version
- `go test -tags acceptance ./acceptance` — e2e: execs the built binary against a real server; requires `MARKPOST_E2E_BASE_URL/USERNAME/PASSWORD` (dev stack: `python3 devops/dev.py start`, creds `markpost`/`markpost`), skips without them. Contains a sleep working around markpost#84.

## Layout

```
cmd/markpost/       entry shim → internal/cmd.Main
internal/cmd/       app assembly + one file per command group; Main owns error printing and exit codes
internal/cliapp/    Factory (lazy memoized closures) + FlagError/exit-code vocabulary
internal/api/       typed REST client: do/send core, 401→refresh→retry-once, wire types mirroring server DTOs
internal/config/    config.toml (XDG paths, 0600/0700) + MARKPOST_* env resolution
internal/iostreams/ injectable streams + TTY facts (prompts need both streams terminal)
internal/agentenv/  agent detection (gh's env conventions)
internal/testserver/ shared httptest fake backend (tests only)
acceptance/         build-tag-gated e2e
```

## Conventions

- Adding a command: new file in `internal/cmd`, constructor takes `*cliapp.Factory`, resolve dependencies via `f.Client()` / `f.AuthenticatedClient()` (auth-required), return plain errors — `Main` classifies (0/1/2/4; auth errors exit 4).
- Wire types in `internal/api/types.go` mirror the server by hand; server DTO changes need a matching edit. Never import the backend module.
- Tests assert exact output against `internal/testserver`; command tests run the whole app in-process via `Main` with `iostreams.Test()`. No new test doubles without a reason.
