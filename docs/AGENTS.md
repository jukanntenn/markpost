# AGENTS.md — The documentation standard

This file defines where documentation lives and the writing rules every gate enforces. The [doc-standards](../.claude/skills/doc-standards/SKILL.md) skill turns it into a workflow; `python3 scripts/doc_sync.py` validates.

## Tiers: one fact, one home

| Tier                                                             | Job                                                                                       |
| ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `README.md` / `README_zh.md`                                     | User-facing product docs                                                                  |
| Root `AGENTS.md`                                                 | Standing orders for every session, 1–3 lines per rule, linking its home                   |
| Subtree `AGENTS.md` (`backend/`, `frontend/`, `e2e/`, this file) | Orders specific to that subtree; never repeat the root                                    |
| `PRINCIPLES.md`                                                  | Frozen archive of behavioral constraints; the live home is root `AGENTS.md` § Conventions |
| `specs/`                                                         | Current-state design reference; `specs/index.md` is the authoritative index               |
| `docs/`                                                          | Operation guides (development, deployment)                                                |
| `.agents/mrfcs/`                                                 | markpost's RFCs — proposals and decision records ([README](../.agents/mrfcs/README.md))   |
| `CHANGELOG.md` / `KNOWN_ISSUES.md`                               | Ledgers — narrate history by design; exempt from prose gates                              |
| `.claude/skills/`                                                | Agent workflows (mirrored to `.agents/skills/` by `scripts/sync_agent_instructions.py`)   |

Elsewhere, link; never restate. Rationale → `.agents/mrfcs/`; procedures → `docs/`; system facts → `specs/`; rules an agent needs every session → `AGENTS.md`.

## Rules and gates

1. **Current state only.** `specs/` and `docs/` describe what is — none of `previously` / `no longer` / `已移除` / `不再` style narration. Link the owning MRFC for the why — `verify_md_current.py`.
2. **Machine-checkable links.** Relative Markdown paths with resolving targets and `#fragment` anchors; never bare filenames — `verify_md_links.py`.
3. **One physical line per paragraph.** Soft-wrap in the editor; code blocks, tables, and lists keep their structure — `verify_md_wrap.py`.
4. **Index completeness.** Every `specs/**/*.md` file has a row in `specs/index.md`, and every index row points at an existing file — `verify_specs_index.py`.
5. **MRFC format.** Header, Status-agrees-with-folder, `## Problem` opener, mandatory `## Alternatives considered` — `verify_mrfc_format.py`.
6. **Word budgets.** The agent-instruction files carry `wc -w` ceilings in [`scripts/doc_budgets.manifest.json`](../scripts/doc_budgets.manifest.json); a budgeted file that is missing fails the gate — `verify_doc_budgets.py`. On red: relocate to the owning tier, condense, raise the ceiling last with the manifest diff justified in the PR.

Ledgers, point-in-time reports under `scripts/loadtest/`, `.zcode/`, and generated Swagger stay outside the gates.

## Slop to hunt

The same rule stated in two homes (a subtree `AGENTS.md` restating the root is the common case); narrated history; implementation-status annotations ("future:", "已实现"); hand-restated catalogs where source or a script is authoritative; paragraph walls carrying several rules; emphasis inflation. Keep one home, link the rest.
