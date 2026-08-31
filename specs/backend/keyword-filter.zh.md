# 关键词过滤表达式

[English](keyword-filter.md) | 中文

本文档规定投递渠道用来判定是否推送文章的关键词过滤表达式语法。它是语法、匹配语义、校验规则，以及 Go 后端（求值器）与 TypeScript 前端（实时预览）之间跨语言契约的权威参考。

<a id="overview"></a>

## 概述

每个投递渠道有一个 `keywords` 文本字段，持有**过滤表达式**。文章投递时，表达式针对文章**标题**求值（子串匹配）。标题满足表达式则推送到该渠道，否则跳过。空表达式匹配每一篇文章（总是投递）。

表达式语言是标准布尔代数（带括号的 OR / AND / NOT），设计目标：

- 常见场景零学习成本：`a, b, c`（OR）、`a & b`（AND）、`!a`（NOT）。
- 空格是关键词内容的一部分——`key word 1` 这类多词短语**无需引号**。
- 恰好七个运算符字符；其余一切字符都是字面关键词内容。
- 畸形表达式**在写入时拒绝**，绝不静默接受。存储的表达式不可能产生歧义或意外匹配。

<a id="grammar"></a>

## 语法

```
expr   := or
or     := and  ( ("," | "|") and )*      # OR — lowest precedence, left-associative
and    := not  ( "&" not )*              # AND — left-associative
not    := "!" not | factor               # NOT — prefix, right-associative
factor := KEYWORD | "(" expr ")"         # terminal or grouped sub-expression
```

**优先级**（从紧到松）：`!` > `&` > `,`/`|`。括号可覆盖。这是人尽皆知的布尔优先级——没有自定义规则。

因此 `a | b & c` 解析为 `a | (b & c)`，`!a & b` 解析为 `(!a) & b`。

<a id="operators"></a>

## 运算符

恰好七个 ASCII 字符是运算符：

| 字符    | 含义        | 备注                             |
| ------- | ----------- | -------------------------------- |
| `,`     | OR          | 等价于 `\|`                      |
| `\|`    | OR          | 等价于 `,`                       |
| `&`     | AND         |                                  |
| `!`     | NOT（前缀） | 右结合；`!!a == a`、`!!!a == !a` |
| `(` `)` | 分组        | 覆盖优先级；可任意嵌套           |
| `"`     | 引号        | 使运算符字符变为字面量；见下文   |

**其余一切字符都是字面关键词内容。**包括字母、数字、CJK、emoji，以及上述七者之外的所有标点：`+ / \ : ; @ # $ % ^ = ~ [ ] { } < > ? * '` 等等。这类关键词无需引号。

<a id="lexing-rules"></a>

## 词法规则

<a id="keywords-model-2-spaces-are-content"></a>

### 关键词（模型 2：空格是内容）

未加引号的关键词是**不含七个运算符字节的最长连续字符串**。读出后修剪首尾空白（含 U+3000 表意空格）。内部空白保留并参与匹配。

- `key word 1` → 一个关键词 `key word 1`。
- `C++`、`a/b`、`a\b`、`🚀go`、`错误` → 各是单个关键词，无需引号。
- `a & b` → 关键词 `a`、运算符 `&`、关键词 `b`。

<a id="quoting"></a>

### 引号 `"..."`

引号段把引号之间的一切当作字面内容——七个运算符在引号内全部失去含义。引号内的首尾空格**保留**。

- `"a,b"` → 关键词 `a,b`。
- `"a & b"` → 关键词 `a & b`。
- `" error "` → 关键词 `error`（首尾空格保留）。

引号段内的字面双引号通过**加倍**书写：`""`。

- `"say ""hi"""` → 关键词 `say "hi"`。
- `""""` → 关键词 `"`（单个双引号）。

反斜杠 `\` **永远是字面量**（没有反斜杠转义）。`a\b` 是关键词 `a\b`，在引号内也一样。

<a id="when-quotes-are-required"></a>

### 何时需要引号

只有当关键词含有 `, | & ! ( ) "` 之一，或需要保留首尾空白时，引号才是必需的。其余情况引号可选——`"key word"` 与 `key word` 完全相同。

<a id="whitespace"></a>

### 空白

运算符周围的空白被忽略：`a & b` ≡ `a&b`。两个相邻因子之间**没有运算符**是语法错误（见下文「拒绝」）——组合因子的唯一方式是显式运算符。这是让语法无歧义的关键性质。

<a id="matching-semantics"></a>

## 匹配语义

每个关键词对标题做**大小写不敏感的子串**比较：

```
keyword'  = strings.ToLower(norm.NFC(keyword))
title'    = strings.ToLower(norm.NFC(title))
match     = strings.Contains(title', keyword')
```

| 维度              | 规则                                                   |
| ----------------- | ------------------------------------------------------ |
| 匹配类型          | 子串（引号与未引号都是子串）                           |
| 大小写            | 不敏感——经 `strings.ToLower` 的 Unicode 默认大小写折叠 |
| 规范化            | 关键词与标题都比较前规范化为 **Unicode NFC**           |
| 匹配字段          | **仅标题**（`post.DeliveryJob.Title`）                 |
| 空 / 纯空白表达式 | 匹配一切（总是投递）                                   |
| 正则 / 通配符     | 不支持（`*`、`?` 等是字面字符）                        |

**NFC 规范化**保证韩文、越南语和带变音符的拉丁文系，无论关键词或标题以预组合（NFC）还是分解（NFD）形式到达，都能正确匹配。例如韩文 `오류`（2 个 NFC rune）与其 4-rune 的 NFD 展开字节不同，但被视为相等。

**子串说明**：`"key word"` 匹配包含 `the key word here` 的标题（短语逐字出现），不匹配 `the keyword here`（缺空格）。引号短语仍是_子串_匹配，不是整标题相等。

<a id="validation-and-error-handling"></a>

## 校验与错误处理

表达式在 service 层**写入时**校验：

- `Create` 与 `Update` 通过 `filter.Compile` 编译表达式。解析失败返回 HTTP `400` 与 `ErrValidation` service error；message 内嵌字节位置，例如 `invalid keywords expression: filter: parse error at pos 5: unexpected ','`。
- 投递时表达式重新编译（它很短、微秒级开销；设计上不做缓存）。存储表达式万一无效的渠道被跳过并记一行日志，而不是让投递循环崩溃。

`UpdateChannelParams.Keywords` 是 `*string`，用于区分「字段未提供」（保持不变）与「显式清除」（设为空 → 匹配一切）。由此清除 keywords 的更新可行。

<a id="rejections-malformed-expressions"></a>

## 拒绝（畸形表达式）

以下全部以 `ParseError` 拒绝。无效输入到不了存储，所以存储的表达式永远不可能产生歧义匹配。

| 类别             | 例子                                      |
| ---------------- | ----------------------------------------- |
| 空操作数         | `a,,b`、`a && b`、`a &`、`& a`、`,`、`a,` |
| 运算符缺操作数   | `!`、`&`、`\|`、`a \| \| b`               |
| 未闭合 / 空分组  | `(a`、`a)`、`()`、`(a,)`、`)(a`           |
| 未终止引号       | `"abc`、`"""`                             |
| 相邻因子无运算符 | `a (b)`、`(a)(b)`、`a"b"`                 |
| 空关键词         | `""`、`a & ""`、`(), a`                   |
| 仅运算符         | `& \| ,`、`! &`、`(!)`                    |

空关键词（`""`）被拒绝，因为空字符串是每个标题的子串、会匹配一切——一个「意外意外」的来源。

<a id="known-limitations"></a>

## 已知限制

这些是有意为之、已记录的限制，而非 bug：

- **德语 `ß`**：`strings.ToLower("Straße") == "straße"`，**不**等于 `strasse`。关键词 `Straße` 不匹配全大写标题 `STRASSE`。完整大小写折叠（ß → ss）刻意不应用，以保持行为与 locale 无关且可预测。
- **土耳其语 `İ` / `ı`**：不做 locale 折叠；使用 Unicode 默认大小写。
- **Emoji 序列**（`👨‍👩‍👧` 这类 ZWJ 序列、肤色修饰符、旗帜区域指示符）在 NFC 之后**逐字节**匹配。不同的 emoji 组合（如有 / 无 ZWJ）被视为不同关键词。

<a id="full-width-characters"></a>

## 全角字符

七个运算符是 **ASCII 专用**。它们的全角变体（`， ＆ ｜ ！ （ ）`）是**字面关键词内容**，不是运算符——这个刻意的选择保持语法无歧义（无需枚举「哪些全角变体算数」）。代价由 UI 承接：中日 IME 常输出全角标点，所以前端表单展示实时解析预览，用户立刻看到 `监控，告警` 是单个关键词而非 OR。

<a id="worked-examples"></a>

## 示例

| #   | 输入                               | 含义                                              |
| --- | ---------------------------------- | ------------------------------------------------- |
| 1   | `alert`                            | 标题含 `alert`                                    |
| 2   | `a, b, c`                          | `a` OR `b` OR `c`                                 |
| 3   | `a \| b \| c`                      | 同 #2（`\|` ≡ `,`）                               |
| 4   | `a & b & c`                        | `a` AND `b` AND `c`                               |
| 5   | `!test`                            | 标题不含 `test`                                   |
| 6   | `prod & (error, warning) & !debug` | `prod` AND（`error` OR `warning`）AND NOT `debug` |
| 7   | `key word 1`                       | 关键词 `key word 1`（多词，无需引号）             |
| 8   | `! key word`                       | NOT 关键词 `key word`                             |
| 9   | `C++`、`a/b`、`🚀go`               | 各是单个关键词，无需引号                          |
| 10  | `"a,b"`                            | 关键词 `a,b`                                      |
| 11  | `"a & b"`                          | 关键词 `a & b`                                    |
| 12  | `"say ""hi"""`                     | 关键词 `say "hi"`                                 |
| 13  | `""""`                             | 关键词 `"`                                        |
| 14  | `!!a`                              | `a`（双重否定）                                   |

<a id="implementation"></a>

## 实现

<a id="backend"></a>

### 后端

全部逻辑位于 `internal/service/delivery/filter/`（约 250 行，零新增依赖——`golang.org/x/text/unicode/norm` 本就是直接依赖）：

| 文件           | 职责                                                                                                                     |
| -------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `lexer.go`     | 分词器：七个运算符、裸 / 引号关键词读取、`""` 加倍、空白跳过                                                             |
| `ast.go`       | AST 节点类型：`orNode`、`andNode`、`notNode`、`keywordNode`、`alwaysTrueNode`                                            |
| `parser.go`    | 遵循优先级文法的递归下降解析器；panic 转换为 `*ParseError{Pos, Msg}`                                                     |
| `evaluator.go` | `normalizeMatch`（NFC + ToLower）与 `containsSubstr`                                                                     |
| `filter.go`    | 公开 API：`Compile(expr) (*Matcher, error)`、`MustCompile(expr) *Matcher`、`(*Matcher).Match(title) bool`、`*ParseError` |

Matcher 在 `internal/service/delivery/post_delivery.go` 被调用。检查被提升到 `switch channel.Kind` **之上**，所有渠道类型共享同一个过滤器。

<a id="frontend"></a>

### 前端

`src/lib/keyword-filter.ts` 是该文法的 TypeScript 移植，**仅用于表单实时校验与预览**——后端仍是权威。它暴露：

- `compileKeywordFilter(expr)` → `{ node, error }`，其中 `error` 是结构化的：`code`（`unterminated_quote` / `unexpected_token` / `missing_rparen` / `empty_keyword` 之一）、指向出错字符的 1 基码点位置 `pos`（输入耗尽时为 null）、以及肇事的 `token` 种类。表单据此映射本地化消息，例如 `语法错误：第 3 个字符附近不应出现「,」，这里需要一个关键词或「(」`。
- `describeFilter(node, phrasebook)` → 各语言的自然语言子句，例如 `包含「prod」且（包含「error」或包含「warning」）且不包含「debug」`。渲染器只管结构——按优先级加括号、双重否定折叠、超过 30 码点的关键词截断——连接词由各语言通过 `Phrasebook` 接口提供；完整句子由 `keywordsPreviewSentence` / `keywordsPreviewAlways` 消息（en、zh-Hans、zh-Hant、ja）拼装。

`src/components/delivery/DeliveryChannelDialog.tsx` 把关键词字段渲染成三层反馈回路：输入框下方的自然语言实时预览（cron-guru 式）、本地化的结构化解析报错、以及承载语法速查、空格/全角标点陷阱和可点击试写示例的 `[?]` 弹出面板（`src/components/ui/popover.tsx`）。

<a id="performance"></a>

### 性能

基准位于 `internal/service/delivery/filter/filter_bench_test.go`（AMD Ryzen 5 5600H）：

| 场景                                                                        | ns/op | allocs/op |
| --------------------------------------------------------------------------- | ----: | --------: |
| Compile，单关键词                                                           |  ~153 |         4 |
| Compile，复合（`prod & (error,warning,fatal) & !debug & !(staging,local)`） | ~1152 |        24 |
| Match（中等标题，命中）                                                     |  ~682 |     **0** |
| Match（长 4KB 标题，命中）                                                  | ~9658 |     **0** |
| Normalize（NFC + ToLower），长标题                                          | ~5549 |         0 |
| Compile + Match（中等标题）——每渠道投递的真实成本                           | ~1618 |        17 |

求值**零分配**；只有编译分配（与 AST 大小成正比）。典型渠道的每次投递成本约 1.6 µs。长标题的主导因素是标题规范化（NFC + ToLower），不是解析。

<a id="test-coverage"></a>

## 测试覆盖

测试位于 `internal/service/delivery/filter/`：

| 文件                          | 范围                                                                                                                                                                                            |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `filter_test.go`              | 语义、优先级、特殊字符 / 引号、空匹配一切、有效边界、无效拒绝                                                                                                                                   |
| `filter_multilingual_test.go` | 中文（无词边界）、日文（混排、长音、小假名）、韩文（NFC/NFD）、泰文、阿拉伯 / 希伯来（RTL）、西里尔（大小写）、德语（变音符、ß 限制）、拉丁变音符、emoji（ZWJ / 肤色 / 旗帜）、混排、全角字面化 |
| `fuzz_test.go`                | `FuzzCompile_NeverPanics`（130 万+ 次执行，无 panic）加五个布尔恒等性质测试：德摩根、双重否定、交换律、分配律、永真 / 永假                                                                      |

<a id="design-decisions"></a>

## 设计决策

记录于此，使理由与规范同在。

1. **标准布尔代数，而非 Ansible 式平面重排。**Ansible 重排 `&`/`!` 项而不是构建 AST，无法表达 `A & (B | C)`。标准优先级 + 括号是同时完备、无歧义、并契合多数用户既有直觉的唯一设计。
2. **空格即内容（模型 2）。**选择它而非「空格作分隔符」（搜索引擎式），因为它逐字匹配动机案例，并免去多词短语的频繁引号。代价——相邻因子需要显式运算符——正是无歧义的保证。
3. **`""` 加倍，而非 `\"` 转义。**加倍不引入转义字符，`\` 无条件是字面量（`C:\`、`a\b` 这类关键词处处可用）。CSV / SQL 先例。
4. **逗号现为 OR（破坏性变更）。**v0.1.0 之前的语义把逗号当作 AND；存储 `a, b` 再交给按新 OR 求值的求值器会静默放宽过滤。在 v0.1.0 作为已记录的破坏性变更接受（渠道功能当时全新，影响面小）。DB 列未变——变的只是其解释。
5. **NFC 规范化。**没有它，韩语 / 越南语 / 带重音拉丁语用户会遭遇静默的跨平台不匹配（关键词 NFC、标题 NFD）。需要 `golang.org/x/text`，本就是依赖。
6. **Unicode 默认大小写折叠（非完整折叠）。**保持行为与 locale 无关且可预测；`ß↔ss` 与土耳其语 `İ/ı` 是已记录的限制。
7. **仅 ASCII 运算符。**全角变体是字面内容，保持语法无歧义。前端实时预览为 CJK IME 用户承接 UX 成本。
