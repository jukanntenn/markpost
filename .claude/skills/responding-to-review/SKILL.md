---
name: responding-to-review
description: Use when a review returns comments or CHANGES_REQUESTED on one layer of a stack — verifying claims, placing fixes on the introducing layer, merge-forward propagation, and in-thread replies.
---

# Responding to review on a stack

Review comments may target several layers of one stack. The stack stays linked through GitHub's official stack object; this skill owns where a fix lands and how it travels.

1. Triage every comment on the merits before acting: verify the claim against the code — a reviewer naming the right symptom can still misdiagnose the cause. Fix or rebut on technical grounds; no performative agreement.
2. Map each accepted finding to the layer that **introduced** the issue (not necessarily the layer it was commented on) and fix it there, in that layer's own worktree. Fixing downstream ships the defect with the lower layer and hides the fix from that layer's reviewer.
3. Propagate the fixed layer through every affected child, bottom-up:
   - **Merge-forward (the default):** merge the fixed parent branch into each child, run the relevant checks on the child, continue upward. Reviewed history stays intact and approvals remain valid.
   - **Cascading rebase (deliberate only):** `gh stack rebase`, validate the rewritten layers, publish with `gh stack push`. Lease-protected only — raw `--force` is forbidden. After any rewritten push, re-read unresolved threads, approvals, mergeability, and checks: earlier commit OIDs and inline anchors are stale evidence.
4. Each review fix is its own commit on the introducing layer; never amend a reviewed fix out of history. Amend only your own unpushed, unreviewed work.
5. Reply **in the original review thread** (`gh api repos/{owner}/{repo}/pulls/{pr}/comments/{id}/replies`), stating the fix and the commit or head that carries it — not as a top-level comment.
6. Re-request the human review on the affected layers once they settle; the board's `In review` ↔ `In progress` movement is owned by the lifecycle workflow, not by you.
7. A decision-level objection (the review contests an approved MRFC's design) → stop and surface it; the human arbitrates between the record and the review.
