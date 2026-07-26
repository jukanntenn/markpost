#!/usr/bin/env bash
# Validate Conventional Commits format on the commit message.
# prek commit-msg hook passes the commit-msg file path as $1.
set -euo pipefail
msg_file="${1:-}"
msg=$(cat "$msg_file" 2>/dev/null || echo "")
# Allow merge commits and revert-of-revert without scope.
pattern='^(feat|fix|chore|docs|refactor|test|build|style|ci|perf|revert)(\([^)]+\))?!?: .+'
if ! printf '%s' "$msg" | grep -qE "$pattern"; then
  echo "ERROR: commit message must follow Conventional Commits."
  echo "  Expected: <type>(<scope>)?: <summary>"
  echo "  Types: feat|fix|chore|docs|refactor|test|build|style|ci|perf|revert"
  echo "  Got: $msg"
  exit 1
fi
