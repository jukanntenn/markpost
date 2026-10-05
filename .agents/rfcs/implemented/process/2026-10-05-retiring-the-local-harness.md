# RFC: Retire the local harness in favor of hdsh

Status: implemented

English | [中文](2026-10-05-retiring-the-local-harness.zh.md)

## Problem

markpost ran a hand-rolled harness of its own design: the `.github/issue-management/policy.py` engine with its two workflows and issue templates, the `scripts/doc_sync.py` gate family (wrap, links, current-state, pairs, budgets, MRFC format, specs index), the `.agents/mrfcs/` decision tree, and per-file manifests under `scripts/`. That harness — the predecessor the hdsh project itself was extracted from — now exists twice: once local, once upstream, maintained in parallel. Every gate fix or contract change had to land in both, and the local copy carried markpost-specific edges (a `Loop status` board field, uppercase `P0-P3` options, a two-link switcher convention) that no longer carry their weight.

## Decision

markpost adopts [hdsh](https://github.com/jukanntenn/harness-deepseek-harness) (pinned by full SHA in [`prek.toml`](../../../../prek.toml) and [`.hdsh/adopt.manifest.json`](../../../../.hdsh/adopt.manifest.json)) and retires the local harness in the same change, per the no-dual-standard adoption contract:

- The issue engine, its workflows, templates, and `.github/issue-management/` are deleted; hdsh's thin workflows and upstream composite action replace them, bound to a new Project board (#6, "markpost Issue Management") whose built-in Status field carries the seven standard statuses; the incumbent board #1 and its `Loop status` field are abandoned with the old engine.
- `.agents/mrfcs/` moves to `.agents/rfcs/` under the class taxonomy (`feature`/`bug-fix`/`simplification`/`architecture`/`process`/`testing`), retitled `# MRFC:` → `# RFC:`, with pairing sidecars recorded per pair.
- The doc-gate scripts and their manifests are deleted; word ceilings fold into [`.hdsh/docs.manifest.json`](../../../../.hdsh/docs.manifest.json) (docs/AGENTS.md: 600 → 750, justified by the merged hdsh standard's irreducible size), the `specs/` corpus subtree into [`.hdsh/pairing.manifest.json`](../../../../.hdsh/pairing.manifest.json) roots, and the seven hdsh gates run through the adopt-managed prek block and [`.github/workflows/docs.yml`](../../../../.github/workflows/docs.yml).
- `scripts/agentlib.py`, `check_agent_instructions.py`, and `sync_agent_instructions.py` stay: the AGENTS↔CLAUDE and skills-mirror contracts are markpost-local and orthogonal to the documentation standard. The `.claude/skills/` mirror is bootstrapped to include the hdsh-installed skills so the direction-free mirror never sees them as phantom deletions.

## Alternatives considered

- **Run both standards side by side** — rejected: two gates over one corpus drift, and hdsh's adoption contract explicitly refuses a dual standard.
- **Keep the local harness, skip hdsh** — rejected: the parallel-maintenance cost is the problem being solved.
- **Reuse board #1** — rejected: its statuses live on a custom `Loop status` field; hdsh's contract addresses the built-in Status field, and the incumbent board carries the old loop's item state.

## Consequences

Gate semantics tighten in places hdsh is stricter than the local family: switchers are single-link per side after the first H1; fenced code blocks are byte-identical between languages (comments included); pairing records are per-pair `.i18n.yaml` sidecars rather than a central language manifest; and `docs/development.md` stays consumer-owned with its pair unrecorded until reviewed. The prek `hdsh` group and the pinned install in `docs.yml` must move together on upgrades (rerun `hdsh adopt apply` under the newer ref). PR and issue mechanics (labels `kind/*`, `type/*`, `p0`-`p3`; the machine account; one-approval branch protection) are unchanged in substance.
