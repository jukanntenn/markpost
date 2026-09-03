# MRFC: Cache and purge observability metrics

Status: implemented

[English](2026-09-03-cache-purge-observability.md) | 中文

## Problem

读路径上的两个缓存机制——进程内渲染缓存(ristretto,`backend/internal/service/post/cache.go`)与尽力而为的 Cloudflare cache-tag purge(`backend/internal/service/post/purger.go`)——此前不产生任何指标。读者反馈内容未失效删除、或源站负载异常时,要回答"缓存到底生效了吗",只能逐请求从响应头推断或查询数据库;可观测文件(`metrics-*.jsonl`、`app-*.jsonl`)里既没有缓存命中/未命中的记录,也没有 purge 的发起与结果(成功、失败、跳过)。渲染缓存还可被配置关闭,而现有信号无法区分"未开启"与"开启但无效"。可观测规范的指标清单中没有这两项,运维方没有任何成文依据来判断缓存有效性——包括源站信号与 Cloudflare 面向访客的 `CF-Cache-Status` 头之间的关系。

## Decision

五个业务计数器、无属性、每个结果一个,运行在既有 OTel 指标通道(meter `markpost`,每 60 秒导出至 `metrics-*.jsonl`),遵循已上线代码的命名与结构——semconv 风格点分命名(`backend/internal/observability/metrics.go`):

| Metric                             | Type    | Counts                                                     |
| ---------------------------------- | ------- | ---------------------------------------------------------- |
| `markpost.render_cache.hit_total`  | counter | 渲染请求由渲染缓存直接服务                                  |
| `markpost.render_cache.miss_total` | counter | 渲染请求未命中,进入渲染/singleflight 路径                   |
| `markpost.cdn.purge_success_total` | counter | cache-tag purge 请求以 HTTP < 300 完成                      |
| `markpost.cdn.purge_failure_total` | counter | purge 尝试失败(marshal、请求构造、传输错误或 HTTP ≥ 300)    |
| `markpost.cdn.purge_skipped_total` | counter | purge 未发起(no-op purger 或凭据未配置)                     |

埋点位置:

- **命中/未命中在请求决策点计数**——`RenderPostHTML` 与 `GetPostMarkdown` 的快路径 `cache.Get`(`backend/internal/service/post/post.go`),仿照 `logger()` 先例经 nil 安全的 `recorder()` 访问器——而非 `ristrettoCache.Get` 内部。`singleflight.Do` 内部的二次检查每次冷未命中都会多执行一次 `Get`;在那里计数会使冷流量下的未命中数虚高约 2 倍。未命中的语义是"该请求进入了 singleflight 路径";极少数竞态下二次检查命中并发 leader 刚填充的条目,接受为少量低估的命中。HTML 与 raw 两个变体共用计数器(不设变体属性)。
- **缓存关闭(`[render] enabled = false`)时仍计未命中**:`noopCache` 永远未命中,命中率如实显示 0%——指标对"缓存是否生效"的回答覆盖关闭场景。
- **purge 结果分类**(`backend/internal/service/post/purger.go`)遵循控制流:no-op 路径记 `skipped`,marshal/构造/传输错误与 HTTP ≥ 300 记 `failure`,其余记 `success`。purge 尝试次数可由 success + failure 推得,不设独立的发起计数器。purge 日志用 `slog` 结构化字段(`qid`、HTTP 状态或错误)替代 `log.Printf`。
- 指标经服务内窄接口 `Metrics` 交付(`post.Service` 的 `WithMetrics` 注入 + `noopMetrics` 回退),其中 purger 子集为 `PurgeMetrics` 接口;`cmd/server/metrics_adapters.go` 适配 `*observability.Metrics`,`NewService` 在选项注入 recorder 之后构建 purger。

文档:五个指标行进入 [`specs/backend/observability.zh.md`](../../../specs/backend/observability.zh.md) 的指标清单,同时把已漂移的行修正为已上线实态(`markpost.auth.login_total` → 分立的 `login_success_total`/`login_failure_total` 计数器;`markpost.delivery.failed_total` 标签 `reason` → 实际的 `error_category` 属性)。[`specs/backend/caching.zh.md`](../../../specs/backend/caching.zh.md) 载有对照 `CF-Cache-Status` 解读源站缓存指标的小节(边缘 HIT/MISS/EXPIRED 与源站命中率的关系;边缘吸收了绝大多数读流量,因此"源站流量低 + 命中率高"是健康稳态而非故障)。两份规范的双语孪生文件在同一变更中同步更新。

## Alternatives considered

**在 `Get` 内部计数的计量版 `renderCache` 包装器。** 机制上最省事(一个包装器覆盖两个实现),但它同样会计入 singleflight 的二次检查 `Get`——每次冷未命中记两次——而排除二次检查所需的标记或上下文,恰恰重新引入了包装器本想隐藏的调用点知识。在请求决策点计数是每结果一行、且语义精确。

**单计数器 + 结果属性(如 `markpost.cdn.purge_total` 配 `outcome=success|failure|skipped`)。** 指标更少,也是规范 login 行曾暗示的形态,但已上线的每个业务指标都是按结果分立的计数器(`login_success_total`、`delivery.failed_total` 等);为两个相邻机制引入第二种风格,会让 jq 查询在整个指标清单内不一致。而暗示标签风格的那行规范,本身就是相对已上线代码漂移的结果。

**导出 ristretto 内置 `Metrics`(命中率、内部计数器)。** 免费获得细节,但那些比率是按 `Get` 的内部记账,不是请求级有效性——不回答"多少比例的渲染请求免于渲染",且把运维方绑定到 ristretto 的内部词汇表。

**独立的 purge 发起计数器。** 冗余:尝试次数恰为 success + failure;skipped 的定义就是未尝试。聚合查询保持简单。

## Consequences

回答"渲染缓存是否生效"与"purge 是否发起且成功"只需可观测文件与 `jq`——无需访问数据库。接受的义务:命中/未命中口径绑定在快路径 `Get` 的位置上,未来移动它的重构必须连带迁移埋点(`post_metrics_test.go` 的冷后热单测钉死一次未命中、一次命中,缓存关闭时只有未命中);`observability/metrics_test.go` 钉死指标名与无属性决策,`purger_test.go` 钉死结果分类;purge 尝试次数是派生量(success + failure),不是一级序列。无 Cloudflare 的自托管实例只会看到 `purge_skipped_total` 增长——这是预期稳态,caching 规范的解读指南已明说,以免误读为故障。基数保持平坦:五个计数器、无属性、每 60 秒导出各一条序列。超出 issue 严格范围的规范修正(login 与 `error_category` 漂移修复)已在交付 PR 中显式声明,供评审否决。
