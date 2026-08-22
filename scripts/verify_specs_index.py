#!/usr/bin/env python3
# specs/index.md and its .zh.md twin are the authoritative index: every
# spec pair under specs/ (the indexes themselves aside) has exactly one
# row in each index — a row covers the pair's stem, the English index
# linking the .md and the Chinese index the .zh.md — and every specs/
# target an index links to must exist. Catches new specs committed
# without an index row and index rows pointing at renamed/deleted files.
# Link locale itself is verify_doc_pairs.py's check. Runs in full
# whenever any specs/ path is in the checked set. Part of doc_sync.py.
from __future__ import annotations

import re
import sys

from doclib import ROOT, collect, mask_source

INDEXES = [ROOT / "specs" / "index.md", ROOT / "specs" / "index.zh.md"]
_LINK = re.compile(r"\]\(([^)]+)\)")


def linked_specs(index: Path) -> set[str]:
    """specs/-relative paths linked from one index (targets resolving
    outside specs/, e.g. ../docs/, are the link gate's concern)."""
    masked = mask_source(index.read_text(encoding="utf-8"))
    out: set[str] = set()
    for target in _LINK.findall(masked):
        if "://" in target or target.startswith(("#", "/")):
            continue
        if target.startswith("../"):
            continue
        out.add("specs/" + target.lstrip("./").split("#", 1)[0].split("?", 1)[0])
    return out


def stem(path: str) -> str:
    """specs/foo.md and specs/foo.zh.md share the stem specs/foo."""
    return path[: -len(".zh.md")] if path.endswith(".zh.md") else path


def main(argv: list[str]) -> int:
    if argv[1:]:
        # Restricted mode (prek passes staged files): run the full check
        # whenever a specs/ file is in the set; otherwise nothing to do.
        if not any(arg.startswith("specs/") for arg in argv[1:]):
            print("verify_specs_index: no specs/ files in scope, nothing to check")
            return 0

    spec_files = {str(p.relative_to(ROOT)) for p in collect(["specs/*.md", "specs/**/*.md"])}
    spec_files -= {"specs/index.md", "specs/index.zh.md"}
    stems = {stem(f) for f in spec_files}

    problems: list[str] = []
    for index in INDEXES:
        rel = index.relative_to(ROOT).as_posix()
        if not index.is_file():
            problems.append(f"  {rel} is missing — the index pairs like every spec")
            continue
        linked = linked_specs(index)
        dangling = sorted(t for t in linked if not (ROOT / t).is_file())
        linked_stems = {stem(t) for t in linked}
        missing = sorted(stems - linked_stems)
        unknown = sorted(linked_stems - stems)
        problems += [f"  {s} has no row in {rel}" for s in missing]
        problems += [f"  {rel} lists {s}, which has no spec file" for s in unknown]
        problems += [f"  {rel} links to {t}, which does not exist" for t in dangling]

    if problems:
        print("verify_specs_index: index and tree disagree:", file=sys.stderr)
        print("\n".join(problems), file=sys.stderr)
        return 1
    print(f"verify_specs_index: {len(stems)} spec pair(s) all listed in both indexes")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
