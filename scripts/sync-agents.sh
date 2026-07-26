#!/usr/bin/env bash
# Sync CLAUDE.md from AGENTS.md (AGENTS.md is the source of truth).
# Run this whenever AGENTS.md changes. Checked by the prek agents-sync hook.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
cp AGENTS.md CLAUDE.md
echo "Synced CLAUDE.md <- AGENTS.md"
