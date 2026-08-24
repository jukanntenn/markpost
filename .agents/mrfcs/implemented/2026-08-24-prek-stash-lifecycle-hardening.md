# MRFC: prek stash-lifecycle incident hardening

Status: implemented

English | [中文](2026-08-24-prek-stash-lifecycle-hardening.zh.md)

## Problem

prek 0.4.14's `WorkTreeKeeper` (`crates/prek/src/cli/run/keeper.rs`) stashes unstaged changes to `~/.cache/prek/patches/<ms>-<pid>.patch`, wipes them from the worktree with `git checkout --`, and re-applies the patch when the process drops. Three consequences were measured in the 2026-08-22/23 incident (#16):

An interruption between clean and restore — process kill, harness termination; the Ctrl-C cleanup path does not cover it — leaves the worktree silently missing the unstaged work, which exists only in the patch file (8-22 23:49: the 16-file deploy state survived solely in a 64.5KB patch while the tree carried a 2-file subset).

Successfully applied patches are never deleted and no patch carries a consumed marker, so the cache grows unboundedly (19MB, July through August) and every stale patch looks like a plausible recovery source.

prek itself never applies an old patch across sessions (the single `git apply` call site restores the current run's own patch), so recovery is manual — and a hand-applied stale patch silently revives day-old state on top of current work, as measured on 2026-08-23 16:07 when the full deploy state reappeared as a fresh 87KB stash. The same keeper window explains the worktree pre-commit misfire disclosed throughout #16: hooks observe the cleaned tree.

## Decision

`scripts/prek_patch_audit.py` (stdlib, unit-tested like the policy suite and wired into prek under the same pattern) flags orphaned patches — owning pid dead, age beyond a one-hour grace window — and prints a preview-first recovery path; the dev-loop skill runs it at session-open triage, before any work continues on top of a silently cleaned tree. Recovery goes through its previewed commands (`git diff --no-index` inspection, `git apply --check`, then apply), never a blind `git apply` of a cache patch; the runbook owns the procedure as a constraint row, alongside the worktree-misfire row. `prek cache gc` — the upstream 30-day stale-patch retention — joins the triage cadence, and the do-not-clean warning retires with these guards. The root fixes (delete or mark consumed patches; warn on dead-pid orphans) are upstream work in prek itself: markpost files the issue as a maintainer action and re-evaluates the repo-side guards when a fixed release is adopted. The [prek MRFC](2026-08-12-prek-single-source-of-format-and-lint.md) keeps ownership of the tooling decision; this record owns the incident hardening, and the worktree-isolation proposal stays `proposed` with its ignition evidence recorded.

## Alternatives considered

**Fix upstream first, no repo-side guard.** The correct long-term home, but gated on an upstream release cycle outside markpost's control, while the measured incident class stays open.

**Disable the stash cycle.** prek 0.4.14 exposes no `--no-stash` or config equivalent (source-verified); wrapping `git` to fake it is heavier than the hazard it removes.

**A commit-time gate instead of session triage.** Commit hooks run inside the keeper's cleaned window — an auditor hooked there cannot reliably see what it must audit; the natural checkpoint is session open, before work stacks on a cleaned tree.

**Do nothing beyond the do-not-clean warning.** The 8-22/23 timeline shows both loss and revival already realized, and the warning lives only in a session's memory.

## Consequences

An interrupted prek run now surfaces at the next session instead of silently stranding work: the audit exits non-zero only when an orphan needs attention, with the newest orphan's recovery commands printed (on this machine's real cache it flags the full never-cleaned accumulation — 319 patches — which `prek cache gc` retires on the 30-day line). Pid-liveness checks stay heuristic (pid reuse falsifies both directions), bounded by the age threshold; the audit adds one fast stdlib step per session. Upstream behavior may change under these guards, re-evaluated at adoption, and the worktree pre-commit misfire remains until upstream — its replication-plus-`--no-verify` workaround stays a runbook row. The unit suite pins the parse, liveness, grace, and silence behaviors, running through prek exactly like the policy tests.
