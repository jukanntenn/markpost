# RFC: 退役本地 harness，改用 hdsh

Status: implemented

[English](2026-10-05-retiring-the-local-harness.md) | 中文

## Problem

markpost 此前运行一套自研 harness：`.github/issue-management/policy.py` 引擎及其两个 workflow 与 issue 模板、`scripts/doc_sync.py` 门禁家族（wrap、links、现状散文、配对、预算、MRFC 格式、specs 索引）、`.agents/mrfcs/` 决策树，以及 `scripts/` 下的各清单文件。这套 harness——hdsh 项目正是从它提炼而来——如今存在两份：一份本地、一份上游，并行维护。每个门禁修复或契约变更都要两边落地，而本地副本还带着不再值得保留的 markpost 特有边缘（`Loop status` 看板字段、大写 `P0-P3` 选项、双链接切换行约定）。

## Decision

markpost 接入 [hdsh](https://github.com/jukanntenn/harness-deepseek-harness)（以完整 SHA 钉在 [`prek.toml`](../../../../prek.toml) 与 [`.hdsh/adopt.manifest.json`](../../../../.hdsh/adopt.manifest.json)），并按「不搞双标准」的接入契约在同一变更中退役本地 harness：

- issue 引擎、其 workflow、模板与 `.github/issue-management/` 删除；由 hdsh 的薄 workflow 与上游 composite action 取代，重绑到既有 Project 看板 #1（"markpost Development"）——其内置 Status 字段改写为七个标准状态（`Loop status` 自定义字段作为惰性遗留数据原地保留），新增 `Start date` 字段，看板链接到仓库。四个职能已被 hdsh 拥有的 markpost 自有 skill 在同一变更中退役——`writing-mrfcs`（RFC 规则 + `archiving-rfcs`）、`doc-standards`（`documenting` + `editing-prose`）、`code-review`（`reviewing`）与 `responding-to-review`（栈评审 cookbook）——保留的 skill 重定向了它们的引用。
- `.agents/mrfcs/` 迁至 `.agents/rfcs/` 并归类（`feature`/`bug-fix`/`simplification`/`architecture`/`process`/`testing`），标题行 `# MRFC:` 改为 `# RFC:`，逐对记录配对 sidecar。
- 文档门禁脚本及其清单删除；词数上限折入 [`.hdsh/docs.manifest.json`](../../../../.hdsh/docs.manifest.json)，`specs/` 语料子树折入 [`.hdsh/pairing.manifest.json`](../../../../.hdsh/pairing.manifest.json) 的 roots，七个 hdsh 门禁经 adopt 管理的 prek 块与 [`.github/workflows/docs.yml`](../../../../.github/workflows/docs.yml) 运行。
- `scripts/agentlib.py`、`check_agent_instructions.py` 与 `sync_agent_instructions.py` 保留：AGENTS↔CLAUDE 与 skills 镜像契约是 markpost 本地事务，与文档标准正交。`.claude/skills/` 镜像在引导时即包含 hdsh 安装的 skills，使方向无关的镜像永远不会把它们当作幻影删除。

## Alternatives considered

- **两套标准并行** —— 否决：一个语料两套门禁必然漂移，且 hdsh 的接入契约明确拒绝双标准。
- **保留本地 harness、不接 hdsh** —— 否决：并行维护成本正是要解决的问题。
- **复用看板 #1** —— 否决：其状态挂在自定义 `Loop status` 字段上；hdsh 契约只认内置 Status 字段，且旧看板承载着旧循环的条目状态。

## Consequences

在 hdsh 比本地家族更严格之处，门禁语义收紧：切换行为首个 H1 之后的单链接形式；围栏代码块在两种语言间逐字节一致（含注释）；配对记录是逐对的 `.i18n.yaml` sidecar 而非中心化语言清单；`docs/development.md` 保持消费者所有，其配对在评审前暂不记录。升级时 prek 的 `hdsh` 组与 `docs.yml` 的钉定安装必须一起动（在新 ref 下重跑 `hdsh adopt apply`）。PR 与 issue 机制（标签 `kind/*`、`type/*`、`p0`-`p3`；机器账号；单批准分支保护）实质不变。
