# MRFC: Cache and purge observability metrics

Status: proposed

[English](2026-09-03-cache-purge-observability.md) | 中文

## Problem

读路径上的两个缓存机制——进程内渲染缓存(ristretto,`backend/internal/service/post/cache.go`)与尽力而为的 Cloudflare cache-tag purge(`backend/internal/service/post/purger.go`)——目前不产生任何指标。读者反馈内容未失效删除、或源站负载异常时,要回答"缓存到底生效了吗",只能逐请求从响应头推断或查询数据库;可观测文件(`metrics-*.jsonl`、`app-*.jsonl`)里既没有缓存命中/未命中的记录,也没有 purge 的发起与结果(成功、失败、跳过)。渲染缓存还可被配置关闭,而现有信号无法区分"未开启"与"开启但无效"。可观测规范的指标清单中没有这两项,运维方没有任何成文依据来判断缓存有效性——包括源站信号与 Cloudflare 面向访客的 `CF-Cache-Status` 头之间的关系。

## Proposal

在既有 OTel 指标通道(meter `markpost`,每 60 秒导出至 `metrics-*.jsonl`)新增五个业务计数器,遵循已上线代码的命名与结构——semconv 风格点分命名、每个结果一个计数器、无属性:

| Metric                             | Type    | Emitted when                                                       |
| ---------------------------------- | ------- | ------------------------------------------------------------------ |
| `markpost.render_cache.hit_total`  | counter | 渲染请求由渲染缓存直接服务                                          |
| `markpost.render_cache.miss_total` | counter | 渲染请求未命中,进入渲染/singleflight 路径                           |
| `markpost.cdn.purge_success_total` | counter | cache-tag purge 请求以 HTTP < 300 完成                              |
| `markpost.cdn.purge_failure_total` | counter | purge 尝试失败(marshal、请求构造、传输错误或 HTTP ≥ 300)            |
| `markpost.cdn.purge_skipped_total` | counter | purge 未发起(no-op purger 或凭据未配置)                            |

埋点位置决策:

- **命中/未命中在请求决策点计数**——`RenderPostHTML` 的快路径 `cache.Get`——而非 `ristrettoCache.Get` 内部。`singleflight.Do` 内部的二次检查每次冷未命中都会多执行一次 `Get`;在那里计数会使冷流量下的未命中数虚高约 2 倍。未命中的语义是"该请求进入了 singleflight 路径";极少数竞态下二次检查命中并发 leader 刚填充的条目,接受为少量低估的命中。HTML 与 raw 两个变体共用计数器(不设变体属性)。
- **缓存关闭(`[render] enabled = false`)时仍计未命中**:`noopCache` 永远未命中,命中率如实显示 0%——指标对"缓存是否生效"的回答覆盖关闭场景。
- **purge 结果分类**遵循现有控制流:no-op 路径记 `skipped`,marshal/构造/传输错误与 HTTP ≥ 300 记 `failure`,其余记 `success`。purge 尝试次数可由 success + failure 推得,不设独立的发起计数器。
- **purge 日志从 `log.Printf` 迁移到 `slog`**,携带结构化字段(`qid`、HTTP 状态或错误),使 purger 对齐可观测规范的日志规则。
- 指标经既有的服务内窄接口 `Metrics` 交付(`post.Service` 的 `WithMetrics` 注入 + `noopMetrics` 回退),扩展新方法;`*observability.Metrics` 实现之。

文档:五个指标行进入 [`specs/backend/observability.zh.md`](../../../specs/backend/observability.zh.md) 的指标清单,同时把已漂移的 `markpost.auth.login_total` 行修正为已上线的分立计数器实态(`login_success_total`/`login_failure_total`);[`specs/backend/caching.zh.md`](../../../specs/backend/caching.zh.md) 新增一小节,讲解如何对照 `CF-Cache-Status` 解读源站缓存指标(边缘 HIT/MISS/EXPIRED 与源站命中率的关系;边缘吸收了绝大多数读流量,因此"源站流量低 + 命中率高"是健康稳态而非故障)。两份规范的双语孪生文件在同一变更中同步更新。

## Alternatives considered

**在 `Get` 内部计数的计量版 `renderCache` 包装器。** 机制上最省事(一个包装器覆盖两个实现),但它同样会计入 singleflight 的二次检查 `Get`——每次冷未命中记两次——而排除二次检查所需的标记或上下文,恰恰重新引入了包装器本想隐藏的调用点知识。在请求决策点计数是每结果一行、且语义精确。

**单计数器 + 结果属性(如 `markpost.cdn.purge_total` 配 `outcome=success|failure|skipped`)。** 指标更少,也是规范 login 行曾暗示的形态,但已上线的每个业务指标都是按结果分立的计数器(`login_success_total`、`delivery.failed_total` 等);为两个相邻机制引入第二种风格,会让 jq 查询在整个指标清单内不一致。而暗示标签风格的那行规范,本身就是相对已上线代码漂移的结果。

**导出 ristretto 内置 `Metrics`(命中率、内部计数器)。** 免费获得细节,但那些比率是按 `Get` 的内部记账,不是请求级有效性——不回答"多少比例的渲染请求免于渲染",且把运维方绑定到 ristretto 的内部词汇表。

**独立的 purge 发起计数器。** 冗余:尝试次数恰为 success + failure;skipped 的定义就是未尝试。聚合查询保持简单。

## Acceptance criteria

- 渲染与删除 post 后,五个计数器出现在 `metrics-*.jsonl`。
- 同一 post 冷后热的两次渲染恰好产生一次未命中、一次命中;缓存关闭时只产生未命中。
- purge 结果按规范分类:HTTP < 300 记 success;各错误分支记 failure;no-op/未配置路径记 skipped。
- purger 日志为 slog 结构化,携带 `qid`(及 status/error)字段。
- 单测覆盖埋点:计数器经真实 meter SDK reader 递增;`noopMetrics` 回退保持无指标的测试不受影响。
- `specs/backend/observability.md` 载有五个指标行(及修正后的 login 行);`specs/backend/caching.md` 载有 `CF-Cache-Status` 解读指南;两份规范的双语孪生文件同变更更新。
- 回答"渲染缓存是否生效"与"purge 是否发起且成功"只需可观测文件与 `jq`——无需访问数据库。

## Risks

- **埋点位置回归**(如未来重构移动快路径 `Get`)会悄然扭曲命中/未命中口径;冷后热的单测钉死该语义。
- **基数与体积**:五个计数器、无属性——各一条时间序列,每 60 秒导出一次;metrics 文件增量可忽略。
- **未配置 Cloudflare 的自托管实例**只会产生 `purge_skipped_total`;这是预期稳态,不应读作故障(已写入 caching 规范的解读指南)。
- **超出 issue 严格范围的规范修正**(login 行漂移修复)触碰了本 issue 未要求的一行;将在 PR 正文显式声明,便于评审否决。
