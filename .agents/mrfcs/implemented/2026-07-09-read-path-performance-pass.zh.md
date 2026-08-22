# MRFC: Read-path performance pass

Status: implemented

[English](2026-07-09-read-path-performance-pass.md) | 中文

## Problem

读路径 —— 产品热路径，目标是在一台 2 核 / 2 GB / 3 Mbps、每月 1 TB 配额的 VPS 上支撑每秒几百次读 —— 在任何一层上都以零缓存服务：每个 `GET /:qid` 都做一次全新的 Postgres 读取、goldmark 渲染、bluemonday 消毒和模板执行；响应不带压缩发出，没有 `Cache-Control`/`ETag`；页面约 8 KB 的内联 `<style>` 块在每次浏览时重传；Postgres 连接池无界且未调优；而一个全局限流器把读节流与写节流耦合在一起。以 375 KB/s 的出口带宽，源站只能承载约每秒 25 个压缩响应 —— 比目标负载低两个数量级 —— 因此 SaaS 参考实例不经过系统性重设计就无法在规模下运转。

## Decision

这一轮优化（自 `ed16a32` 起落地，2026-07-09）把传输最小化当作主导目标，在应用代码与部署模板中发布了：三个按失效能力区分 TTL 的缓存层 —— 浏览器 `max-age=300`、CDN `s-maxage=3600`（Cloudflare 免费档），以及一个无界的进程内渲染缓存，键为 `qid:buildID:{html,raw}`（singleflight 折叠、ristretto 承载并带 TinyLFU 准入、可在 `[render]` 配置） —— 配上对**渲染后输出**做哈希的 ETag（压缩后 HTML / 原始字符串的 xxhash64），使渲染器、模板或 CSS 的升级自动改变 ETag；删除端点（`DELETE /api/v1/posts/:id` 及管理员变体）同步失效源站缓存，并经 `Purger` 接口异步发出尽力而为的 Cloudflare cache-tag 清除（未配置 Cloudflare 时为 no-op），而 `PruneExpired` 从不清除；可被边缘缓存的 404（`max-age=60, s-maxage=60`），用于吸收 QID 枚举探测；Caddy `encode zstd gzip`；CSS 抽取到 `templates/post.css`，由 `cmd/buildcss` 压缩并加内容哈希指纹（`go:embed` + `csshash.go`，以 `/static/post.<hash>.css` 提供并带 `immutable`），HTML 则在渲染时由同一个 `tdewolff/minify` 库压缩；四个 tollbooth 限流器 —— 读（每 IP 100/s）、公开写（每 `user_id` 10/min + 1000/day）、已认证写（每 `user_id` 30/min）、登录（每 IP 5/min） —— 以 gin `ClientIP()` 沿 XFF 链解析，Caddy `trusted_proxies` 钉在 Cloudflare CIDR 上；以及 Postgres 调优 —— 连接池 25/10/30 分钟，五个 GUC（`shared_buffers=256MB`、`effective_cache_size=1GB`、`maintenance_work_mem=128MB`、`max_connections=50`、`synchronous_commit=off`）由部署模板应用，lz4 TOAST 压缩在版本化迁移中声明，以及一条到兄弟 Postgres 容器的 Unix socket 连接。Specs：[`specs/backend/caching.md`](../../../specs/backend/caching.zh.md)、[`compression.md`](../../../specs/backend/compression.zh.md)、[`rate-limiting.md`](../../../specs/backend/rate-limiting.zh.md)、[`postgres-tuning.md`](../../../specs/backend/postgres-tuning.zh.md)。帖子一次写入且不可变 —— `UpdatePost` 路径已从契约中移除，正是这一点把缓存失效折叠为删除事件。

## Alternatives considered

**在 SaaS 规模下直接由源站服务。** 在 3 Mbps 下数学上不可能；CDN 是参考实例的前置条件。计量型 CDN（AWS CloudFront、Bunny）被拒绝，因为按字节计费把 1 TB 配额以成本上限的形式重新施加（按预计的 7.8 TB 约每月 $660），而 Cloudflare 免费档不计量。

**对输入做 ETag（`sha256(body+title+cssHash+templateVersion)`）、仅对 body 哈希、或干脆用 SHA-256。** 输入哈希漏掉渲染器升级，并经 `304` 续签陈旧外壳；SHA-256 的密码学强度对缓存校验纯属多余，且比 xxhash 慢约 20 倍，而 xxhash 经 ristretto 传递性地到来。

**统一的单一 TTL、`stale-while-revalidate`、或 HTML 上的 `immutable`。** 统一 TTL 要么让升级传播得太慢，要么放弃 CDN 收益；SWR 在有 `s-maxage` 时是文档明确的 no-op（RFC 9111 proxy-revalidate 语义），启用它就意味着为换取 ≥2 小时的免费档 Edge Cache TTL 而放弃 `s-maxage`；在一个可删除、非内容寻址的 URL 上用 `immutable` 事实上就是错的，还会向浏览器隐藏删除。`"Purge Everything"` 作为失效机制被认定为必然惊群而拒绝；完全避开清除则在 Cloudflare 向免费档开放全部清除方法（2025-04）之后被拒绝 —— markpost 的删除量远在其限额之下。

**brotli、Caddy `precompressed`、为 CSS 引入 Vite/Node 或 esbuild 工具链。** brotli 需要自定义 Caddy 构建，收益边际；预压缩只帮到那一个约 1.8 KB 的静态资产；为单个自包含 CSS 文件引入 Node 工具链没有必要（若 CSS 有朝一日需要打包，esbuild 仍是记录在案的升级路径）。

**渲染缓存用朴素 LRU / `golang-lru` / `bigcache` / `freecache`，或不含 singleflight 的 ristretto。** 朴素 LRU 在 Zipf 型读取下易受扫描污染；bigcache/freecache 面向大量小而均匀的条目，而非少数大 HTML 块；ristretto 单独能吸收重复但吸收不了踩踏 —— singleflight 才是承重的另一半（冷突发、发布窗口、重启）。

**静态文件物化或渲染后 HTML 的数据库列。** 到 10 000 篇帖子时，进程内缓存已把源站渲染成本压到近零；两个方案都为可忽略的收益加上写入时渲染、跨进程协调或 schema 耦合（每次渲染器升级都要重渲染），以及存储。

**单一全局限流器；按 `post_key` 做 L2 键控；按日期键控的每日计数器。** 全局限流器把读节流与写节流耦合在一起（最初的缺陷）；L2 按 `post_key` 键控会让轮换绕开限额；UTC 午夜翻转的日期计数器重复了令牌桶以 `1000/86400` 每秒速率表达的东西。

**用 `gin.PlatformCloudflare` 或 tollbooth `SetIPLookup` 取客户端 IP。** 两者都信任一个可伪造的头且无 CIDR 校验；XFF 遍历配合 Caddy 侧的 `trusted_proxies` 在 TCP 层校验对端，并覆写非受信对端的 XFF —— 即便源站防火墙被伪造源 IP 绕过也能自保。

**专用数据库服务器（shared_buffers 占内存的 25%）、markpost 容器内的 Postgres、或裸金属部署。** 这台机器不是专用数据库服务器；容器内 Postgres 把故障耦合在一起并使备份/扩缩复杂化；裸金属省下个位数的微秒，代价是放弃可复现镜像、一条命令的自托管安装和多架构 CI —— 真正的约束是 3 Mbps 链路，任何容器层都碰不到它。

**应用层把帖子 body gzip 进 BYTEA。** 压缩比约 4 倍，对 TOAST-lz4 的约 3 倍，但它把解压挪进每次读取的 Go 热路径，并改变列类型。保留为磁盘压力超出约 11 GB 估计时的第一升级步骤，与冷层卸载到对象存储并列。

## Consequences

源站以约每月 $0.20 的边际成本撑过其负载包络，升级在一小时内传播而无需运维者动手，删除在 CDN 上近乎即时。接受的义务：缓存正确性依赖输出哈希 ETag 与删除驱动的失效（任何新的响应变体都必须对它所服务的内容做哈希），Caddyfile 中的 Cloudflare CIDR 列表是运维者维护的职责，GUC/lz4/socket 各项是部署模板事务、靠人工而非 Go 测试验证，而负载测试套件（`scripts/loadtest/`、容量报告）是包络仍然成立的常备证据。灾备层（备份工具）刻意不在本记录的范围内，活在[灾难恢复 MRFC](../proposed/2026-07-09-wal-archival-disaster-recovery.zh.md)里。
