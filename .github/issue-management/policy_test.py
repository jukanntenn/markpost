#!/usr/bin/env python3
"""Unit tests for the issue/pull-request policy logic (stdlib unittest)."""
from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path

spec = importlib.util.spec_from_file_location(
    "policy", Path(__file__).with_name("policy.py")
)
policy = importlib.util.module_from_spec(spec)
spec.loader.exec_module(policy)


def body(visible: str, details: str = "- 验收条件：\n") -> str:
    return f"{visible}\n\n<details>\n<summary>细节</summary>\n\n{details}\n\n</details>\n"


class CountUnits(unittest.TestCase):
    def test_han_only(self):
        self.assertEqual(policy.count_units(body("修复登录失败的问题")), 9)

    def test_latin_tokens(self):
        self.assertEqual(policy.count_units(body("修复 pnpm build 的失败")), 7)

    def test_links_count_text_not_url(self):
        text = body("详见 [说明文档](https://example.com/very/long/path)")
        self.assertEqual(policy.count_units(text), 6)

    def test_html_comments_ignored(self):
        self.assertEqual(policy.count_units(body("<!-- hidden -->修复失败")), 4)

    def test_details_content_excluded(self):
        self.assertEqual(
            policy.count_units(body("修复失败", "- 验收条件：这是一段很长的验收内容")),
            4,
        )


class DetailsShape(unittest.TestCase):
    def test_balanced_collapsed(self):
        shape = policy.extract_outside_details(body("一句话"))
        self.assertTrue(shape["balanced"])
        self.assertEqual(shape["details_count"], 1)
        self.assertTrue(shape["all_collapsed"])

    def test_unbalanced(self):
        self.assertFalse(policy.extract_outside_details("<details><summary>x</summary>")["balanced"])

    def test_open_forbidden(self):
        shape = policy.extract_outside_details("<details open><summary>x</summary></details>")
        self.assertFalse(shape["all_collapsed"])


class References(unittest.TestCase):
    def test_plain_and_closing(self):
        refs = policy.parse_references("Related to #12 and fixes #34.")
        self.assertEqual(refs["all"], [12, 34])
        self.assertEqual(refs["resolving"], [34])
        self.assertEqual(refs["related"], [12])

    def test_same_repo_explicit(self):
        refs = policy.parse_references("Fixes jukanntenn/markpost#7", "jukanntenn/markpost")
        self.assertEqual(refs["resolving"], [7])

    def test_cross_repo_excluded(self):
        refs = policy.parse_references("Fixes other/repo#7 and #9")
        self.assertEqual(refs["all"], [9])

    def test_issue_url(self):
        refs = policy.parse_references(
            "Resolves https://github.com/jukanntenn/markpost/issues/42"
        )
        self.assertEqual(refs["resolving"], [42])

    def test_fenced_code_ignored(self):
        refs = policy.parse_references("text\n```\nfixes #5\n```\nand fixes #6")
        self.assertEqual(refs["resolving"], [6])

    def test_pr_reference_not_issue(self):
        refs = policy.parse_references("implements #13")
        self.assertEqual(refs["all"], [13])
        self.assertEqual(refs["resolving"], [])


class ValidateIssue(unittest.TestCase):
    def issue(self, **over):
        base = {
            "title": "修复登录失败的问题",
            "body": body("修复登录时的崩溃"),
            "labels": ["type/bug"],
            "state": "open",
        }
        base.update(over)
        return base

    def test_valid(self):
        self.assertEqual(policy.validate_issue(self.issue()), [])

    def test_title_needs_han(self):
        self.assertTrue(policy.validate_issue(self.issue(title="fix login crash")))

    def test_title_prefix_banned(self):
        self.assertTrue(policy.validate_issue(self.issue(title="[Bug] 修复登录")))
        self.assertTrue(policy.validate_issue(self.issue(title="Feature: 加导出")))

    def test_type_label_required_exactly_once(self):
        self.assertTrue(policy.validate_issue(self.issue(labels=[])))
        self.assertTrue(policy.validate_issue(self.issue(labels=["type/bug", "type/task"])))
        self.assertTrue(policy.validate_issue(self.issue(labels=["type/unknown"])))

    def test_body_limit(self):
        self.assertTrue(policy.validate_issue(self.issue(body=body("修复" * 30))))

    def test_details_required(self):
        self.assertTrue(policy.validate_issue(self.issue(body="一句话，没有折叠区")))

    def test_status_state_consistency(self):
        self.assertTrue(policy.validate_issue(self.issue(status="Done")))
        self.assertEqual(
            policy.validate_issue(
                self.issue(status="Done", state="closed", state_reason="completed")
            ),
            [],
        )
        self.assertTrue(policy.validate_issue(self.issue(status="In review", state="closed")))

    def test_priority(self):
        self.assertTrue(policy.validate_issue(self.issue(priority="P9")))
        self.assertEqual(policy.validate_issue(self.issue(priority="p2")), [])


class ValidatePR(unittest.TestCase):
    def pr(self, **over):
        base = {
            "title": "fix(post): stamp referrerpolicy on rendered images",
            "labels": ["area/backend"],
        }
        base.update(over)
        return base

    def test_valid_with_reference(self):
        self.assertEqual(policy.validate_pr(self.pr(), {"all": [3], "resolving": [3], "related": []}), [])

    def test_conventional_title(self):
        self.assertTrue(policy.validate_pr(self.pr(title="stamp referrerpolicy"), {"all": [1]}))
        self.assertEqual(policy.validate_pr(self.pr(title="fix: 无空格描述"), {"all": [1]}), [])
        self.assertEqual(policy.validate_pr(self.pr(title="fix(auth): 修复令牌刷新"), {"all": [1]}), [])

    def test_area_label_required(self):
        self.assertTrue(policy.validate_pr(self.pr(labels=[]), {"all": [1]}))

    def test_reference_required(self):
        self.assertTrue(policy.validate_pr(self.pr(), {"all": [], "resolving": [], "related": []}))


class RequiresPolicy(unittest.TestCase):
    def test_matrix(self):
        self.assertTrue(policy.requires_policy(False, "User", 1, 0))
        self.assertTrue(policy.requires_policy(False, "User", 0, 1))
        self.assertFalse(policy.requires_policy(True, "User", 1, 0))
        self.assertFalse(policy.requires_policy(False, "Bot", 1, 2))
        self.assertFalse(policy.requires_policy(False, "App", 0, 3))
        self.assertFalse(policy.requires_policy(False, "User", 0, 0))


class ResolvingCommand(unittest.TestCase):
    def test_pull_request_events(self):
        self.assertEqual(
            policy.resolving_command("pull_request", {"action": "review_requested"}),
            "review-requested",
        )
        self.assertEqual(
            policy.resolving_command("pull_request", {"action": "synchronize"}),
            "implementation",
        )
        self.assertIsNone(policy.resolving_command("pull_request", {"action": "converted_to_draft"}))

    def test_review_states(self):
        self.assertEqual(
            policy.resolving_command(
                "pull_request_review", {"action": "submitted", "review": {"state": "CHANGES_REQUESTED"}}
            ),
            "changes-requested",
        )
        self.assertIsNone(
            policy.resolving_command(
                "pull_request_review", {"action": "submitted", "review": {"state": "APPROVED"}}
            )
        )


class NextStatus(unittest.TestCase):
    def test_forward_only(self):
        self.assertEqual(policy.next_status(None, "implementation"), "In progress")
        self.assertEqual(policy.next_status("Inbox", "implementation"), "In progress")
        self.assertEqual(policy.next_status("In progress", "review-requested"), "In review")
        self.assertIsNone(policy.next_status("In review", "implementation"))
        self.assertIsNone(policy.next_status("Done", "implementation"))

    def test_backward_needs_automation_actor(self):
        target = policy.next_status("In review", "changes-requested", "github-actions[bot]")
        self.assertEqual(target, "In progress")
        self.assertIsNone(policy.next_status("In review", "changes-requested", "jukanntenn"))
        self.assertIsNone(policy.next_status("In review", "changes-requested", None))
        self.assertIsNone(policy.next_status("In progress", "changes-requested", "github-actions[bot]"))

    def test_unknown_command(self):
        with self.assertRaises(ValueError):
            policy.next_status("Inbox", "bogus")


if __name__ == "__main__":
    unittest.main()
