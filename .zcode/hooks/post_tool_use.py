#!/usr/bin/env python3
# ZCode PostToolUse (Edit|Write): delegate formatting to prek (the fmt group,
# single source of truth). No formatter logic here. ZCode's payload uses
# camelCase keys (toolInput).
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


def main() -> None:
    try:
        payload = json.loads(sys.stdin.read())
    except json.JSONDecodeError:
        return

    file_path = (payload.get("toolInput") or {}).get("file_path")
    if not isinstance(file_path, str) or not (Path(file_path).is_file() or (ROOT / file_path).is_file()):
        return

    r = subprocess.run(
        ["prek", "run", "--group", "fmt", "--files", file_path],
        cwd=str(ROOT),
        capture_output=True,
        text=True,
    )
    # exit 0 = clean, 1 = files modified (expected for a formatter); surface only real errors.
    if r.returncode not in (0, 1):
        print(f"[zcode-post-tool-use] prek fmt exited {r.returncode}:", file=sys.stderr)
        if r.stdout:
            print(r.stdout, file=sys.stderr)
        if r.stderr:
            print(r.stderr, file=sys.stderr)


if __name__ == "__main__":
    main()
    sys.exit(0)
