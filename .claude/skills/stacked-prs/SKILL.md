---
name: stacked-prs
description: Use when creating or maintaining a stack of dependent pull requests for one issue — worktree layout, branch naming, layer creation, gh stack linking, and where the closing keyword goes.
---

# Building a PR stack

The loop's stack rules are owned by [the design record](../../../.agents/mrfcs/implemented/2026-08-22-agent-driven-development-loop.md); this skill is the mechanics.

## Worktrees

- One worktree per layer branch, at `.local/worktrees/<branch>/` — inside the repository's scratch root, already excluded by `.gitignore` and `.dockerignore`, so the path is repository-relative wherever the repo is cloned.
- Branch names: `rfc/<issue>-<slug>` for RFC layers, `impl/<issue>-<slug>` for implementation layers.
- Build bottom-up: each higher layer branches from the one below it (`git worktree add .local/worktrees/<branch> -b <branch> <parent-branch>`); fixes later land in the worktree of the layer that introduced them, never in a downstream checkout.

## Layer content

- RFC phase: one `proposed/` MRFC pair per layer; the PR body references the issue as `Related to #N` — never a closing keyword, so RFC layers neither drive the board nor close the issue.
- Implementation phase: layers ordered by dependency (schema → API → UI); **only the top layer carries `Fixes #N`** — the issue closes exactly when the whole stack lands and stays open under a partial landing.
- Every PR: a conventional-commit title, at least one `area/*` label, and the PR template's Ask-first and change/verification sections filled in.

## Linking the stack

After creating all PRs, link them bottom-to-top into GitHub's official stack:

```sh
gh stack link --base main <bottom-pr> <next-pr> ... <top-pr>
```

GraphQL `PullRequest.stack` is the membership authority — re-query and verify one stack with the expected entries before requesting review; never treat a matching branch chain as a stack without checking. Same-author chains link automatically; mixed authors need the user's confirmation first.

## Discipline

- Merge-forward is the default propagation for fixes; rebase is deliberate, lease-protected, and re-audited ([responding-to-review](../responding-to-review/SKILL.md)).
- Pushing a branch uses the machine account's credentials; never force-push a reviewed layer (`--force-with-lease` only, and only on the rebase path).
- One worktree per layer stands for the whole life of the stack — [merging-stacked-prs](../merging-stacked-prs/SKILL.md) removes them in the cleanup pass.
