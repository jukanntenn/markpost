# MRFC: Issue-policy exemption for release pull requests

Status: implemented

English | [中文](2026-08-24-issue-policy-release-exemption.zh.md)

## Problem

The [`issue-policy` workflow](../../../.github/workflows/issue-policy.yml) validates every non-draft, non-bot pull request against the intake contract: a conventional-commit title, at least one `area/*` label, and at least one issue reference. A release pull request fails this by nature: it carries a version bump and a changelog, references no issue, and historically carries no area label. Measured 2026-08-23: PR #23 (`chore: release v0.2.0-rc.6`) failed `Issue policy` with both errors and merged red anyway.

Once [PR conclusion jobs and required checks](../implemented/2026-08-24-pr-conclusion-jobs-required-checks.md) makes gate 3 platform-enforced, red-but-mergeable stops existing: either release pull requests are exempt and green, or every release blocks on a check that cannot apply to it.

## Decision

[`policy.py`](../../../.github/issue-management/policy.py) `pr` skips the three intake checks — title form, area label, issue reference — when the pull request's head branch matches `release/**`: `is_release_pull` is a pure function of the head ref on the pull-request snapshot `run_pr_check` already fetches, so detection adds no API call and covers every event shape the workflow triggers on. Everything else still runs: any references a release pull request carries are validated as usual, and the issue-side checks are unchanged. `Issue policy` joins `main`'s required checks alongside the five conclusion checks — applying the settings is the maintainer's branch-protection step. Release branch naming stays owned by the release skill (`release/v0.2.0-rc.6`, measured).

## Alternatives considered

**Convention: release pull requests carry `area/devops` and a standing reference.** Measured against: #23 was filed bare under release pressure with no tracking issue to reference; the convention would also force one throwaway issue per release to satisfy a check that exists to route review attention, not to gate releases.

**Keep `Issue policy` out of the required set and red on releases.** Enforcement-free for the policy layer, but it normalizes a check that is "always red on releases" — exactly the ambiguity gate 3's platform enforcement exists to remove — and the never-merge-red rule becomes unteachable the first time it must be waived.

**Skip the issue-policy workflow itself for `release/**` heads.** Loses the checks that do apply — referenced-issue validation whenever a reference exists — and hides the workflow from release pull requests entirely; the exemption belongs in the policy's intake rules, where the contract lives.

## Consequences

Release pull requests can be all-green: a bare `release/**` pull request with no labels and no references passes `policy.py pr`, while a non-release pull request missing any of the three checks still fails — both directions pinned in the policy unit-test suite (`is_release_pull` release, near-miss, and missing-ref cases) that prek runs on every commit and CI re-runs; the next real release pull request is the end-to-end measurement. The exemption is branch-name-scoped: a release cut from a misnamed branch fails policy and, once required checks are on, blocks — visible immediately, fixed by renaming, and the release skill owns the naming. Exempting releases from the intake contract means area mislabeling on a release pull request can never be caught by policy — nothing about releases routes by area today, so the cost is theoretical.
