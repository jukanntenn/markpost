# MRFC: Layered agent instructions and direction-free CLAUDE.md mirrors

Status: implemented

[English](2026-08-22-layered-agent-instructions-and-direction-free-mirrors.md) | 中文

## Problem

根 `AGENTS.md` —— 连同其逐字节相同的 `CLAUDE.md` 拷贝 —— 承载了四个技术栈的全部子树细节：分领域的命令块、迁移规则、Go 与 TypeScript 风格、分栈的测试约定和混杂的边界条款，共 1,172 词，无论任务是什么，每次会话都全量加载。分层其实早已开始（[`docs/AGENTS.md`](../../../docs/AGENTS.md) 拥有文档标准，[`../AGENTS.md`](../AGENTS.md) 拥有 MRFC 常备守则），但三个代码工作区没有自己的层，没有任何机制把子树细节推出根文件，也没有任何东西约束这份"每会话都要付费"的文件 —— [2026-08-21 的裁决](./2026-08-21-documentation-gates-and-mrfc-system.zh.md) 只选择性地移植了参考项目的闸门，并把词数预算推迟到出现触发信号为止；四个技术栈在根文件里的堆积正是那个信号。镜像机制本身也有缺陷：`sync_agents.py` 是单向拷贝 `AGENTS.md` → `CLAUDE.md`，一种主/从关系。工具加载 `CLAUDE.md` 的 agent 编辑了它，下一次同步时这份编辑就会被更旧的 `AGENTS.md` 静默覆盖 —— 恰是镜像绝不能犯的"旧的覆盖新的"错误。

## Decision

**带预算的分层。** 根文件只留每次会话都需要的内容 —— 身份、技术栈、布局地图、跨领域命令索引（`dev.py`、`doc_sync`）、git 工作流、跨领域边界，以及 "Editing these instructions" 一节 —— 以 576 词落在 800 词上限之内。三个子树文件承接工作区专属守则：`backend/AGENTS.md`（Go 命令、迁移规则、Go 风格、testcontainers 约定、后端专属禁令）≤ 500 词，`frontend/AGENTS.md`（pnpm 命令、React/TypeScript 风格、静态导出约束、Vitest）≤ 300 词，`e2e/AGENTS.md`（命令、仅 chromium、dagger 保真）≤ 150 词。子树文件补充根文件、绝不复述根文件；根文件的布局地图逐一链接它们。`devops/`、`scripts/`、`specs/` 不建文件：部署过程在 `docs/`，闸门契约在 [`docs/AGENTS.md`](../../../docs/AGENTS.md)，`dev.py` 命令是根文件拥有的全仓入口。

**无方向的镜像。** 四棵 agent 指令树 —— 根、`backend/`、`frontend/`、`e2e/` —— 每棵都在 `AGENTS.md` 旁持有一份逐字节相同的 `CLAUDE.md`；`docs/AGENTS.md` 与 `.agents/mrfcs/AGENTS.md` 是关于文档的常备守则，保持单文件。文件对没有主从。方向按对基于 git HEAD 判定 —— 绝不用 mtime，clone 和 checkout 会把它重置：恰好一侧与 HEAD 不同，则该侧复制到另一侧，新的更新旧的；两侧相等则通过；两侧都有改动且互不相同是冲突，工具拒绝猜测。HEAD 中不存在的路径按该侧已变更计，新文件对由此自举。

**更名的工具链与自愈闸门。** `check_agent_instructions.py`（闸门）与 `sync_agent_instructions.py`（修复器）共用判定模块 [`agentlib.py`](../../../scripts/agentlib.py) —— "agent instructions" 是 agents.md 生态自己对这类文件的称谓，同时去掉易与 subagent 混淆的 `agents` 命名，hook id `agents-sync` 一并更名。[prek](../../../prek.toml) 闸门 `agent-instructions-sync` 的文件过滤扩到子树文件对；单侧改动且方向明确时，闸门自行更新并暂存旧的一侧；只有真冲突或删除才失败，且失败输出写明两侧、修复命令、以及手工调和步骤 —— 读到它的 agent 知道该怎么做。[CI](../../../.github/workflows/docs.yml) 路径过滤跟随更名。

**词数预算。** [`verify_doc_budgets.py`](../../../scripts/verify_doc_budgets.py) 与 [`doc_budgets.manifest.json`](../../../scripts/doc_budgets.manifest.json) —— `wc -w` 整文件上限，预算文件缺失即失败，改名无法让预算成为孤儿 —— 加入 [`doc_sync.py`](../../../scripts/doc_sync.py)。预算是上限而非压缩目标：闸门变红时，先把内容归位到其所属层级，再压缩措辞，最后才是带着在 PR 中说明的 manifest diff 上调上限。上限全部低于参考项目（根 800 对 1,600；子树 500/300/150 对默认 600）：切分后的内容用不了那么多；`docs/AGENTS.md` 定 600，为承接子树层行与预算规则而增长。

**文档跟随。** `docs/AGENTS.md` 增加了子树 AGENTS 层行、预算规则，以及"子树文件复述根文件"这一 slop 案例；doc-standards skill 承接更新的工作流；根文件的 "Editing these instructions" 一节写明镜像契约（改任一侧均可，工具保持文件对相等）、预算与子树清单。

**双语记录对。** [`verify_mrfc_format.py`](../../../scripts/verify_mrfc_format.py) 现在接受与英文主记录同名的 `.zh.md` 镜像 —— 闸门扩展与本记录同落地，闸门与触发它的需求在一起。两份骨架都被校验，机器 token 与节标题保持英文，一对文件一同更新。本记录是第一对，依 maintainer 指示而为，放宽了[树契约](./2026-08-22-agents-mrfcs-tree-and-contract-split.zh.md)中"记录仅英文"的事实；[MRFC README](../README.zh.md) 写明了镜像对契约。skills 镜像维持方向性（`.claude/skills/` → `.agents/skills/`）；把 `.agents/` 提升为源仍是该契约记录在案的后续工作。

## Alternatives considered

**软链镜像 —— 参考项目的机制。** 构造上零漂移：一个 inode，两个名字。它输在：git 把软链存为内容为目标路径的 mode-120000 blob，Windows 检出时若非 `core.symlinks=true` 加开发者模式或管理员权限，`CLAUDE.md` 会退化为一个内容是九个字节 `AGENTS.md` 的普通文件；在那儿创建符号链接需要特权，归档传输与若干网络文件系统也会破坏链接。没有任何东西把贡献者约束在类 Unix 检出环境上，第一次 Windows clone 就会把漂移风险换成坏掉的镜像。

**维持单一的大根文件。** 它输在：无论任务是否触及，每次会话都为所有技术栈的细节付费，而加载路径本身已支持渐进披露 —— Claude Code 读到子树文件时会拉取该子树的 `CLAUDE.md`，Codex 沿 cwd 链合并 `AGENTS.md` —— 分层让支持该机制的工具把上下文税局部化，其余工具靠根文件的链接。没有机械预算，堆积还会继续：参考项目自己的 tiers-and-budgets 记录实证过没有护栏的文字纪律守不住。

**维持单向同步（现状）。** 它输在：方向性就是缺陷本身，不是可调参数 —— 被命名为从的那一侧，其编辑会被修复器静默回退，而那一侧恰是其工具加载、其 agent 最可能编辑的一侧。

**基于 mtime 的方向判定。** 它输在：clone 和 checkout 把 mtime 重置为当下，两侧变得同等"新"；基于 HEAD 的内容比较是确定性的，且不依赖任何文件系统状态。

**各工具的原生规则文件（`.cursor/rules`、copilot instructions 之流）。** 它输在：一个事实获得每个工具一个家，并在它们之间漂移；镜像让一份内容顶着各工具的文件名存在，边际撰写成本为零。

**对齐参考项目的预算数字（根 1,600、子树 600）。** 它输在：切分后的内容远低于这些数字 —— 根文件落在 576 词 —— 更高的上限只会为不需要的增长发放许可。

## Consequences

镜像矩阵已在干净的 scratch clone 中逐场景演练：任一侧的单侧编辑都被闸门自动修复并暂存；两侧相同编辑通过；两侧相异编辑以指明两侧与调和步骤的输出失败；只添加新文件对的一侧即可自举出另一侧；无 HEAD（尚无首次提交）时仍然强制文件对相等。`doc_sync` 携预算闸门在全语料上全绿，删除任一被预算文件会使之失败；根↔子树的每条交叉链接经链接闸门解析，新 `AGENTS.md` 文件作为普通 Markdown 被其覆盖。接受的代价：闸门现在会改动已暂存内容，由确定性判定兜底（只有"恰好一侧变更"的情形会自动修复，双侧变更永不猜测）；方向判定读取 HEAD，合并或变基后文件对不一致会以带指引的冲突暴露，绝不静默拷贝；偏紧的上限诱发上调琐碎化，由余量与文档化的上调路径吸收；成对记录为该记录的每次更新增加翻译义务，对每个记录而言成对是可选的。文件对已在 [bilingual-pairs MRFC](../proposed/2026-08-22-bilingual-documentation-pairs.zh.md) 中成为每条记录的强制项。
