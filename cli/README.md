# markpost CLI

English | [简体中文](README.zh.md)

The official command-line client for a [markpost](../README.md) server — a standalone binary (`cli/` is its own Go module, built on urfave/cli/v2) designed to be driven by both humans and AI agents.

```bash
cd cli && make build          # or: go build -o markpost ./cmd/markpost
./markpost config set server https://mp.example.com
./markpost auth login          # prompts on a terminal; flags for scripts
./markpost posts create --title "Hello" "# Hello World

Some **markdown**."
# → https://mp.example.com/p-AbCdEf...
```

## Commands

| Command                                | Purpose                                                                                                                   |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `auth login / status / token / logout` | session lifecycle; login via prompts, flags, `--password-stdin`, or `--token`                                             |
| `posts create / list / view / delete`  | publish (prints the post URL), list (table or `--json`), view (`--format raw/html/url`), delete (`--yes` in scripts)      |
| `post-key show / rotate`               | the publishing key                                                                                                        |
| `api <endpoint>`                       | gh-style passthrough for anything typed commands do not cover; relative paths hit `/api/v1` (`markpost api me/retention`) |
| `status`                               | server health / readiness / version (`--json`)                                                                            |
| `config get/set`                       | local settings (`server`)                                                                                                 |
| `version`                              | CLI version                                                                                                               |

## Agent- and script-friendly by design

- **Never blocks**: prompts appear only when stdin and stdout are both terminals; everywhere else, actionable flag errors. `--file -` and `--password-stdin` accept piped input.
- **Machine-readable output**: results on stdout, diagnostics on stderr; read commands support `--json` (compact when piped, indented on a TTY).
- **Stable exit codes** (gh's contract): `0` success, `1` failure, `2` canceled, `4` authentication — auth failures print a login hint.
- **Agent detection**: with `AI_AGENT` (or a known agent's env vars) set, usage errors print the full help so an agent can self-correct in one round trip; the User-Agent gains `Agent/<name>`.
- **Automatic session refresh**: a 401 triggers one token refresh, persisted to the config file, then a retry.

## Configuration

Session and settings live in `config.toml` (0600) at `$MARKPOST_CONFIG_DIR`, else `$XDG_CONFIG_HOME/markpost`, else `~/.config/markpost`. Environment variables override the file:

| Variable            | Effect                                                        |
| ------------------- | ------------------------------------------------------------- |
| `MARKPOST_SERVER`   | server base URL (same as the `--server` flag)                 |
| `MARKPOST_TOKEN`    | access token for headless use — no refresh; a 401 is terminal |
| `MARKPOST_POST_KEY` | publish without a login session                               |

## Development

```bash
make test         # unit tests (httptest fake backend; no Docker)
make test-race
make lint
make acceptance   # e2e: needs MARKPOST_E2E_BASE_URL/USERNAME/PASSWORD, skips otherwise
```

Design: [specs/cli.md](../specs/cli.md) · decisions: [MRFC](../.agents/mrfcs/implemented/2026-09-03-standalone-agent-cli.md) · tree orders: [AGENTS.md](AGENTS.md)
