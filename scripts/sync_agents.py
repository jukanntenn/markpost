#!/usr/bin/env python3
# Sync AI tool configs from their source of truth:
#   - AGENTS.md → CLAUDE.md (Claude Code reads CLAUDE.md)
#   - .claude/skills/ → .agents/skills/ (ZCode reads .agents/skills/)
# Run this whenever AGENTS.md or .claude/skills/ changes.
# Checked by the prek agents-sync hook.
from __future__ import annotations

import shutil
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

shutil.copy2(ROOT / "AGENTS.md", ROOT / "CLAUDE.md")
print("Synced CLAUDE.md <- AGENTS.md")

dst = ROOT / ".agents" / "skills"
if dst.exists():
    shutil.rmtree(dst)
shutil.copytree(ROOT / ".claude" / "skills", dst)
print("Synced .agents/skills/ <- .claude/skills/")
