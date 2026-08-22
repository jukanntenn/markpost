# MRFC: The .agents/mrfcs tree and its two-document contract

Status: implemented

[English](2026-08-22-agents-mrfcs-tree-and-contract-split.md) | 中文

## Problem

MRFC 系统唯一的 README 同时扛着两份工作：机制的持久契约（布局、生命周期、格式），以及 agent 每次触碰都需要的会话关键守则（写前先 grep、绝不改写决策）。落进这棵树的 agent 没有可加载的 AGENTS.md，这些守则只活在一篇 agent 得知道去打开的文档里。树还坐在仓库根部（`mrfc/`），而仓库的 agent 加载资源正汇聚到 `.agents/` 之下 —— AI agent 工具当作自己加载路径的目录，deepseek-harness 参考项目放自己笔记的地方。一次天真的移动会无声剥掉语料的全部文档闸门：prek 的 `doc-check` hook 整体排除 `^\.agents/`，且没有任何闸门的 PATTERNS 扫进 `.agents/`。最后，README 自述为 "a decision record" —— ADR 框架只覆盖语料回望的那一半，而这棵树的生命周期还盛着请求评审的提案与被拒的裁决。

## Decision

树活在 `.agents/mrfcs/`（以 `git mv` 移动，历史保留）：`.agents/` 是 AI agent 工具加载仓库资源的地方 —— 今天是 skills，现在加上 MRFC —— 与参考项目的落位一致而不引入其机器。树携带两份工作正交的文档：[`AGENTS.md`](../AGENTS.md) 只装常备守则 —— 查重、替代而非改写、保持双语文件对同步 —— 每条都是一个链到规则之家的触发器；[MRFC README](../README.zh.md) 仍是唯一的规范契约。`README.zh.md` 逐节镜像 README，机器 token（`# MRFC:`、`Status:`）保持英文并带显式 ASCII 锚点，使 fragment 链接在两种语言中同样解析；记录保持纯英文，直到双语记录对落地（[layered-instructions MRFC](./2026-08-22-layered-agent-instructions-and-direction-free-mirrors.zh.md)）。术语校准到本质：MRFC 是 markpost 的 RFC —— 持久留存的提案与决策记录 —— "MRFC" 处处是名词，ADR 腔的自述退役。闸门跟随树：[`verify_mrfc_format.py`](../../../scripts/verify_mrfc_format.py)、[`verify_md_links.py`](../../../scripts/verify_md_links.py) 与 [`verify_md_wrap.py`](../../../scripts/verify_md_wrap.py) 把范围定为 `.agents/mrfcs/**`，格式闸门把树根文件（`README.md`、`AGENTS.md`、`README.zh.md`）列入白名单，[`prek.toml`](../../../prek.toml) 中的 prek 排除从 `^\.agents/` 收窄到 `^\.agents/skills/`，使提交时闸门在移动后存活；[`.github/workflows/docs.yml`](../../../.github/workflows/docs.yml) 的路径过滤跟随。记录在案的方向：`.agents/` 计划成为 agent 加载资源的真正源（今天 `.claude/skills/` 是镜像进 `.agents/skills/` 的源）；那次提升是后续工作，按子树的排除让闸门范围在其间保持正确。

## Alternatives considered

**把树留在仓库根部、只加一个 AGENTS.md。** 输在：`.agents/` 是 agent 工具既定的加载路径，仓库的方向使它成为仓库的 agent 侧 —— 决策记忆属于消费它的工作流旁边，而根目录保持产品聚焦。ADR 社区可见 `docs/adr/` 目录的惯例被权衡后搁置：markpost 的语料在作者与读者两端都是 agent 优先。

**整包移植参考项目的机器 —— 分类目录、冻结档案、每笔记 `.zh.md` + `.i18n.yaml` 三元组、每生命周期一个 AGENTS.md 文件、词数预算闸门。** 输在：这重申了 [2026-08-21 的裁决](./2026-08-21-documentation-gates-and-mrfc-system.zh.md) —— markpost 的语料小一个数量级，记录是单语的；在这个规模上，三元组与清单维护给每次编辑上税却什么也买不到。部件只在触发信号上到来；AGENTS.md/README 拆分与一个双语 README 正是本次变更的信号。

**继续以 ADR 决策记录自述。** 输在：语料里还有评审中的提案与被拒的裁决，不只是回望的决策 —— 参考项目自己就把它的笔记定义为 agent 写的 RFC，而 "markpost's RFCs" 命名了这棵树实际管理的整个生命周期。

**只移动、不动排除规则 —— 让闸门跳过 `.agents/`。** 输在：链接、换行与格式闸门会无声地从语料消失 —— 格式闸门将报告零文件并通过绿灯；强制对等正是闸门存在的理由，所以排除改为按子树收窄。

## Consequences

MRFC 语料在新家保有完整的提交时与 CI 闸门；在树里工作的 agent 自动加载常备守则，而每条规则仍单一家居于 README。跨树链接加深一级 —— specs、docs、skills 与 Go 注释中的 30+ 处引用经机械更新并由链接闸门复验。zh 镜像增加一项翻译同步义务，以树的 AGENTS.md 中一行守则承载，而非配对清单。prek 排除现在点名 `.agents/skills/`，于是 `.agents/` 下未来的 Markdown 默认受闸门约束而非默认被排除 —— 正确的默认，也是 skills 镜像升格为真正源时要记住的一条。
