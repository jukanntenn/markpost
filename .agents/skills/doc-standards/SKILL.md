---
name: doc-standards
description: Use when writing, moving, reviewing, or auditing documentation in markpost — deciding where a fact lives (README/specs/docs/mrfc/skills), adding a spec to specs/index.md, trimming history narration from current-state docs, responding to a doc_sync.py gate failure, or requests like "improve the docs", "where should this be documented", "this doc is too long".
---

# Applying the markpost documentation standard

The rules live in [docs/AGENTS.md](../../../docs/AGENTS.md). This workflow covers placement, writing discipline, and validation. Guidance, not a script.

## Placement: one fact, one home

Before writing, check [specs/index.md](../../../specs/index.md) for an existing home; grep a distinctive phrase to catch duplicates. New content goes to the tier whose job it is; everywhere else links there.

| Tier | Job |
| --- | --- |
| `README.md` / `README_zh.md` | User-facing product docs |
| `AGENTS.md` | Standing orders for agents, 1–3 lines per rule, linking its home |
| `PRINCIPLES.md` | Behavioral constraints, each anchored to a real incident |
| `specs/` | Current-state design reference; `specs/index.md` is the authoritative index |
| `docs/` | Operation guides (development, deployment); [docs/AGENTS.md](../../../docs/AGENTS.md) owns the doc rules |
| `mrfc/` | Decision records — the why and what was given up ([mrfc/README.md](../../../mrfc/README.md)) |
| `CHANGELOG.md` / `KNOWN_ISSUES.md` | Ledgers — narrate history by design, exempt from prose gates |
| `.claude/skills/` | Reusable workflows (then run `scripts/sync_agents.py`) |

Rationale and change stories go to `mrfc/`, never into `specs/` prose. Procedures ("how to deploy") go to `docs/`; facts about the system ("what the delivery scheduler does") go to `specs/`.

## Writing discipline

- **Current state only** in `specs/` and `docs/`: no "previously/now/no longer/已移除/不再". Name the live mechanism; link the owning MRFC for the why. `verify_md_current.py` gates this.
- **One physical line per paragraph**; let the editor soft-wrap. Code blocks, tables, and list structure keep their formatting. `verify_md_wrap.py` gates this.
- **Relative Markdown links with real targets and `#fragment` anchors**; never bare filenames or free prose references. `verify_md_links.py` gates this.
- **New spec file ⇒ new row in `specs/index.md` in the same change** (and index rows point at existing files). `verify_specs_index.py` gates this.
- Deleting or renaming a doc is atomic: move the content, fix every inbound link, update the index — one change.

## Splitting oversized pages

A spec page over ~40KB is a split candidate: break it into focused pages under the same directory, update `specs/index.md`, fix inbound links, and keep each new page current-state. Split for findability, not word count alone.

## Validate

From the repo root: `python3 scripts/doc_sync.py` (all gates over the full corpus; add file paths to restrict). It also runs as the prek `doc-check` hook on commit and in CI. When a gate fails: fix the doc, not the gate; if the gate itself is wrong, change it in the same change and say why in the MRFC.
