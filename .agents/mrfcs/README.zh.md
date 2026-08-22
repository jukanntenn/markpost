# Markpost Request for Comments (MRFC)

[English](README.md) | 中文

MRFC 是 markpost 的 RFC:持久留存的提案与决策记录 —— *为什么*、*放弃了什么*,以及代码和 specs 无法承载的部分。Specs 描述当前状态;MRFC 解释该状态为何如此。

<a id="layout-and-naming"></a>
## 布局与命名

每个 MRFC 位于 `.agents/mrfcs/{lifecycle}/yyyy-mm-dd-topic-title.md`。日期是主题首次提出的时间(以 git 历史为准)。生命周期目录树即清单 —— 浏览它或 grep 仓库;没有需要维护的索引文件。

- **`proposed/`** —— 实现前接受评审的提案。尚未构建,或只部分构建。
- **`implemented/`** —— 决策已发布。文件以现在时记录决定了什么、拒绝了什么。当代码随后重命名文件或更改默认值时,在同一变更中更新 MRFC 的事实(路径、名称、结构)—— 但绝不把 MRFC 改写成另一个决策;用新 MRFC 替代并互链两者。
- **`rejected/`** —— 提案经过考虑后被拒绝。仅当其理由能防止一个有诱惑力的错误时保留;否则删除。

MRFC 之间的交叉引用使用相对 Markdown 链接,绝非裸散文,这样 [`verify_md_links`](../../scripts/verify_md_links.py) 能检查它们,且在目录间移动后依然有效。

<a id="when-to-write-one"></a>
## 何时撰写

每个非平凡变更都在同一 PR 中新增或更新至少一个 MRFC。当变更改变行为、架构、跨文件契约、工具链、测试策略、磁盘或线上格式,或维护者可能合理重审的任何内容时,即为非平凡。纯机械或局部编辑豁免。更新已拥有该决策的 MRFC 即满足规则 —— 不要创建重复;先 grep `.agents/mrfcs/` 查找主题。

<a id="the-file-format"></a>
## 文件格式

头部块严格为:

```markdown
# MRFC: <title>

Status: <status>
```

`Status:` 值必须与所在目录一致,取三种形式之一:`proposed`、`implemented`、或 `rejected — <why, in one line>`(拒绝理由是读者前来的事实)。正文以 `## Problem` 开篇,须脱离解决方案独立成立。

`implemented/` 接续 `## Decision`(现在时,发布了什么)… `## Alternatives considered` … `## Consequences`。提案期标题 —— `## Proposal`、`## Plan`、`## Migration plan`、`## Acceptance criteria` —— 在此被格式门禁拒绝。

`proposed/` 接续 `## Proposal` … `## Alternatives considered` … `## Acceptance criteria` … `## Risks`。工作未构建时,提案可以用将来时表述。

`rejected/` 保持其提案期的全部章节,冻结;结论写在 `Status:` 行。

每条记录都是双语文档对:英文主文件旁携带 `.zh.md` 镜像 —— 骨架相同,机器 token 与节标题保持英文 —— 且一对文件一同更新([bilingual-pairs MRFC](./proposed/2026-08-22-bilingual-documentation-pairs.zh.md))。

**`## Alternatives considered` 在每个 MRFC 中强制存在** —— 每个真实替代方案一个加粗引导的段落及其落败原因。没有记录手下败将的决策会招致重新争论,这正是 MRFC 要防止的失败。替代方案按当时论证的原样记录,绝不在事后虚构。

在生命周期目录之间移动文件意味着在同一变更中更新其 `Status:` 行并重新满足目标目录的骨架:`proposed/` → `implemented/` 把 `## Proposal` 改写为现在时的 `## Decision`,并把 `## Acceptance criteria`/`## Risks` 折叠进 `## Consequences`;`proposed/` → `rejected/` 只在 `Status:` 上添加理由并冻结文件。

[`verify_mrfc_format.py`](../../scripts/verify_mrfc_format.py) 强制执行以上全部;它作为 [`doc_sync.py`](../../scripts/doc_sync.py) 的一部分运行。
