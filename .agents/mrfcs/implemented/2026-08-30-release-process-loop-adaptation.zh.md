# MRFC: Adapting the release process to the agent-driven loop

Status: implemented

[English](2026-08-30-release-process-loop-adaptation.md) | 中文

## Problem

release 技能从头到尾教的是闭环之前的仓库运营模式：把版本号提交直推 `main`、push、再推 tag。[开发闭环](2026-08-22-agent-driven-development-loop.zh.md)封闭了那个世界——今日实测，`main` 的分支保护要求 pull request 携带一个批准评审（管理员同样受限）并解决全部会话，禁止 force push，并点名六个 required checks（五个 conclusion 检查加 `Issue policy`）；仓库只允许以 merge commit 合并。技能的核心动作在机制上已不可能——对 `main` 的直接 push 会被拒绝——而新体制下尝试过的唯一一次发版，即经 PR #23 走的 v0.2.0-rc.6，是在 `Issue policy` 变红的情况下合并的：正是那起事故催生了 [PR conclusion jobs and required checks](2026-08-24-pr-conclusion-jobs-required-checks.zh.md) 与 [release 豁免](2026-08-24-issue-policy-release-exemption.zh.md) 两份记录。豁免让裸的 `release/**` pull request 能通过 policy，并把分支命名留给了 release 技能——而技能仍在教那条死路，也对闭环的异步门禁无言以对：发版等待的批准，在开出它的会话结束之后才到来。

## Decision

发版与任何其他变更走同一套轨道，只是带一个 release 特有的形状，[release 技能](../../skills/release/SKILL.md)现在教的正是这套。

**版本落地经由 pull request。** 从当前 `main` 切出分支 `release/vX.Y.Z[-rc.N]`——技能拥有这个命名，policy 豁免正是按它匹配——携带一个 `chore: release vX.Y.Z` 提交（`frontend/package.json` + `CHANGELOG.md`）、一个 conventional 标题、不带 `area/*` 标签也不带 issue 引用：依豁免有意裸报。

**交付门禁是 pull request 的批准评审。** 与每个闭环 PR 一样由机器账户署名——维护者署名的 PR 会在 include-administrators 上死锁，因为没人能批准自己的 pull request，而批准只属于人类。会话内的暂停在它们是决策而非合并之处原样保留：版本号确认留下；旧的推送前确认被吸收，因为 PR body 声明了合并之后将打 tag `vX.Y.Z`、其推送触发 `release.yml` 与 `docker-publish.yml`——批准即是对发布的授权。

**落地是单 PR。** 预检（open、非草稿、approved、六项检查全绿、无未解决的 change request），然后 `gh pr merge --merge`——仓库唯一的合并方法；一个基于 main 的孤立 PR 没有栈对象可链接、也没有 retarget 风险，闭环对逐 PR 合并的禁令（栈的后备路径）因此不适用。分支清理仍循栈规则：仅当没有 open PR 再基于该分支时才删除。

**发布由 tag 驱动，严格处于门禁下游。** 合并之后，为 `main` 上的合并提交打 tag 并推送——绝不打在合并前的分支头上，那会发布一个交付门禁尚未接受的提交。分支推送不发布任何东西；`docker-publish.yml` 由 `v*` tag 触发，滚动的 `main` 镜像是 `docker/build.py` 的另一条车道。

**流程可续跑。** 任何会话在 triage 中发现 `release/**` pull request 已批准且全绿，即可完成它——合并、打 tag、验证——无需重新推导发版，release 由此像其他一切工作一样嵌入闭环的异步门禁。

## Alternatives considered

**给分支保护开例外，让维护者仍能直推发版到 `main`。** 发版压力下最快，但 include-administrators 恰恰为了让维护者自己的推送也过闸而存在，而压力正是检查起作用的时刻——实测即 #23 带红合并。例外还会为用户最先消费的那些提交重新打开无审计的直接推送历史。

**满足 issue policy 而不是倚仗豁免**（conventional 标题 + `area/devops` + 一个长期 release issue）。所有 pull request 一套 policy，但豁免记录已实测并否决：每次发版造一个一次性 issue，去喂一个为评审注意力分流而生的检查——它不该管发版。

**在合并前给 release 分支打 tag，让 tag 在 PR 开出那一刻就存在。** 发布会在 `main` 尚未接受的提交上开始；评审一旦改动，就意味着删除已发布的 tag 及其 Docker 标签。给合并提交打 tag 让发布严格处于交付门禁下游。

**把发版跑成闭环 issue（模板、看板、两阶段栈）。** 与功能工作完全一致，但多出了 `Ready` 门禁——对一个版本号毫无内容的人类决策——又逼出了豁免本要避免的 issue 引用。发版由维护者定时；唯一的交付批准才是它们真正包含的决策。

**对孤立的 release PR 也用 `gh stack merge`。** 每次落地一条机械路径，但为无所获而支付扩展依赖：没有栈可验证、没有可 retarget 的东西，`gh pr merge --merge` 在同一保护下行使同一合并方法。

## Consequences

下一次发版将端到端地度量这套流程：一个裸的 `release/**` pull request 应当展示六项全绿的检查——`Issue policy` 经豁免变绿，是必需检查集应用以来的首次实跑——并且只在维护者批准后合并。发版多出一个人类触点，即其他一切变更早已支付的同一个交付门禁；版本校验规则、CHANGELOG 文风与失败即停的姿态原样未动。回滚如今分阶段：合并之前，关闭 pull request 并删除分支；合并与打 tag 之间，经 pull request revert 合并提交；打 tag 之后，删除 tag（本地与远端）并在 web UI 里编辑或删除 GitHub Release——直接推送的撤销随直接推送一同消失。豁免记录那句"release 分支命名仍归 release 技能所有"，如今有了所有者的正文：`release/vX.Y.Z[-rc.N]`。
