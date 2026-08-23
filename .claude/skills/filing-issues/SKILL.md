---
name: filing-issues
description: Use when creating a GitHub issue for markpost — picking the template, the title and body contract, type labels, and the self-check before filing.
---

# Filing an issue

Issues enter the development loop through five templates; the same contract binds humans and the agent, and the `issue-policy` workflow audits it (a failing issue gets an audit comment listing the violations).

1. Pick the template by intent — `Idea` (an uncommitted possibility worth recording), `Feature` (new or intentionally changed observable behavior), `Bug` (expected behavior failing), `Research` (a conclusion, evidence, or decision to produce), `Task` (clear work that is neither Feature nor Bug). Blank issues are disabled; the template choice applies the `type/*` label automatically.
2. Title: one Chinese action-or-result sentence. No `[Type]`, `Type:`, `P0`, or status prefixes — classification lives in the label, and the policy rejects prefixed titles.
3. Body: exactly one visible line above the collapsed `<details>` block, within 50 units (a Chinese character is one unit; a contiguous Latin/number token is one; links count their text, not their URL). The details block stays collapsed by default and carries the template's acceptance/evidence fields — fill them meaningfully; they are what the implementation phase maps deliverables against.
4. Self-check before filing: re-read the draft against rules 1–3. The policy also validates after the fact — if the audit comment appears, fix the issue body (`gh issue edit <n> --body …`) rather than closing and refiling.
5. Never advance the board yourself: newly filed issues land in `Inbox`, and the move into `Ready` is the human's gate. Priority (P0–P3) is likewise the human's call on the board.
