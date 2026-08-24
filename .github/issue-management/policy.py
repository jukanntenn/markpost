#!/usr/bin/env python3
"""Issue and pull-request policy for the agent-driven development loop.

One logic serves two callers: `policy.py pr` runs as a pull-request check
(validates a reviewed PR and its referenced issues against the intake
contract), `policy.py lifecycle` runs on repository events and moves the
GitHub Project board (Inbox/Backlog/Ready/In progress/In review/Done/No
action). The agent also runs the same checks locally before filing. All
GitHub access goes through the `gh` CLI with GH_TOKEN; pure logic lives in
functions covered by policy_test.py. Board writes record their actor in a
marker comment on the issue, so a backward transition (In review -> In
progress on a changes-requested review) only fires when automation set the
current status — a human's manual placement is never overridden.
"""
from __future__ import annotations

import json
import os
import re
import subprocess
import sys
from pathlib import Path

CONFIG_PATH = Path(__file__).with_name("config.json")
CONFIG = json.loads(CONFIG_PATH.read_text(encoding="utf-8"))
FULL_NAME = f"{CONFIG['owner']}/{CONFIG['repository']}"
TERMINAL = {"Done", "No action"}
ACTIVE_ORDER = [s for s in CONFIG["statuses"] if s not in TERMINAL]
BODY_LIMIT = CONFIG["bodyUnitLimit"]
TYPE_LABELS = {f"type/{t}" for t in CONFIG["types"]}
LIFECYCLE_ACTOR = CONFIG["lifecycleActor"]
MARKER = "<!-- markpost-issue-policy -->"
LIFECYCLE_MARKER = "<!-- markpost-lifecycle:"

CONVENTIONAL = re.compile(
    r"^(feat|fix|chore|docs|refactor|test|build|style|perf)"
    r"(\([a-z0-9][a-z0-9-]*\))?: \S"
)
TITLE_PREFIX = re.compile(
    r"^\s*(?:\[(?:Idea|Feature|Bug|Research|Task|P[0-3]|Inbox|Backlog|Ready|"
    r"In progress|In review|Done|No action|Owner|area/[^\]]+)[^\]]*\]|"
    r"(?:Idea|Feature|Bug|Research|Task|P[0-3]|Inbox|Backlog|Ready|"
    r"In progress|In review|Done|No action|Owner|area/[^:： ]+)\s*[:：-])",
    re.IGNORECASE,
)
HAN = re.compile(r"[\u3400-\u4dbf\u4e00-\u9fff]")
LATIN_TOKEN = re.compile(r"[A-Za-z0-9_./:@+-]+")
REFERENCE = re.compile(
    r"(?:([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)#|#)(\d+)"
    r"|https://github\.com/([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)/issues/(\d+)",
    re.IGNORECASE,
)
CLOSING = re.compile(
    r"\b(?:close(?:s|d)?|fix(?:es|ed)?|resolve(?:s|d)?)\s*:?\s+"
    r"(?:(?:([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)#|#)(\d+)"
    r"|https://github\.com/([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)/issues/(\d+))",
    re.IGNORECASE,
)


def strip_html_comments(body: str) -> str:
    return re.sub(r"<!--[\s\S]*?-->", "", body or "")


def extract_outside_details(body: str) -> dict:
    """Return Markdown outside balanced details elements plus their shape."""
    source = strip_html_comments(body)
    tag = re.compile(r"</?details\b[^>]*>", re.IGNORECASE)
    depth = 0
    cursor = 0
    balanced = True
    text = ""
    details_count = 0
    all_collapsed = True
    for match in tag.finditer(source):
        if depth == 0:
            text += source[cursor : match.start()]
        if match[0].startswith("</"):
            if depth == 0:
                balanced = False
            else:
                depth -= 1
        else:
            depth += 1
            details_count += 1
            if re.search(r"\sopen(?:\s|=|>)", match[0], re.IGNORECASE):
                all_collapsed = False
        cursor = match.end()
    if depth == 0:
        text += source[cursor:]
    if depth != 0:
        balanced = False
    return {
        "text": text,
        "balanced": balanced,
        "details_count": details_count,
        "all_collapsed": all_collapsed,
    }


def count_units(body: str) -> int:
    """Count Chinese characters plus contiguous Latin/numeric tokens."""
    outside = extract_outside_details(body)
    visible = re.sub(r"!\[([^\]]*)\]\([^)]*\)", r"\1", outside["text"])
    visible = re.sub(r"\[([^\]]+)\]\([^)]*\)", r"\1", visible)
    visible = re.sub(r"\[([^\]]+)\]\[[^\]]*\]", r"\1", visible)
    visible = re.sub(r"<((?:https?://|mailto:)[^>]+)>", r"\1", visible, flags=re.IGNORECASE)
    visible = re.sub(r"<[^>]+>", " ", visible)
    visible = re.sub(r"&(?:[A-Za-z]+|#\d+|#x[0-9A-Fa-f]+);", " ", visible)
    visible = re.sub(r"[`*~[\]{}()<>#!|]", " ", visible)
    return len(HAN.findall(visible)) + len(LATIN_TOKEN.findall(visible))


def first_nonblank_line(body: str) -> str | None:
    for line in (body or "").splitlines():
        if line.strip():
            return line
    return None


def strip_fences_and_comments(body: str) -> str:
    """Drop HTML comments and fenced-code lines so references parse prose only."""
    kept: list[str] = []
    fence: str | None = None
    for line in strip_html_comments(body).splitlines():
        marker = re.match(r"^\s{0,3}([`~]{3,})", line)
        if marker:
            if fence is None:
                fence = marker[1][0]
            elif marker[1][0] == fence:
                fence = None
            continue
        if fence is None:
            kept.append(line)
    return "\n".join(kept)


def parse_references(body: str, repo_full_name: str = FULL_NAME) -> dict:
    """Parse same-repository resolving and informational issue references."""
    source = strip_fences_and_comments(body)
    expected = repo_full_name.lower()
    all_refs: set[int] = set()
    resolving: set[int] = set()

    def collect(match: re.Match, resolving_ref: bool) -> None:
        groups = match.groups()
        explicit = next((g for g in groups[::2] if g), "").lower()
        number = int(next(g for g in groups[1::2] if g))
        if not explicit or explicit == expected:
            all_refs.add(number)
            if resolving_ref:
                resolving.add(number)

    for match in REFERENCE.finditer(source):
        collect(match, False)
    for match in CLOSING.finditer(source):
        collect(match, True)
    return {
        "all": sorted(all_refs),
        "resolving": sorted(resolving),
        "related": sorted(all_refs - resolving),
    }


def validate_issue(issue: dict) -> list[str]:
    """Validate one issue snapshot (labels, title, body; status/priority optional)."""
    errors: list[str] = []
    body = issue.get("body") or ""
    outside = extract_outside_details(body)
    units = count_units(body)
    labels = issue.get("labels") or []
    types = [label for label in labels if label.startswith("type/")]

    if not HAN.search(issue.get("title") or ""):
        errors.append("Issue 标题必须包含中文")
    if TITLE_PREFIX.match(issue.get("title") or ""):
        errors.append("Issue 标题不得带 Type、Priority、Status、area 或 Owner 前缀")
    if len(types) != 1 or types[0] not in TYPE_LABELS:
        errors.append(f"Issue 必须恰好一个 type/* 标签，取值 {sorted(TYPE_LABELS)}")
    if not outside["balanced"]:
        errors.append("details 标签必须成对闭合")
    if outside["details_count"] == 0:
        errors.append("正文必须包含默认收起的 <details> 区域")
    if not outside["all_collapsed"]:
        errors.append("details 必须默认收起，不得设置 open")
    if units > BODY_LIMIT:
        errors.append(f"正文外露部分为 {units} 单位，超过 {BODY_LIMIT} 单位")

    status = issue.get("status")
    state = issue.get("state")
    if status is not None:
        if status not in CONFIG["statuses"]:
            errors.append(f"Issue 的看板状态非法：{status}")
        elif status == "Done" and (state != "closed" or issue.get("state_reason") != "completed"):
            errors.append("Done 必须对应 Completed 关闭原因")
        elif status == "No action" and (state != "closed" or issue.get("state_reason") != "not_planned"):
            errors.append("No action 必须对应 Not planned 关闭原因")
        elif status not in TERMINAL and state != "open":
            errors.append(f"{status} 必须对应开放 Issue")
    priority = issue.get("priority")
    if priority is not None and priority.lower() not in {"p0", "p1", "p2", "p3"}:
        errors.append("Priority 必须为空或为 P0–P3")
    return errors


def validate_pr(pull: dict, references: dict) -> list[str]:
    """Validate one PR snapshot: conventional title, area label, issue reference."""
    errors: list[str] = []
    title = pull.get("title") or ""
    if not CONVENTIONAL.match(title):
        errors.append(f"PR 标题不符合 Conventional Commits：{title!r}")
    areas = [label for label in pull.get("labels") or [] if label.startswith("area/")]
    if not areas:
        errors.append("PR 至少需要一个 area/* 标签")
    if not references["all"]:
        errors.append("PR 必须以 #N 或 Fixes/Related 引用至少一个本仓库 issue")
    return errors


def is_release_pull(head_ref: str | None) -> bool:
    """Release PRs (version bump + changelog on a release/** head) are exempt
    from the three intake checks; references they do carry still validate
    (.agents/mrfcs/implemented/2026-08-24-issue-policy-release-exemption.md)."""
    return bool(head_ref) and head_ref.startswith("release/")


def requires_policy(is_draft: bool, author_type: str, review_requests: int, reviews: int) -> bool:
    """Policy applies once a human-authored PR is non-draft and under review."""
    automated = author_type in {"Bot", "App"}
    return not is_draft and not automated and (review_requests > 0 or reviews > 0)


def resolving_command(event_name: str, event: dict) -> str | None:
    """Translate a repository event into a resolving-issue lifecycle command."""
    if event_name == "pull_request":
        if event.get("action") == "review_requested":
            return "review-requested"
        implemented = {"opened", "edited", "synchronize", "reopened", "labeled", "unlabeled"}
        return "implementation" if event.get("action") in implemented else None
    if event_name == "pull_request_review" and event.get("action") == "submitted":
        state = ((event.get("review") or {}).get("state") or "").lower()
        return "changes-requested" if state == "changes_requested" else None
    return None


def next_status(current: str | None, command: str, status_actor: str | None = None) -> str | None:
    """Plan one event-directed transition; forward-only, automation-set backward."""
    target = "In review" if command == "review-requested" else "In progress"
    if command not in {"review-requested", "implementation", "changes-requested"}:
        raise ValueError(f"未知 lifecycle command：{command}")
    if (
        command == "changes-requested"
        and current == "In review"
        and status_actor == LIFECYCLE_ACTOR
    ):
        return target
    try:
        current_index = ACTIVE_ORDER.index(current) if current else -1
    except ValueError:
        return None
    target_index = ACTIVE_ORDER.index(target)
    return target if current_index < target_index else None


def _gh(*args: str) -> str:
    result = subprocess.run(
        ["gh", *args], capture_output=True, text=True, check=False
    )
    if result.returncode != 0:
        raise RuntimeError(f"gh {' '.join(args)}: {result.stderr.strip()}")
    return result.stdout


def gh_api(path: str) -> dict:
    return json.loads(_gh("api", path))


def gh_graphql(query: str, **fields: str) -> dict:
    args = ["api", "graphql", "-f", f"query={query}"]
    for key, value in fields.items():
        args += ["-F", f"{key}={value}"]
    return json.loads(_gh(*args))["data"]


def project_item(number: int) -> dict | None:
    """Fetch the issue's project Status/Priority, or None without a project."""
    if not CONFIG["requireProject"] or not CONFIG["projectNumber"]:
        return None
    data = gh_graphql(
        """
        query($owner: String!, $repo: String!, $number: Int!, $issue: Int!) {
          user(login: $owner) {
            projectV2(number: $number) { id }
          }
          repository(owner: $owner, name: $repo) {
            issue(number: $issue) {
              projectItems(first: 10) {
                nodes {
                  id
                  project { number }
                  fieldValueByName(name: "%s") { ... on ProjectV2ItemFieldSingleSelectValue { name } }
                }
              }
            }
          }
        }
        """
        % CONFIG["statusField"],
        owner=CONFIG["owner"],
        repo=CONFIG["repository"],
        number=str(CONFIG["projectNumber"]),
        issue=str(number),
    )
    items = data["repository"]["issue"]["projectItems"]["nodes"]
    for item in items:
        if item["project"]["number"] == CONFIG["projectNumber"]:
            status = (item["fieldValueByName"] or {}).get("name")
            return {"item_id": item["id"], "status": status}
    return None


def marker_status_actor(body: str) -> tuple[str | None, str | None]:
    """Read the latest lifecycle marker (status + actor) from an issue body comment."""
    status = actor = None
    for match in re.finditer(
        rf"{re.escape(LIFECYCLE_MARKER)} ([^:]+) by ([^\]]+)\]", body or ""
    ):
        status, actor = match[1].strip(), match[2].strip()
    return status, actor


def upsert_comment(number: int, body: str, marker: str) -> None:
    slug = f"repos/{FULL_NAME}/issues/{number}/comments"
    for comment in gh_api(f"{slug}?per_page=100"):
        if marker in (comment.get("body") or ""):
            # Creating posts under issues/{n}/comments; updating an existing
            # issue comment is the flat issues/comments/{id} endpoint.
            _gh(
                "api",
                "-X",
                "PATCH",
                f"repos/{FULL_NAME}/issues/comments/{comment['id']}",
                "-f",
                f"body={body}",
            )
            return
    _gh("api", "-X", "POST", slug, "-f", f"body={body}")


def set_status(number: int, status: str) -> None:
    """Move the board and record the write in a marker comment."""
    if not CONFIG["requireProject"] or not CONFIG["projectNumber"]:
        return
    item = project_item(number)
    if item is None:
        data = gh_graphql(
            'mutation($projectId: ID!, $contentId: ID!) { addProjectV2ItemById(input: '
            '{ projectId: $projectId, contentId: $contentId }) { item { id } } }',
            projectId=_project_id(),
            contentId=gh_api(f"repos/{FULL_NAME}/issues/{number}")["node_id"],
        )
        item = {"item_id": data["addProjectV2ItemById"]["item"]["id"], "status": None}
    if item["status"] == status:
        return
    field = _status_field()
    option = next((o for o in field["options"] if o["name"] == status), None)
    if option is None:
        raise RuntimeError(f"Status 不存在：{status}")
    gh_graphql(
        "mutation($project: ID!, $item: ID!, $field: ID!, $option: String!) { "
        "updateProjectV2ItemFieldValue(input: { projectId: $project, itemId: $item, "
        "fieldId: $field, value: { singleSelectOptionId: $option } }) "
        "{ projectV2Item { id } } }",
        project=_project_id(),
        item=item["item_id"],
        field=field["id"],
        option=option["id"],
    )
    upsert_comment(
        number,
        f"{LIFECYCLE_MARKER} {status} by {LIFECYCLE_ACTOR}]",
        LIFECYCLE_MARKER,
    )


def _project_id() -> str:
    cache = getattr(set_status, "_project_id", None)
    if cache is None:
        data = gh_graphql(
            "query($owner: String!, $number: Int!) { user(login: $owner) "
            "{ projectV2(number: $number) { id } } }",
            owner=CONFIG["owner"],
            number=str(CONFIG["projectNumber"]),
        )
        cache = data["user"]["projectV2"]["id"]
        set_status._project_id = cache
    return cache


def _status_field() -> dict:
    data = gh_graphql(
        "query($owner: String!, $number: Int!) { user(login: $owner) "
        "{ projectV2(number: $number) { fields(first: 20) { nodes { ... on "
        "ProjectV2SingleSelectField { id name options { id name } } } } } } }",
        owner=CONFIG["owner"],
        number=str(CONFIG["projectNumber"]),
    )
    for field in data["user"]["projectV2"]["fields"]["nodes"]:
        if field and field["name"] == CONFIG["statusField"]:
            return field
    raise RuntimeError("Project 缺少 Status 字段")


def issue_snapshot(number: int) -> dict | None:
    issue = gh_api(f"repos/{FULL_NAME}/issues/{number}")
    if "pull_request" in issue:
        return None
    snapshot = {
        "title": issue["title"],
        "body": issue.get("body") or "",
        "labels": [label["name"] for label in issue["labels"]],
        "state": issue["state"],
        "state_reason": issue.get("state_reason"),
    }
    item = project_item(number)
    if item is not None:
        snapshot["status"] = item["status"]
    comments = gh_api(f"repos/{FULL_NAME}/issues/{number}/comments?per_page=100")
    body = "\n".join(c.get("body") or "" for c in comments)
    status, actor = marker_status_actor(body)
    snapshot["_marker_status"] = status
    snapshot["_marker_actor"] = actor
    return snapshot


def run_pr_check(event: dict) -> int:
    number = event["pull_request"]["number"]
    pull = gh_api(f"repos/{FULL_NAME}/pulls/{number}")
    reviewers = gh_api(f"repos/{FULL_NAME}/pulls/{number}/requested_reviewers")
    reviews = gh_api(f"repos/{FULL_NAME}/pulls/{number}/reviews?per_page=100")
    mandatory = requires_policy(
        pull.get("draft", False),
        (pull.get("user") or {}).get("type", "User"),
        len(reviewers.get("users", [])) + len(reviewers.get("teams", [])),
        len(reviews),
    )
    if not mandatory:
        print("PR 尚未进入 Issue policy 强制范围。")
        return 0
    references = parse_references(pull.get("body") or "")
    snapshot = {
        "title": pull["title"],
        "labels": [label["name"] for label in pull["labels"]],
    }
    errors = [] if is_release_pull((pull.get("head") or {}).get("ref")) else validate_pr(snapshot, references)
    for issue_number in references["all"]:
        issue = issue_snapshot(issue_number)
        if issue is not None:
            errors += [f"#{issue_number}: {e}" for e in validate_issue(issue)]
    if errors:
        upsert_comment(
            number,
            f"{MARKER}\n⚠️ Issue policy 未通过：\n\n"
            + "\n".join(f"- {e}" for e in errors),
            MARKER,
        )
        for error in errors:
            print(f"::error::{error}")
        print(f"Issue policy 未通过，共 {len(errors)} 项", file=sys.stderr)
        return 1
    print("Issue policy 通过。")
    return 0


def run_lifecycle(event_name: str, event: dict) -> int:
    if event_name == "issues":
        number = event["issue"]["number"]
        action = event.get("action")
        if action == "opened" or action == "reopened":
            set_status(number, "Inbox")
        elif action == "closed":
            reason = event["issue"].get("state_reason")
            set_status(number, "No action" if reason == "not_planned" else "Done")
        return 0
    command = resolving_command(event_name, event)
    if not command:
        return 0
    number = event["pull_request"]["number"]
    pull = gh_api(f"repos/{FULL_NAME}/pulls/{number}")
    references = parse_references(pull.get("body") or "")
    for issue_number in references["resolving"]:
        issue = issue_snapshot(issue_number)
        if issue is None:
            continue
        target = next_status(
            issue.get("status", issue.get("_marker_status")),
            command,
            issue.get("_marker_actor"),
        )
        if target:
            set_status(issue_number, target)
    return 0


def read_event() -> dict:
    path = os.environ.get("GITHUB_EVENT_PATH")
    if not path:
        raise RuntimeError("GITHUB_EVENT_PATH 未设置")
    return json.loads(Path(path).read_text(encoding="utf-8"))


def main(argv: list[str]) -> int:
    if not os.environ.get("GH_TOKEN"):
        raise RuntimeError("GH_TOKEN 未设置")
    if argv[:1] == ["pr"]:
        return run_pr_check(read_event())
    if argv[:1] == ["lifecycle"]:
        return run_lifecycle(os.environ.get("GITHUB_EVENT_NAME", ""), read_event())
    raise SystemExit("用法：policy.py pr|lifecycle")


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
