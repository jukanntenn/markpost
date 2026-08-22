# 读路径缓存

[English](caching.md) | 中文

本页规定 markpost 的读路径缓存设计：三个缓存层（浏览器 / CDN / 源站渲染缓存）、ETag/304 方案、CDN 清除契约，以及由删除驱动的失效。压缩与页面权重的工作见 [`compression.zh.md`](./compression.zh.md)；请求限流见 [`rate-limiting.zh.md`](./rate-limiting.zh.md)。决策记录 —— 为何选 Cloudflare、为何是这些 TTL、拒绝了什么 —— 是[性能优化 MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.zh.md)。运维层的 Cloudflare（接入、SSL 模式、免费版边界）见 [`cloudflare.zh.md`](./cloudflare.zh.md)。

<a id="scope-and-workload"></a>

## 范围与负载

markpost 的核心业务是 **Markdown 内容的存储与分发** —— 一条通知、一次临时分享、一段粘贴 —— 不是社交文章。四个事实决定缓存设计：

- **文章一次写入且不可变。** 没有发布后编辑，没有 `UpdatePost` 路径。这把缓存失效收拢为删除事件，并允许对文章_正文_做激进的边缘缓存。
- **文章短寿命。** 7 天保留期下限是用户体验要求。
- **读路径是热路径。** 产品由点击分享链接的读者消费；写路径是低频的创作操作（平均约 0.12 次写入/秒，每用户硬上限为每分钟 10 篇、每天 1000 篇）。
- **两种部署语境。** 项目既作为可自托管软件交付，也作为官方 SaaS 实例运行。任何 SaaS 专属内容都没有固化进应用代码或配置默认值。

<a id="hardware-envelope-saas-reference-instance"></a>

## 硬件包络（SaaS 参考实例）

| 资源   | 限制   | 说明                                                                        |
| ------ | ------ | --------------------------------------------------------------------------- |
| CPU    | 2 核   | 由 markpost 容器内的 Caddy + Go +（Next.js）共享；Postgres 运行在同级容器中 |
| 内存   | 2 GB   | 所有进程共享                                                                |
| 磁盘   | 40 GB  | Postgres 数据 + WAL + 容器层                                                |
| 带宽   | 3 Mbps | 出站峰值 **375 KB/s**                                                       |
| 月流量 | 1 TB   | 375 KB/s 持续约 30.86 天 ≈ 1 TB                                             |

**链路即配额。** 把 3 Mbps 链路打满一个月就耗尽全部 1 TB 配额。省下的每个字节同时是链路余量与配额余量 —— 一份预算，不是两份。这正是传输最小化（见 [`compression.zh.md`](./compression.zh.md)）而非 CPU 优化主导设计的原因。带宽更充裕的自托管实例感受这个约束较浅，但同样受益于这些优化。

3 Mbps 的源站无法直接支撑"每秒几百次读取"的目标：375 KB/s ÷ 每个压缩页面约 10 KB ≈ 25 次源站响应/秒。无缓存的源站物理上无法承载该读负载，因此 **CDN 是 SaaS 参考实例的前置条件** —— 不是可选增强。管道更粗的自托管实例没有 CDN 也能运行并接受更高的源站负载；没有 CDN 时没有任何东西损坏，因为全部缓存逻辑都在源站（见下文_自托管兼容性_）。

<a id="three-cache-layers-three-invalidation-stories"></a>

## 三层缓存，三个失效故事

```
Browser ──[1]──> Cloudflare edge ──[2]──> Origin VPS (Caddy → Go)
 (private)        (shared)                  (render cache + DB)
```

| 层           | TTL                               | 失效方式                                          |
| ------------ | --------------------------------- | ------------------------------------------------- |
| 浏览器       | `max-age=300`                     | 仅到期 —— 服务器无法清除                          |
| CDN          | `s-maxage=3600`                   | 到期 + 源站同步再验证 + 缓存标签清除              |
| 源站渲染缓存 | 无界（键 = QID + buildID + 变体） | 进程重启；`DeletePost` / `PruneExpired`；发布轮换 |

**决定性的细微之处：只有文章_正文_不可变 —— HTML_响应_不是。** Go 渲染的 HTML 响应把不可变正文与一个可变外壳捆绑在一起：指向 CSS 文件的 `<link>` 标签、页脚品牌字符串、页面骨架。外壳随 CSS 或模板升级而变化。这就是三个 TTL 各不相同的原因，也是 `immutable` 与一年期 CDN TTL 都不适用于 HTML 响应的原因。

- **浏览器**无法被服务器清除，因此给它短 TTL（300 秒）。外壳变化后，300 秒之内的下一次再验证会拿到新版本。
- **CDN** 可以对源站再验证，因此它持有页面一小时；外壳变化后，源站返回带新 ETag 与新正文的 `200`，CDN 换掉它的副本。刻意不使用一年期 TTL —— 它会把陈旧外壳冻结到手动清除为止，而一小时 TTL 让渲染器/CSS 升级在一小时内传播，无需全站清除。（Cloudflare 免费版确实支持按文章的缓存标签清除 —— 见下文_CDN 缓存_ —— 但清除 API 仍是_主动删除_机制，不是发布部署机制。）
- **源站渲染缓存**以 QID _加发布维度_为键；一次发布自动轮换整个键命名空间（发布携带新二进制，重启进程，本来就会清空内存缓存）。

<a id="etag-design--hash-the-rendered-response-not-its-inputs"></a>

## ETag 设计 —— 对渲染后的响应做哈希，而非对其输入

`ETag` 是_响应正文_的指纹，而响应把不可变正文与可变外壳以及渲染器本身（goldmark + bluemonday + 原生 HTML 中和器，其中任何一项都可能在版本间变化）捆绑在一起。保证 ETag 反映**决定渲染字节的全部因素**的唯一办法是对渲染输出做哈希：

```
ETag (HTML) = xxhash64( minified renderedHTML )          // the exact bytes served
ETag (raw)  = xxhash64( "# " + title + "\n\n" + body )   // the exact bytes of the raw response
```

对输入（`body + title + cssHash + templateVersion`）做哈希行不通：goldmark 或 bluemonday 升级会改变渲染出的 HTML 却让输入保持不变，于是 CDN 的再验证会命中 `If-None-Match` 相等，返回 `304`，并不断续期一个由旧代码渲染的陈旧外壳。对客户端实际收到的字节做哈希让这类 bug 不可能发生 —— 渲染器、模板或 CSS 的任何变化（CSS 变化会改变外壳中 `<link>` 的 href）都自动产生不同的 ETag；不需要维度清单。

选择 `xxhash64`（`github.com/cespare/xxhash/v2`）而非 SHA-256：ETag 生成不需要密码学抗碰撞性，xxhash 快约 20 倍，且经 ristretto 作为传递依赖到达。64 位值以十六进制编码为 16 个字符；碰撞概率（2⁻⁶⁴）对缓存验证而言可忽略。ETag 在 `singleflight.Do` 内部、**每次缓存未命中只计算一次** —— 对完整渲染 HTML 做哈希的成本只由冷未命中突发的 leader 支付，热路径从不支付。

<a id="render-cache-key--qid--buildid--variant"></a>

## 渲染缓存键 —— QID + buildID + 变体

```
cache key (HTML) = qid + ":" + buildID + ":html"
cache key (raw)  = qid + ":" + buildID + ":raw"
cache value      = { title, body, etag, createdAt }    // stored together
```

在一个进程生命周期内，渲染器、模板与 CSS 都是常量（在 `NewService` 中一次性构建），因此仅 QID 就决定输出；一次发布携带新二进制、重启进程并清空内存缓存。`buildID`（`internal/web/buildid.go`，编译期注入的构建短哈希）仅作为对未来不重启热更新模板的防御而保留。`:html`/`:raw` 后缀分开两个变体，使它们互不碰撞。值把 `createdAt` 与正文、ETag 存在一起，让 handler 无需 DB 往返即可发出 `Last-Modified`。

<a id="the-render-pipeline-behind-a-cache-miss"></a>

## 缓存未命中背后的渲染管线

未命中时，leader 运行完整管线（`RenderPostHTML`，`internal/service/post/post.go`）：Postgres `GetByQID` 读取（唯一 QID 索引上亚毫秒级）→ goldmark 渲染（一个共享的并发安全 `goldmark.Markdown` 实例）→ 原生 HTML 中和（一个正则 pass，转义 raw-text/RCDATA 元素的起始 `<`，使未闭合标签无法吞掉文档）→ bluemonday 净化（共享的 `UGCPolicy` 衍生策略，最昂贵的一步 —— 一次完整的 HTML5 分词器 pass）→ `addNoReferrerToImages` → HTML 最小化（见 [`compression.zh.md`](./compression.zh.md)）。DB 读取之后的步骤是 `Body` 的纯函数；对不可变文章，它们在进程重启前产生逐字节相同的输出。`?format=raw` 变体的"渲染"是纯字符串拼接（`"# " + title + "\n\n" + body`）—— 没有 goldmark/bluemonday pass。

<a id="singleflight--ristretto-composed"></a>

## `singleflight` + ristretto，组合使用

快路径是 ristretto `Get`；仅未命中时请求才进入 `singleflight.Do`，而 `Do` 内部会再次检查缓存，避免与并发填充竞争：

```go
func (s *Service) RenderPostHTML(ctx context.Context, qid string) (title, html, etag string, createdAt time.Time, err error) {
    key := qid + ":" + buildID + ":html"

    if v, ok := s.cache.Get(key); ok {          // fast path — no lock, no Do
        return v.title, v.body, v.etag, v.createdAt, nil
    }

    v, err, _ := s.group.Do(key, func() (any, error) {
        if v, ok := s.cache.Get(key); ok {      // double-check inside Do
            return v, nil
        }
        ... render, minify, etag := etagHex(minified) ...
        s.cache.Set(key, r, int64(len(r.body)))
        return r, nil
    })
    ...
}
```

| 层                         | 防御                  | 机制             |
| -------------------------- | --------------------- | ---------------- |
| ristretto `Get`（`Do` 外） | 跨时间的重复          | map 查找，纳秒级 |
| `singleflight.Do`          | 瞬时内的并发          | `WaitGroup` 屏障 |
| ristretto `Get`（`Do` 内） | leader 执行期间的竞争 | 双重检查填充     |

选择 **ristretto**（`internal/service/post/cache.go`）是为了 TinyLFU 准入控制：读访问呈 Zipf 分布，普通 LRU 会把热集丢给一次性的冷访问突发（爬虫扫描、批量分享）。TinyLFU 的频率草图只在新条目比它将逐出的条目更"热"时才准许进入。`MaxCost` 以_字节_设置（默认 128 MiB，经 `[render] cache_size_bytes`；条目按其正文长度计价），写入异步批量进行，热路径从不在逐出簿记上阻塞，`NumCounters`（约 10 倍于预期键数）保持草图的准确性。缓存由配置驱动：`[render] enabled` 可以禁用它，尺寸也可以为小实例缩小。

<a id="http-cache-headers-in-detail"></a>

## HTTP 缓存头，逐项说明

`RenderPost` 的 HTML 响应携带：

```http
ETag: "<xxhash64(minified renderedHTML)>"
Last-Modified: <Post.CreatedAt as HTTP date>
Cache-Control: public, max-age=300, s-maxage=3600
Cache-Tag: post-<qid>
Vary: Accept-Encoding
```

`?format=raw` 响应携带相同的 `Cache-Control`/`Cache-Tag`/`Vary`/`Last-Modified`，ETag 为 `<xxhash64("# "+title+"\n\n"+body)>`。带哈希的 CSS 资源（serve 于 `/static/post.<cssHash>.css`）携带 `Cache-Control: public, max-age=31536000, immutable`（见 [`compression.zh.md`](./compression.zh.md)）。文章页的 404 携带 `Cache-Control: public, max-age=60, s-maxage=60`（`setNotFoundCache`，`internal/api/rest/v1/post.go`），使 QID 枚举探测被 CDN 边缘吸收，而不是每次请求都回源；只有 not-found 情形被标记 —— 其他错误保持不可缓存。

- **`public`** 允许共享缓存（CDN）在浏览器之外存储该响应。
- **`max-age=300`** 是浏览器的新鲜度生命周期：300 秒内浏览器完全从磁盘服务，没有任何网络活动。
- **`s-maxage=3600`** 仅对共享缓存覆盖 `max-age` —— 这是让 CDN 吸收绝大多数读取的旋钮。
- **不使用 `stale-while-revalidate`。** 依 RFC 9111，`s-maxage` 吸收了 `proxy-revalidate` 的语义，后者禁止共享缓存服务陈旧内容；在 Cloudflare 该指令是空操作，再验证是同步的（`EXPIRED`）而非后台刷新。保留一个什么都不做的指令会误导读者。同步再验证在这里很便宜 —— `304` 无正文，且源站渲染缓存无需重新渲染即可给出 ETag。（要让 SWR 生效需要去掉 `s-maxage` 并经 Edge Cache TTL 规则设置 CDN TTL，其免费版下限为 2 小时 —— 与一小时的升级传播目标冲突。）
- **`immutable` 只出现在 CSS 资源上**，绝不出现在 HTML 或 raw 上。HTML 与 raw 响应位于发布时不变化的 URL（`/:qid`、`/:qid?format=raw`），而文章可能被主动删除；把任一标记为 `immutable` 在事实上是错误的，并会让浏览器永远无从得知一次删除。raw 正文按 QID 不可变，但其 URL 不按内容寻址，因此它使用与 HTML 相同的 TTL 方案。
- **`Last-Modified`** 是 `Post.CreatedAt` —— 文章一次写入，因此它就是真实的最后修改时间。它充当第二验证器（RFC 9110 建议两者都发）；`If-None-Match` 优先于 `If-Modified-Since`，因此陈旧的 `Last-Modified` 不能造成错误的 `304` —— ETag 获胜，而 ETag 跟踪外壳/渲染器升级。
- **`Cache-Tag: post-<qid>`** 是 Cloudflare 的代理键。一篇文章的 HTML 与 raw 两个变体携带相同标签，因此一次按标签清除同时失效两者，无论 CDN 持有多少个 `Accept-Encoding` 变体。Cloudflare 会把该头从访客可见的响应中剥除。
- **`Vary: Accept-Encoding`** 让 gzip 与 zstd 的缓存条目分开，使支持 zstd 的浏览器永远不会收到不匹配的 gzip 正文。Caddy 的 `encode` 在压缩时加上该头；在 handler 中显式设置则覆盖无压缩回退的情形。
- **API 响应上的 `Cache-Control: no-store`。** `/api/v1` 组用 `NoStore` 中间件包裹每个响应 —— 动态载荷（尤其是 `/oauth/url`，其正文携带一次性 CSRF state）绝不能被共享缓存存储。handler 之后仍可为刻意可缓存的响应覆盖该头。

<a id="who-handles-the-304"></a>

### 谁处理 304

| 情形                         | 处理者                                                                        |
| ---------------------------- | ----------------------------------------------------------------------------- |
| CDN 边缘命中，浏览器再验证   | **Cloudflare** —— 用它存储的 ETag 应答 `304`；源站完全看不到该请求            |
| CDN 副本过 TTL，对源站再验证 | **Gin handler** —— 将 `If-None-Match` 与渲染缓存 ETag 比较；相等 → 无正文 304 |
| 无 CDN                       | **Gin handler**，同一路径                                                     |

缓存命中时 handler 完全跳过 goldmark/bluemonday；未命中时执行渲染并为后续请求填充缓存。Caddy 既不生成也不比较 ETag —— 它是透传头部并压缩正文的反向代理。`304` 响应没有正文，也不被压缩。

<a id="cdn-caching-cloudflare-free-tier-and-the-purge-contract"></a>

## CDN 缓存：Cloudflare 免费版与清除契约

Cloudflare 免费版是承重选择：不限量带宽（预计约 7.8 TB/月的边缘出站流量花费 $0；按量计费的 CDN 以 $0.085/GB 约 $660/月，并会以账单重新施加 1 TB 约束）、不计量 DDoS 防护（3 Mbps 之后的 2 核源站扛不住未缓存的洪水），以及全球任播边缘（约 330 个 POP）。100k/天的免费版限制适用于 **Workers**（边缘计算），markpost 不使用它；CDN 缓存路径是静态的、由头部驱动，没有请求上限。锁定风险接近零：Cloudflare 是纯靠 DNS 即可达的反向代理，应用中没有嵌入任何专有 API（`[cloudflare]` 配置节是可选的）。

**清除 API。** 所有清除方式在每个套餐上都可用（按 URL、缓存标签、前缀、主机名、"purge everything" 清除一切）。免费版限制，按账户经令牌桶施加：每分钟 5 次清除请求、桶容量 25、每次请求 100 个操作（标签/URL）；清除延迟文档记载全球低于 150 ms。设计使用**按缓存标签清除**：删除一篇文章发出一次 `POST /zones/{zone}/purge_cache`，负载为 `{"tags":["post-<qid>"]}`。即使在假设的每天 3 000 次删除下，平均清除速率（约 2 次/分钟）也远低于上限。拒绝 "purge everything"（全站清除）：它会迫使每个缓存文章同时回源 —— 一个能压垮源站的惊群。缓存标签机制在没有该风险的情况下提供按文章的粒度，且存在按 URL 的回退（免费版每秒 800 个 URL）以防缓存标签可用性发生变化。

**清除是尽力而为且异步的。** 删除 handler 同步移除源站渲染缓存条目（必须），随后经 `Purger` 接口（`internal/service/post/purger.go`）在后台 goroutine 上排队 Cloudflare 清除调用：配置了 `[cloudflare] api_token` + `zone_id` 时是 `cloudflarePurger`，否则是 `noopPurger`（无 Cloudflare 的自托管 —— CDN 副本退回自然 TTL 到期）。QID 在进入 JSON 负载前经净化（`sanitizeCacheTag`）；失败被记录并吞掉，不做重试。主动删除因此是**源站即时、CDN 近即时（通常 <150 ms）、浏览器至多 5 分钟陈旧**（`max-age=300`）。

<a id="deletion-and-invalidation"></a>

## 删除与失效

删除端点：`DELETE /api/v1/posts/:id`（JWT 属主）与 `DELETE /api/v1/admin/posts/:id`（管理员）。`DeletePostByQID(ctx, qid, ownerID)` 移除 DB 行（`ownerID > 0` 时限定属主，管理员路径不加约束；无匹配行时 `ErrNotFound`），同步移除两个渲染缓存条目 —— ristretto 包装器的 `Delete` 调用 `cache.Wait()`，因此并发渲染的一个待处理缓冲 `Set` 无法在删除后重新准入该条目 —— 并排队上述尽力而为的缓存标签清除。

`PruneExpired`（对已过期内容的例行清理）移除 DB 行与源站缓存条目（repo 返回被清理的 QID，service 借此使其失效），但**不**清除：已过期临时内容的陈旧但无害投递是被接受的权衡，且清理量可能很大。被清理文章的 CDN 边缘副本存留至多一小时 TTL；该窗口内的读者得到一个陈旧但无害的 200。

<a id="request-flow-walkthrough"></a>

## 请求流程走查

1. **首次访问，浏览器与 CDN 皆冷。** 浏览器 → Cloudflare 边缘 → 源站。Go 渲染（或从源站渲染缓存服务），Caddy 压缩，响应带着缓存头流回。边缘存储一小时；浏览器 300 秒。
2. **300 秒内重复访问。** 浏览器使用本地副本 —— 零网络流量。
3. **300 秒后重复访问，CDN 仍新鲜。** 浏览器发送条件请求；边缘副本未过期，Cloudflare 自己应答 `304`。源站从不被联系。
4. **CDN 副本过一小时。** Cloudflare 向源站发送条件请求。无变化时源站返回无正文 `304`；发布之后返回带新正文的 `200`，边缘换掉副本。
5. **来自新地理区域的首次访问。** 该区域的边缘节点执行一次回源；随后的区域访客命中该边缘。源站负载随_见过该 URL 的边缘节点数_扩展，而非随总请求数。
6. **文章被保留期清理删除。** 源站从缓存与 DB 移除；边缘副本存留至多一小时（陈旧但无害）。
7. **文章被用户或管理员删除。** 源站移除 + 尽力而为的缓存标签清除（见上文）。

<a id="deployment-window-analysis-release-induced-origin-load"></a>

## 发布窗口分析（发布引发的源站负载）

一次发布携带新二进制、重启进程并清空内存渲染缓存 —— 之后每个 CDN 副本再验证的文章都未命中并重新渲染。这不会以尖峰形式到来：CDN 副本惰性再验证，各自遵循自己的 TTL 日程（设定于副本上次填充或验证之时，而非发布时刻），因此再验证分布在发布之后的一小时内，而约 330 个 POP 甚至把单篇文章的区域副本也错开。最坏算术 —— 1 000 000 篇缓存文章全部在一小时内再验证 —— 给出约 278 req/s ≈ 两个核心的 42%；真实的活跃缓存群体远小于此，因此 42% 是上限而非预期。真正的惊群情形（一篇极热文章从许多 POP 同时再验证）正是 `singleflight` 击败的：五十个对同一 QID 的并发再验证坍缩为一次渲染，其余四十九个等待者接收它的结果。在低流量窗口部署始终是换取余量的免费运维杠杆。

<a id="self-hosting-compatibility"></a>

## 自托管兼容性

每个组件落入两档之一：

| 档                     | 组件                                                                                                                                                                                    | 自托管行为                                                                        |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| **镜像内置，始终开启** | Caddy `encode`、CSS 外置 + 最小化 + `go:embed`、HTML 最小化、HTTP 缓存头 + ETag/304、Postgres 连接池/lz4/GUC 调优、singleflight+ristretto、三限流器 + 登录限流、删除端点 + 源站缓存失效 | 纯代码；随镜像交付，零配置。                                                      |
| **外部，可选**         | Cloudflare CDN、B2/WAL 备份                                                                                                                                                             | 挂在镜像前/后的运维层；使用、忽略或替换为等价物。应用代码中没有任何东西引用它们。 |

CDN 是建议而非要求：管道更粗的自托管实例可以没有 CDN 运行，并接受更高的源站 CPU 与带宽占用。配置由配置驱动而非编译期驱动 —— 渲染缓存有 `[render] enabled` 开关与可调尺寸，`config.go` 或 `config.example.toml` 中没有 SaaS 专属值。三种部署模式（SaaS / 有域名自托管 / 家庭实验室）只在 Caddyfile、DNS 与可选的 `[cloudflare]` 节上不同；Go 二进制完全相同。完整拓扑与接入：[`cloudflare.zh.md`](./cloudflare.zh.md)；备份与恢复：[`disaster-recovery.zh.md`](./disaster-recovery.zh.md)。
