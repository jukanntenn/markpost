# MRFC: Content-sensitive version strings for dirty-tree image builds

Status: implemented

[English](2026-09-03-dirty-tree-image-version-string.md) | 中文

## Problem

`docker/build.py` 曾以 `git describe --tags --always --dirty` 的输出烘焙镜像版本串（经 Dockerfile 的 `VERSION` build-arg 传入二进制的 `-X main.version`）。dev 部署的验证方式是：在部署用的 checkout 上重算同一条命令（`devops/ansible/deploy.yml`），与 `/api/v1/version` 的报告做字符串精确比对（`scripts/check_deploy.py`）。

脏工作树下，`git describe --dirty` 的输出只取决于基底提交——`v0.2.0-dirty` 或 `abc1234-dirty`——因此同一提交的每一次脏构建都烘焙出相同的版本串。版本比对由此无法区分同一提交的不同脏构建，两个真实隐患会静默通过：

- **改了树却不重建就部署。** 镜像构建完成后又编辑了 checkout；部署时重算的 describe 串没有变化，版本检查对着 registry 里仍在服务的过期镜像照常通过。
- **迭代构建不可区分。** 两台机器（或两个会话）在同一提交上构建不同的脏树，产出的镜像版本串完全相等，实例实际运行的是哪一次构建，从上报的版本无从得知。

Issue 将验收表述为「发布与推送场景拒绝脏树，或以提交哈希加脏标记消歧，使版本比对能区分同提交的不同脏构建」。其中真正的驱动目标是最后一个分句——而提交哈希只是提交的函数，对同提交的情形无力满足它。

## Decision

镜像版本串由唯一一份共享实现计算：`scripts/build_version.py`，此外无他。`docker/build.py` 在构建时调用它解析 `VERSION` build-arg；部署 playbook 的 dev 检查（`devops/ansible/deploy.yml`）在控制器 checkout 上调用它，产出 `scripts/check_deploy.py` 比对所用的期望串。

- **干净树** → `git describe --tags --always`，与旧干净树行为逐字节一致；发布镜像（CI 从干净 tag checkout 构建）不受影响。
- **脏树** → `<describe>-dirty.<8 hex>`，基底提交加工作区增量的确定性摘要：tracked diff（`git diff HEAD --binary`，重命名检测、前缀与算法均锁定以防配置漂移）加未跟踪且未 ignore 文件的内容（status 用 `--untracked-files=all`，新目录内的文件逐个枚举）。内容敏感且可复现：同提交、不同内容的两场脏构建互不相等；重建完全相同的树会复现同一字符串。

dev 检查从构建镜像的同一 checkout 重算版本串，因此检查是内容真实的：部署重建过的相同树照常通过；构建后编辑 checkout 会改变重算结果，镜像重建前部署直接失败——驱动本决策的部署期隐患从静默通过变成硬错误。无提交的仓库产出 `dev`（历史回退值）；其余任何计算失败都以退出码 2 中止构建，而不是烘焙一个含混的版本串。

`--push` 不拒绝脏树。shipping 流程在 `docker/build.py --push` 之前先提交，标准路径本就是干净树，拒绝规则只会去管束手动的迭代推送；而它想消除的歧义在部署检查的另一侧照样回来——期望串是从控制器树重算的。内容敏感的版本串使脏推送可区分、可验证；拒绝规则想担保的身份性由摘要直接承载。

## Alternatives considered

**推送（及发布）场景拒绝脏树。** Issue 的第一个选项。推送的镜像总能映射到提交——但标准 ship 流程在推送前就已提交，规则只管束得到手动迭代推送，且修不好它针对的部署检查：期望串从控制器树重算，镜像一旦提交精确，控制器稍脏就部署失败。本地 load——不落入任何拒绝规则——依旧不可区分。在摘要已让比对内容真实的地方，一条只强制约定的规则落选。

**脏树时总是追加提交哈希。** Issue 第二个选项的字面读法：把 `v0.2.0-dirty` 变成 `v0.2.0-gabc1234-dirty`（覆盖 describe 省略哈希的恰在 tag 上的情形）。它锚定了脏在哪个提交上，但仍然只是提交的函数——同一提交的两个脏构建照样相撞，而这正是 issue 提出的问题本身，故不能作为决策。

**脏构建追加构建时间戳。** 区分构建毫不费力，但验证是在部署时重算期望串的；时间戳不可复现，连完全相同树的合法重建都会挂掉部署检查。非确定性恰好破坏了驱动本次变更的那个消费方。

## Consequences

Issue 所指的版本比对现在能区分同提交的不同脏构建。单元测试断言支撑这一点的各项性质：确定性（相同树状态 → 相同字符串）、内容敏感性（不同增量 → 不同字符串）、可逆性（还原树后回归裸 describe 串）；PR 附带真实构建的干净树与脏树版本串输出。

代价：摘要仅覆盖 git 可见的增量，被 git ignore 但仍在构建上下文中（且未被 `.dockerignore` 排除）的文件仍是相撞窗口——上下文卫生本就是 `.dockerignore` 的职责，接受而非重新推导；子模块内部的脏状态不进摘要（子模块指针变更会进）；每次构建对未跟踪集合的哈希开销有界，相对一次多架构镜像构建可忽略；部署控制器需调用 `scripts/build_version.py`——`scripts/check_deploy.py` 本就要求控制器具备 Python。
