# ZCode Hooks

Project-local hook scripts for the ZCode agent (same multi-language pipeline as
the Claude/Codex hooks — dispatch matches `prek.toml` and CI `lint.yml`).

- `hooks/post_tool_use.py` — PostToolUse (`Edit|Write`): formats the edited file
  by extension — Go (`gofmt`+`goimports`), frontend/root (`prettier`), `.toml`
  (`oxfmt`), `.j2` (`djlint`), `Caddyfile` (`caddy fmt`), `.py`/`.pyi`
  (`ruff check --fix` + `ruff format`). Never blocks; missing tools are skipped.
- `hooks/stop.py` — Stop: full-tree gate running `golangci-lint run` (backend) +
  `pnpm lint` + `pnpm typecheck` (frontend); on error it prints
  `{"decision":"block","reason":"..."}` (once per turn, guarded by the
  `stopHookActive` flag). ZCode caps Stop continuations at 3 natively.

## Why there is no `.zcode/config.json` here

The ZCode client UI and official docs support workspace-scope hooks in
`.zcode/config.json`, but the agent runtime on this machine (v2.1.0, WSL server)
**strips them unconditionally** — a "security policy" warning
(`config_project_hooks.ignored`) is logged and no hook runs. Only hooks in the
**user-level** `~/.zcode/cli/config.json` are executed (verified in source and
by log evidence).

The user-level config therefore points at these scripts via
`${ZCODE_PROJECT_DIR}` and guards on file existence, so other workspaces without
this directory are unaffected. Changing the runtime behavior would require a
ZCode update that honors workspace hooks.
