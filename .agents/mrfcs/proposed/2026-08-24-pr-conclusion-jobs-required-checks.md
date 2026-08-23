# MRFC: PR conclusion jobs and required checks

Status: proposed

English | [中文](2026-08-24-pr-conclusion-jobs-required-checks.zh.md)

## Problem

Gate 3 — delivery approval — has no platform enforcement: branch protection on `main` requires nothing today, so a pull request with a red check can merge. Measured 2026-08-23: PR #23 (`chore: release v0.2.0-rc.6`) merged with `Issue policy` failing, breaching the loop's never-merge-red rule the moment nobody was looking.

Required checks cannot simply be pointed at today's jobs. The five quality workflows — Lint, Test, Build, E2E, Docs — all filter at the workflow level on `pull_request`: Lint/Test/Build/E2E via `paths-ignore` (`**/*.md`, `docs/**`, `.gitignore`, `LICENSE`), Docs via a positive `paths` list. A pull request whose changes fall outside a workflow's path set never triggers that workflow, so its check would sit at "Expected — waiting for status" forever and block the merge. Measured on this repository: docs-only PR #24 ran Docs and Issue policy only; Lint, Test, Build, and E2E never started.

The [development-loop MRFC](../implemented/2026-08-22-agent-driven-development-loop.md) left platform enforcement to a maintainer settings step; this record designs what that step turns on.

## Proposal

All five workflows drop workflow-level path filtering on `pull_request` (the `push: main` filters stay verbatim — required checks concern pull requests, and main-push runner economy is preserved). Path selectivity moves to job level: Lint and Docs gain the `changes` job (dorny/paths-filter) that Test, Build, and E2E already have, and every real job keeps running conditionally on it.

Each workflow adds a `conclusion` job: `needs` lists every real job, `if: always()` makes it run past skips and failures alike, and a single step fails exactly when any need result is `failure` — skipped jobs count as success. That yields five stable, always-reported checks per pull request: `Lint / conclusion`, `Test / conclusion`, `Build / conclusion`, `E2E / conclusion`, `Docs / conclusion`.

The maintainer then sets those five as `main`'s required checks. `Issue policy` is deliberately not in that first set: it runs on every pull request unconditionally (stable as a name) but fails release pull requests by nature — the release-exemption question is settled by its own paired proposal before it joins the set.

## Alternatives considered

**Require today's job names directly.** Zero redesign, but out-of-path pull requests never trigger the path-filtered workflows (measured on #24), leaving required checks stuck at "Expected — waiting for status"; and once filters move to job level, the required set must enumerate every conditional job, where each future job added without a protection update silently escapes gating — the conclusion job exists to collapse that list to one name per workflow.

**One umbrella gate workflow.** Cross-workflow `needs` does not exist, so a single gate file means duplicating every job definition into one workflow, abandoning the per-domain files that mirror prek's structure and the path-based runner economy of main pushes — a rewrite of CI for a problem two small jobs per workflow solve.

**Merge queue instead of required checks.** A merge queue presupposes required checks and serializes merges into queued groups behind full CI runs; it is a freshness mechanism layered on this one, not an alternative to it — and it adds queue babysitting for a single maintainer.

**Convention only (do nothing).** Already measured failing: #23 merged red with no platform signal, and the loop's stopping rules bind the agent, not a hurried human mid-release.

## Acceptance criteria

On a docs-only pull request, all five conclusion checks run and are green while every path-gated real job is skipped. On a full pull request, one failing matrix job turns exactly its workflow's conclusion red. With required checks set, a pull request with any red conclusion cannot merge (verified from a scratch branch before the settings change is trusted). `push: main` behavior is unchanged: same filters, same skips. The development-loop MRFC's branch-protection step gains a pointer to this record.

## Risks

Every pull request now starts all five workflows, costing a `changes` and a `conclusion` micro-job each on out-of-path PRs — bounded: neither installs toolchains, and in-path behavior is unchanged. The conclusion job's `needs` list is explicit; GitHub Actions has no all-jobs wildcard, so a future job added without extending `needs` escapes that workflow's gating — the acceptance test and review convention carry that duty. Required checks need a maintainer settings step after the workflows land; until it is taken, conclusions are advisory and gate 3 remains convention-only, exactly as today. And moving path filters from workflow level to job level changes what the runs page shows: out-of-path pull requests now list skipped real jobs instead of no run at all — more rows, same verdicts.
