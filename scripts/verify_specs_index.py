#!/usr/bin/env python3
# specs/index.md is the authoritative index: every file under specs/
# (except index.md itself) must be listed there, and every specs/ file
# the index links to must exist. Catches new specs committed without an
# index row and index rows pointing at renamed/deleted files. Runs in
# full whenever any specs/ path is in the checked set. Part of
# doc_sync.py.
from __future__ import annotations

import re
import sys

from doclib import ROOT, collect, mask_source

INDEX = ROOT / "specs" / "index.md"
_LINK = re.compile(r"\]\(([^)]+)\)")


def linked_specs() -> set[str]:
    """specs/-relative paths linked from the index (targets resolving
    outside specs/, e.g. ../docs/, are the link gate's concern)."""
    masked = mask_source(INDEX.read_text(encoding="utf-8"))
    out: set[str] = set()
    for target in _LINK.findall(masked):
        if "://" in target or target.startswith(("#", "/")):
            continue
        if target.startswith("../"):
            continue
        out.add("specs/" + target.lstrip("./").split("#", 1)[0].split("?", 1)[0])
    return out


def main(argv: list[str]) -> int:
    if argv[1:]:
        # Restricted mode (prek passes staged files): run the full check
        # whenever a specs/ file is in the set; otherwise nothing to do.
        if not any(arg.startswith("specs/") for arg in argv[1:]):
            print("verify_specs_index: no specs/ files in scope, nothing to check")
            return 0

    spec_files = {str(p.relative_to(ROOT)) for p in collect(["specs/*.md", "specs/**/*.md"])}
    spec_files.discard("specs/index.md")
    linked = linked_specs()

    missing = sorted(spec_files - linked)
    dangling = sorted(m for m in linked if not (ROOT / m).is_file())

    problems = [f"  {f} is not listed in specs/index.md" for f in missing]
    problems += [f"  specs/index.md links to {f}, which does not exist" for f in dangling]

    if problems:
        print("verify_specs_index: index and tree disagree:", file=sys.stderr)
        print("\n".join(problems), file=sys.stderr)
        return 1
    print(f"verify_specs_index: {len(spec_files)} spec file(s) all listed in specs/index.md")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
