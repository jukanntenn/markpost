---
name: writing-mrfcs
description: Use when a non-trivial change lands in markpost (new capability, schema change, architectural shift, tooling/process change) and needs an MRFC, when searching .agents/mrfcs/ for prior decisions on a topic, when superseding or rejecting an existing MRFC, or when asked "why is X like this" / "record this decision".
---

# Writing MRFCs

MRFCs are markpost's RFCs: durable proposals and decision records — the why, the alternatives that lost, and what the trade-off bought. The full contract — naming, lifecycle, format skeleton — lives in [.agents/mrfcs/README.md](../../../.agents/mrfcs/README.md); `scripts/verify_mrfc_format.py` enforces it.

## Before writing

1. `grep -ri "<topic keywords>" .agents/mrfcs/` — the decision may already have a home. Updating the owning MRFC in the same PR satisfies the rule; never create a duplicate.
2. If a new decision supersedes an old one: write the new MRFC, cross-link both (old → new with a one-line pointer, new → old in Alternatives), and keep the old file unless fully consolidated.
3. Pick the lifecycle: `proposed/` for anything not yet built (substantial future work gets written down *before* implementation), `implemented/` for decisions that shipped, `rejected/` for declines worth remembering.

## Writing it

- Filename: `yyyy-mm-dd-topic-title.md` — the date the topic was first proposed, a lowercase slug. Every record is a bilingual pair: write the `foo.md` and `foo.zh.md` sides together, machine tokens and section headings in English (terminology: [docs/AGENTS.md](../../../docs/AGENTS.md) rule 7).
- `## Problem` must stand without the solution: what forced the decision?
- `implemented/` states `## Decision` in present tense describing shipped reality — paths and names must match the code today. Proposal-era headings (`## Proposal`, `## Plan`, `## Migration plan`, `## Acceptance criteria`) are rejected here.
- `## Alternatives considered` is mandatory: one bold-led paragraph per genuine alternative and why it lost. Record the alternatives as they were actually argued; never invent them.
- `## Consequences` records what the trade-off cost *and* bought, including required verification.
- Keep facts current: when code moves what an implemented MRFC references, update the MRFC's facts in the same change.

## Lifecycle moves

Moving a file between folders re-satisfies the target folder's skeleton in the same change, moving both languages of the pair: `proposed/ → implemented/` rewrites Proposal into a present-tense Decision and folds Acceptance criteria/Risks into Consequences; `proposed/ → rejected/` adds the one-line reason to `Status:` and freezes the file. The format gate fails the move otherwise.

Validate with `python3 scripts/doc_sync.py .agents/mrfcs/<path>` before committing; the prek `doc-check` hook runs it on commit.
