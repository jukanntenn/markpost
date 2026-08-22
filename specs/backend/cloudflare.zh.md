# Cloudflare CDN 集成

[English](cloudflare.md) | 中文

本文档规定 markpost 如何为 SaaS 参考实例集成 Cloudflare 免费版 CDN，并记录项目支持的三种部署模式。它是 DNS、TLS、缓存与缓存清除决策的权威参考。所有 Cloudflare 行为主张均引用 `~/Workspace/contexts/cloudflare/cloudflare-docs/`（仓库版本 2026-07-08）的 Cloudflare 文档；引用使用相对该根目录的路径。

缓存与失效的_设计_（三层缓存、ETag 方案、缓存标签清除、渲染缓存）见 [`caching.zh.md`](./caching.zh.md)。本文档覆盖_运维_层：如何把一台 VPS 源站接入 Cloudflare、选哪种 SSL 模式、清除 API 如何调用、免费版的边界是什么。

<a id="deployment-modes"></a>

## 部署模式

markpost 既作为可自托管软件交付，也作为官方 SaaS 实例运行。每个设计选择必须在两种语境中都可行：任何 SaaS 专属内容都没有固化进应用代码或配置默认值（[`caching.zh.md`](./caching.zh.md#self-hosting-compatibility)）。三种模式只在部署拓扑、Caddyfile 与 DNS 上不同 —— Go 二进制完全相同。

| 模式                 | 源站      | DNS                                 | TLS 终结                                                               | CDN | Caddyfile                                                                          | 典型用途             |
| -------------------- | --------- | ----------------------------------- | ---------------------------------------------------------------------- | --- | ---------------------------------------------------------------------------------- | -------------------- |
| **SaaS**             | VPS       | 域名 → Cloudflare（Proxied / 橙云） | Cloudflare 边缘（访客侧）+ Caddy 上的 Origin CA（源站侧，Full strict） | 有  | `Caddyfile.production.j2`（TLS 经 Origin CA，`trusted_proxies` = Cloudflare CIDR） | 官方实例             |
| **自托管（有域名）** | VPS / NAS | 域名 → 源站 IP（DNS-only / 灰云）   | Caddy 自动 Let's Encrypt + HTTPS 重定向                                | 无  | 按域名的 site 块（仓库中尚无模板）                                                 | 个人 / 小团队自托管  |
| **家庭实验室**       | NAS       | 无（局域网 IP:port）                | 无（明文 HTTP）                                                        | 无  | `docker/Caddyfile`（`:2053`，无 TLS）                                              | 家庭网络，可信局域网 |

**CDN 只是 SaaS 参考实例的前置条件。** 3 Mbps 的源站无法直接支撑每秒几百次读取（[`caching.zh.md`](./caching.zh.md#hardware-envelope-saas-reference-instance)）。管道更粗的自托管实例没有 CDN 也能运行，并接受更高的源站负载。没有 CDN 时没有任何东西损坏：全部缓存逻辑都在源站，CDN 只是在前面加一层边缘。

<a id="saas-mode-cloudflare-onboarding"></a>

## SaaS 模式：Cloudflare 接入

拓扑是：

```
Visitor ──HTTPS:443──> Cloudflare edge (orange cloud, Full strict) ──HTTPS──> VPS :2053 ──> Caddy :2053 ──> Go :7330 / static export
         [TLS-A: Cloudflare edge certificate]                  [TLS-B: Cloudflare Origin CA certificate]
                                 [Origin Rule rewrites the destination port 443 → 2053]
```

边缘监听面向访客的 443 端口；一条 Origin Rule 把目标端口改写为源站发布的 2053（宿主端口 2053 → 容器端口 2053）。Caddy 从 `/app/frontend` 服务 Next.js 静态导出，并把 API 路径反向代理到 Go 二进制 —— 没有 Node 服务器。

SaaS 模式中 Caddy **不**使用按域名的 site 块，也**不**运行自动 Let's Encrypt。TLS-B 由手动安装的 Cloudflare Origin CA 证书处理。下面两节解释原因。

<a id="dns-and-the-orange-cloud"></a>

### DNS 与橙云

使用完整 DNS 设置（`fundamentals/manage-domains/add-site.mdx`）：

1. 把域名添加进 Cloudflare（Free 套餐）。
2. 创建一条 **A 记录**，其内容是 **VPS 真实 IP**。这个 IP 是 Cloudflare 回源时使用的源站地址 —— "Your origin server address (cannot be a Cloudflare IP)"（`dns/manage-dns-records/reference/dns-record-types.mdx`）。
3. 把该记录设为 **Proxied（橙云）**。
4. 在注册商处把 nameserver 改为 Cloudflare 分配的 NS 对。

决定性的一点常被误解：橙云开启时，**公共 DNS 查询返回的是 Cloudflare 任播 IP，不是 VPS IP**。A 记录里的 VPS IP 只有 Cloudflare 看到（用于回源），访客永远看不到：

> "对已代理记录 `blog.example.com` 的 DNS 查询将由 Cloudflare 任播 IP 地址应答……而非 `192.0.2.1`。这确保该名称的 HTTP/HTTPS 请求会被送往 Cloudflare 网络并可以被代理……" —— `dns/proxy-status/index.mdx`

这就是"把源站藏在 CDN 之后"的含义。DNS-only（灰云）则会"以你服务器的真实 IP 应答……这暴露了你的源站 IP"（`dns/proxy-status/index.mdx`）。只有 A/AAAA/CNAME 记录可被代理；MX/TXT 永远 DNS-only。

已代理记录的 TTL 固定为 Auto（300 秒，不可编辑）。Cloudflare 建议代理所有服务 Web 流量的 A/AAAA/CNAME 记录。

<a id="tls-two-segments-and-the-ssl-mode-choice"></a>

### TLS：两段与 SSL 模式选择

Cloudflare 把 TLS 拆成两段独立的部分：

- **TLS-A（访客 ↔ 边缘）：** 以 Cloudflare 管理的证书在边缘终结，自动且免费。
- **TLS-B（边缘 ↔ 源站）：** 由仪表盘中选择的 **SSL/TLS encryption mode** 决定。

四种模式（`ssl/origin-configuration/ssl-modes/*.mdx`）：

| 模式              | 边缘 → 源站         | 需要源站证书                                    | 失败码 | 安全性                |
| ----------------- | ------------------- | ----------------------------------------------- | ------ | --------------------- |
| Off               | 无                  | 否                                              | —      | 最差（TLS-A 也关闭）  |
| Flexible          | 明文 HTTP           | 否                                              | —      | 中（TLS-B 明文）      |
| Full              | HTTPS，**不校验**   | 是（自签/CA）                                   | 525    | 中高（MITM 可换证书） |
| **Full (strict)** | HTTPS，**严格校验** | 是（未过期、公共 CA 或 Origin CA、CN/SAN 匹配） | 526    | 最高                  |

markpost 选择 **Full (strict)**。Cloudflare 强烈推荐它：

> "为获得最佳安全，尽可能选择 Full (strict) 模式。……你可以使用来自公共可信证书颁发机构（CA）的证书，或从 Cloudflare 生成免费的 Origin CA 证书。" —— `ssl/get-started.mdx`

对带登录的应用，Flexible 被明确劝阻：

> "如果你的应用包含敏感信息（用户登录），请改用 Full 或 Full (Strict) 模式。" —— `ssl-modes/flexible.mdx`

markpost 有用户登录、管理员写路径与改密码 —— 它正是这样的应用。更早的"无 TLS Caddy"拓扑等价于 Flexible，被本决策取代。

**端口。** 访客经规范 HTTPS 端口 443 到达边缘。源站发布 2053（`host_port: 2053` 映射到容器的 `caddy_port: 2053`），一条 **Origin Rule** 把该区域主机名的目标端口从 443 改写为 2053 —— 规则由一个匹配表达式（主机名等于域名）加一个目标端口覆盖组成（仪表盘：Rules → Overview → Create rule → Origin Rule）。Origin Rules 在每个套餐上可用，Free 亦然，Free 为 10 条（`rules/origin-rules/index.mdx`、`rules/origin-rules/create-dashboard.mdx`、套餐矩阵 `rules.origin_rules`）。

<a id="origin-ca-certificate"></a>

### Origin CA 证书

Origin CA 证书是 Cloudflare 免费的长期（15 年）源站证书，只被 Cloudflare 信任 —— 对橙云之后的自托管 VPS 源站是理想选择，没有续期负担。

**步骤：**

1. 仪表盘 → **SSL/TLS → Origin Server → Create Certificate**。
2. 密钥类型 RSA（2048）；主机名覆盖该域名；有效期 **15 年**。
3. 下载 **Origin Certificate** 与 **Private Key** 两者（私钥只展示一次）。
4. 存到 VPS 上，如 `/app/certs/origin.pem` 与 `/app/certs/origin.key`。

Caddyfile 为 TLS-B 出示这张证书。Caddy 仍监听裸端口（不是按域名的 site 块）—— `tls` 指令直接指向证书文件。已部署的模板是 `devops/ansible/templates/Caddyfile.production.j2`；示意：

```caddyfile
{
    auto_https off
}

:2053 {
    tls /app/certs/origin.pem /app/certs/origin.key
    encode zstd gzip
    handle /api/v1/* {
        reverse_proxy 127.0.0.1:7330 {
            trusted_proxies {{ cloudflare_cidrs }}
            header_up X-Forwarded-For {http.request.header.CF-Connecting-IP}
        }
    }
    # /static/*, /swagger/*, /mpk-*, /p-* blocks are identical
    handle {
        root * /app/frontend
        file_server
    }
}
```

SaaS 模式**不**使用自动 Let's Encrypt：已代理域名解析到 Cloudflare IP，Caddy 的 ACME HTTP-01 challenge 到不了 VPS，签发会失败。Origin CA 证书替代它。

<a id="origin-protection-only-cloudflare-may-connect"></a>

### 源站防护：只允许 Cloudflare 连接

橙云本身**不**阻止得知 VPS IP 的人直连它、完全绕过 Cloudflare：

> "如果有人发现了你源站服务器的 IP 地址……他们可以直接向你的服务器发送流量，完全绕过 Cloudflare 的安全防护。为防止这一点，封锁所有非来自 Cloudflare IP 地址的流量……" —— `fundamentals/concepts/cloudflare-ip-addresses.mdx`

两层，施加在 **VPS 宿主防火墙**（iptables，不是 Caddy）上：

1. **白名单 Cloudflare CIDR，丢弃其余一切**（推荐基线）。权威 CIDR 列表是 `https://www.cloudflare.com/ips/`（文档不内联这些段）。这是"中等安全"且"易受 IP 欺骗"（`partials/learning-paths/limit-external-connections-network.mdx`）。
2. **Authenticated Origin Pulls（AOP，可选加固）。** mTLS：Cloudflare 向源站出示客户端证书，因此"Cloudflare 之外的任何 HTTPS 请求都不会从你的源站得到响应"（`ssl/origin-configuration/authenticated-origin-pull/explanation.mdx`）。AOP 要求 Full 或 Full (strict) —— 这正是上文 SSL 模式决策成为前置的原因。AOP 是后续的加固步骤，不属于初始接入。

这层宿主防火墙区别于 Caddy 的 `trusted_proxies`（转发头处理，而非包过滤）—— CIDR 白名单是包层的执行点，它与 Cloudflare 边缘对 `CF-Connecting-IP` 的覆写一起，使下文的客户端 IP 接力可信。

<a id="client-ip-detection"></a>

### 客户端 IP 检测

橙云之后，源站看到的直接对端只有 Cloudflare IP。Cloudflare 在回源时添加携带真实访客 IP 的头（`fundamentals/reference/http-headers.mdx`）：

- **`CF-Connecting-IP`** —— 真实客户端 IP，单值，只在边缘→源站一跳上设置。文档推荐。
- **`True-Client-IP`** —— 取值与 `CF-Connecting-IP` 相同，但仅 Enterprise 可用。
- **`X-Forwarded-For`** —— 代理链（逗号分隔）。

markpost **不**使用 `TrustedPlatform = gin.PlatformCloudflare`（它在应用层无条件信任 `CF-Connecting-IP`，无 CIDR 检查）。取而代之，`Caddyfile.production.j2` 的每个 `reverse_proxy` 块都带 `header_up X-Forwarded-For {http.request.header.CF-Connecting-IP}`。Caddy 在其默认转发头处理**之后**应用用户头操作，因此到达 Go 的 `X-Forwarded-For` 恒为单个 `CF-Connecting-IP` 值 —— 多跳链（访客提供的 XFF、Cloudflare 填充的链、Caddy 自己的一跳）从不抵达应用。

这一收敛是正确性所必需，而非仅为整洁：gin 只信任回环对端（`SetTrustedProxies(["127.0.0.1", "::1"])`，与 Caddy 的回环代理跳一致），`ClientIP()` 从右到左遍历 `X-Forwarded-For`，返回首个不受信 IP。若 Caddy 转发追加后的链 `<real-client>, <cloudflare-hop>`，gin 会把 Cloudflare 边缘 IP 当作每个访客的客户端 IP，以 IP 为键的 L1/login 限流器将坍缩到少数边缘地址上。

该单值的信任锚：Cloudflare 在边缘覆写访客提供的任何 `CF-Connecting-IP`，因此凡是流经 Cloudflare 的流量，该头由 Cloudflare 断言；宿主防火墙（见上文_源站防护_）在包层把其余对端挡在端口之外。残余暴露：绕过防火墙的对端可以伪造 `CF-Connecting-IP`，改写会传播该伪造值 —— 头真实性的执行点是防火墙。每个块上保留 `trusted_proxies {{ cloudflare_cidrs }}`，使 Caddy 默认转发头处理与同一 CIDR 集保持一致；XFF 取值本身由 `header_up` 改写固定。详细机制见 [`rate-limiting.zh.md`](./rate-limiting.zh.md#ip-resolution-gin-not-tollbooth)。

运维要求：`devops/ansible/group_vars/production/vars.yml` 的 `cloudflare_cidrs` Ansible 变量承载当前 Cloudflare CIDR 列表（同时镜像在 VPS 防火墙中）。Cloudflare 偶尔更新这些段；运维者必须同步两处。这一维护职责在此处与 [`rate-limiting.zh.md`](./rate-limiting.zh.md#ip-resolution-gin-not-tollbooth)中均有记录。

<a id="caching"></a>

## 缓存

<a id="what-cloudflare-caches-by-default"></a>

### Cloudflare 默认缓存什么

Cloudflare **按文件扩展名缓存，而非按 MIME 类型**，且**默认不缓存 HTML 或 JSON**（`cache/concepts/default-cache-behavior.mdx`）。默认缓存的扩展名是一组静态资源类型（CSS、JS、JPG、PNG、PDF、WOFF2、SVG……）；HTML/JSON/API 响应返回 `DYNAMIC` 并直通源站，除非源站的头或某条 Cache Rule 使其合格。

markpost 依赖**源站头**路径：`RenderPost` 以 `Cache-Control: public, max-age=300, s-maxage=3600` 服务 HTML（`backend/internal/api/rest/v1/post.go:72`）。因为 `public` + `max-age>0` 满足 Cloudflare 的缓存条件，HTML 文章页进入边缘缓存，尽管 HTML 不在默认缓存之列。

Cloudflare 在 **Origin Cache Control（OCC）**之下尊重源站缓存头，OCC 默认开启且在 Free/Pro/Business 上无法关闭（`cache/concepts/cache-control.mdx`）。这就是 markpost 头中的 `s-maxage`/`max-age` 指令真实生效的原因。

<a id="why-writeauthdynamic-traffic-is-safe-behind-the-cdn"></a>

### 为何写/认证/动态流量在 CDN 之后是安全的

把全部流量（包括写与已认证请求）放在橙云之后是安全的，因为 Cloudflare 默认**不缓存**这些：

- **非 GET 方法永不缓存。** "当……HTTP 请求方法不是 GET 时，Cloudflare 不缓存该资源。"（`cache/concepts/default-cache-behavior.mdx`）markpost 的全部写（创建文章、删除、改密码、投递写入）都是 POST/PUT/DELETE → 透明代理到源站。
- **`Authorization` 响应在 OCC 之下不被缓存**，除非同时存在 `must-revalidate`、`public` 或 `s-maxage`（`cache/concepts/cache-control.mdx`）。markpost 的已认证路由携带 `Authorization: Bearer …`，其响应从不携带这些指令 → BYPASS，每次回源。
- **`Set-Cookie` 响应在 OCC 之下不被缓存**（`cache/concepts/cache-control.mdx`）。登录/OAuth/刷新令牌响应设置 cookie → BYPASS。

唯一被刻意边缘缓存的是公开读 `GET /:qid`。其余一切在每次请求时流向源站。

<a id="cf-cache-status-reference"></a>

### `CF-Cache-Status` 参考

`CF-Cache-Status` 响应头诊断缓存行为（`cache/concepts/cache-responses.mdx`）：

| 取值         | 含义                                                            |
| ------------ | --------------------------------------------------------------- |
| HIT          | 由边缘缓存服务                                                  |
| MISS         | 不在缓存中，从源站取回                                          |
| EXPIRED      | 曾被缓存但 TTL 已过；同步再验证                                 |
| REVALIDATED  | 源站经条件请求（304）确认未变；从缓存服务                       |
| UPDATING     | 已过期，后台再验证（异步 SWR 路径）期间服务陈旧内容             |
| STALE        | 已过期且源站不可达；服务陈旧内容                                |
| DYNAMIC      | 不具缓存资格；直通源站（无正确头的 HTML）                       |
| BYPASS       | 源站指示绕过（`no-cache`/`private`/`max-age=0`，或 Set-Cookie） |
| NONE/UNKNOWN | Cloudflare 在到达缓存前应答（Worker、WAF 拦截、重定向）         |

若文章页持续显示 `DYNAMIC`，说明缓存头没有生效；重复访问先 `MISS` 后 `HIT` 确认缓存工作正常。

<a id="cache-purge"></a>

## 缓存清除

<a id="the-purge-api-contract"></a>

### 清除 API 契约

markpost 在删除文章时发出尽力而为的**缓存标签清除**。经 Cloudflare API 核实的正确调用：

- **端点：** `POST https://api.cloudflare.com/client/v4/zones/{zone_id}/purge_cache`（`cache/how-to/purge-cache/purge-zone-versions.mdx`）。
- **认证：** `Authorization: Bearer <api_token>`，使用 **API Token**（按 zone 范围限定；不是账户级 Global API Key）。令牌需要 **Zone → Cache Purge** 权限（`fundamentals/api/reference/template.mdx`："Zone Cache Purge | Cache Purge | Zone"）。
- **负载（按缓存标签清除）：** `{"tags":["post-<qid>"]}`。这是 `purge_cache` 端点的"按标签、主机或前缀清除缓存内容"形式。
- **尽力而为，无重试。** 丢弃或被限流的请求退回自然的 `s-maxage=3600` TTL 到期 —— 绝不差于设计本就依赖的自愈。

**实现核对。** `backend/internal/service/post/purger.go` 与该契约完全一致：

| 契约要素         | `purger.go` 位置 | 代码                                                                                         |
| ---------------- | ---------------- | -------------------------------------------------------------------------------------------- |
| 端点             | `:48`            | `fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/purge_cache", cfg.ZoneID)`       |
| Bearer 认证      | `:68`            | `req.Header.Set("Authorization", "Bearer "+p.apiToken)`                                      |
| 标签负载         | `:56-57`         | `tag := "post-" + sanitizeCacheTag(qid)`; `json.Marshal(map[string][]string{"tags": {tag}})` |
| 尽力而为         | `:77-79`         | `if resp.StatusCode >= 300 { log.Printf(...) }`（吞掉，不返回错误）                          |
| 未配置时为 no-op | `:95-101`        | `APIToken` 或 `ZoneID` 为空时 `newPurger` 返回 `noopPurger`                                  |

实现是正确的。刻意**不**引入 `cloudflare-go` SDK 依赖：清除是单次尽力而为的 POST，裸 `net/http` 调用把依赖面保持在零。五种清除类型（everything / 按 URL / 按标签 / 按前缀 / 按主机名）都打到同一端点、不同的负载；markpost 只需要按标签。

<a id="cache-tags-how-purge-by-tag-works"></a>

### Cache-Tags：按标签清除如何工作

Cache-Tags 是 Cloudflare 的代理键。源站用特殊的头给响应打标签；Cloudflare 把标签与缓存对象关联，使你可以按标签批量清除而无需枚举 URL（`cache/how-to/purge-cache/purge-by-tags.mdx`）：

1. 源站在响应上设置 `Cache-Tag: post-<qid>`。
2. 流量经 Cloudflare 代理（橙云）。
3. Cloudflare 把标签与缓存内容关联，并在**投递给访客之前剥除该头**（它绝不泄漏给客户端）。
4. 按该标签的清除迫使所有携带它的内容缓存 MISS。

markpost 在 `RenderPost` 中设置 `Cache-Tag: post-<qid>`（`backend/internal/api/rest/v1/post.go:73`）。一篇文章的 HTML 与 raw 变体携带相同标签，因此一次清除同时失效两者，无论边缘持有多少个 `Accept-Encoding` 条目。**这个头是整个清除机制的硬前置** —— 没有它，按标签清除什么都匹配不到。

<a id="purge-limits-free-tier"></a>

### 清除限制（Free 套餐）

Free 套餐的清除限制，按**账户**经令牌桶模型施加（`cache/how-to/purge-cache/index.mdx`，数据在 `plans/index.json`）：

| 维度                         | Free             | Pro       | Business  | Enterprise |
| ---------------------------- | ---------------- | --------- | --------- | ---------- |
| 请求速率（标签/前缀/主机名） | **5 / 分钟**     | 5 / 秒    | 10 / 秒   | 50 / 秒    |
| 令牌桶容量                   | **25**           | 25        | 50        | 500        |
| 每次请求最大操作数           | **100**          | 100       | 100       | 100        |
| 按 URL 速率                  | **800 URL / 秒** | 1500 / 秒 | 1500 / 秒 | 3000 / 秒  |

令牌桶不是硬性的"每分钟 5 次上限"：它最多持有 25 个令牌、以每分钟 5 个补充，因此短促突发（一次至多 25 个）被吸收；只有桶空时请求才被限流。markpost 每次删除清除一个标签 —— 远低于任何限制。按假设的每天 3000 次删除，平均约 2 次/分钟，从容低于每分钟 5 次的速率。

只有**用户/管理员的主动删除**（`DeletePostByQID`）触发清除。例行清理 `PruneExpired` 刻意**不**清除：它收割的是已过期的临时内容，陈旧但无害的投递可以接受，且清理量可能很大（[`caching.zh.md`](./caching.zh.md#deletion-and-invalidation)）。

<a id="free-tier-limits-at-a-glance"></a>

## 免费版限制速览

聚合自 `plans/index.json`（cache 段）与缓存文档：

| 能力                              | 免费版                     | 说明                                                                                                                                            |
| --------------------------------- | -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| CDN 存储 / 带宽 / 请求数          | 免费，无计量配额           | 唯一明确表述："users can continue to use Cloudflare's CDN (without Cache Reserve) for free"（`cache/advanced-configuration/cache-reserve.mdx`） |
| 可缓存文件大小上限                | 512 MB                     | `cache/concepts/default-cache-behavior.mdx`                                                                                                     |
| 上传（请求体）上限                | **100 MB**                 | `plans/index.json` network.max_upload_size —— 约束创建文章的正文（平均 32 KB，约 3000 倍余量）                                                  |
| Edge Cache TTL 下限               | 2 小时                     | 这是不使用 `stale-while-revalidate` 的原因（见 [`caching.zh.md`](./caching.zh.md#http-cache-headers-in-detail)）                                |
| Browser Cache TTL 下限            | 2 分钟                     |                                                                                                                                                 |
| Cache Rules                       | 10                         |                                                                                                                                                 |
| 清除（含缓存标签在内的全部 5 种） | 有                         | Free 支持 URL/Hostname/Tag/Prefix/Everything（`plans/index.json:454`）                                                                          |
| 清除速率                          | 5/分，桶 25，100 操作/请求 | 按账户                                                                                                                                          |
| Proxy Read Timeout（→ 524）       | **120 秒**                 | Free 上不可配置；只有 Enterprise 能提高（`fundamentals/reference/connection-limits.mdx`）                                                       |
| Tiered Cache（含 Smart Topology） | 免费                       | `plans/index.json` cache.tiered_cache                                                                                                           |
| ETag / Vary                       | 有                         |                                                                                                                                                 |
| Cache Reserve                     | 付费附加                   | 未使用                                                                                                                                          |
| Cache Analytics                   | 无                         |                                                                                                                                                 |
| 按状态码缓存                      | 仅 Enterprise              |                                                                                                                                                 |
| Origin Cache Control              | 默认开启，**无法关闭**     | 因此 `s-maxage` 等被严格遵守                                                                                                                    |

**每天 100,000 次请求的限制不适用于 markpost。** 那是 **Workers**（边缘计算）的配额（`workers/platform/limits.mdx`）；markpost 不使用 Workers，只使用静态的、由头驱动的 CDN 代理路径，后者没有这样的请求上限。即使 Worker 在 fail-open 模式下超出 100k/天，"请求表现为未配置 Worker" —— 即到源站的正常代理继续进行，证实代理路径本身不受限。

<a id="caddyfile-selection-by-mode"></a>

## 按模式选择 Caddyfile

Caddyfile 因模式而异，因为 TLS 处理不同：

**家庭实验室** —— `docker/Caddyfile`（当前仓库基线），无 TLS 的 `:2053`，无 `trusted_proxies`。局域网明文 HTTP；没有反向代理链，`ClientIP()` 直接工作。

**自托管（有域名）** —— 按域名的 site 块启用 Caddy 的自动 HTTPS（Let's Encrypt）与 HTTP→HTTPS 重定向。仓库中尚无模板；目标形态：

```caddyfile
markpost.example.com {
    encode zstd gzip
    # Caddy automatically provisions a Let's Encrypt cert and redirects HTTP→HTTPS.

    handle /api/v1/* { reverse_proxy 127.0.0.1:7330 }
    handle /static/* { reverse_proxy 127.0.0.1:7330 }
    handle /swagger/* { reverse_proxy 127.0.0.1:7330 }
    handle /mpk-*    { reverse_proxy 127.0.0.1:7330 }
    handle /p-*      { reverse_proxy 127.0.0.1:7330 }
    handle           { reverse_proxy 127.0.0.1:3000 }

    log { output stderr format console }
}
```

因为 Caddy 只服务 `Host: markpost.example.com`，直连的 `http://<IP>:<port>` 请求不匹配、不会被路由到应用 —— 这就是"IP:port 不可访问"的实施方式。compose 的 `ports` 应为 ACME 暴露 80 与 443。该模式不使用 Cloudflare，因此没有 `[cloudflare]` 配置，purger 是 no-op。

**SaaS** —— `devops/ansible/templates/Caddyfile.production.j2`：`:2053` 带 `auto_https off`、`tls /app/certs/origin.pem /app/certs/origin.key`，每个 `reverse_proxy` 带 `trusted_proxies {{ cloudflare_cidrs }}` 与 `header_up X-Forwarded-For {http.request.header.CF-Connecting-IP}` 改写（见_客户端 IP 检测_）。没有按域名的 site 块（Origin CA，而非 Let's Encrypt）。它以不可变的 `Cache-Control` 服务 `/_next/static/*`、导出 HTML 带 5 分钟 `max-age`，并服务品牌化 404 页。

三种模式下 Go 二进制与配置 schema 完全相同 —— 只有 Caddyfile、DNS 与可选的 `[cloudflare]` 节不同。
