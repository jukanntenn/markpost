#!/usr/bin/env bash
# Sync AI tool configs from their source of truth:
#   - AGENTS.md → CLAUDE.md (Claude Code reads CLAUDE.md)
#   - .claude/skills/ → .agents/skills/ (ZCode reads .agents/skills/, not .claude/)
# Run this whenever AGENTS.md or .claude/skills/ changes.
# Checked by the prek agents-sync hook.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

cp AGENTS.md CLAUDE.md
echo "Synced CLAUDE.md <- AGENTS.md"

rsync -a --delete .claude/skills/ .agents/skills/
echo "Synced .agents/skills/ <- .claude/skills/"
