#!/usr/bin/env python3
# ZCode Stop: full-tree lint gate delegated to prek (the lint group). prek is
# the single source of truth; this adapter only translates prek's exit code
# into ZCode's block decision. Fires once per turn (stopHookActive guard).
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]

REASON = """Lint errors must be fixed before finishing.

Diagnostics:
<lint_output>
{diagnostics}
</lint_output>

Re-run the failing linter(s) and verify they exit 0 before finishing. This gate
fires once per turn; if you stop again with errors remaining they slip through to CI."""


def main() -> None:
    try:
        payload = json.loads(sys.stdin.read())
    except json.JSONDecodeError:
        return
    if payload.get("stopHookActive"):
        return

    r = subprocess.run(
        ["prek", "run", "--group", "lint", "--all-files"],
        cwd=str(ROOT),
        capture_output=True,
        text=True,
    )
    if r.returncode != 0:
        print(
            json.dumps(
                {"decision": "block", "reason": REASON.format(diagnostics=(r.stdout + r.stderr).strip())}
            )
        )


if __name__ == "__main__":
    main()
    sys.exit(0)
