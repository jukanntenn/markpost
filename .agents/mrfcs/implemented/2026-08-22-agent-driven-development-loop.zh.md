# MRFC: The agent-driven development loop

Status: implemented

English | [中文](2026-08-22-agent-driven-development-loop.md)

## Problem

markpost 曾以直推 `main` 的方式交付：历史里一个 pull request 都没有（仅一次 release 回合除外），没有 issue 入口——无模板、无看板、无分诊——review 即便发生，也发生在 agent 会话内部，不留持久记录。本仓库需要的运作模式把这一切倒转过来：人类做决策，agent 驱动百分之百的执行——issue 进、成栈受审的 pull request 出、合并后工作自动关闭——构成一个可重复到能排程的闭环。这样的闭环强制出的决策（哪些点留给人类、review 由谁做、stacked pull request 与 worktree 如何组织、看板如何反映进度、门禁靠什么机械强制）是代码与 spec 都装不下的流程设计，本记录是它们的家。机制词汇借鉴自 `.local/contexts/deepseek-harness` 参考实现，在 markpost 属于单人维护的个人仓库而非组织之处逐一修剪。

## Decision

**三门人类门禁；其余全部 agent 驱动。** 决定做什么（issue 进入 `Ready`）、批准设计（RFC 栈）、验收交付（实施栈）是每个非平凡变更中人类的三次决策；平凡变更只保留交付门禁。`Ready` 前移靠 skill 纪律——agent 永不把 issue 推入 `Ready`，只认领已在其中的 issue——因为项目看板权限表达不了这道边界；另外两道门禁在分支保护开启后由平台强制（见下的维护者设置项；required checks 的设计见 [PR conclusion jobs and required checks](2026-08-24-pr-conclusion-jobs-required-checks.zh.md)）。既有的 Ask-first 边界（schema 迁移、新依赖、CI 与 Docker 变更）延续为 pull request 模板必有的"Ask-first 项"一节，作为人类 review 的强制关注清单。agent 在请求人类 review 前对每层先出预审：对照 `AGENTS.md`、specs 与文档门禁的结构化 comment review。approve 单属人类——不是靠约定，而是因为机器账号无法 approve 自己的 pull request，平台让"comment review 归 agent、approve 归人类"这组分工成为承重结构。

**机器账号执笔。** 所有 agent 驱动的 GitHub 操作——认领、push 分支、开 pull request、评论、落栈——以专属机器账号执行（`gh2bda`，本仓库的 collaborator，并被有意设计为可在维护者多个 agent 驱动项目间复用），凭一张 classic PAT——其四个 scope 各自解开一堵实测撞过的墙：`repo`（fine-grained 替代方案根本选不了协作者仓库）、`project`（看板写入）、`read:org`（gh CLI 的内部查询，如 `gh pr edit`）、`workflow`（HTTPS 推送触及 `.github/workflows/**`；SSH 推送不受 scope 限制）——经 agent 环境的 `GH_TOKEN` 生效，维护者自己的 `gh` 登录原封不动地留给决策动作。classic token 不是仓库级 scope 的——爆炸半径由账号的协作清单而非 token 界定，因此该账号只在运行闭环的仓库获得访问。这组分离买到共享身份买不到的三样东西：审批门禁能区分 agent 预审与人类签核、审计轨迹能把每个时间线动作归因到真实驱动者、agent 的爆炸半径是这些协作仓库而非整个人类账号。

**模板化 issue 入口。** 五张中文 issue 模板——`Idea`、`Feature`、`Bug`、`Research`、`Task`——禁空白 issue；每张模板强制一句话陈述（外露不超过 50 单位），折叠的 `<details>` 区承载验收条件与证据字段。类型由模板 frontmatter 自动打的 `type/*` 标签承载，因为个人仓库没有原生 issue types（实测：GraphQL `repository.issueTypes` 在此返回 `null`）。一个个人 GitHub Project 作为看板：状态列 `Inbox / Backlog / Ready / In progress / In review / Done / No action` 存于自定义的 `Loop status` 字段（项目默认的 `Status` 字段不可删除、亦无编辑其选项的公开 API——对 `deleteProjectV2Field` 与选项编辑面均已实测——因此闲置并在视图中隐藏），旁置 `P0–P3` 的 `Priority` 字段。`issue-lifecycle` workflow 自动流转——打开或重开进 `Inbox`；resolving pull request 上的实现动作进 `In progress`；请求 review 进 `In review`；changes-requested review 退回 `In progress`（仅当当前状态由自动化设置，该事实记录在标记评论里，人类手动摆放的位置绝不被静默覆盖）；关闭按原因进 `Done` 或 `No action`。`issue-policy` workflow 作为 pull request check：非草稿、非 bot 的 pull request 一旦进入评审，必须带合法的 Conventional Commit 标题、至少一个 `area/*` 标签、形态正确的引用，且每个被引 issue 满足模板契约。两者共用一份 Python 纯标准库 policy 脚本（`.github/issue-management/policy.py`，32 个 `unittest` 用例挂入 prek），agent 在提交前也对照同一份——一套逻辑，自检与强制同源。pull request 的 kind 留在 Conventional 标题前缀而非标签；`area/backend|frontend|e2e|devops|docs` 是可扩展的起点集；无 source 标签、无迁移机制。看板写入已激活：`config.json` 指向 1 号项目并置 `requireProject: true`，lifecycle workflow 以 `MARKPOST_PROJECT_TOKEN` secret（机器账号的 classic PAT）认证，因为 `GITHUB_TOKEN` 没有面向用户级 Projects v2 的权限 scope（实测：actionlint 的 scope 清单里不存在此类授权），该 PAT 是唯一的看板写入路径——它的每次写入都以 `gh2bda` 身份记录，标记评论守卫据此读回。

**认领与分解。** agent 认领处于 `Ready` 且无 assignee 的 issue，按优先级再按创建时间取序；认领即 assign 加一条声明评论，绝不写看板。分解的单位是决策而非文件：agent 枚举该 issue 强制的非平凡决策，每个决策写一对 MRFC，决策间有依赖（schema、然后 API、然后 UI）则 stack pull request，相互独立则平铺独立开 PR。符合既有"机械局部修改"豁免标准的 issue 走快速通道，没有 RFC 层——直达实施 pull request，只剩交付门禁。

**两段式栈。** RFC 栈——每层一对 `proposed/` MRFC，各层以 `Related to #N` 引用 issue——在设计门禁受审；批准后先行合并，MRFC 以 `proposed/` 落上 `main`，实施栈从新 `main` 起建、引用稳定。只有实施栈顶层带 closing 关键词 `Fixes #N`，issue 恰在整栈落地时关闭，部分落地时保持开启；agent 在实施启动时写一次看板的 `In progress`，因为底层实施层没有 closing 引用、lifecycle workflow 看不见实施开始。使决策成为现实的那一层实施（通常是顶层）在同一变更里把该 MRFC 对从 `proposed/` 移入 `implemented/` 并改写为现在时（本记录正是在实施它的变更里如此完成的）。实施中发现的小设计缺口在实施 pull request 内就地修订仍是 `proposed` 的 MRFC；决策级反转停下来等人类。

**Worktree。** 每层一个 worktree，位于 `.local/worktrees/<分支>/`——在仓库既定的杂物根之内，已被 `.gitignore` 与 `.dockerignore` 双排除，skill 指令因此是仓库相对路径，克隆到哪里都成立。分支命名 `rfc/<issue>-<slug>` 与 `impl/<issue>-<slug>`；修复落在引入问题那一层的 worktree，绝不在下游 checkout 就地修。`prek` 的 hooks 经共享 hooks 目录分发、却在当前 worktree 内从 `PATH` 解析 `prek`，因此未移植任何 per-worktree 安装器。在[隔离提案](../proposed/2026-08-22-worktree-dev-environment-isolation.zh.md)落地之前 worktree 串行使用：compose dev 环境的固定容器名使环境互斥。

**会话恢复协议。** 门禁是异步的，因此每个 agent 会话以 triage 开场：枚举机器账号的开放 pull request 及其官方栈；对每一个，若 review 决定为 approved、checks 全绿、无未决变更，则门禁已过——合并该栈并推进下一阶段（RFC 栈合并意味着实施开始；实施栈合并意味着清理与关闭）；若 review 带回反馈，走响应循环；若无在途工作，认领下一个 `Ready` issue。会话由人类明确请求启动；执行同一协议的定时轮询是指名的后续项，仅在手动轮次证明稳定后加入。

**交付物与证据。** 每个实施层交付代码、测试与其变更触及的文档——新 spec 页在同一 pull request 里加 `specs/index.md` 行。pull request body 写明引用、Ask-first 项与证据：跑过的命令及结果、对照 issue 验收条件的映射、UI 变更附 Playwright 截图。动画交互证据推迟到[浏览器 GIF 提案](../proposed/2026-08-22-browser-gif-evidence-chain.zh.md)；当前验收由截图承载。

**反馈传播。** 每条 review 评论先对照代码核实再行动——指出正确症状的 review 也可能诊错病因。接受的修复落在引入问题的层上，以 merge-forward 向上传播，此为默认，因为它不重写已 review 的历史、approvals 保持有效；rebase 路径保留为刻意之选，lease 保护、事后全面重审。每个修复独立成 commit，绝不 amend 已 review 的工作；回复写进原 review 线程并注明携带修复的 commit。

**落栈。** 栈只经 GitHub 原生 stack 对象与 `gh stack merge <stack> --yes --merge` 落地，preflight 先重查官方栈并逐层独立判定——open、非草稿、approved、checks 全绿。原生合并报出的阻塞在 owning pull request 内解决或停下整轮；绝不回退到逐 PR `gh pr merge` 加手动 retarget。完成的判定是每层查到 `MERGED`；单独的清理轮在验证无开放 pull request 再以某分支为 base 之后才删它，随后移除对应 worktree。

**守护。** 仓库的合并方式为仅 merge commit，v1 分支保护为要求 pull request 加一个 approving review、新推送作废过期批准、要求会话内线程解决、包含管理员、禁止向 `main` force push——两者自 bootstrap 合并之后即已生效，且只能在其后：required approval 加 include-administrators 会让唯一维护者卡死在自己署名的 pull request 上（无人可 approve 自己的 PR），保护无法在 bootstrap 合并前开启；而 agent 的 pull request 以机器账号署名，正是 required-approval 门禁自此可运转的前提。暂不设 required checks：路径过滤的 workflow 对纯 Markdown pull request 整体跳过，一个永不报告的 required check 会卡死每个 RFC 栈；在此期间 checks 汇总由 agent 的 triage 自行核验。补上这道缺口的后续项见下（结论 job）。`deleteBranchOnMerge` 保持 false；验证过的手动清理覆盖部分落地场景。不引入 merge queue——单人维护者没有并发落地压力。

**skills 承载工作流。** 六个 skill 各管一段机制——`dev-loop`（恢复协议与阶段状态机）、`stacked-prs`（worktree、分支命名、建层、链接）、`merging-stacked-prs`（落栈）、`responding-to-review`（反馈传播）、`code-review`（预审规范）、`filing-issues`（模板纪律）——与 `iterating`、`writing-mrfcs`、`commit` 组成从讨论到落地的完整管线。根 `AGENTS.md` 持有一节紧凑的 Development Loop，写明门禁、机器账号与引用纪律；本记录持有全部理由。

## Alternatives considered

**agent 共用维护者的 GitHub 身份。** 零配置，但审批门禁失去机械意义——共享身份无法区分 agent 的预审 approve 与人类签核——审计轨迹无法归因，agent 还握着覆盖整个账号的 token。机器账号花一次手工 setup，换回强制性、可归因与单仓库爆炸半径。

**用 GitHub App 而非机器账号。** scope 最小的正式之选，因实测被否：本机 `gh` 的 `gh auth login` 没有 App 凭据流程（仅 web OAuth、`--with-token`、环境变量 token 三种），App 意味着本地维护一套私钥换 installation token 的铸造脚本，面对每个会话一小时过期的 token。这对有平台工具的组织合比例，对个人仓库不合。

**更少的人类门禁。** 两门（agent 自行采纳 issue）或一门（agent 自审设计、人类只验收交付）都放弃了维护者明确要保留的决策：做什么与怎么设计同级；设计错误拖到交付才发现，代价是整个实施。三门意味着每个非平凡变更约三次人类接触，快速通道把平凡工作裁到一次。

**人类 review 每个 pull request，agent review 仅作参考。**"人类或 agent review"最保守的读法，但它把闭环变回人类节奏的串行 review；所选模型是 agent 先审、人类批准，Ask-first 类别在既有边界已要求之处强制人类关注。

**一个巨型栈（RFC 层在底、实施层在其上），一次落地。** 单一评审上下文，但设计门禁期间的任一次 RFC 修订都会级联 rebase 其上的实施层，两道门禁在同一栈里界限模糊。两段式让每道门禁恰好对应一次合并，每个栈都小。

**所有实施层都带 `Fixes #N`。** workflow 独立给出正确的 `In progress` 时机，但底层合并会在栈的其余部分落地前几秒关闭 issue——整栈落地时不可见，部分落地时错误。仅顶层带关键词加一次 agent 写入的 `In progress` 在两种场景下都正确。

**worktree 放仓库外。** 不依赖任何 ignore 清单的物理隔离，也是参考实现的实际模型。因实测被否：声称的硬理由——构建上下文污染——已被覆盖（`.dockerignore` 排除 `.local/`），剩余成本是外观层面的，而 `.local/worktrees/` 让 skill 指令保持仓库相对，并把项目产物收在一个可统一清理的根下。

**外部兄弟目录 worktree。** 同一组权衡从另一侧实测；因同样理由落败——仓库相对指令与既定的 `.local/` 杂物惯例，重于新工具边界已基本覆盖的残余风险所带来的物理隔离。

**完整标签体系。** 每个 pull request 恰好一个 `kind/*` 加至少一个 `area/*`、issue 用原生类型、source 标签、迁移机制——参考实现的组织级答案。markpost 的 Conventional 标题已在被查询处承载 kind，个人仓库没有原生 issue types，单一机器作者使 source 标签冗余；修剪版保留 `area/*` 供审查分流，其余丢弃。

**rebase 作为默认传播历史。** 历史紧凑，但它把重写已 review 分支变成常规动作，每轮反馈都作废 approvals 与评论锚点。merge-forward 保全已 review 的历史；rebase 保留为 lease 保护下的刻意逃生通道。

**squash 合并。** 换一个线性的 `main`，但栈式子层对 base 分支算 diff；squash 的父层改写了子层所坐落的提交，把父层变更拖进每个子层 diff，拆掉闭环存在的理由——逐层 review 结构。

**第一天就上带 required checks 的分支保护。** 立即机械完备，但路径过滤的 workflow 跳过纯 Markdown 变更，每个 RFC 栈都会永远等待永不报告的 checks。分阶段计划先上零 CI 改动即可用的审批门禁，再在要求 checks 之前补齐结论 job。

**第一天就定时驱动。** 立即完全无人值守，但首轮失败会在无人监督下重复。手动会话先把闭环端到端验证；排程器之后包裹同一套 triage 协议。

**从 GitHub timeline 事件读看板状态写入者。** 参考实现查询 timeline 元数据获知谁设置了状态；确切的事件类型文档不足、依赖脆弱。标记评论——workflow 记录自己的每次写入——以纯 REST 数据提供同样的守卫。

## Consequences

闭环的仓库侧在本变更中完整落地：五张 issue 模板与 `config.yml`、pull request 模板、policy 脚本及其挂入 prek 的 32 用例套件、两个 workflow、六个 skill、紧凑的 `AGENTS.md` 章节，以及本记录自身的生命周期移动——bootstrap pull request 顺手演练了它引入的基础 PR 机制。仓库之外的激活已完成：机器账号连同其四 scope PAT 与 collaborator 权限就位，个人 Project 带上了 `Loop status` 与 `Priority` 字段且机器账号是项目的 `WRITER` 协作者，`MARKPOST_PROJECT_TOKEN` secret 已设置，`config.json` 已翻转，仅 merge commit 与 v1 分支保护均已生效。`type/*` 与 `area/*` 标签已由本变更的 setup 命令建好。首环验证：快速通道此后已在生产中跑通两次——标记端点修复与看板可见性修复各自走了 issue → 机器署名 pull request → 人类批准 → agent 合并 → issue 自动关闭 → 看板 `Done`。在其他仓库重复此激活的操作清单与背后的实测平台约束，见[闭环运行手册](../../../docs/agent-loop-runbook.zh.md)。

接受的代价与常在风险：看板自动化以一张长期 PAT secret 认证（实测 `GITHUB_TOKEN` 写不了用户级 Projects v2），是多一份需要轮换的凭据；dev 环境在隔离提案落地前跨 worktree 互斥；三门是每个非平凡变更的三次人类接触，快速通道减轻但不消除；`gh stack` 是架在年轻原生 API 之上的外部扩展，任一方变动时落地将硬停而非降级为不安全的手动合并；agent 的预审与其作者共享盲区——它是过滤器和简报，不是后续批准的替身；而 `Ready` 门禁始终是 skill 纪律而非平台强制——未来某个 agent 的看板误判可在审计轨迹中察觉，但仅凭权限无法预防。

指名的后续项各带触发条件：逐 workflow 的结论 job 与五个 required checks（首批闭环跑通之后）；triage 协议的定时轮询（手动轮次证明稳定之后）；[浏览器 GIF 证据](../proposed/2026-08-22-browser-gif-evidence-chain.zh.md)（当人类反复亲手重验交互主张时）；以及 [worktree 开发环境隔离](../proposed/2026-08-22-worktree-dev-environment-isolation.zh.md)（当全栈验证争用真实咬合时）。
