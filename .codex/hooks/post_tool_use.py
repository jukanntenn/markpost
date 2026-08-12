#!/usr/bin/env python3
# Codex PostToolUse (apply_patch): extract edited paths from the patch text,
# then delegate formatting to prek — the single source of truth for the
# file→formatter mapping (the fmt group in prek.toml). No formatter logic
# lives here, so it can never drift from prek/CI.
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]

MOVE_TO_PREFIX = "*** Move to: "
PATCH_FILE_PREFIXES = ("*** Update File: ", "*** Add File: ")


def extract_edited_paths(command: str) -> list[str]:
    paths: list[str] = []
    pending: str | None = None
    for raw in command.splitlines():
        line = raw.strip()
        if pending is not None and line.startswith(MOVE_TO_PREFIX):
            paths.append(line[len(MOVE_TO_PREFIX) :].strip())
            pending = None
            continue
        if pending is not None:
            paths.append(pending)
            pending = None
        if line.startswith(MOVE_TO_PREFIX):
            continue
        for prefix in PATCH_FILE_PREFIXES:
            if line.startswith(prefix):
                pending = line[len(prefix) :].strip()
                break
    if pending is not None:
        paths.append(pending)
    return paths


def main() -> None:
    try:
        payload = json.loads(sys.stdin.read())
    except json.JSONDecodeError:
        return

    command = (payload.get("tool_input") or {}).get("command")
    if not isinstance(command, str):
        return

    paths = [
        p
        for p in extract_edited_paths(command)
        if (ROOT / p).is_file() or Path(p).is_file()
    ]
    if not paths:
        return

    r = subprocess.run(
        ["prek", "run", "--group", "fmt", "--files", *paths],
        cwd=str(ROOT),
        capture_output=True,
        text=True,
    )
    # exit 0 = clean, 1 = files modified (expected for a formatter); only surface real errors.
    if r.returncode not in (0, 1):
        print(f"[codex-post-tool-use] prek fmt exited {r.returncode}:", file=sys.stderr)
        if r.stdout:
            print(r.stdout, file=sys.stderr)
        if r.stderr:
            print(r.stderr, file=sys.stderr)


if __name__ == "__main__":
    main()
    sys.exit(0)
