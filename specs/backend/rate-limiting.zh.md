# 限流

[English](rate-limiting.md) | 中文

面向公开读路径、API 认证写路径与登录端点的请求限流：构建于 tollbooth v8 之上的四个独立令牌桶限流器，各限定一个路由类别，并以真正标识行为者的维度为键。接线位于 `cmd/server/main.go`（`SetupRoutes`）与 `internal/middleware/rate_limit.go`；配置位于 `[ratelimit]` TOML 节。决策记录（为何四个限流器、为何这些键维度、拒绝了什么）见[性能优化 MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.zh.md)。

<a id="the-four-limiters"></a>

## 四个限流器

| 限流器                 | 路由                                                                                                                                                | 键        | 速率（默认）                                                                   |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------------------------------------------------------------------------------ |
| **read**（L1）         | `GET /:qid`（公开文章渲染）                                                                                                                         | 客户端 IP | 100/秒，突发 200                                                               |
| **public_write**（L2） | `POST /:post_key`                                                                                                                                   | `user_id` | 10/分（0.1667/秒），突发 20；**外加**每日上限 1000/天（0.01157/秒，突发 1000） |
| **authed_write**（L3） | JWT 组写路由（`POST /auth/logout`、`POST /auth/change-password`、`POST /post-key/rotate`、投递渠道写入、`DELETE /posts/:id`、会话吊销、管理员写入） | `user_id` | 30/分（0.5/秒），突发 60                                                       |
| **login**              | `POST /auth/login`、`POST /oauth/login`                                                                                                             | 客户端 IP | 5/分（0.0833/秒），突发 5                                                      |

- **L1** 宽松，因为 CDN 吸收绝大多数读取；它只治理对源站再验证的那一小部分。IP 是公开读路径上唯一可用的标识。
- **L2** 以 `user_id` 为键，由在限流器运行之前校验每用户凭据的 `PostKey` 中间件解析。以 `user_id` 而非裸 `post_key` 为键意味着轮换 post_key 无法逃避限额，并与 L3 统一了维度。10/分与 1000/天上限是业务硬限制。
- **L3** 以 JWT（`AuthWithBlacklist`）中的 `user_id` 为键。读（GET）留在限流器之外，因此列表不消耗写预算。
- **login** 是面向凭据端点的专用按 IP 限流器 —— 登录尝试没有可作键的已认证身份，而 5/分的紧默认值正是为了挫败撞库。

**每日上限的实现（L2）。** tollbooth 的令牌桶是固定的 1 秒窗口，因此每日上限表达为 `rate.Limit(1000.0/86400)`、突发 1000 —— 数学上即"每天 1000 个，可一次性突发花完"。代价：在 UTC 午夜花光全部 1000 个令牌的用户，每个额外令牌要等约 86 秒，对低频创作操作可以接受，且避免了第二个按日期作键的计数器数据结构。

**429 响应携带 `Retry-After`**（从桶的补充时间计算）与经 `apierr.RespondError` 的自定义 i18n JSON 正文；tollbooth 自己的响应写入路径被绕过。`RateLimit-Limit` / `RateLimit-Reset` / `RateLimit-Remaining` 头未设置 —— CORS 暴露列表为未来可能的使用保留它们。

**匿名客户端。** 若 `c.ClientIP()` 返回空（无法解析），按 IP 作键的限流器立即返回 `429`，而不是把所有匿名客户端折叠进一个共享的 `"unknown"` 桶 —— 一个匿名攻击者绝不能耗尽其他所有人的限额。选 `429` 而非 `400`，因为语义是"你正在被限流"（无身份 → 无配额），不是"请求格式错误"。

**豁免。** `GET /api/v1/health` 与 `GET /api/v1/version` 注册在每个限流器组之外；Docker 健康检查在回环定时器上访问 health，让它受 L1 约束会在负载下造成假阳性健康失败。

<a id="ip-resolution-gin-not-tollbooth"></a>

## IP 解析：gin，而非 tollbooth

中间件调用 `c.ClientIP()`（应用下述可信代理逻辑）并把结果传给 `tollbooth.LimitByKeys`。tollbooth 自己的 `SetIPLookup` 不做可信代理校验，把 IP 解析委托给它会让可信代理配置所关闭的伪造风险重新出现。

SaaS 拓扑中的全部流量走一条路径：`Client → Cloudflare → Caddy → Go`。Cloudflare 在回源时重写 `X-Forwarded-For` 并设置 `CF-Connecting-IP`；源站防火墙锁定到 Cloudflare 的 CIDR（见 [`cloudflare.zh.md`](./cloudflare.zh.md) _源站防护_）。客户端 IP 恢复使用标准 XFF 链与两层信任配置，作为对 IP 白名单经伪造被绕过这一残余威胁的**纵深防御**保留：

**Caddy 层。** `reverse_proxy` 以 Cloudflare 公布的 CIDR 段作为 `trusted_proxies` 运行。对来自 Cloudflare 对端的请求（`trusted=true`），Caddy 把自己这一跳追加到 Cloudflare 已用真实客户端 IP 填好的 `X-Forwarded-For` 上。对来自任何其他对端的请求（只可能发生在白名单被绕过时），Caddy **覆写**客户端提供的任何 `X-Forwarded-For`，只留真实的对端 IP —— 伪造的 `X-Forwarded-For: 1.2.3.4` 被丢弃，而非被追加。只有可信对端才能向链前插值。

**gin 层。** `SetTrustedProxies(["127.0.0.1", "::1"])` 对应 Caddy 经回环代理到 Go 的事实。`ClientIP()` 先检查直接 TCP 对端，然后从右到左遍历 `X-Forwarded-For`，跳过可信代理 IP 并返回第一个不可信 IP —— MDN 的"可信代理列表"算法。由于合法流量以 `<real-client>, <cloudflare-hop>` 加回环跳到达，gin 得出真实客户端 IP。

刻意不使用 `gin.PlatformCloudflare`（无条件信任 `CF-Connecting-IP`）：它不做 CIDR 检查，因此一个把 Cloudflare 源 IP 伪造过防火墙的攻击者也能伪造该头并逃避限流。XFF 设计在 TCP 层验证对端，而 TCP 层的伪造是不可能的。

**Cloudflare CIDR 维护。** CIDR 列表由运维者在 Caddyfile 模板中提供（`trusted_proxies` 只接受 CIDR 字面量；它无法抓取 URL）。Cloudflare 偶尔更新其公布的段（https://www.cloudflare.com/ips/）；运维者必须重新同步 —— 这一明确的运维职责记录在 [`cloudflare.zh.md`](./cloudflare.zh.md)。

<a id="configuration"></a>

## 配置

```toml
[ratelimit.read]          # per_second = 100, burst = 200
[ratelimit.public_write]  # per_second = 0.1666666667, burst = 20,
                          # daily_per_second = 0.0115740741, daily_burst = 1000
[ratelimit.authed_write]  # per_second = 0.5, burst = 60
[ratelimit.login]         # per_second = 0.0833333333, burst = 5
```

所有值对运维者可调，默认值如上（经 `MARKPOST_RATELIMIT__*` 环境映射）；`internal/middleware/rate_limit_fuzz_test.go` 对键构造做模糊测试，`rate_limit_test.go` 覆盖限流器隔离（L1 不计入 L2）、匿名 429 路径与健康豁免。
