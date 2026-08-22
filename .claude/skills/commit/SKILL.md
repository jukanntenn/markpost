---
name: commit
description: Use when the user asks to commit or stage changes (commit/stage/save/submit), when a task ends with dirty files to commit, or when multiple files should be split into logical commits.
---

# Commit

Group by logical change, not by file. Draft a plan, confirm, then execute. Never push, never amend.

1. `git status --porcelain` + `git log --oneline -5` for current changes and history style.
2. Separate AI-edited files from unrecognized ones; list unrecognized separately, never mix them in.
3. Group by logical unit (handler+service+repository, page+fetcher+types); order: `build/chore` → `feat` → `fix` → `refactor` → `style` → `docs` → `test`, `chore(release)` last.
4. Present the plan once; after confirmation run `git add` + `git commit` batch by batch. Rejected → stop, no second plan.
5. Verification is owned by prek — never run parallel lint/format/test commands by hand. `git commit` itself runs the pre-commit stage (fmt + lint + generated-files drift + agent-instructions-sync) and `commit-msg` checks the message; never `--no-verify`. To gate by hand first, from the repo root: `prek run --all-files` (CI's Lint gate). Tests are the push gate, not a commit gate — `prek run --stage pre-push --all-files` runs them on demand.
6. Single file → skip the plan, commit directly.

Message: `<type>(<scope>): <desc>` — lowercase, imperative, no trailing period. Types: `feat`/`fix`/`refactor`/`docs`/`test`/`chore`/`ci`/`build`/`style`/`perf`. Scopes: `backend`/`frontend`/`auth`/`post`/`delivery`/`admin`/`i18n`/`devops`/`ci`/`tooling`/`db` (omit for cross-cutting). Match the change's language.

- Generated files (`backend/docs/` Swagger, embedded CSS in `backend/internal/web/`, `go.sum`, `pnpm-lock.yaml`) bundle into the producing commit, or as a standalone `chore` — regenerate via `go generate ./...`, never hand-edit.
- Pair every GORM struct tag change with its migration file; migration + code stay in one commit.
- Same change across all locale files (`en`/`ja`/`zh-Hans`/`zh-Hant`), and language pairs (`foo.md` + `foo.zh.md`), one commit.
- Never silently include unrecognized files. Never amend, never push, never placeholder messages (wip, update files).
