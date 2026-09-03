#!/usr/bin/env python3
"""Compute markpost's image version string — the single shared home.

docker/build.py bakes the output into the image as the VERSION build-arg
(`-X main.version`, reported at /api/v1/version), and devops/ansible/deploy.yml
recomputes it on the deploying checkout for the dev post-deploy version check.
Two consumers that must agree byte-for-byte, so the computation lives here and
nowhere else (MRFC 2026-09-03-dirty-tree-image-version-string):

  clean tree  ->  `git describe --tags --always`
                  (release images are CI builds from clean tag checkouts, so
                  a release string is exactly the tag)

  dirty tree  ->  <describe>-dirty.<8 hex>
                  a deterministic digest of the base commit plus the
                  working-tree delta: the tracked diff and the contents of
                  untracked non-ignored files. Different dirty builds of the
                  same commit compare unequal; rebuilding an identical tree
                  reproduces the string.

The digest sees what git sees: files ignored by git stay out even when
.dockerignore lets them into the build context, and submodule-internal dirt is
not digested (the submodule pointer change is). Determinism is required only
per checkout, which is what the deploy check compares.

No git repository, or one without commits, yields "dev" — build.py's historic
fallback. Any git failure after describe succeeds is fatal (exit 1): content
identity cannot be verified, and guessing would bake an ambiguous string.
"""
from __future__ import annotations

import argparse
import hashlib
import os
import subprocess
import sys
from pathlib import Path


class VersionError(RuntimeError):
    """A git or filesystem failure that leaves content identity unverifiable."""


def _git(repo: Path, *args: str, binary: bool = False) -> bytes | str:
    result = subprocess.run(
        ["git", "-C", str(repo), *args], capture_output=True, check=False
    )
    if result.returncode != 0:
        raise VersionError(
            f"git {' '.join(args)} failed: {result.stderr.decode(errors='replace').strip()}"
        )
    return result.stdout if binary else result.stdout.decode(errors="replace")


# Pinned diff config: the output bytes are hashed, so config-dependent output
# (rename detection, prefixes, algorithm) must not vary with the caller's git
# configuration.
_DIFF_ARGS = (
    "-c",
    "diff.renames=false",
    "-c",
    "diff.noprefix=false",
    "-c",
    "diff.algorithm=histogram",
    "diff",
    "--binary",
    "--no-ext-diff",
    "--no-textconv",
    "HEAD",
)


def version_string(repo: Path) -> str:
    try:
        base = _git(repo, "describe", "--tags", "--always").strip()
    except (VersionError, FileNotFoundError):
        # Not a git repository, or no commits yet: build.py's historic fallback.
        return "dev"
    if not base:
        return "dev"

    head = _git(repo, "rev-parse", "HEAD").strip()
    status = _git(
        repo, "status", "--porcelain=v1", "-z", "--untracked-files=all", binary=True
    )
    if not status:
        return base

    digest = hashlib.sha256()
    digest.update(head.encode())
    digest.update(b"\x00")
    digest.update(_git(repo, *_DIFF_ARGS, binary=True))
    digest.update(status)

    # Untracked files carry no diff bytes of their own — hash their contents.
    # With --untracked-files=all every file is enumerated, never a collapsed
    # directory, so content inside a fresh directory still moves the digest.
    for entry in sorted(e for e in status.split(b"\x00") if e.startswith(b"?? ")):
        path = entry[3:]
        digest.update(path)
        digest.update(b"\x00")
        try:
            digest.update((repo / os.fsdecode(path)).read_bytes())
        except OSError as e:
            raise VersionError(f"cannot read untracked file {path!r}: {e}") from e

    return f"{base}-dirty.{digest.hexdigest()[:8]}"


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Compute the markpost image version string (see module docstring)."
    )
    parser.add_argument(
        "--repo",
        default=str(Path(__file__).resolve().parent.parent),
        help="repository root (default: the checkout this script lives in)",
    )
    args = parser.parse_args(argv)
    try:
        print(version_string(Path(args.repo).resolve()))
    except VersionError as e:
        print(f"ERROR: {e}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
