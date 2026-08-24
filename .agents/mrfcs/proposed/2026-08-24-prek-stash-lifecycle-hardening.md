# MRFC: prek stash-lifecycle incident hardening

Status: proposed

English | [中文](2026-08-24-prek-stash-lifecycle-hardening.zh.md)

## Problem

prek 0.4.14's `WorkTreeKeeper` (`crates/prek/src/cli/run/keeper.rs`) stashes unstaged changes to `~/.cache/prek/patches/<ms>-<pid>.patch`, wipes them from the worktree with `git checkout --`, and re-applies the patch when the process drops. Three consequences were measured in the 2026-08-22/23 incident (#16):

An interruption between clean and restore — process kill, harness termination; the Ctrl-C cleanup path does not cover it — leaves the worktree silently missing the unstaged work, which exists only in the patch file (8-22 23:49: the 16-file deploy state survived solely in a 64.5KB patch while the tree carried a 2-file subset).

Successfully applied patches are never deleted and no patch carries a consumed marker, so the cache grows unboundedly (19MB, July through August) and every stale patch looks like a plausible recovery source.

prek itself never applies an old patch across sessions (the single `git apply` call site restores the current run's own patch), so recovery is manual — and a hand-applied stale patch silently revives day-old state on top of current work, as measured on 2026-08-23 16:07 when the full deploy state reappeared as a fresh 87KB stash. The same keeper window explains the worktree pre-commit misfire disclosed throughout #16: hooks observe the cleaned tree.

## Proposal

Anti-loss: `scripts/prek_patch_audit.py` (stdlib, unit-tested) flags orphaned patches — owning pid dead, age beyond a grace window — and runs at session-open triage as a dev-loop skill step, before any work continues on top of a silently cleaned tree.

Anti-revive: recovery goes through the audit script's previewed path (`git apply --check` plus diff inspection), never a blind `git apply` of a cache patch; the runbook owns the procedure as a constraint row.

Retention: `prek cache gc` — the upstream stale-patch retention, 30 days — joins the same triage cadence, and the agent workspace schedules it weekly; the do-not-clean warning retires once these guards land.

Upstream: the root fixes — delete or mark consumed patches, and warn about orphaned patches from dead pids — belong in prek itself; markpost files the upstream issue (a maintainer action) and re-evaluates the repo-side guards when a fixed release is adopted. The [prek MRFC](2026-08-12-prek-single-source-of-format-and-lint.md) keeps ownership of the tooling decision; this record owns the incident hardening.

## Alternatives considered

**Fix upstream first, no repo-side guard.** The correct long-term home, but gated on an upstream release cycle outside markpost's control, while the measured incident class stays open.

**Disable the stash cycle.** prek 0.4.14 exposes no `--no-stash` or config equivalent (source-verified); wrapping `git` to fake it is heavier than the hazard it removes.

**A commit-time gate instead of session triage.** Commit hooks run inside the keeper's cleaned window — an auditor hooked there cannot reliably see what it must audit; the natural checkpoint is session open, before work stacks on a cleaned tree.

**Do nothing beyond the do-not-clean warning.** The 8-22/23 timeline shows both loss and revival already realized, and the warning lives only in a session's memory.

## Acceptance criteria

The audit script flags a synthetic orphaned patch (dead pid, past grace) and stays silent on live-pid and fresh patches; its unit suite is wired into prek like the policy tests; the dev-loop skill carries the triage step; the runbook gains the lifecycle row and the worktree-misfire row; `prek cache gc` is documented at the triage cadence; the worktree-isolation proposal record gains its ignition evidence and stays `proposed` (disposition: defer until parallel verification is next needed).

## Risks

Pid-liveness checks are heuristic (pid reuse falsifies both directions), bounded by the age threshold; the audit adds one fast stdlib step per session and exits non-zero only when an orphan needs attention; upstream behavior may change under our guards, re-evaluated at adoption; and the worktree pre-commit misfire remains until upstream — its replication-plus-`--no-verify` workaround stays a runbook row.
