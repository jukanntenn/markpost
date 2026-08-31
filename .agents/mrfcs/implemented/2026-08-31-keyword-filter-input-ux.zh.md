# MRFC: 关键词过滤输入 UX——自然语言实时反馈

Status: implemented

[English](2026-08-31-keyword-filter-input-ux.md) | 中文

## Problem

关键词过滤字段要求用户从一行压缩口诀学会七个运算符的布尔文法——`语法：逗号/竖线=或，&=且，!=非，""=精确短语，()=分组`——而且口诀本身是错的：`""=精确短语` 误述了引号语义（引号只是让运算符字符变字面内容，匹配仍是子串；`""` 是双写的字面引号）。解析失败直接透出内部英文报错（`unexpected comma`、`expected ')', got eof`）。`[?]` 帮助的 hover 从未生效：`group-hover:block` 没有任何 `group` 祖先，只剩点击开关。与此同时规范承诺过更好的东西：[关键词过滤规范](../../../specs/backend/keyword-filter.zh.md)把实时解析预览指定为 CJK IME 用户全角 `，` 问题的缓解手段，其 § 前端还文档化了 `describeFilter` 与反馈组件——但七月 Dialog 重构（D5.6）"保留语法校验，移除语义解释"，规范没跟着改。用户看不到表达式的效果，规范与代码的契约也断了。

## Decision

`src/components/delivery/DeliveryChannelDialog.tsx` 的关键词字段是三层反馈回路，由 `src/lib/keyword-filter.ts` 支撑：

**自然语言实时预览（cron-guru 式）。** `describeFilter(node, phrasebook)` 把 AST 渲染为 locale 句子——zh-Hans：`标题包含「prod」且（包含「error」或包含「warning」）且不包含「debug」时推送`；en：`Delivers when the title contains “prod” and (contains “error” or contains “warning”) and does not contain “debug”`。渲染器只管结构——按优先级加括号（AND 内包 OR、OR 内包 AND 都加括号）、双重否定折叠、超过 30 码点的关键词截断——连接词由各语言通过 `Phrasebook` 接口提供；句子由四个 locale（en、zh-Hans、zh-Hant、ja）的 `keywordsPreviewSentence` / `keywordsPreviewAlways` 消息拼装。空表达式预览为"推送全部文章"。规范的全角缓解由此兑现：`监控，告警` 一眼可见是单个关键词。

**结构化本地化报错。** `FilterParseError` 携带 `{ code, pos, token }`——四个错误码之一（`unterminated_quote`、`unexpected_token`、`missing_rparen`、`empty_keyword`）、1 基码点位置（星形字符算一个；输入耗尽为 null）、肇事 token 种类。Dialog 映射为 locale 文案，如 `语法错误：第 3 个字符附近不应出现「,」，这里需要一个关键词或「(」`；内部 token 名不再到达用户。

**带可点击示例的语法帮助弹出面板。** `[?]` 触发 `src/components/ui/popover.tsx` 面板（base-ui Popover，键盘可达，按 Dialog 内弹层惯例 `z-[100]`），承载运算符速查表、三个陷阱（空格是内容、全角标点是字面、留空匹配全部）、五个示例 chip。点击示例填入输入框并关闭面板，让预览立即可见。坏掉的 hover span 已移除。

规范的 § 前端重写为与此一致的现状，终结 D5.6 重构引入的漂移。

## Alternatives considered

**维持只做校验（D5.6 姿态）。** 最便宜，但规范早已承诺实时预览作为 CJK 全角缓解——没有它，`监控，告警` 会静默匹配出用户根本不想的东西，且漂移依旧。

**条件清单式渲染**（顶层层 AND/OR 拆成 bullet 条件）。复合 AND 链读起来最轻松，但在 `sm:max-w-lg` 的 Dialog 里占多行，对 `mark, post` 这种简单场景过度铺陈。选择单句式——cron-guru 对齐，简单表达式退化为自然的短句。

**复活旧版 `describeFilter`（重排规范化表达式，`a | (b & c)`）。** 移植成本最低，但用同一记号重排表达式什么也没解释——诉求是自然语言，不是漂亮打印。

**报错在服务端本地化。** 后端 400 消息需要 API 层的 locale 感知；表单在发请求前就用同一文法做了客户端校验，客户端才是用户遇到错误的地方。直连 API 的使用者继续看到带字节位置的英文消息。

**帮助用 tooltip 而非 popover。** 一行 hover 装不下速查表加示例，且 hover-only 在触屏上不可用；popover 由点击/键盘驱动。

## Consequences

用户如今可以看着预览朗读效果来写表达式、在指向具体字符位置的母语报错里恢复、并通过就地试写示例从速查表学会文法。代价：四个 locale 文件各增加约 28 个消息键；未来第五语言必须实现 `Phrasebook` 连接词；TS 渲染器必须与 Go 文法保持语义对齐——这是 `keyword-filter.ts` 既有的移植契约（后端权威、前端镜像）的延伸。验证随变更一同落地：`keyword-filter.test.ts` 的逐 locale 渲染快照（en、zh-Hans）与结构化错误位置、Dialog 组件测试覆盖预览/报错/帮助/chip、`e2e/tests/delivery-channel.spec.ts` 的预览断言、以及 zh-Hans 全流程的交互验证（复合预览、全角单关键词预览、本地化报错、面板填入即关闭）。
