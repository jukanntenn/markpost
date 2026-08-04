#!/usr/bin/env bash
# Verify AI tool configs are in sync, fail if not.
# Companion to scripts/sync-agents.sh which performs the fix.
# prek pre-commit hook (agents-sync).
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

if ! diff -q AGENTS.md CLAUDE.md >/dev/null; then
  echo "ERROR: AGENTS.md != CLAUDE.md — run scripts/sync-agents.sh"
  diff -u AGENTS.md CLAUDE.md || true
  exit 1
fi

if ! diff -rq .claude/skills/ .agents/skills/ >/dev/null; then
  echo "ERROR: .claude/skills/ != .agents/skills/ — run scripts/sync-agents.sh"
  diff -rq .claude/skills/ .agents/skills/ || true
  exit 1
fi
