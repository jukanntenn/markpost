# AGENTS.md

markpost is a Go (Gin/GORM) backend and a Next.js 16 + React 19 frontend, shipped as multi-arch Docker images. You are a senior pair-programming partner for this codebase: write secure, maintainable, performant code that matches the patterns already in the repo. Design and behavior rules live in [Conventions](#conventions); [`PRINCIPLES.md`](PRINCIPLES.md) is the frozen predecessor, kept until migration completes.

Subtree orders supplement this file and never repeat it: [`backend/AGENTS.md`](backend/AGENTS.md) (Go commands, migrations, testcontainers), [`frontend/AGENTS.md`](frontend/AGENTS.md) (pnpm, static export), [`cli/AGENTS.md`](cli/AGENTS.md) (standalone client module), [`e2e/AGENTS.md`](e2e/AGENTS.md) (Playwright, dagger), [`mcp/AGENTS.md`](mcp/AGENTS.md) (markpost-mcp, go-sdk, e2e). Read the one for the tree you are touching.

## Commands

Prefer the dev environment in containers over host services:

- `python3 devops/dev.py start` — backend + frontend + postgres in Docker Compose (`stop`, `logs [backend|frontend|postgres]`)
- `docker exec markpost-postgres psql -U markpost` — inspect the dev DB (postgres has no published port)
- `hdsh pairing verify`, `hdsh rfc verify`, `hdsh docs wrap|links|budgets`, `hdsh adopt verify` — the documentation and adoption gates (prek runs them on staged Markdown; CI runs the full corpus)

Backend, frontend, and e2e command blocks live in their `AGENTS.md` files linked above.

## Tech Stack

- **Frontend**: Next.js 16 (`output: "export"` static export), React 19, TypeScript, Tailwind CSS 4, Zustand, TanStack Query, next-intl, @base-ui/react, Prettier
- **Backend**: Go 1.26, Gin, GORM, JWT, Swagger (swag), Viper, OpenTelemetry
- **Database**: PostgreSQL 17 (the only supported database)
- **Testing**: Vitest (frontend unit), testcontainers-go + postgres (backend), Playwright chromium (e2e), httptest fake + tag-gated acceptance (cli)
- **Tooling**: golangci-lint v2 (lint+format), prek (pre-commit), air (Go hot reload), hdsh (harness gates)

## Project Structure

```
backend/           Go service — orders in backend/AGENTS.md
frontend/          Next.js static export — orders in frontend/AGENTS.md
cli/               standalone markpost client (own Go module) — orders in cli/AGENTS.md
mcp/               standalone markpost-mcp MCP server — orders in mcp/AGENTS.md
e2e/               Playwright workspace (own package.json) — orders in e2e/AGENTS.md
devops/            dev.py, docker-compose.yml, Dockerfiles, ansible/
docker/            production image (s6 multi-process), build.py
docs/              operation guides + the documentation standard (docs/AGENTS.md)
specs/             current-state design reference (index: specs/index.md)
.agents/           rfcs/ (decision records) + skills/
.github/workflows/ CI (lint/test/build/e2e with path filters)
scripts/           deployment, vault, and load-test tooling
```

## Conventions

Standing design rules, each 1–3 lines; this section is their live home as rules migrate out of the frozen [`PRINCIPLES.md`](PRINCIPLES.md).

- **Derive from essence, not incumbency.** An inherited name, wording, or arrangement is a data point, never authority — least of all when the change exists because that era was wrong. Two reviews defended incumbent terms (`ai_configs` aligned to an old hook label; "a decision record" over MRFC's own definition) while holding the essence in hand.

## Development Loop

Work flows issue-first: template-filed issues enter the board as `Inbox`; only the maintainer moves an issue to `Ready` (gate 1). The agent — a machine account via `GH_TOKEN` — claims `Ready` issues, decomposes by decision into RFCs, and drives two-phase PR stacks: RFC stack (gate 2), then implementation stack (gate 3), landed only through `gh stack merge`. RFC layers reference `Related to #N`; only a stack's top implementation layer carries `Fixes #N`. The [`merging-stacked-prs`](.agents/skills/merging-stacked-prs/SKILL.md) skill owns the mechanics; [the loop record](.agents/rfcs/implemented/process/2026-08-22-agent-driven-development-loop.md) holds the rationale.

## Git Workflow

- Conventional Commits, optional scope: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`, `build:`, `style:` — e.g. `fix(test): relax singleflight burst assertion`
- Commits are signed off by the author; do not commit on behalf of others
- `prek` runs on pre-commit (format + lint + agent-instruction sync + hdsh gates) and pre-push (tests); a `commit-msg` hook checks the commit format

## Documentation

[`docs/AGENTS.md`](docs/AGENTS.md) owns the standard: one fact, one home across tiers, current-state prose, machine-checkable links, and word budgets for the agent-instruction files. Documentation is bilingual — every doc pairs `foo.md` with `foo.zh.md` plus an `.i18n.yaml` record, equal authority, updating together ([pairing contract](docs/i18n/README.md)). New spec file ⇒ a row in [`specs/index.md`](specs/index.md) in the same change; placement decisions use the `documenting` skill.

## RFCs

Every non-trivial change adds or updates an RFC in the same PR ([`.agents/rfcs/README.md`](.agents/rfcs/README.md)) — grep `.agents/rfcs/` for the topic first; only mechanical/local edits are exempt.

## Run relevant checks locally

Hooks are worktree-local and intentionally narrow (`hdsh worktree install` installs them); CI owns exhaustive coverage and the platform matrix. Before pushing, run the checks owning your diff: the focused tests for changed behavior, `hdsh pairing verify` (re-record reviewed pairs with `hdsh pairing record <pair>`) plus `hdsh rfc verify` for decision records, the owning formatter/linter for touched trees, and `hdsh adopt verify` after adoption-related edits. The [pushing](.agents/skills/pushing/SKILL.md) skill carries the full inventory.

## Boundaries

- **Always**: read a file in full before editing it; run the tree's formatter/linter before finishing (golangci-lint in `backend/`, pnpm format+lint in `frontend/`).
- **Ask first**: database schema changes / migrations; new dependencies (`go get` / `pnpm add`); changes to CI workflows or Docker images.
- **Never**: edit generated files (Swagger docs in `backend/docs/`, lock files); commit secrets or `.env` files.

## Editing these instructions

This file loads in every agent session — keep it to standing orders and link everything else to its home. `CLAUDE.md` is a byte-identical copy with no primary: edit either file; [`scripts/sync_agent_instructions.py`](scripts/sync_agent_instructions.py) copies the newer side over the older and refuses to guess when both changed. `.claude/skills/` mirrors `.agents/skills/` through the same tooling. Word ceilings live in [`.hdsh/docs.manifest.json`](.hdsh/docs.manifest.json): relocate or condense before raising one, and justify any raise in the PR.
