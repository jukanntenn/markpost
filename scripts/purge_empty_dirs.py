#!/usr/bin/env python3
# Interactively list and delete empty directories (utility, not a prek hook).
from __future__ import annotations

import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EXCLUDE_PARTS = {".git", "node_modules", ".next"}


def _excluded(p: Path) -> bool:
    return any(part in EXCLUDE_PARTS for part in p.parts)


def main() -> int:
    empties = [
        p
        for p in ROOT.rglob("*")
        if p.is_dir() and not _excluded(p) and not any(p.iterdir())
    ]
    empties.sort(reverse=True)
    print(f"=== {len(empties)} empty directory(s) ===")
    for p in empties[:20]:
        print(f"  {p.relative_to(ROOT)}")
    if len(empties) > 20:
        print("  ...")

    if not empties:
        return 0
    confirm = input("Delete these directories? (y/n) ").strip().lower()
    if confirm != "y":
        return 0
    for p in empties:
        try:
            p.rmdir()
        except OSError:
            pass
    print("Done")
    return 0


if __name__ == "__main__":
    sys.exit(main())
