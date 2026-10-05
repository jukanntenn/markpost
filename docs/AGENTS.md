# AGENTS.md — The documentation standard

This file defines the document tiers, writing rules, and the documentation-gate budgets; [docs/i18n/README.md](i18n/README.md) owns the pairing contract. Use [documenting](../.agents/skills/documenting/SKILL.md) for placement and [editing-prose](../.agents/skills/editing-prose/SKILL.md) for editorial judgment.

## The tier taxonomy: one home per fact

Each fact has one home: the tier whose job it is; elsewhere, link there.

| Tier                                                                             | Job                                                                                                                                   | Does NOT belong there                                                                          |
| -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `README.md` pair                                                                 | User-facing product docs                                                                                                              | Procedures, rationale                                                                          |
| Root `AGENTS.md`                                                                 | Standing orders an agent needs in context in every session, linking its home                                                          | Stories, worked examples, situational procedures, anything restated from a linked home         |
| Subtree `AGENTS.md` (`backend/`, `frontend/`, `e2e/`, `cli/`, `mcp/`, this file) | Orders specific to that subtree                                                                                                       | Repo-wide rules the root file already carries                                                  |
| [architecture.md](architecture.md)                                               | Ordered map: components and where new behavior goes; read before changing the source tree                                             | Per-domain detail (→ the owning document), decision rationale (→ RFCs)                         |
| [development.md](development.md)                                                 | Contributor setup, daily workflow, and a summary of CI                                                                                | Runtime/version rationale (→ RFCs), check-by-check lists that drift from the command inventory |
| `specs/`                                                                         | Current-state design reference; every spec pair has a row in `specs/index.md` and its zh twin, and index rows point at existing files | Procedures (→ docs), rationale (→ RFCs)                                                        |
| `docs/` (guides)                                                                 | Operation guides — deployment, backup, monitoring                                                                                     | System facts specs already own                                                                 |
| [cookbook/](cookbook/responding-to-pr-review-on-a-stack.md)                      | Step-by-step how-tos with observable verify steps                                                                                     | Design rationale (→ the RFC each guide links)                                                  |
| `.agents/rfcs/**`                                                                | Decision records: the why, what was given up, required verification ([rules](../.agents/rfcs/README.md))                              | Current-state contracts (→ specs/docs), procedures (→ cookbooks)                               |
| Skills (`.agents/skills/`)                                                       | Reusable workflow instructions                                                                                                        | Product and runtime contracts (→ specs/docs or source)                                         |
| `PRINCIPLES.md`                                                                  | Frozen archive of behavioral constraints; the live home is root `AGENTS.md` § Conventions                                             | Anything current                                                                               |
| `CHANGELOG.md` / `KNOWN_ISSUES.md`                                               | Ledgers — narrate history by design; outside the prose gates                                                                          | Anything gated                                                                                 |

Placement: rationale → RFCs; procedures → cookbooks or docs guides; contracts → the owning spec or reference; standing orders → root `AGENTS.md` with a rationale link.

## Pairing rules

- Every in-scope document is an English + Simplified Chinese pair — `foo.md`, `foo.zh.md`, and the `foo.i18n.yaml` record — under the [pairing contract](i18n/README.md), enforced by `hdsh pairing verify`; pairs update together in one PR and re-record with `hdsh pairing record <pair>`.
- Instruction files are English-only and exempt from pairing: every `AGENTS.md` and everything under `.agents/skills/`.
- Archived RFC triplets are frozen and outside the corpus ([archive policy](../.agents/rfcs/archived/AGENTS.md)); raw load-test data stays outside the gates.

## Writing rules

- Current-state prose: document what is, not change history; put change stories in RFCs and commits; link the owning RFC for the why.
- Concise titles that name their subject.
- Write directly: name actors and facts instead of metaphorical placeholders. Name the exact check, command, type, or behavior instead of a generic label.
- One physical line per prose paragraph, enforced by `hdsh docs wrap`; use editor soft-wrap. Code blocks, tables, and list structure keep their formatting.
- Cross-reference documents by relative Markdown link, never bare prose or a number, so `hdsh docs links` can prove every target and `#fragment` resolves; links into a pair use the reader's own locale.
- Files end with exactly one trailing newline.

## Wordcount budgets

`.hdsh/docs.manifest.json` sets standing-doc ceilings; `hdsh docs budgets` rejects excess or missing files.

When the gate goes red:

1. **Relocate** content that belongs in another tier; leave a one-line link if needed.
2. **Condense** content that belongs here but can be shorter.
3. **Raise** the ceiling only when the words need the space; justify the manifest diff in the PR. A too-low ceiling is a budget bug.

Ceilings are guardrails, not reduction targets. At or below target, retain at least 5% headroom; above target, freeze the ceiling until relocation or condensation brings the document under target. Review governs unbudgeted documents.

## The slop checklist

Hunt duplicately-homed rules, narrated history ("previously", "已移除", PRs), implementation-status annotations ("future:", "已实现"), hand-restated catalogs where source is authoritative, reasoning transcripts, paragraph walls, emphasis inflation, and one-sided language-pair edits. The [editing-prose](../.agents/skills/editing-prose/SKILL.md) skill owns the full editorial checklist.
