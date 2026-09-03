# AGENTS.md

markpost is a Go (Gin/GORM) backend and a Next.js 16 + React 19 frontend, deployed as a single multi-arch Docker image. You are a senior pair-programming partner for this codebase: write secure, maintainable, performant code that matches the patterns already in the repo. Design and behavior rules live in [Conventions](#conventions); [`PRINCIPLES.md`](PRINCIPLES.md) is the frozen predecessor, kept until migration completes.

Subtree orders supplement this file and never repeat it: [`backend/AGENTS.md`](backend/AGENTS.md) (Go commands, migrations, testcontainers), [`frontend/AGENTS.md`](frontend/AGENTS.md) (pnpm, static export), [`cli/AGENTS.md`](cli/AGENTS.md) (standalone client module), [`e2e/AGENTS.md`](e2e/AGENTS.md) (Playwright, dagger). Read the one for the tree you are touching.

## Commands

Prefer the dev environment in containers over host services:

- `python3 devops/dev.py start` — backend + frontend + postgres in Docker Compose (`stop`, `logs [backend|frontend|postgres]`)
- `docker exec markpost-postgres psql -U markpost` — inspect the dev DB (postgres has no published port)
- `python3 scripts/doc_sync.py` — all documentation gates (prek runs them on staged Markdown; CI runs the full corpus)

Backend, frontend, and e2e command blocks live in their `AGENTS.md` files linked above.

## Tech Stack

- **Frontend**: Next.js 16 (`output: "export"` static export), React 19, TypeScript, Tailwind CSS 4, Zustand, TanStack Query, next-intl, @base-ui/react, Prettier
- **Backend**: Go 1.26, Gin, GORM, JWT, Swagger (swag), Viper, OpenTelemetry
- **Database**: PostgreSQL 17 (the only supported database)
- **Testing**: Vitest (frontend unit), testcontainers-go + postgres (backend), Playwright chromium (e2e), httptest fake + tag-gated acceptance (cli)
- **Tooling**: golangci-lint v2 (lint+format), prek (pre-commit), air (Go hot reload)

## Project Structure

```
backend/           Go service — orders in backend/AGENTS.md
frontend/          Next.js static export — orders in frontend/AGENTS.md
cli/               standalone markpost client (own Go module) — orders in cli/AGENTS.md
e2e/               Playwright workspace (own package.json) — orders in e2e/AGENTS.md
devops/            dev.py, docker-compose.yml, Dockerfiles, ansible/
docker/            production image (s6 multi-process), build.py
docs/              operation guides + the documentation standard (docs/AGENTS.md)
specs/             current-state design reference (index: specs/index.md)
.agents/           mrfcs/ (markpost's RFCs) + skills/ (mirrored from .claude/skills/)
.github/workflows/ CI (lint/test/build/e2e with path filters)
scripts/           documentation + agent-instruction gates
```

## Conventions

Standing design rules, each 1–3 lines; this section is their live home as rules migrate out of the frozen [`PRINCIPLES.md`](PRINCIPLES.md).

- **Derive from essence, not incumbency.** An inherited name, wording, or arrangement is a data point, never authority — least of all when the change exists because that era was wrong. Two reviews defended incumbent terms (`ai_configs` aligned to an old hook label; "a decision record" over MRFC's own definition) while holding the essence in hand.

## Development Loop

Work flows issue-first: template-filed issues enter the board as `Inbox`; only the maintainer moves an issue to `Ready` (gate 1). The agent — a machine account via `GH_TOKEN` — claims `Ready` issues, decomposes by decision into MRFCs, and drives two-phase PR stacks: RFC stack (gate 2), then implementation stack (gate 3), landed only through `gh stack merge`. RFC layers reference `Related to #N`; only a stack's top implementation layer carries `Fixes #N`. The [`dev-loop` skill](.agents/skills/dev-loop/SKILL.md) owns the mechanics; [the loop record](.agents/mrfcs/implemented/2026-08-22-agent-driven-development-loop.md) holds the rationale.

## Git Workflow

- Conventional Commits, optional scope: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`, `build:`, `style:` — e.g. `fix(test): relax singleflight burst assertion`
- Commits are signed off by the author; do not commit on behalf of others
- `prek` runs on pre-commit (format + lint + agent-instruction sync + doc gates) and pre-push (tests); a `commit-msg` hook checks the commit format

## Documentation

[`docs/AGENTS.md`](docs/AGENTS.md) owns the standard: one fact, one home across tiers, current-state prose, machine-checkable links, and word budgets for the agent-instruction files. Documentation is bilingual — every doc pairs `foo.md` with `foo.zh.md`, equal authority, updating together ([rule 7](docs/AGENTS.md)). New spec file ⇒ a row in [`specs/index.md`](specs/index.md) in the same change; placement decisions use the `doc-standards` skill.

## MRFCs

Every non-trivial change adds or updates an MRFC in the same PR ([`.agents/mrfcs/README.md`](.agents/mrfcs/README.md)) — grep `.agents/mrfcs/` for the topic first; only mechanical/local edits are exempt. The `writing-mrfcs` skill owns the workflow.

## Boundaries

- **Always**: read a file in full before editing it; run the tree's formatter/linter before finishing (golangci-lint in `backend/`, pnpm format+lint in `frontend/`).
- **Ask first**: database schema changes / migrations; new dependencies (`go get` / `pnpm add`); changes to CI workflows or Docker images.
- **Never**: edit generated files (Swagger docs in `backend/docs/`, lock files); commit secrets or `.env` files.

## Editing these instructions

This file loads in every agent session — keep it to standing orders and link everything else to its home. `CLAUDE.md` is a byte-identical copy with no primary: edit either file; [`scripts/sync_agent_instructions.py`](scripts/sync_agent_instructions.py) copies the newer side over the older and refuses to guess when both changed. Word ceilings live in [`scripts/doc_budgets.manifest.json`](scripts/doc_budgets.manifest.json) ([`verify_doc_budgets.py`](scripts/verify_doc_budgets.py)): relocate or condense before raising one, and justify any raise in the PR.
