# MRFC: Content-sensitive version strings for dirty-tree image builds

Status: proposed

[English](2026-09-03-dirty-tree-image-version-string.md) | 中文

## Problem

`docker/build.py` 以 `git describe --tags --always --dirty` 的输出烘焙镜像版本串（经 Dockerfile 的 `VERSION` build-arg 传入二进制的 `-X main.version`）。dev 部署的验证方式是：在部署用的 checkout 上重算同一条命令（`devops/ansible/deploy.yml`），与 `/api/v1/version` 的报告做字符串精确比对（`scripts/check_deploy.py`）。

脏工作树下，`git describe --dirty` 的输出只取决于基底提交——`v0.2.0-dirty` 或 `abc1234-dirty`——因此同一提交的每一次脏构建都烘焙出相同的版本串。版本比对由此无法区分同一提交的不同脏构建，两个真实隐患会静默通过：

- **改了树却不重建就部署。** 镜像构建完成后又编辑了 checkout；部署时重算的 describe 串没有变化，版本检查对着 registry 里仍在服务的过期镜像照常通过。
- **迭代构建不可区分。** 两台机器（或两个会话）在同一提交上构建不同的脏树，产出的镜像版本串完全相等，实例实际运行的是哪一次构建，从上报的版本无从得知。

Issue 将验收表述为「发布与推送场景拒绝脏树，或以提交哈希加脏标记消歧，使版本比对能区分同提交的不同脏构建」。其中真正的驱动目标是最后一个分句——而提交哈希只是提交的函数，对同提交的情形无力满足它。

## Proposal

版本串收敛为一个共享计算，唯一安家于 `scripts/build_version.py`，由两个必须达成一致的消费方调用——`docker/build.py` 在构建时、`devops/ansible/deploy.yml` 在部署验证时：

- **干净树** → `git describe --tags --always`，与今天干净树的输出逐字节一致；发布镜像（tag 触发的 CI 构建，干净 checkout）不受影响。
- **脏树** → describe 输出追加后缀 `-dirty.<8 hex>`，摘要是基底提交与工作区增量的确定性函数：tracked diff 加上未跟踪且未被 ignore 的文件内容（即同一提交两个脏状态之间的差异，也是进入构建上下文的内容）。

确定性让部署检查变得更强而非更弱：重建完全相同的树会复现相同的版本串，合法的重建后部署照常通过；构建后编辑树则改变重算结果，部署的版本检查在镜像重建前直接失败——上面的第一个隐患从静默通过变成硬错误。

`--push` 不拒绝脏树。shipping 流程在 `docker/build.py --push` 之前先提交，标准路径本就是干净树，拒绝规则只会去管束手动的迭代推送；而它想消除的歧义在部署检查的另一侧照样回来——期望串是从控制器树重算的，镜像一旦提交精确，控制器稍脏就部署失败，规则沦为机械的「先提交再部署」闹钟，而非内容真实的比对。内容敏感的版本串使脏推送可区分、可验证；拒绝规则想担保的身份性由摘要直接承载。

## Alternatives considered

**推送（及发布）场景拒绝脏树。** Issue 的第一个选项。推送的镜像总能映射到提交——但标准 ship 流程在推送前就已提交，规则只管束得到手动迭代推送，且修不好它针对的部署检查：期望串从控制器树重算，镜像一旦提交精确，控制器稍脏就部署失败。本地 load——不落入任何拒绝规则——依旧不可区分。在摘要已让比对内容真实的地方，一条只强制约定的规则落选。

**脏树时总是追加提交哈希。** Issue 第二个选项的字面读法：把 `v0.2.0-dirty` 变成 `v0.2.0-gabc1234-dirty`（覆盖 describe 省略哈希的恰在 tag 上的情形）。它锚定了脏在哪个提交上，但仍然只是提交的函数——同一提交的两个脏构建照样相撞，而这正是 issue 提出的问题本身，故不能作为决策。

**脏构建追加构建时间戳。** 区分构建毫不费力，但验证是在部署时重算期望串的；时间戳不可复现，连完全相同树的合法重建都会挂掉部署检查。非确定性恰好破坏了驱动本次变更的那个消费方。

## Acceptance criteria

- 干净树输出与今天的 `git describe --tags --always` 逐字节一致。
- 脏树输出为 `<describe>-dirty.<8 hex>`；确定性（相同树状态 → 相同字符串）且内容敏感（不同增量 → 不同字符串）。
- 计算只存在一份实现；`docker/build.py` 与 `devops/ansible/deploy.yml` 都调用它，两个文件中不再存有该命令的第二份拷贝。
- 证据：真实构建的干净树与脏树版本串输出；编辑树后的重算与先前构建串不匹配；相同树的重建与之匹配。

## Risks

- **摘要范围仅限 git 可见。** 被 git ignore 但仍在构建上下文中（且未被 `.dockerignore` 排除）的文件对摘要不可见，仅在这些文件上不同的两棵树会相撞。上下文卫生本就是 `.dockerignore` 的职责；接受这一缺口，而不去重新推导 git 对上下文的视图。
- **摘要成本。** 哈希未跟踪且未 ignore 的文件（如 `dist/`）给每次构建增加开销；受未跟踪集合大小约束，相对一次多架构镜像构建可忽略。
- **控制器依赖。** `deploy.yml` 需在部署 checkout 上调用 `scripts/build_version.py`；控制器本就运行 Python（`scripts/check_deploy.py`），不引入新的运行时。
