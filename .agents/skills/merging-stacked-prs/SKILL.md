---
name: merging-stacked-prs
description: Use when landing a stack of dependent PRs (A ← B ← C) after a human gate passes — preflight, gh stack merge, MERGED verification, and the branch/worktree cleanup pass. Never land stacked PRs one at a time.
---

# Landing a PR stack

Landing happens only after the human approval gate. Work from a clean checkout, fetch live state, and never trust an earlier report — heads, approvals, and checks move while you are not looking.

## Preflight

1. `gh stack --version`; hard-stop if the extension is unavailable — there is no fallback to per-PR `gh pr merge` plus manual retargeting.
2. Verify one official stack in the expected bottom-to-top order via GraphQL `PullRequest.stack` / `stackEntry.position`; branch-name chains are not proof.
3. Judge **every layer independently** — a ready top layer proves nothing about the bottom: open, non-draft, `reviewDecision: APPROVED`, `statusCheckRollup` green, no unresolved change requests.

## Merge

```sh
gh stack merge <stack-number> --yes --merge
```

A partial landing happens only on an explicit user-named boundary PR (the merge covers the bottom prefix through it). If the native merge reports a blocker, resolve it through the owning PR or stop and report — never bypass merge requirements and never fall back to per-PR merges.

## Verify and clean up

- Completion requires every selected layer to report `MERGED` (`gh pr view <n> --json state,mergedAt`); a queued request is not a landed one.
- Branch deletion is a separate final pass. Before deleting each branch, require that no open PR still bases on it (`gh pr list --state open --base <branch> --json number --jq length` must print `0`); then `git branch -d` and `git worktree remove` the layer's worktree.
- After an implementation stack lands, verify the issue auto-closed (the top layer's `Fixes #N`) and the board shows `Done`.
