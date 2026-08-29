# MRFC: Adapting the release process to the agent-driven loop

Status: implemented

English | [中文](2026-08-30-release-process-loop-adaptation.zh.md)

## Problem

The release skill taught the repository's pre-loop operating model end to end: commit the version bump straight to `main`, push, push the tag. [The development loop](2026-08-22-agent-driven-development-loop.md) closed that world — measured today, `main`'s branch protection requires a pull request with one approving review (administrators included) and conversation resolution, forbids force pushes, and names six required checks (five conclusion checks plus `Issue policy`); the repository merges through merge commits only. The skill's central move is mechanically impossible now — a direct push to `main` is rejected — and the one release attempted under the new regime, v0.2.0-rc.6 through PR #23, merged with `Issue policy` red: the incident that produced both [PR conclusion jobs and required checks](2026-08-24-pr-conclusion-jobs-required-checks.md) and [the release exemption](2026-08-24-issue-policy-release-exemption.md). The exemption made bare `release/**` pull requests pass policy and left the branch naming to the release skill — which still teaches the dead path, and has no answer for the loop's asynchronous gates: the approval a release waits for arrives after the session that opened it has ended.

## Decision

A release rides the same rails as any other change, in a release-specific shape, and [the release skill](../../skills/release/SKILL.md) now teaches exactly this.

**The bump lands through a pull request.** Branch `release/vX.Y.Z[-rc.N]` cut from current `main` — the skill owns the naming the policy exemption matches on — carrying one `chore: release vX.Y.Z` commit (`frontend/package.json` + `CHANGELOG.md`), a conventional title, no `area/*` label and no issue reference: bare by design, per the exemption.

**The delivery gate is the pull request's approving review.** The PR is machine-authored like every loop PR — a maintainer-authored one deadlocks on include-administrators, since nobody approves their own pull request and approval belongs to the human alone. The in-session pauses survive where they are decisions rather than merges: the version-number confirmation stays; the old confirm-before-push pause is subsumed, because the PR body declares that merging is followed by tag `vX.Y.Z`, whose push triggers `release.yml` and `docker-publish.yml` — approval is the consent to publish.

**Landing is single-PR.** Preflight (open, non-draft, approved, six checks green, no unresolved change requests), then `gh pr merge --merge` — the repository's merge method; a solitary main-based PR has no stack object to link and no retargeting risk, so the loop's ban on per-PR merges (a stack fallback) does not apply. Branch cleanup follows the stack rule anyway: delete only when nothing open bases on the branch.

**Publication is tag-driven and strictly downstream of the gate.** After the merge, tag the merge commit on `main` and push the tag — never a pre-merge branch head, which would publish a commit the delivery gate has not accepted. Branch pushes publish nothing; `docker-publish.yml` triggers on `v*` tags, and the rolling `main` image is `docker/build.py`'s separate lane.

**The flow is resumable.** Any session whose triage finds a `release/**` pull request approved and green finishes it — merge, tag, verify — without re-deriving the release, so releases fit the loop's asynchronous gates like every other piece of work.

## Alternatives considered

**A branch-protection exception so the maintainer can still push releases to `main`.** Fast under release pressure, but include-administrators exists precisely so the maintainer's own pushes gate, and pressure is when checks matter — measured as #23 merging red. An exception would also reopen unaudited direct-push history for exactly the commits users consume first.

**Satisfy issue policy instead of leaning on the exemption** (conventional title + `area/devops` + a standing release issue). One policy for all pull requests, but it was measured against and rejected in the exemption record: one throwaway issue per release to feed a check that routes review attention, not one that gates releases.

**Tag the release branch before merging, so the tag exists the moment the PR opens.** Publication would start on a commit `main` has not accepted; any review change then means deleting an already-published tag and its Docker tags. Tagging the merge commit keeps publication strictly downstream of the delivery gate.

**Run releases as loop issues (template, board, two-phase stack).** Uniform with feature work, but it adds the `Ready` gate — a human decision with no content for a version bump — and forces the issue reference the exemption exists to avoid. Releases are maintainer-timed; the single delivery approval is the one decision they contain.

**`gh stack merge` for the solitary release PR too.** One mechanical path for every landing, but it pays the extension dependency for nothing: there is no stack to verify and nothing to retarget, and `gh pr merge --merge` exercises the same merge method under the same protection.

## Consequences

The next release measures the flow end to end: a bare `release/**` pull request should show six green checks — `Issue policy` green through the exemption, its first live run since the required set was applied — and merge only after the maintainer's approval. Releases gain one human touchpoint, the same delivery gate every other change already pays; the version-validation rules, the CHANGELOG style, and the STOP-on-failure posture are untouched. Rollback is now staged: before the merge, close the pull request and delete the branch; between merge and tag, revert the merge commit through a pull request; after the tag, delete the tag (local and remote) and edit or delete the GitHub Release in the web UI — direct-push undos are gone with direct pushes. The exemption record's "release branch naming stays owned by the release skill" now has its owner's text: `release/vX.Y.Z[-rc.N]`.
