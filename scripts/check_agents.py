#!/usr/bin/env python3
# Verify AI tool configs are in sync, fail if not.
# Companion to scripts/sync_agents.py which performs the fix.
# prek pre-commit hook (agents-sync).
from __future__ import annotations

import filecmp
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def _collect(root: Path) -> dict[Path, Path]:
    return {p.relative_to(root): p for p in root.rglob("*") if p.is_file()}


def _dirs_equal(a: Path, b: Path) -> bool:
    if not a.is_dir() or not b.is_dir():
        return False
    fa, fb = _collect(a), _collect(b)
    if set(fa) != set(fb):
        return False
    return all(filecmp.cmp(fa[k], fb[k], shallow=False) for k in fa)


def main() -> int:
    agents, claude = ROOT / "AGENTS.md", ROOT / "CLAUDE.md"
    if not filecmp.cmp(agents, claude, shallow=False):
        print("ERROR: AGENTS.md != CLAUDE.md — run scripts/sync_agents.py")
        return 1
    if not _dirs_equal(ROOT / ".claude" / "skills", ROOT / ".agents" / "skills"):
        print("ERROR: .claude/skills/ != .agents/skills/ — run scripts/sync_agents.py")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
