# CLI Specification

English | [中文](cli.zh.md)

Current-state design of the `markpost` CLI (`cli/`, standalone Go module `markpost/cli`). Rationale lives in the [standalone agent CLI MRFC](../.agents/mrfcs/implemented/2026-09-03-standalone-agent-cli.md).

## Scope and stack

A client for one markpost server per config file: session handling, publishing, and generic API passthrough — designed to be driven by humans and AI agents. Framework: `urfave/cli/v2` (the backend's framework; cobra-style interspersed parsing does not apply — flags precede positional arguments). Architecture follows gh (cli/cli): a lazy `Factory` of dependencies (`internal/cliapp`), `IOStreams` (`internal/iostreams`), and a typed REST client (`internal/api`) over one request core.

## Commands

| Command                    | What it does                                                                                                | Auth                                                                                     |
| -------------------------- | ----------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `auth login`               | username/password (flags, prompts, or `--password-stdin`) or `--token`; stores the session                  | —                                                                                        |
| `auth status`              | server, session, expiry; verifies via `/api/v1/me/retention`                                                | optional                                                                                 |
| `auth token`               | prints the access token                                                                                     | session                                                                                  |
| `auth logout`              | revokes server-side (best effort), clears local credentials, keeps the server configured                    | session                                                                                  |
| `posts create`             | publishes markdown (`--file <path                                                                           | ->`, positional args, or piped stdin) and prints the post URL; `--json`gives`{qid, url}` | post key (flag/env) or session lookup |
| `posts list`               | paginated table or `--json` envelope; `--search/--page/--limit`                                             | session                                                                                  |
| `posts view <qid>`         | `--format raw` (default), `html`, or `url`                                                                  | — (public route)                                                                         |
| `posts delete <qid>`       | `--yes` required when non-interactive; prompts on a TTY                                                     | session                                                                                  |
| `post-key show` / `rotate` | prints the key (rotate invalidates the old one immediately)                                                 | session                                                                                  |
| `api <endpoint>`           | passthrough; relative paths resolve under `/api/v1`, absolute paths verbatim; `-X <method>`, `--input <file | ->`; body printed verbatim                                                               | session when present                  |
| `status`                   | health + readiness + version; `--json`                                                                      | —                                                                                        |
| `config get/set`           | local keys: `server`                                                                                        | —                                                                                        |
| `version`                  | CLI version                                                                                                 | —                                                                                        |

## Session and configuration

- Config file: `config.toml`, resolved `$MARKPOST_CONFIG_DIR` > `$XDG_CONFIG_HOME/markpost` > `~/.config/markpost`, written atomically 0600 in a 0700 directory. Fields: server, user identity, access token, refresh token, expiry.
- Environment: `MARKPOST_SERVER` (also the `--server` global flag's env), `MARKPOST_TOKEN` (session without refresh — a 401 is terminal), `MARKPOST_POST_KEY`.
- Precedence: `--server` flag > `MARKPOST_SERVER` > stored server; `MARKPOST_TOKEN` > stored token. A stored session is offered only to its own server.
- On a 401 with a stored refresh token, the client refreshes once, persists the new pair, and retries the original request; unrecoverable cases surface as auth errors.
- `auth login --token` verifies the token before storing it.

## Output and exit codes

- stdout carries results (parseable); diagnostics go to stderr. Read commands accept `--json` — compact when piped, indented on a TTY.
- Exit codes (gh-aligned): `0` success, `1` any failure, `2` cancellation, `4` authentication — printed with a login hint.
- Agent detection (`internal/agentenv`, gh's env conventions, `AI_AGENT` first) switches usage errors to full help and adds `Agent/<name>` to the User-Agent. Prompts never block non-interactive callers.

## Testing

- Unit (default `go test ./...`): per-package tests with exact-output assertions against the shared httptest backend in `internal/testserver`; command tests run the whole app in-process with buffer streams.
- Acceptance (`go test -tags acceptance ./acceptance`): execs the compiled binary against a real server; requires `MARKPOST_E2E_BASE_URL/USERNAME/PASSWORD`, skips otherwise. Carries a documented sleep working around [#84](https://github.com/jukanntenn/markpost/issues/84) (same-second logout/re-login blacklists the reissued token).
