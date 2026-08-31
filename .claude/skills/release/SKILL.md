---
name: release
description: >
  Execute the markpost release process. Use this skill when the user wants to
  publish a new version, create a release, bump version numbers, or says things like
  "release v0.2.0", "publish a new version", "cut a release", "ship it", or "prepare
  release". Also use when the user mentions CHANGELOG updates combined with version
  bumping in this project.
---

# Release Process

Safe release workflow for markpost on the development loop's rails: the version bump lands through a `release/**` pull request — `main` accepts nothing else — and the tag pushed after that merge triggers publication. Every step validates before proceeding; failures report the problem + suggestion then STOP — never auto-fix.

## Prerequisites

Verify clean tree and identify current state:

```bash
git status                         # must be clean
git fetch origin && git status -sb # must not be behind origin/main
grep '"version"' frontend/package.json  # current version (field "version": "X.Y.Z")
git describe --tags --abbrev=0 2>/dev/null || echo "NO_TAGS"  # last tag (may not exist yet)
```

If `NO_TAGS` — this is the first release. Use the initial commit (`git rev-list --max-parents=0 HEAD`) as the baseline for CHANGELOG generation.

## Steps 1–4: Preparation Phase

### Step 1: Quality Checks

Run the prek gates from the repo root (the single source of quality gates):

```bash
prek run --all-files                    # fmt + lint + generated-files drift (CI's Lint gate)
prek run --stage pre-push --all-files   # backend go build + go test -race; frontend test:run + build (the push gate)
```

Fail → report which check failed (lint/test/build) + the specific violations, STOP.

### Step 2: Version Bump

1. Show commits since last tag (or since initial commit if no tags) — skip merge commits, they carry no change information:
   ```bash
   git log --oneline --no-merges <last_tag_or_initial>..HEAD
   ```
2. **Determine version number:**
   - If the user provided a version number → **validate it:**
     1. Semver format: `X.Y.Z` (stable) or `X.Y.Z-rc.N` (prerelease) where X/Y/Z are non-negative integers. The hyphen before the prerelease segment is REQUIRED — `X.Y.Zrc1` is invalid semver and breaks every semver-aware tool (docker/metadata-action, npm, Go modules).
     2. Higher than current version per semver precedence (no downgrades; a prerelease of the next version sorts below that version)
     3. No skipped intermediate versions (e.g. 0.1.1 → 0.3.0 skips 0.2.0 → soft warning, NOT a blocker)
     4. Tag does not already exist (`git tag -l "vX.Y.Z[-rc.N]"` must be empty)
     - Present validation results as **non-binding suggestions** — the user may override any warning.
   - If the user did NOT provide a version → recommend one based on semver principles (patch=fixes, minor=features, major=breaking), using your best judgment. **Show your reasoning** (e.g. "3 feat + 2 fix commits since v0.1.1 → recommending minor bump to 0.2.0").
3. **PAUSE — confirm version number.** Present the chosen version + reasoning/validation results and wait for explicit user confirmation. Do NOT proceed until the user gives a clear affirmative response (e.g. ok / 确认 / yes / 行 / 好的 / LGTM / 没问题 / 可以 / proceed / confirm).

   **⚠️ Anti-ambiguity rule:** You MUST NOT interpret silence, vague acknowledgments, or topic-adjacent replies as consent. Only explicit affirmative words count. When in doubt, ask.

4. Update `frontend/package.json` with the confirmed version `X.Y.Z` (the `"version"` field).

5. Verify:
   ```bash
   grep '"version"' frontend/package.json
   ```

### Step 3: Update CHANGELOG

1. Derive changes from `git log --oneline --no-merges <last_tag_or_initial>..HEAD`
2. If `CHANGELOG.md` does not exist yet, create it with header:
   ```
   # Changelog

   ```
3. Draft entry matching this format, present to user for review, then insert at TOP (after `# Changelog` header + blank line):

   ```
   ## [X.Y.Z] - YYYY-MM-DD

   ### Added
   - ...
   ### Changed
   - ...
   ### Fixed
   - ...
   ```

   Omit empty sections. Keep `# Changelog` header + blank line at the top.

   **Writing style — user-facing, not developer-facing:**
   - Each entry is **one sentence describing what the user experiences**, not what the code does.
   - ✅ Good: "Posts now render with syntax highlighting for code blocks."
   - ❌ Bad: "Added chroma renderer to markdown pipeline."
   - ❌ Bad: "Refactored rendering middleware."
   - Do NOT mention implementation details (function names, architecture, internal APIs).
   - **Completely omit** chore commits, CI changes, internal refactorings, and dependency updates unless they directly affect what users see or do.

4. Verify the entry was inserted correctly (first section after header).

### Step 4: README Consistency Check

Compare `README.md` and `README.zh.md` for conflicting or contradictory information:

- Feature descriptions must agree (content may differ in detail, but must not contradict)
- API examples must match
- Quick Start instructions must agree

If inconsistencies found → report them to the user and STOP.

---

## PAUSE — User Confirmation Required

Present summary:

```
Steps 1–4 complete:
- Version: X.Y.Z (frontend/package.json)
- CHANGELOG: updated
- READMEs: verified consistent

Ready to open the release pull request? Confirm to continue.
```

Do NOT proceed until the user gives a **clear affirmative response** (e.g. ok / 确认 / yes / 行 / 好的 / LGTM / 没问题 / 可以 / proceed / confirm). If the user objects or requests changes, address them and re-present the updated summary.

**⚠️ Anti-ambiguity rule:** You MUST NOT interpret silence, vague acknowledgments, or topic-adjacent replies as consent. Only explicit affirmative words count. When in doubt, ask.

---

## Steps 5–9: Landing & Publish Phase

### Step 5: Verify Release Workflows

Check BOTH workflows that trigger on `v*` tags:

```bash
cat .github/workflows/release.yml
cat .github/workflows/docker-publish.yml
```

**release.yml** MUST have:

- Trigger on `v*` tags
- CHANGELOG extraction step (awk to extract version-specific notes)
- `softprops/action-gh-release` with `body_path` pointing to extracted notes
- Prerelease/make_latest decided by the exact-match regex `^v[0-9]+\.[0-9]+\.[0-9]+$` (stable iff it matches)

**docker-publish.yml** MUST have:

- Trigger on `v*` tags
- Native multi-arch Docker build (amd64 + arm64, one runner per arch, no QEMU)
- Push to Docker Hub (`jukanntenn/markpost`) with the SAME stable regex;
  Docker tags strip the leading `v` (`v0.1.3` → `0.1.3`, `v0.1.3-rc.1` →
  `0.1.3-rc.1`); `latest` moves only on stable releases

If either file is missing or incomplete → report, explain what's expected, and ask user to confirm before continuing.

### Step 6: Release Branch + Commit

```bash
git switch main && git pull --ff-only   # the release cuts from current main
git switch -c release/vX.Y.Z            # prerelease: release/vX.Y.Z-rc.N (hyphen required)
git add frontend/package.json CHANGELOG.md
git commit -m "chore: release vX.Y.Z"
```

Verify: `git log --oneline -1` shows the commit, `git status` is clean. Hook failure → report output, STOP. Never `--no-verify`.

The branch name is load-bearing: `policy.py` exempts exactly `release/**` head branches from the issue-policy intake checks, so a release cut from any other branch name fails the required `Issue policy` check and can never merge.

### Step 7: Open the Pull Request

Follow the loop's machine-account rule for every GitHub operation: the release PR is machine-authored, so the maintainer's approving review is its gate — a maintainer-authored PR deadlocks on include-administrators (nobody approves their own pull request, and approval belongs to the human alone).

```bash
git push -u origin release/vX.Y.Z
gh pr create --base main --title "chore: release vX.Y.Z" --body "$(cat <<'EOF'
关联 Issue：（无 —— release PR，走 issue-policy 的 release/** 豁免）

**Ask-first 项**：无

<details>
<summary>变更与验证</summary>

- 变更：版本号 X.Y.Z（frontend/package.json）+ CHANGELOG 的 [X.Y.Z] 条目
- 验证：prek run --all-files 与 pre-push 阶段全绿；README 双语一致性已核对
- 合并后将为 main 上的合并提交打 tag vX.Y.Z 并推送，该 tag 触发 release.yml 与 docker-publish.yml 发布
</details>
EOF
)"
gh pr edit <number> --add-reviewer jukanntenn
```

No `area/*` label and no issue reference — bare by design; the `release/**` head branch carries the exemption. Record the PR number for the steps below.

The PR approval is the delivery gate AND the consent to publish: merging is followed mechanically by the tag push (Step 9), so the reviewer approves both at once — no separate publish confirmation exists.

### PAUSE — Human Approval Gate

Wait for the maintainer's approving review; gates are asynchronous and the session may end here. Any later session resumes at Step 8 when triage finds the `release/**` PR approved and green.

### Step 8: Preflight, Merge, Cleanup

Fetch live state; never trust an earlier report:

```bash
gh pr view <number> --json state,isDraft,reviewDecision,statusCheckRollup,mergeStateStatus
```

Require ALL: open, non-draft, `reviewDecision: APPROVED`, every check green (the five `* conclusion` checks + `Issue policy`), no unresolved change requests. Anything less → report, STOP.

```bash
gh pr merge <number> --merge   # a merge commit — the repository's only merge method
```

Verify `gh pr view <number> --json state,mergedAt` reports `MERGED` — a queued request is not a landed one. Then clean up:

```bash
gh pr list --state open --base release/vX.Y.Z --json number --jq length   # must print 0
git push origin --delete release/vX.Y.Z
git switch main && git branch -D release/vX.Y.Z
```

### Step 9: Tag & Publish

```bash
git switch main && git pull --ff-only   # HEAD is now the release merge commit
git tag -a vX.Y.Z -m "Release vX.Y.Z"   # prerelease: vX.Y.Z-rc.N (hyphen required)
git tag -l "vX.Y.Z"                     # must return exactly the tag; exists already → report, STOP
git push origin vX.Y.Z
```

Tag the merge commit on `main`, never a pre-merge branch head — the tag publishes what the delivery gate accepted. The tag push — not any branch push — is what triggers both workflows; the PR approval already covers it, so no further pause. Rejected push → report error, STOP. Never force push.

Then provide monitoring URLs:

- GitHub Release: `https://github.com/jukanntenn/markpost/actions/workflows/release.yml`
- Docker publish: `https://github.com/jukanntenn/markpost/actions/workflows/docker-publish.yml`

---

## Step 10: Post-Release Verification

Provide user with:

1. **GitHub Release**: check body matches CHANGELOG entry
   → `https://github.com/jukanntenn/markpost/releases/tag/vX.Y.Z`
2. **GitHub Actions**: verify both workflows succeeded
   → Release: `https://github.com/jukanntenn/markpost/actions/workflows/release.yml`
   → Docker: `https://github.com/jukanntenn/markpost/actions/workflows/docker-publish.yml`
3. **Docker Hub**: verify the new tag is published (`X.Y.Z` or `X.Y.Z-rc.N`,
   no `v` prefix); `latest` must also have moved, but ONLY for stable releases
   → `https://hub.docker.com/r/jukanntenn/markpost/tags`
4. **Version checklist**: confirm `frontend/package.json` shows X.Y.Z
5. **Rollback options** (one per stage):
   - PR open, not merged: close the PR, `git push origin --delete release/vX.Y.Z`, delete the local branch — nothing landed.
   - Merged, not yet tagged: revert the merge commit through a pull request — `main` is protected, no direct pushes.
   - Tagged: delete the tag `git tag -d vX.Y.Z && git push origin :refs/tags/vX.Y.Z`, and edit or delete the GitHub Release manually via the web UI.
