# MRFC: Content-sensitive version strings for dirty-tree image builds

Status: implemented

English | [中文](2026-09-03-dirty-tree-image-version-string.zh.md)

## Problem

`docker/build.py` baked the image's version string with `git describe --tags --always --dirty` (passed to the Dockerfile's `VERSION` build-arg, which lands in the binary via `-X main.version`). The dev deploy verifies what actually runs by recomputing the exact same command on the deploying checkout (`devops/ansible/deploy.yml`) and demanding string equality against `/api/v1/version` (`scripts/check_deploy.py`).

On a dirty working tree, `git describe --dirty` output depends only on the base commit — `v0.2.0-dirty` or `abc1234-dirty` — so every dirty build of the same commit bakes an identical string. Version comparison then cannot distinguish different dirty builds of one commit, and two real hazards pass silently:

- **Deploying without rebuilding.** The checkout is edited after the image was built; the deploy recomputes the unchanged describe string and the version check passes against the stale image the registry still serves.
- **Indistinguishable iteration builds.** Two machines (or two sessions) building different dirty trees at the same commit produce images whose version strings are equal, so which build an instance actually runs is unknowable from the reported version.

The issue framed the acceptance as "reject dirty trees in push/release scenarios, or disambiguate with commit hash plus dirty marker, so version comparison can distinguish different dirty builds of the same commit". The forcing goal is the last clause — and a commit hash, being a function of the commit alone, cannot meet it for the same-commit case.

## Decision

The image version string is computed by one shared implementation, `scripts/build_version.py`, and by nothing else: `docker/build.py` calls it at build time to resolve the `VERSION` build-arg, and the deploy playbook's dev check (`devops/ansible/deploy.yml`) calls it on the controller checkout to produce the expected string `scripts/check_deploy.py` compares against.

- **Clean tree** → `git describe --tags --always`, byte-identical to the old clean-tree behavior; release images (CI builds from clean tag checkouts) are untouched.
- **Dirty tree** → `<describe>-dirty.<8 hex>`, a deterministic digest of the base commit plus the working-tree delta: the tracked diff (`git diff HEAD --binary` with rename detection, prefixes, and the algorithm pinned against config drift) and the contents of untracked non-ignored files (status with `--untracked-files=all`, so files inside fresh directories are enumerated). Content-sensitive and reproducible: two dirty builds of the same commit with different content compare unequal, and rebuilding an identical tree reproduces the string.

Because the dev check recomputes the string from the same checkout that built the image, it is now content-true: deploying a rebuilt identical tree passes, while editing the checkout after the build changes the recomputed string and fails the deploy until the image is rebuilt — the deploy-time hazard that motivated this decision is a hard error, not a silent pass. A repository without commits yields `dev` (the historic fallback); any other computation failure aborts the build with exit 2 rather than baking an ambiguous string.

`--push` does not reject dirty trees. The shipping flow commits before `docker/build.py --push`, so the standard path is clean and a rejection rule would only police manual iteration pushes; the ambiguity it would close re-enters on the other side of the deploy check anyway, whose expected string is recomputed from the controller tree. With content-sensitive strings a dirty push is distinguishable and verifiable; the digest carries the identity rejection was meant to guarantee.

## Alternatives considered

**Reject dirty trees in push (and release) scenarios.** The issue's first option. Pushed images would always map to a commit — but the standard ship path already commits before pushing, so the rule only polices manual iteration pushes, and it does not repair the deploy check it targets: the expected string is recomputed from the controller tree, so with a commit-exact image any dirty controller fails the deploy. Local loads — not covered by any rejection rule — would stay ambiguous. A rule that enforces a convention where the digest makes the comparison content-true loses.

**Always append the commit hash when dirty.** The issue's second option read literally: turn `v0.2.0-dirty` into `v0.2.0-gabc1234-dirty` (covering the at-tag case where describe omits the hash). It anchors which commit the dirt sits on but is still a function of the commit alone — two dirty builds at the same commit keep colliding, which is exactly the complaint the issue raises, so it cannot be the decision.

**Append a build timestamp to dirty builds.** Distinguishes builds trivially, but verification recomputes the expected string at deploy time; a timestamp is not reproducible, so even a legitimate rebuild of an identical tree would fail the deploy check. Non-determinism breaks the consumer that motivates the change.

## Consequences

The version comparison the issue targeted now distinguishes same-commit dirty builds. Unit tests assert the properties that make it so: determinism (same tree state → same string), content sensitivity (different delta → different string), and reversion (restoring the tree returns the bare describe string); the PR carries the clean-tree and dirty-tree outputs from real builds.

The costs: the digest sees only git-visible delta, so files ignored by git but present in the build context (and not excluded by `.dockerignore`) remain a collision window — context hygiene is `.dockerignore`'s existing job, accepted rather than re-derived; submodule-internal dirt is not digested (the submodule pointer change is); per-build hashing of the untracked set is bounded and negligible against a multi-arch image build; and the deploy controller now invokes `scripts/build_version.py`, Python it already requires for `scripts/check_deploy.py`.
