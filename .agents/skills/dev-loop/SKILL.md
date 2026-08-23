---
name: dev-loop
description: Use when a session should drive the agent-driven development loop — triaging in-flight stacks after asynchronous human gates, claiming Ready issues, or progressing any phase from claim to landed work. Covers session-open triage, the two-phase RFC-to-implementation state machine, and the stopping rules.
---

# Driving the development loop

The design record is [the agent-driven development loop MRFC](../../../.agents/mrfcs/implemented/2026-08-22-agent-driven-development-loop.md). The three human gates (issue → Ready, RFC approval, delivery approval) are asynchronous: a session ends waiting for them, and the next session detects their passage. Operate through the machine account's `GH_TOKEN`; never advance an issue into `Ready` yourself — that move is the human's first gate.

## Session-open triage (always, in order)

1. Enumerate in-flight work: `gh pr list --author @me --state open --json number,title`, then `gh pr view <n> --json isDraft,reviewDecision,statusCheckRollup,mergeStateStatus` per PR, plus the GraphQL stack for each chain.
2. Act per stack and phase:
   - **RFC stack, approved, checks green, no unresolved change requests** → land it with [merging-stacked-prs](../merging-stacked-prs/SKILL.md), then start the implementation phase for its issue.
   - **Implementation stack, approved, green** → land it, clean up branches and worktrees, verify the issue auto-closed and the board shows `Done`.
   - **changes_requested or new review comments** → run [responding-to-review](../responding-to-review/SKILL.md).
   - **Awaiting the gate with nothing to do** → leave it standing; report the waiting state if the session was manually kicked.
3. Nothing in flight → claim: issues in `Ready`, unassigned, ordered by priority (P0 first) then age. Claiming is `gh issue edit <n> --add-assignee @me` plus a comment declaring the decomposition entry point — never a board write.

## Phase: claim → decomposition → RFC stack

1. Judge triviality against the [MRFC README](../../../.agents/mrfcs/README.md) mechanical-local-edit exemption; trivial issues take the fast path — straight to the implementation phase, delivery gate only.
2. Decompose by decision, not by file: every non-trivial decision the issue forces gets one MRFC pair ([writing-mrfcs](../writing-mrfcs/SKILL.md)); layers stack when decisions depend on each other ([stacked-prs](../stacked-prs/SKILL.md)).
3. Build the RFC stack; each layer references the issue as `Related to #N` — never a closing keyword.
4. Pre-review every layer ([code-review](../code-review/SKILL.md)), then request the human review on every layer (`gh pr edit <n> --add-reviewer jukanntenn`) and end the session with a stack report.

## Phase: RFC merged → implementation

1. Write the board's `In progress` once at implementation start (`gh project item-edit` on the issue) — lower implementation layers carry no closing reference, so the lifecycle workflow cannot see the start.
2. Decompose implementation from the merged MRFCs; order layers by dependency; **only the top layer carries `Fixes #N`**.
3. The layer that makes an MRFC decision real — usually the top — moves that pair from `proposed/` to `implemented/` and rewrites it into the present tense in the same change.
4. Each layer delivers code, tests, and affected documentation (a new spec page adds its `specs/index.md` row); the PR body carries references, the Ask-first items, and evidence: commands with outcomes, Playwright screenshots for UI changes, and a mapping against the issue's acceptance conditions. Local checks first (prek, focused tests); CI owns the exhaustive matrix.
5. Pre-review, request human review on all layers, end the session.

## Stopping rules

- A decision-level design reversal discovered mid-implementation → stop and ask; small gaps amend the still-`proposed` MRFC inside the implementation PR.
- Ask-first items (schema migrations, new dependencies, CI, Docker) → always named in the PR body; they gate human attention.
- A native stack-merge blocker → resolve through the owning PR or stop and report; the per-PR merge fallback is forbidden.
- Never: advance an issue into `Ready`, approve a pull request, or merge with a red check.
