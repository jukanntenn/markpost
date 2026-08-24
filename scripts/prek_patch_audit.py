#!/usr/bin/env python3
"""Audit prek's patch cache for orphaned stash files (interrupted hook runs).

prek's WorkTreeKeeper saves unstaged changes to
~/.cache/prek/patches/<epoch-ms>-<pid>.patch, wipes them from the worktree,
and re-applies the patch when the process drops. A run killed in between
leaves the worktree silently without that work — the patch file is the only
copy. This script flags patches whose owning pid is dead and whose age
exceeds the grace window, and prints a preview-first recovery path; it never
applies anything itself — the decision record is
.agents/mrfcs/implemented/2026-08-24-prek-stash-lifecycle-hardening.md.
"""
from __future__ import annotations

import os
import sys
import time
from dataclasses import dataclass
from pathlib import Path

GRACE_SECONDS = 3600  # a live hook run takes minutes; an hour is unambiguously orphaned


def default_patches_dir() -> Path:
    cache = os.environ.get("PREK_CACHE")
    base = Path(cache) if cache else Path.home() / ".cache" / "prek"
    return base / "patches"


def parse_patch_name(name: str) -> tuple[int, int] | None:
    """`<epoch-ms>-<pid>.patch` → (epoch_seconds, pid); anything else is not a keeper patch."""
    if not name.endswith(".patch"):
        return None
    stem = name.removesuffix(".patch")
    ms, sep, pid = stem.partition("-")
    if not sep or not ms.isdigit() or not pid.isdigit():
        return None
    return int(ms) // 1000, int(pid)


def pid_alive(pid: int) -> bool:
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True  # exists but is owned by someone else
    return True


@dataclass
class Finding:
    path: Path
    age_seconds: int


def audit(
    patches_dir: Path,
    now: float,
    alive=pid_alive,
    grace_seconds: int = GRACE_SECONDS,
) -> list[Finding]:
    findings: list[Finding] = []
    if not patches_dir.is_dir():
        return findings
    for entry in sorted(patches_dir.iterdir()):
        if not entry.is_file():
            continue
        parsed = parse_patch_name(entry.name)
        if parsed is None:
            continue
        created, pid = parsed
        age = int(now - created)
        if age < grace_seconds or alive(pid):
            continue
        findings.append(Finding(entry, age))
    return findings


def main() -> int:
    patches_dir = default_patches_dir()
    findings = audit(patches_dir, time.time())
    if not findings:
        print(f"prek patch audit: no orphaned patches in {patches_dir}")
        return 0
    print(f"prek patch audit: {len(findings)} orphaned patch(es) in {patches_dir}")
    print("an interrupted prek run may hold work that is missing from the worktree")
    for finding in findings:
        size = finding.path.stat().st_size
        print(f"  {finding.path.name}  (pid dead, {finding.age_seconds // 3600}h old, {size}B)")
    newest = max(findings, key=lambda finding: finding.path.name)
    print("recover preview-first, never a blind apply:")
    print(f"  git diff --no-index /dev/null {newest.path} | less   # inspect the patch")
    print(f"  git apply --check {newest.path}                       # does it fit the tree")
    print(f"  git apply {newest.path}                               # only after both")
    return 1


if __name__ == "__main__":
    sys.exit(main())
