# MRFC: Documentation gates and the MRFC system

Status: implemented

[English](2026-08-21-documentation-gates-and-mrfc-system.md) | 中文

## Problem

Markdown 语料已漂离现实，却没有机械手段抓它：specs 与指南叙述变更史、描述已不存在的机制，交叉链接无声腐烂，spec 页面游离于索引之外，硬换行的散文在每个 diff 里制造重排噪音，而决策理由以 "Decision Record" / "Implementation Plan" 章节的形式活在 spec 页面内部 —— 对已发布工作的计划腔。单靠评审纪律没能让约五十个文件保持诚实，于是每个文档任务都从代码重新推导事实开始，然后才能相信一个字。

## Decision

标准活在唯一的家 [`docs/AGENTS.md`](../../../docs/AGENTS.md)：一张为每类事实指派单一归宿（README、AGENTS、PRINCIPLES、specs、docs、mrfcs、账本、skills）的层级图，外加五条写作规则，每条由 `scripts/` 里一个 stdlib-Python 闸门背书 —— [`verify_md_links.py`](../../../scripts/verify_md_links.py)（相对链接与 `#fragment` 锚点可解析）、[`verify_md_wrap.py`](../../../scripts/verify_md_wrap.py)（每段一个物理行）、[`verify_md_current.py`](../../../scripts/verify_md_current.py)（README/docs/specs 的当前状态散文）、[`verify_specs_index.py`](../../../scripts/verify_specs_index.py)（`specs/index.md` 双向完备）与 [`verify_mrfc_format.py`](../../../scripts/verify_mrfc_format.py) —— 在 [`doclib.py`](../../../scripts/doclib.py) 中共享掩码与 slug 助手，由 [`doc_sync.py`](../../../scripts/doc_sync.py) 聚合，后者按序运行它们、保持各自可独立运行、并把范围限定于给定的文件参数。prek 的 `doc-check` hook 只对已暂存的 Markdown 运行 `doc_sync`（prek 在 hook 运行期间贮藏未暂存的变更，更宽的扫描会看到一棵过时的树）；全量语料经 [`.github/workflows/docs.yml`](../../../.github/workflows/docs.yml) 在 CI 中运行 —— 这个工作流存在是因为 `lint.yml` 按 path 忽略 `*.md`。MRFC —— markpost 的 RFC：持久留存的提案与决策记录 —— 活在 `.agents/mrfcs/`，受 [MRFC README](../README.zh.md) 的生命周期与格式契约约束，树的落位与双文档契约由 [mrfcs-tree MRFC](./2026-08-22-agents-mrfcs-tree-and-contract-split.zh.md) 拥有：每个非平凡变更在同一 PR 中新增或更新一条。两个 agent 工作流日常承载这套系统 —— `.claude/skills/` 里的 `doc-standards` 与 `writing-mrfcs`，由 [`sync_agent_instructions.py`](../../../scripts/sync_agent_instructions.py) 镜像到 `.agents/skills/`。账本、`scripts/loadtest/` 报告、生成的 Swagger 与 agent 配置镜像按设计坐在闸门范围之外。

## Alternatives considered

**整包移植参考仓库的完整文档分类学。** deepseek-harness 把每一页配成双语（`.md` + `.zh.md` + i18n 清单），把笔记归类到 `.agents/notes` 下并带冻结档案与替代闸门，维护一个不变式系统、逐文件覆盖清单、词数预算、堆叠 PR 规则和一条跑在 mdast 上的 doc-typecheck 管线。它输在：markpost 的语料是单语且小一个数量级，没有根部 Node 工具链来跑 mdast 检查，而配对加清单维护会给每次编辑上税、在这个规模上什么也买不到 —— 部件只在触发信号出现时移植，绝不整包。触发信号已到来；配对层已在 [bilingual-pairs MRFC](../implemented/2026-08-22-bilingual-documentation-pairs.zh.md)中移植。

**闸门调度器。** 参考仓库把所有闸门路由经过一个依赖图运行器，带聚合模式、`needs`/`after` 边、分区和 `allowFailure`。它输在：五个顺序的 stdlib 闸门在几十个 Markdown 文件上几秒钟跑完，而一个调度器会成为树里最复杂的脚本，服务的调度需求 markpost 一个也没有。

## Consequences

文档回归如今在提交时和 CI 中机械地失败，理由有了归宿使 `specs/` 得以保持当前状态，整套机制是 Python-stdlib、零新依赖。规则约束它自己的引入 —— 发布这些闸门与 `mrfc/` 的那批变更本身就是一次非平凡的流程变更，落地时却没有它的记录；本文件就是那条记录。它接受的代价：每个非平凡变更携带一条新增或更新的 MRFC，每个新 spec 文件在同一变更中需要它的 `specs/index.md` 行，闸门变红时改的是文档而非闸门（闸门变更与促成它的文档需求同变更发布，并在此说明），且因为 pre-commit hook 只检查已暂存的文件，全量语料的真相活在 CI 里。
