#!/usr/bin/env python3
"""Unit tests for scripts/prek_patch_audit.py (stdlib unittest)."""
from __future__ import annotations

import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path

spec = importlib.util.spec_from_file_location(
    "prek_patch_audit", Path(__file__).with_name("prek_patch_audit.py")
)
audit = importlib.util.module_from_spec(spec)
sys.modules["prek_patch_audit"] = audit
spec.loader.exec_module(audit)


def always_alive(pid: int) -> bool:
    return True


def never_alive(pid: int) -> bool:
    return False


class ParsePatchName(unittest.TestCase):
    def test_valid(self):
        self.assertEqual(audit.parse_patch_name("1787413756161-2670204.patch"), (1787413756, 2670204))

    def test_rejects_non_keeper_names(self):
        for name in ("readme.md", "abc-def.patch", "123-.patch", "-456.patch", "1787413756161-2670204"):
            self.assertIsNone(audit.parse_patch_name(name))


class Audit(unittest.TestCase):
    now = 1_800_000_000

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.patches = Path(self.tmp.name)

    def tearDown(self):
        self.tmp.cleanup()

    def touch(self, name: str) -> None:
        (self.patches / name).write_bytes(b"x")

    def test_orphan_flagged(self):
        self.touch("1799990000000-999999999.patch")
        findings = audit.audit(self.patches, self.now, alive=never_alive)
        self.assertEqual([f.path.name for f in findings], ["1799990000000-999999999.patch"])

    def test_live_pid_stays_silent(self):
        self.touch("1799990000000-999999999.patch")
        self.assertEqual(audit.audit(self.patches, self.now, alive=always_alive), [])

    def test_fresh_patch_stays_silent(self):
        self.touch(f"{(self.now - 60) * 1000}-999999999.patch")
        self.assertEqual(audit.audit(self.patches, self.now, alive=never_alive), [])

    def test_missing_directory_is_silent(self):
        self.assertEqual(audit.audit(self.patches / "nope", self.now), [])

    def test_non_keeper_files_ignored(self):
        (self.patches / "README").write_bytes(b"x")
        self.assertEqual(audit.audit(self.patches, self.now, alive=never_alive), [])

    def test_age_reported_in_seconds(self):
        self.touch("1799990000000-999999999.patch")
        findings = audit.audit(self.patches, self.now, alive=never_alive)
        self.assertEqual(findings[0].age_seconds, self.now - 1799990000)


if __name__ == "__main__":
    unittest.main()
