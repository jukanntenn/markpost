#!/usr/bin/env python3
"""PostToolUse hook: format the edited file. Never blocks (exit 0 always)."""

import json
import subprocess
import sys
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[2]
BACKEND = PROJECT_ROOT / "backend"
FRONTEND = PROJECT_ROOT / "frontend"


def run(cmd, cwd=None):
    try:
        subprocess.run(
            cmd,
            cwd=str(cwd) if cwd else None,
            capture_output=True,
            text=True,
            timeout=30,
        )
    except (FileNotFoundError, subprocess.TimeoutExpired):
        pass


def main():
    try:
        data = json.load(sys.stdin)
    except Exception:
        sys.exit(0)

    file_path = data.get("tool_input", {}).get("file_path", "")
    if not file_path:
        sys.exit(0)

    abs_path = Path(file_path)
    if not abs_path.is_absolute():
        abs_path = PROJECT_ROOT / file_path
    suffix = abs_path.suffix

    # Go: gofmt + goimports on the single file (fast; golangci-lint fmt has no single-file mode).
    if suffix == ".go":
        run(["gofmt", "-w", str(abs_path)])
        run(["goimports", "-w", str(abs_path)])
    # Frontend: prettier --write on the single file.
    elif suffix in (".ts", ".tsx", ".js", ".jsx", ".json", ".css", ".md") and str(
        abs_path
    ).startswith(str(FRONTEND)):
        try:
            rel = abs_path.relative_to(FRONTEND)
            run(["prettier", "--write", str(rel)], cwd=FRONTEND)
        except ValueError:
            pass
    # Python: ruff format (devops scripts, hooks).
    elif suffix == ".py":
        run(["ruff", "format", str(abs_path)])

    sys.exit(0)


if __name__ == "__main__":
    main()
