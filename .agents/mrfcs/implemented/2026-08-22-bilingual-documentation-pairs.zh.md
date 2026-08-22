# MRFC: Bilingual documentation pairs

Status: implemented

[English](2026-08-22-bilingual-documentation-pairs.md) | 中文

## Problem

语料的读者使用两种语言 —— 维护者以中文工作，第一对记录正是依此指示而生 —— 但没有任何机制约定一个页面该用哪种语言书写。`specs/` 逐文件单语，或英文或中文或句中混杂（`auth.md`、`api-design.md`、两篇 `i18n` spec、`e2e/HANDBOOK.md` 为中文原生；十余篇混合），`backend/docs/API_SPECIFICATION.md` 是躲在生成 Swagger 豁免之后的过时中文页，根 README 对的命名则早于 MRFC 树确立的约定（`README_zh.md` 对 `.zh.md`）。读者只能以母语读一半语料，另一半完全读不了，而哪一半取决于所在目录。没有任何页面与另一语言的对应页保持同步，因为根本不存在对应页：既有的双语机制 —— 可选的 `.zh.md` 镜像，仅被树的 README 与恰好一条记录使用 —— 使单语文件成为默认，于是 `PRINCIPLES.md`、操作指南和全部历史记录只有英文，而一半 specs 只有中文或混合。[2026-08-21 的裁决](./2026-08-21-documentation-gates-and-mrfc-system.zh.md)曾将参考项目的双语机制推迟到触发信号出现为止；在整个语料上服务两个语言社区，正是那个信号。

## Decision

**同等权威的文件对，统一命名。** 除 agent instructions 外，每个文档文件成对交付：英文 `foo.md` 与中文 `foo.zh.md` 并置于同一目录，英文侧不带语言中缀 —— `README_zh.md` 已更名为 `README.zh.md`，对齐 MRFC 树、首条记录镜像与格式闸门文件名正则早已共用的约定。两种语言同等权威：任一侧都可先行撰写或编辑，被编辑侧即该次变更之源，镜像在同一变更内以最小补丁跟进 —— 绝不整篇重译；当文件对出现实质分歧，改错的一侧，任何语言都不默认获胜。范围为整个文档语料 —— 根 README、`PRINCIPLES.md`、`docs/`、`specs/`（含索引）、`.agents/mrfcs/`（含每条历史记录）、`e2e/HANDBOOK.md` —— 豁免载于仅列排除项的清单 [`scripts/doc_languages.manifest.json`](../../../scripts/doc_languages.manifest.json)：agent instructions（全部 `AGENTS.md`/`CLAUDE.md`/`SKILL.md` 及 skills 镜像）、账本 `CHANGELOG.md` 与 `KNOWN_ISSUES.md`（按设计叙述历史，逐条翻译是义务而读者价值为零）、`scripts/loadtest/` 时点报告、生成的 Swagger、`.zcode/`。`backend/docs/API_SPECIFICATION.md` 已直接删除：[`specs/backend/api-schema.zh.md`](../../../specs/backend/api-schema.zh.md) —— 端点参考的唯一家 —— 的过时中文重复页，且在基本事实上与之矛盾（基础路径 `/api` 对 `/api/v1`）。

**一个闸门，五项检查。** [`scripts/verify_doc_pairs.py`](../../../scripts/verify_doc_pairs.py) —— 基于 `doclib` 的 stdlib Python，与同级闸门一样支持暂存范围收缩，并把暂存文件展开为整对 —— 强制执行：双向配对完整性（范围内每个 `.md` 都有其 `.zh.md`；没有任何 `.zh.md` 是孤儿）；每侧头部区域存在链接其镜像的语言切换行 —— 唯一被许可的跨语言链接；链接语言区（`.zh.md` 内指向语料内的相对链接定位到 `.zh.md` 变体，`.md` 对应 `.md`；语料之外的目标保持原路径）；简单形态的结构对等 —— 标题深度序列一致、围栏代码块逐字节相同（含注释，因为示例不翻译）；以及语言纯度 —— 范围内 `.md` 的散文不得含 CJK，主题确需 CJK 的页面走清单逐文件豁免，`.zh.md` 不做反向检查 —— 中文页合法携带英文术语。它作为 [`doc_sync.py`](../../../scripts/doc_sync.py) 的一部分运行，prek 的暂存运行与 CI 的全量运行无需新增接线即可强制执行；暂存运行宽容尚未写出的 `.zh.md` 链接目标，使并行翻译波次落地时不产生跨波链接竞态，而严格的终态归 CI 全量运行。

**闸门与规则之家随之适配。** `specs/index.md` 配对出带翻译行摘要的 [`index.zh.md`](../../../specs/index.zh.md)，[`verify_specs_index.py`](../../../scripts/verify_specs_index.py) 校验每对一行 —— 剥去两种语言后缀的词干 —— 出现在两份索引中，并跳过索引彼此之间的切换行链接。[MRFC README](../README.zh.md) 写明强制契约 —— 每条记录都是文件对，[`verify_mrfc_format.py`](../../../scripts/verify_mrfc_format.py) 以英文机器 token 与节标题校验两份骨架，中文标题旁的 ASCII 锚点使 fragment 链接与语言无关。`verify_md_*.py` 的模式已将 `README_zh.md` 换为 `README.zh.md`（目录 glob 本已覆盖 `.zh.md` 孪生）；docs 工作流的路径过滤跟随新清单。契约本身是 [`docs/AGENTS.md`](../../../docs/AGENTS.md) 的规则 7：同等权威、切换行、链接语言区、代码块同一、ASCII 锚点、中西文之间半角空格、全角标点，以及行内承载的种子术语 —— 600 词上限使行内列表优于表格，独立术语文件留待信号点触发 —— 另附"文档 `.zh` 对应应用的 `zh-Hans` locale"的说明（应用自身 i18n 不动）。根 `AGENTS.md` 以一行指向规则 7；`doc-standards`、`writing-mrfcs`、`commit` 三个 skill 承载工作流。

**语料在单个变更集内归一。** 逐文件语言审计将范围内每页归入三类：纯英文（撰写镜像）、中文原生（英文内容写入 `.md`，中文清洗入 `.zh.md`）、混合（妥善拆分为两者） —— 约一百个文件上的四十九对，以分层提交落地（机制、核心对、MRFC 存量、随后 `specs/` 分波），每次提交暂存完整的文件对。历史记录同样翻译：没有双类语料，中文读者保有决策史。`PRINCIPLES.md` 亦配对；其既定删除会把文件对一并带走。归一顺手纠正了遭遇的过时事实 —— `pkg/apierr` 路径、与 locale 文件不再相符的 delivery-queue UI 字符串、更新到当前代码的 frontend build/i18n 围栏 —— 各记录在其所在层的提交里。

**机制随信号生长，不投机建设。** 暂缓项各记其复活触发条件：逐对 blob-hash 边车与新鲜度记录（单侧散文编辑反复漏过评审）；PR 级 co-change diff 闸门（文件对单侧变更事故）；覆盖表格与列表的完整结构签名（结构性漂移）；带禁译列的独立术语文件（评审中出现术语翻译不一致）；翻译 briefing、专门翻译 skill 或 MT 管线（整文件翻译成为常态）；配对元数据的 merge driver（参考项目规模）。参考项目的机器 —— 约 5,300 行 TypeScript 服务约 1,150 对 —— 是这条路尽头的存档天花板，不是起点。

## Alternatives considered

**整包移植参考项目的双语机器** —— `.md` + `.zh.md` + `.i18n.yaml` blob-hash 一致性记录、覆盖标题/表格/列表/链接的结构签名、翻译 briefing 生成器、封存档案、git merge driver。它输在：每对约 4.6 行工具对一个五十对的语料、每次文档编辑都要重录边车的税负，以及 Python stdlib 闸门套件刻意规避的依赖形态（Node 工具链、mdast）。本记录按设计执行 2026-08-21 的触发条款，不推翻该裁决。

**只有规则、没有闸门。** 在 `docs/AGENTS.md` 里用散文请求作者保持文件对同步。它输在：失守的漂移正是 2026-08-21 各闸门存在的理由，标准中的每条规则都点名强制它的闸门 —— 没有机器的双语规则会用两种语言重演那次失败。

**语言目录（`en/` 与 `zh/` 子树）。** 它输在：并置的文件对让镜像留在同一个 diff、同一场评审里，链接靠后缀即可切换语言；目录把每个页面与其自身的维护拆开，把链接图变成需要分别守诚的两棵树。

**英文为主、中文为译。** 它输在：需求就是同等权威，而语料早已否定主从 —— 数个页面中文原生，正因为中文是维护者更强的语言；主从规则会贬低更可能被先行撰写的一侧，并把既存的中文页重新归档为并不存在的译本之衍生物。

**沿用 `README_zh.md` 命名。** 它输在：`.zh.md` 已是既定约定 —— 树 README、首条记录镜像、格式闸门的文件名正则都在用；一个概念两种拼写本身就是漂移。

**存量记录豁免（grandfather）** —— 历史记录与 `PRINCIPLES.md` 保持单语，新文件才成对。它输在：双类语料让中文那一半为有界的一次性成本失去决策史，且随着记录累积，这个豁免类别会比它的理由活得更久。

## Consequences

全语料在全部七道闸门下运行绿：49 对文件通过配对完整性、切换行、链接语言区、结构对等与纯度校验；30 个 spec 对在两份索引中各有一行；24 个 MRFC 文件符合格式。暂存闸门按设计演练过 —— 从一侧删去一个标题使配对检查变红，演练后还原 —— 而单侧散文编辑仍超出本级机器能力，已点名为首个信号点升级项。实施揭示了两条随本变更修复的闸门事实：`verify_specs_index.py` 的词干必须剥去两种语言后缀，否则两份索引永不相等；索引的语言切换行不得计为 spec 行。归一还顺手修复了过时事实（`pkg/apierr` → `internal/apierr`、delivery-queue 的 locale 字符串、frontend build/i18n 围栏），并标记了一个超出本决策范围的后续项：`specs/frontend/testing.md` 携带与仓库不符的 e2e 事实（不存在的 `frontend/tests/` 目录与 `pnpm test:e2e` 脚本），已如实镜像进两种语言，待该文件对上的内容修复。接受的代价：没有闸门检查语义 —— 结构对等证明形状而非实质，潦草的镜像照样绿，语义由评审承载；种子术语在 600 词上限拒绝表格后行内落于规则 7（`docs/AGENTS.md` 落在 575 词 —— 先压缩成立，未上调上限）；此后每次文档变更都携带双写义务，与一个比参考项目小一个数量级的语料相称；约一百文件的归一以抽查加闸门评审，这是该规模翻译被接受的评审方式。
