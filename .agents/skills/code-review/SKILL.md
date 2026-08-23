---
name: code-review
description: Use before requesting human review on any pull request or stack layer — the agent's structured pre-review of its own work: conventions compliance, correctness and security checks, evidence completeness, and the written assessment the human approval decides on.
---

# Pre-reviewing a pull request

The pre-review is a filter and a briefing, not the approval: the human gate follows it (and the machine account cannot approve its own PRs — the platform pins that assignment). Produce a structured **comment review** on each layer before requesting the human.

## Verify first

- Re-establish the layer's live base and head (`gh pr view <n> --json baseRefName,headRefOid`); for upper layers, read the diff against the parent layer, not `main`.
- Run the relevant local checks for the tree touched — prek gates, focused tests — and match evidence to the surface; CI owns the exhaustive matrix.

## Check

- **Conventions:** the tree's `AGENTS.md` rules; changed behavior matches `specs/` (update the spec in the same PR where behavior moved); documentation duties — bilingual pairs updated together, new spec pages add their `specs/index.md` row; MRFC lifecycle moves where a decision lands (`proposed/` → `implemented/` on the finalizing layer).
- **Correctness:** trace both sides of every changed interface; error and cancellation paths; cleanup and ownership for anything long-lived; new query shapes against indexes and N+1 loads; concurrency around shared state.
- **Security:** authorization on new endpoints; input validation at parse boundaries; injection in SQL, templates, and shell; secrets never logged or committed.
- **Tests:** assertions fail on the intended regression; behavior verified through external state or the real entry path rather than restating the implementation.
- **Evidence:** the PR body's verification section states the commands run and their outcomes; UI changes carry Playwright screenshots; the issue's acceptance conditions are explicitly mapped.

## Report

Separate blockers from suggestions; omit what a green gate already enforces. State the defect, the location, the impact, and the evidence. Name the Ask-first items (schema migrations, new dependencies, CI, Docker changes) explicitly — they are the human reviewer's forced-attention list. Post as a review of type COMMENT on the layer's PR.
