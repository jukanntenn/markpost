#!/usr/bin/env bash
# Sync .codex/hooks from .claude/hooks (.claude is the source of truth).
# The hook scripts are agent-agnostic (both read tool_input.file_path, both
# honor Stop decision: block). Run after editing .claude/hooks/*.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
mkdir -p .codex/hooks
cp .claude/hooks/format.py .codex/hooks/format.py
cp .claude/hooks/lint_stop.py .codex/hooks/lint_stop.py
# Remove stale scripts no longer used.
rm -f .codex/hooks/lint.py
echo "Synced .codex/hooks <- .claude/hooks"
