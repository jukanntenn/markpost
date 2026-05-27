#!/usr/bin/env python3
"""
PostToolUse hook for linting files after editing.
This hook runs after Edit, Write, or MultiEdit operations:
- Python (.py, .pyi): ruff check --fix
- Go (.go): golangci-lint run --fix
- TypeScript (.ts, .tsx): eslint --fix (from frontend/ directory)
"""

import json
import os
import subprocess
import sys


def run_linter(cmd, file_path, cwd=None):
    """Run a linting tool, returning the CompletedProcess or None if not found."""
    try:
        return subprocess.run(
            [*cmd, file_path],
            capture_output=True,
            text=True,
            cwd=cwd,
        )
    except FileNotFoundError:
        return None


def handle_result(result, tool_name):
    """Handle subprocess result from a linter.
    Exit 0 on success/warning, exit 2 on lint errors to signal agent."""
    if result is None:
        print(f"{tool_name} not found.", file=sys.stderr)
        sys.exit(0)

    if result.returncode == 0:
        if result.stdout:
            sys.stdout.write(result.stdout)
        if result.stderr:
            sys.stderr.write(result.stderr)
        sys.exit(0)
    elif result.returncode == 1:
        # Lint errors found
        if result.stdout:
            sys.stderr.write(result.stdout)
        if result.stderr:
            sys.stderr.write(result.stderr)
        sys.exit(2)
    else:
        # Tool error (non-lint failure)
        if result.stdout:
            sys.stdout.write(result.stdout)
        if result.stderr:
            sys.stderr.write(result.stderr)
        sys.exit(0)


def main():
    """Main function to run lint hooks."""
    try:
        ...
    #     hook_data = json.load(sys.stdin)
    #     tool_input = hook_data.get("tool_input", {})

    #     file_path = tool_input.get("file_path", "")

    #     if not file_path:
    #         print("No file path provided, skipping lint.")
    #         sys.exit(0)

    #     # Python: ruff check --fix
    #     if file_path.endswith((".py", ".pyi")):
    #         result = run_linter(
    #             [
    #                 "ruff",
    #                 "check",
    #                 "--fix",
    #                 "--select",
    #                 "F,E,W,I",
    #                 "--line-length",
    #                 "120",
    #             ],
    #             file_path,
    #         )
    #         if result is None:
    #             print("Ruff not found. Please install ruff.", file=sys.stderr)
    #             sys.exit(0)

    #         if result.returncode == 0:
    #             if result.stdout:
    #                 sys.stdout.write(result.stdout)
    #             if result.stderr:
    #                 sys.stderr.write(result.stderr)
    #             sys.exit(0)
    #         elif result.returncode == 1:
    #             if result.stdout:
    #                 sys.stderr.write(result.stdout)
    #             if result.stderr:
    #                 sys.stderr.write(result.stderr)
    #             sys.exit(2)
    #         else:
    #             if result.stdout:
    #                 sys.stdout.write(result.stdout)
    #             if result.stderr:
    #                 sys.stderr.write(result.stderr)
    #             sys.exit(0)

    #     # Go: golangci-lint run --fix (from backend/ where go.mod lives)
    #     if file_path.endswith(".go"):
    #         script_dir = os.path.dirname(os.path.abspath(__file__))
    #         project_root = os.path.normpath(os.path.join(script_dir, "..", ".."))
    #         abs_file = os.path.normpath(os.path.join(project_root, file_path))
    #         backend_dir = os.path.join(project_root, "backend")
    #         rel_path = os.path.relpath(abs_file, backend_dir)
    #         result = run_linter(
    #             ["golangci-lint", "run", "--fix", "--build-tags=integration"],
    #             rel_path,
    #             cwd=backend_dir,
    #         )
    #         handle_result(result, "golangci-lint")

    #     # TypeScript/TSX: eslint --fix (from frontend/)
    #     if file_path.endswith((".ts", ".tsx")):
    #         # Resolve frontend directory relative to project root
    #         # The hook script is at .claude/hooks/lint.py, project root is ../../
    #         script_dir = os.path.dirname(os.path.abspath(__file__))
    #         project_root = os.path.normpath(os.path.join(script_dir, "..", ".."))
    #         frontend_dir = os.path.join(project_root, "frontend")
    #         # Convert file_path to be relative to frontend/ since eslint runs from there
    #         abs_file = os.path.normpath(os.path.join(project_root, file_path))
    #         rel_path = os.path.relpath(abs_file, frontend_dir)
    #         result = run_linter(
    #             ["npx", "eslint", "--fix"],
    #             rel_path,
    #             cwd=frontend_dir,
    #         )
    #         handle_result(result, "eslint")

    #     # Unrecognized file type: no-op
    #     sys.exit(0)

    except Exception as e:
        print(f"Hook error: {e}", file=sys.stderr)
        sys.exit(0)


if __name__ == "__main__":
    main()
