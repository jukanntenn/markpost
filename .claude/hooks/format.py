#!/usr/bin/env python3
"""
PostToolUse hook for formatting files after editing.
This hook runs formatting tools after Edit or Write operations:
- Python (.py, .pyi): ruff format
- Go (.go): gofmt + goimports
- TypeScript (.ts, .tsx): no-op (ESLint handles in lint.py)
"""

import json
import subprocess
import sys


def run_tool(cmd, file_path):
    """Run a formatting tool, returning (returncode, stdout, stderr).
    Returns (None, None, None) if the tool is not found."""
    try:
        result = subprocess.run(
            [*cmd, file_path],
            capture_output=True,
            text=True,
        )
        return result.returncode, result.stdout, result.stderr
    except FileNotFoundError:
        return None, None, None


def main():
    """Main function to run format hooks."""
    try:
        hook_data = json.load(sys.stdin)
        tool_input = hook_data.get("tool_input", {})

        file_path = tool_input.get("file_path", "")

        if not file_path:
            sys.exit(0)

        # Python: ruff format
        if file_path.endswith((".py", ".pyi")):
            run_tool(["ruff", "format"], file_path)
            sys.exit(0)

        # Go: gofmt + goimports
        if file_path.endswith(".go"):
            run_tool(["gofmt", "-w"], file_path)
            run_tool(["goimports", "-w"], file_path)
            sys.exit(0)

        # TypeScript: no-op (ESLint handles in lint.py)
        # Unrecognized: no-op

        sys.exit(0)

    except Exception:
        sys.exit(0)


if __name__ == "__main__":
    main()
