#!/usr/bin/env python3

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[2]
BACKEND = PROJECT_ROOT / "backend"
FRONTEND = PROJECT_ROOT / "frontend"

# (cwd, command, label) — full-tree quality gates, mirroring prek.toml pre-commit
# and CI lint.yml. No --fix (Stop is a gate, not an auto-formatter).
LINTS: list[tuple[Path, list[str], str]] = [
    (BACKEND, ["golangci-lint", "run", "./..."], "golangci-lint (backend)"),
    (FRONTEND, ["pnpm", "lint"], "eslint (frontend)"),
    (FRONTEND, ["pnpm", "typecheck"], "tsc (frontend)"),
]

REASON_TEMPLATE = """Lint errors must be fixed before finishing.

Diagnostics:
<lint_output>
{diagnostics}
</lint_output>

Required:
1. Fix every diagnostic above with a real code change. Do not silence them with `// eslint-disable`, `@ts-ignore`, inline rule disables, or `type: ignore` - only treat a diagnostic as a false positive if you can justify why.
2. After editing, re-run the failing linter(s) yourself to verify they exit 0 with no output.
3. Only attempt to finish again once those commands are clean.

This enforcement fires once per turn - the stop hook will not block a second time. If you stop again with lint errors remaining, they will slip through to CI. Verify before you finish."""


def active_lints() -> list[tuple[Path, list[str], str]]:
    out = []
    for cwd, cmd, label in LINTS:
        sentinel = cwd / ("go.mod" if cwd == BACKEND else "package.json")
        if sentinel.exists():
            out.append((cwd, cmd, label))
    return out


def run_lint(cwd: Path, cmd: list[str]) -> str | None:
    try:
        r = subprocess.run(
            cmd, cwd=str(cwd), capture_output=True, text=True, timeout=180
        )
    except (FileNotFoundError, subprocess.TimeoutExpired) as e:
        print(f"[stop-hook] {cmd[0]} unavailable ({e}); skipping", file=sys.stderr)
        return None
    return None if r.returncode == 0 else (r.stdout + r.stderr).strip()


def main() -> None:
    try:
        payload = json.loads(sys.stdin.read())
    except json.JSONDecodeError:
        return

    if payload.get("stop_hook_active"):
        return

    errors = []
    for cwd, cmd, label in active_lints():
        out = run_lint(cwd, cmd)
        if out:
            errors.append(f"{label}:\n{out}")

    if errors:
        diagnostics = "\n\n".join(errors)
        print(
            json.dumps(
                {
                    "decision": "block",
                    "reason": REASON_TEMPLATE.format(diagnostics=diagnostics),
                }
            )
        )


if __name__ == "__main__":
    main()
    sys.exit(0)
