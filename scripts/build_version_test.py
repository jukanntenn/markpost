#!/usr/bin/env python3
"""Unit tests for the shared version-string computation (stdlib unittest)."""
from __future__ import annotations

import os
import re
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("build_version.py")

DIRTY_SUFFIX = re.compile(r"^v0\.0\.1-dirty\.[0-9a-f]{8}$")


# Scratch-repo subprocesses must not inherit the outer repository's git env:
# when prek runs this suite from a git hook it exports GIT_DIR/GIT_WORK_TREE/
# GIT_INDEX_FILE, which override git's -C and route the scratch repo's
# add/commit straight into the real repository's index.
def clean_env() -> dict[str, str]:
    return {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}


def git(repo: Path, *args: str) -> None:
    subprocess.run(
        ["git", "-C", str(repo), *args], check=True, capture_output=True,
        env=clean_env(),
    )


def version(repo: Path) -> str:
    result = subprocess.run(
        [sys.executable, str(SCRIPT), "--repo", str(repo)],
        check=True,
        capture_output=True,
        text=True,
        env=clean_env(),
    )
    return result.stdout.strip()


class BuildVersionTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.repo = Path(self.tmp.name)
        git(self.repo, "init", "-q")
        (self.repo / "README.md").write_text("markpost test repo\n")
        git(self.repo, "-c", "user.name=t", "-c", "user.email=t@example.com",
            "add", "README.md")
        git(self.repo, "-c", "user.name=t", "-c", "user.email=t@example.com",
            "commit", "-q", "-m", "initial")
        git(self.repo, "tag", "v0.0.1")

    def write(self, name: str, content: str) -> None:
        path = self.repo / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

    def test_clean_tree_is_bare_describe(self):
        self.assertEqual(version(self.repo), "v0.0.1")

    def test_repo_without_commits_yields_dev(self):
        repo = Path(tempfile.mkdtemp())
        self.addCleanup(self._rm, repo)
        git(repo, "init", "-q")
        self.assertEqual(version(repo), "dev")

    def test_directory_outside_git_yields_dev(self):
        repo = Path(tempfile.mkdtemp())
        self.addCleanup(self._rm, repo)
        self.assertEqual(version(repo), "dev")

    def test_dirty_tracked_file_gets_deterministic_digest(self):
        self.write("README.md", "changed\n")
        first = version(self.repo)
        self.assertRegex(first, DIRTY_SUFFIX)
        self.assertEqual(version(self.repo), first)

    def test_different_dirty_content_compares_unequal(self):
        self.write("README.md", "changed one way\n")
        one = version(self.repo)
        self.write("README.md", "changed another way\n")
        two = version(self.repo)
        self.assertNotEqual(one, two)

    def test_reverting_restores_the_clean_string(self):
        self.write("README.md", "changed\n")
        self.assertRegex(version(self.repo), DIRTY_SUFFIX)
        self.write("README.md", "markpost test repo\n")
        self.assertEqual(version(self.repo), "v0.0.1")

    def test_staged_change_is_dirty(self):
        self.write("README.md", "staged but uncommitted\n")
        git(self.repo, "add", "README.md")
        self.assertRegex(version(self.repo), DIRTY_SUFFIX)

    def test_untracked_file_contents_move_the_digest(self):
        self.write("notes.txt", "one\n")
        one = version(self.repo)
        self.assertRegex(one, DIRTY_SUFFIX)
        self.write("notes.txt", "two\n")
        self.assertNotEqual(version(self.repo), one)
        (self.repo / "notes.txt").unlink()
        self.assertEqual(version(self.repo), "v0.0.1")

    def test_untracked_files_inside_new_directories_are_enumerated(self):
        # A collapsed directory entry ("?? newdir/") would hash the path but
        # not the contents, so two states differing only inside a fresh
        # directory must still compare unequal.
        self.write("newdir/nested.txt", "one\n")
        one = version(self.repo)
        self.assertRegex(one, DIRTY_SUFFIX)
        self.write("newdir/nested.txt", "two\n")
        self.assertNotEqual(version(self.repo), one)

    def _rm(self, path: Path) -> None:
        import shutil

        shutil.rmtree(path, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
