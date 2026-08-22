# MRFC: prek is the single source of format and lint

Status: implemented

English | [中文](2026-08-12-prek-single-source-of-format-and-lint.zh.md)

## Problem

Formatter and linter invocations were defined in parallel: AI post-edit hooks ran one set of commands, CI another, and developers' muscle memory a third. The definitions drifted — a bare `prettier` entry depended on an ambient binary that only some machines had, and "run the linter" meant different things in different contexts, so formatting fights and lint skips reproduced locally.

## Decision

Every check the pipeline runs is declared exactly once, in prek: `prek.toml` at the root (builtin read-only checks, formatters, deploy-file gates, Conventional Commits), `backend/prek.toml` (Go fmt/lint/generate gates), and `frontend/prek.toml` (ESLint/Prettier). The invocation contracts are fixed: AI post-edit hooks run `prek run --group fmt --files <edited>`, the AI Stop hook runs `prek run --group lint --all-files`, and CI's Lint job runs `prek run --all-files` — installing only the toolchains the hooks call. `git commit` runs the pre-commit stage, `commit-msg` checks the message, pre-push runs tests. No parallel formatter or lint definition exists outside these files; hooks that depend on optional tools skip gracefully with a notice.

## Alternatives considered

**Make/just targets per environment.** One place to look, but nothing keeps AI hooks, CI, and humans pointed at the same targets — the drift this decision exists to remove.

**lefthook / husky.** Capable hook runners; prek was selected for native multi-language hook management and config validation, and a second runner alongside it would reintroduce the parallel definition being eliminated.

**CI-only enforcement.** Single definition, but feedback arrives minutes late and local commits accumulate unformatted work; the tiered model (fast fixers at commit, full suite in CI) needs local hooks anyway.

## Consequences

`prek install` is the one-time setup, and every environment that runs the hooks sees the same behavior. Adding a check means editing one prek file (and providing its toolchain in CI) — the docs gate added by this batch follows that path. Optional-tool hooks must keep the graceful-skip contract so a missing local binary degrades to a notice instead of a blocked commit. Committing requires the hook chain; `--no-verify` is out of bounds.
