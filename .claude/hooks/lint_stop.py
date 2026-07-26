#!/usr/bin/env python3
"""Stop hook: run full lint before the agent finishes; block on errors.

Reads `stop_hook_active` from stdin; if true, exits 0 to avoid infinite loops
(Claude forces stop after 8 consecutive blocks anyway). Otherwise runs
golangci-lint (backend) + eslint (frontend) on changed files; on error prints
JSON {decision: block, reason: ...} and exits 0 (exit 0 + JSON is the
documented way to block Stop; exit 2 ignores JSON).
"""

import json
import subprocess
import sys
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[2]
BACKEND = PROJECT_ROOT / "backend"
FRONTEND = PROJECT_ROOT / "frontend"


def changed_go() -> bool:
    try:
        r = subprocess.run(
            ["git", "diff", "--name-only", "--cached", "--diff-filter=ACMR", "HEAD"],
            cwd=PROJECT_ROOT,
            capture_output=True,
            text=True,
            timeout=10,
        )
        return any(f.endswith(".go") for f in r.stdout.split())
    except Exception:
        return False


def run_lint(cmd, cwd):
    try:
        r = subprocess.run(
            cmd, cwd=str(cwd), capture_output=True, text=True, timeout=180
        )
        return None if r.returncode == 0 else (r.stdout + r.stderr).strip()
    except Exception as e:
        return None  # tool missing/unavailable: don't block


def block(reason):
    print(json.dumps({"decision": "block", "reason": reason}))
    sys.exit(0)


def main():
    try:
        data = json.load(sys.stdin)
    except Exception:
        sys.exit(0)

    if data.get("stop_hook_active"):
        sys.exit(0)  # already looping; let it stop

    errors = []

    # Backend: full golangci-lint run (has caching; acceptable per-turn).
    if (BACKEND / "go.mod").exists():
        err = run_lint(["golangci-lint", "run", "./..."], BACKEND)
        if err:
            errors.append("golangci-lint (backend):\n" + err)

    # Frontend: full eslint.
    if (FRONTEND / "package.json").exists():
        err = run_lint(["pnpm", "lint"], FRONTEND)
        if err:
            errors.append("eslint (frontend):\n" + err)

    if errors:
        block("Lint errors must be fixed before finishing:\n\n" + "\n\n".join(errors))
    sys.exit(0)


if __name__ == "__main__":
    main()
