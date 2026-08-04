#!/usr/bin/env python3

from __future__ import annotations

import json
from pathlib import PurePath
import subprocess
import sys


PRETTIER_EXTS = {
    ".ts",
    ".tsx",
    ".js",
    ".jsx",
    ".json",
    ".css",
    ".md",
    ".yaml",
    ".yml",
    ".html",
}
CADDYFILE_STAGES = {"dev", "staging", "local", "production"}


def is_caddyfile(path: PurePath) -> bool:
    if path.name == "Caddyfile":
        return True
    parts = path.name.split(".")
    return parts[0] == "Caddyfile" and len(parts) == 2 and parts[1] in CADDYFILE_STAGES


def commands_for(path: PurePath) -> list[list[str]]:
    match path.suffix:
        case ".py" | ".pyi":
            return [
                ["uv", "run", "ruff", "check", "--fix", str(path)],
                ["uv", "run", "ruff", "format", str(path)],
            ]
        case ".go":
            return [["gofmt", "-w", str(path)], ["goimports", "-w", str(path)]]
        case s if s in PRETTIER_EXTS:
            return [["prettier", "--write", str(path)]]
        case ".toml":
            return [["oxfmt", "--write", str(path)]]
        case ".j2":
            return [["djlint", "--reformat", "--profile=jinja", str(path)]]
        case _:
            return [["caddy", "fmt", str(path)]] if is_caddyfile(path) else []


def tool_name(cmd: list[str]) -> str:
    return cmd[2] if cmd[:2] == ["uv", "run"] else cmd[0]


def main() -> None:
    try:
        payload = json.loads(sys.stdin.read())
    except json.JSONDecodeError:
        return

    raw_path = (payload.get("toolInput") or {}).get("file_path")
    if not isinstance(raw_path, str):
        return

    for cmd in commands_for(PurePath(raw_path)):
        name = tool_name(cmd)
        try:
            result = subprocess.run(cmd, capture_output=True, text=True)
        except FileNotFoundError:
            print(
                f"[zcode-post-tool-use] {name} not found on PATH; skipped",
                file=sys.stderr,
            )
            continue
        if result.returncode != 0:
            print(
                f"[zcode-post-tool-use] {name} reported issues for {raw_path}:",
                file=sys.stderr,
            )
            if result.stdout:
                print(result.stdout, file=sys.stderr)
            if result.stderr:
                print(result.stderr, file=sys.stderr)


if __name__ == "__main__":
    main()
    sys.exit(0)
