# MRFC: Content-sensitive version strings for dirty-tree image builds

Status: proposed

English | [中文](2026-09-03-dirty-tree-image-version-string.zh.md)

## Problem

`docker/build.py` bakes the image's version string with `git describe --tags --always --dirty` (passed to the Dockerfile's `VERSION` build-arg, which lands in the binary via `-X main.version`). The dev deploy verifies what actually runs by recomputing the exact same command on the deploying checkout (`devops/ansible/deploy.yml`) and demanding string equality against `/api/v1/version` (`scripts/check_deploy.py`).

On a dirty working tree, `git describe --dirty` output depends only on the base commit — `v0.2.0-dirty` or `abc1234-dirty` — so every dirty build of the same commit bakes an identical string. Version comparison then cannot distinguish different dirty builds of one commit, and two real hazards pass silently:

- **Deploying without rebuilding.** The checkout is edited after the image was built; the deploy recomputes the unchanged describe string and the version check passes against the stale image the registry still serves.
- **Indistinguishable iteration builds.** Two machines (or two sessions) building different dirty trees at the same commit produce images whose version strings are equal, so which build an instance actually runs is unknowable from the reported version.

The issue frames the acceptance as "reject dirty trees in push/release scenarios, or disambiguate with commit hash plus dirty marker, so version comparison can distinguish different dirty builds of the same commit". The forcing goal is the last clause — and a commit hash, being a function of the commit alone, cannot meet it for the same-commit case.

## Proposal

The version string becomes one shared computation with a single home, `scripts/build_version.py`, called by both consumers that must agree — `docker/build.py` at build time and `devops/ansible/deploy.yml` at deploy verification:

- **Clean tree** → `git describe --tags --always`, byte-identical to today's clean-tree output; release images (tag-driven CI builds from clean checkouts) are unaffected.
- **Dirty tree** → the describe output suffixed `-dirty.<8 hex>`, where the digest is a deterministic function of the base commit and the working-tree delta: the tracked diff plus the contents of untracked non-ignored files (what differs between two dirty states of one commit, and what enters the build context).

Determinism is the property that makes the deploy check stronger, not weaker: rebuilding an identical tree reproduces the same string, so a legitimate rebuild-and-deploy still passes; editing the tree after building changes the recomputed string, so the deploy fails the version check until the image is rebuilt — the first hazard above becomes a hard error instead of a silent pass.

`--push` does not reject dirty trees. The shipping flow commits before `docker/build.py --push`, so the standard path is clean and a rejection rule would only police manual iteration pushes; the ambiguity it would close re-enters on the other side of the deploy check anyway, whose expected string is recomputed from the controller tree — with a commit-exact image, any dirty controller fails the deploy, making the rule a mechanical "commit before deploy" nag rather than a content-true comparison. With content-sensitive strings a dirty push is distinguishable and verifiable; the digest carries the identity rejection was meant to guarantee.

## Alternatives considered

**Reject dirty trees in push (and release) scenarios.** The issue's first option. Pushed images would always map to a commit — but the standard ship path already commits before pushing, so the rule only polices manual iteration pushes, and it does not repair the deploy check it targets: the expected string is recomputed from the controller tree, so with a commit-exact image any dirty controller fails the deploy. Local loads — not covered by any rejection rule — would stay ambiguous. A rule that enforces a convention where the digest makes the comparison content-true loses.

**Always append the commit hash when dirty.** The issue's second option read literally: turn `v0.2.0-dirty` into `v0.2.0-gabc1234-dirty` (covering the at-tag case where describe omits the hash). It anchors which commit the dirt sits on but is still a function of the commit alone — two dirty builds at the same commit keep colliding, which is exactly the complaint the issue raises, so it cannot be the decision.

**Append a build timestamp to dirty builds.** Distinguishes builds trivially, but verification recomputes the expected string at deploy time; a timestamp is not reproducible, so even a legitimate rebuild of an identical tree would fail the deploy check. Non-determinism breaks the consumer that motivates the change.

## Acceptance criteria

- Clean-tree output is byte-identical to today's `git describe --tags --always`.
- Dirty-tree output is `<describe>-dirty.<8 hex>`; deterministic (same tree state → same string) and content-sensitive (different delta → different string).
- Exactly one implementation of the computation exists; `docker/build.py` and `devops/ansible/deploy.yml` both call it, and no second copy of the command survives in either file.
- Evidence: clean-tree and dirty-tree version-string outputs from real builds; a recompute after editing the tree mismatches the previously built string; a rebuild of the identical tree matches.

## Risks

- **Digest scope is git-visible.** Files ignored by git but present in the build context (and not excluded by `.dockerignore`) are invisible to the digest, so two trees differing only there collide. Context hygiene is `.dockerignore`'s existing job; the gap is accepted rather than re-deriving git's view of the context.
- **Digest cost.** Hashing untracked non-ignored files (e.g. `dist/`) adds work per build on the local machine; bounded by the size of the untracked set and negligible against a multi-arch image build.
- **Controller dependency.** `deploy.yml` gains a call into `scripts/build_version.py` on the deploying checkout; the controller already runs Python (`scripts/check_deploy.py`), so no new runtime is required.
