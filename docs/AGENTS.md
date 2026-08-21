# AGENTS.md — The documentation standard

This file defines where documentation lives and the writing rules every gate enforces. The [doc-standards](../.claude/skills/doc-standards/SKILL.md) skill turns it into a workflow; `python3 scripts/doc_sync.py` validates.

## Tiers: one fact, one home

| Tier                               | Job                                                                            |
| ---------------------------------- | ------------------------------------------------------------------------------ |
| `README.md` / `README_zh.md`       | User-facing product docs                                                       |
| `AGENTS.md`                        | Standing orders for agents, 1–3 lines per rule, linking its home               |
| `PRINCIPLES.md`                    | Behavioral constraints                                                         |
| `specs/`                           | Current-state design reference; `specs/index.md` is the authoritative index    |
| `docs/`                            | Operation guides (development, deployment)                                     |
| `mrfc/`                            | Decision records — the why and what was given up ([README](../mrfc/README.md)) |
| `CHANGELOG.md` / `KNOWN_ISSUES.md` | Ledgers — narrate history by design; exempt from prose gates                   |
| `.claude/skills/`                  | Agent workflows (mirrored to `.agents/skills/` by `scripts/sync_agents.py`)    |

Elsewhere, link; never restate. Rationale → `mrfc/`; procedures → `docs/`; system facts → `specs/`; rules an agent needs every session → `AGENTS.md`.

## Rules and gates

1. **Current state only.** `specs/` and `docs/` describe what is — none of `previously` / `no longer` / `已移除` / `不再` style narration. Link the owning MRFC for the why — `verify_md_current.py`.
2. **Machine-checkable links.** Relative Markdown paths with resolving targets and `#fragment` anchors; never bare filenames — `verify_md_links.py`.
3. **One physical line per paragraph.** Soft-wrap in the editor; code blocks, tables, and lists keep their structure — `verify_md_wrap.py`.
4. **Index completeness.** Every `specs/**/*.md` file has a row in `specs/index.md`, and every index row points at an existing file — `verify_specs_index.py`.
5. **MRFC format.** Header, Status-agrees-with-folder, `## Problem` opener, mandatory `## Alternatives considered` — `verify_mrfc_format.py`.

Ledgers, point-in-time reports under `scripts/loadtest/`, `.zcode/`, and generated Swagger stay outside the gates.

## Slop to hunt

The same rule stated in two homes; narrated history; implementation-status annotations ("future:", "已实现"); hand-restated catalogs where source or a script is authoritative; paragraph walls carrying several rules; emphasis inflation. Keep one home, link the rest.
