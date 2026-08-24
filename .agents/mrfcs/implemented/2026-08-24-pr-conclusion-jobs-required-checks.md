# MRFC: PR conclusion jobs and required checks

Status: implemented

English | [中文](2026-08-24-pr-conclusion-jobs-required-checks.zh.md)

## Problem

Gate 3 — delivery approval — had no platform enforcement: branch protection on `main` required nothing, so a pull request with a red check could merge. Measured 2026-08-23: PR #23 (`chore: release v0.2.0-rc.6`) merged with `Issue policy` failing, breaching the loop's never-merge-red rule the moment nobody was looking.

Required checks could not simply be pointed at the job names of that era. The five quality workflows — Lint, Test, Build, E2E, Docs — all filtered at the workflow level on `pull_request`: Lint/Test/Build/E2E via `paths-ignore` (`**/*.md`, `docs/**`, `.gitignore`, `LICENSE`), Docs via a positive `paths` list. A pull request whose changes fell outside a workflow's path set never triggered that workflow, so its check would sit at "Expected — waiting for status" forever and block the merge. Measured on this repository: docs-only PR #24 ran Docs and Issue policy only; Lint, Test, Build, and E2E never started.

The [development-loop MRFC](2026-08-22-agent-driven-development-loop.md) left platform enforcement to a maintainer settings step; this record designs what that step turns on.

## Decision

All five quality workflows run on every `pull_request` targeting `main` with no workflow-level path filtering; the `push: main` filters stay verbatim, preserving main-push runner economy. Path selectivity lives at job level: each workflow gates its real jobs on a `changes` job (dorny/paths-filter) — Lint and Docs gained one, Test, Build, and E2E already had theirs — and the job-level conditions apply on push and dispatch exactly as Test, Build, and E2E already practiced.

Each workflow ends in a `conclusion` job whose `needs` lists every other job and whose `if: always()` lets it run past skips and failures; a single step fails exactly when any need result is `failure` or `cancelled` — skipped jobs count as success. That yields five stable, always-reported checks per pull request: `Lint / conclusion`, `Test / conclusion`, `Build / conclusion`, `E2E / conclusion`, `Docs / conclusion`.

`main`'s required checks are those five conclusion checks, plus `Issue policy` once its release exemption (its own paired record) is in place; applying the settings is the maintainer's branch-protection step — the workflow side ships here.

## Alternatives considered

**Require the job names of that era directly.** Zero redesign, but out-of-path pull requests never triggered the path-filtered workflows (measured on #24), leaving required checks stuck at "Expected — waiting for status"; and once filters move to job level, the required set must enumerate every conditional job, where each future job added without a protection update silently escapes gating — the conclusion job exists to collapse that list to one name per workflow.

**One umbrella gate workflow.** Cross-workflow `needs` does not exist, so a single gate file means duplicating every job definition into one workflow, abandoning the per-domain files that mirror prek's structure and the path-based runner economy of main pushes — a rewrite of CI for a problem two small jobs per workflow solve.

**Merge queue instead of required checks.** A merge queue presupposes required checks and serializes merges into queued groups behind full CI runs; it is a freshness mechanism layered on this one, not an alternative to it — and it adds queue babysitting for a single maintainer.

**Convention only (do nothing).** Already measured failing: #23 merged red with no platform signal, and the loop's stopping rules bind the agent, not a hurried human mid-release.

## Consequences

Every pull request now reports one green-or-red row per workflow regardless of touched paths, and required checks can point at five stable names that never hang at "Expected". Out-of-path PRs cost two micro-jobs per workflow (`changes` and `conclusion`; neither installs toolchains); the runs page lists skipped real jobs instead of no run at all — more rows, same verdicts. The conclusion job's `needs` list is explicit — GitHub Actions has no all-jobs wildcard — so a future job added without extending `needs` escapes that workflow's gating; review carries that duty. Until the maintainer applies the branch-protection settings, conclusions are advisory and gate 3 remains convention-only, exactly as before; the settings step is verified by a scratch branch whose red conclusion must fail to merge before it is trusted. On the landing stack's own pull requests — the first to run this shape — a docs-and-workflow change runs all five workflows with real jobs selectively skipped, which is the standing acceptance evidence.
