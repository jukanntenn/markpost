# 压缩与页面权重

[English](compression.md) | 中文

读路径的传输最小化：Caddy 上的 HTTP 压缩、CSS 外置 + 最小化 + 内容哈希指纹，以及渲染期 HTML 最小化。在 SaaS 参考实例的 3 Mbps / 1 TB 预算下，省下的每个字节同时是链路余量与配额余量（见 [`caching.zh.md`](./caching.zh.md) _硬件包络_），这就是字节削减在本设计中压倒 CPU 优化的原因。决策记录（zstd 而非 brotli、不做预压缩、无 Node 工具链）见[性能优化 MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.zh.md)。

<a id="why-page-weight-dominates"></a>

## 页面权重为何主导

文章页在字节层面的实测拆解（真实 `post.html` + 典型 1.8 KB 正文）：

| 组件                  |  未压缩 |    gzip -9 |
| --------------------- | ------: | ---------: |
| `post.html` 模板总计  |  8789 B |          — |
| 内联 `<style>` 块     |  8073 B | **1798 B** |
| HTML 骨架（不含 CSS） |   247 B |     ~120 B |
| 典型 1.8 KB 正文      |  1850 B |       90 B |
| 整页（内联 CSS）      | 10639 B |     2183 B |
| 整页（CSS 外置）      |  2099 B |  **283 B** |

外置 + 压缩 + 缓存 CSS 把一次重复访问从约 10 KB（压缩后、内联）降到几百字节的正文，因为压缩后的 CSS 被取回一次，随后在整个站点范围内由浏览器缓存服务。文章正文受 `[post] body_max_bytes` 封顶（生产配置模板中为 262144，默认 32768）；压缩性的实测见 `scripts/loadtest/CAPACITY_REPORT.md`。

<a id="http-compression-zstd--gzip-via-caddy"></a>

## HTTP 压缩：经 Caddy 的 zstd + gzip

仓库中的每个 Caddyfile（`docker/Caddyfile*`、`devops/ansible/templates/Caddyfile*`）都携带 `encode zstd gzip`；Caddy 按 `Accept-Encoding` 逐请求选择。

- **zstd（Zstandard）** 以约 3 倍的速度达到或超过 gzip 的压缩比，并在同等 CPU 成本下对文本/HTML 通常产生比 gzip 小 5–10% 的输出。
- **gzip** 作为通用回退保留，面向不声明 zstd 的客户端。
- **brotli** 与**服务端预压缩**（Caddy 的 `precompressed` 指令）被拒绝：brotli 需要非默认的 Caddy 构建却只有边际收益，预压缩只帮助静态资源 —— 这里唯一的静态资源（带指纹的 CSS）压缩后已约 1.8 KB。动态 HTML/raw 响应无论如何都无法预压缩。

`Vary: Accept-Encoding`（压缩时由 Caddy 设置，其余情况由 handler 显式设置）让 CDN 的 gzip 与 zstd 变体各占独立的缓存条目。`304` 响应没有正文，也不被压缩。

<a id="css-externalization-minification-and-fingerprinting"></a>

## CSS 外置、最小化与指纹

CSS 是页面上最大的单项字节成本，因此被整体从页面中抽出：

1. **构建期抽取并最小化。** CSS 源位于 `backend/templates/post.css`；`cmd/buildcss`（由 `go generate ./...` 调用）用 `github.com/tdewolff/minify/v2` 最小化它（事实上的 Go 最小化器 —— 纯 Go、无 Node 工具链；该 CSS 是单一自包含文件，没有 `@import` 或 `url()` 引用，因此不需要打包器），计算最小化输出的 `xxhash64`，写出 `backend/internal/web/post.<hash>.css` 与生成的 `backend/internal/web/csshash.go`（`var CSSHash = "<hash>"`）。CSS 文件经 `go:embed` 嵌入二进制。
2. **文件名按内容寻址。** 模板引用 `<link rel="stylesheet" href="/static/post.{{.CSSHash}}.css">`。CSS 升级时新的最小化字节哈希不同 → 不同的文件名 → 不同的 URL；每个浏览器都取回新 CSS，因为它位于一个从未见过的 URL；`Cache-Control: public, max-age=31536000, immutable` 是严格正确的，因为 URL 随内容变化而变化（MDN 的 "cache busting"）。HTML 的一小时 CDN TTL 自然轮换到新外壳 —— `<link>` href 的变化改变渲染出的 HTML，输出哈希 ETag 自动捕捉到它 —— 且静态资源不需要任何清除 API：URL _就是_版本。
3. **从内存服务。** 一个 gin 路由（`v1.StaticCSS`）以不可变缓存头在 `GET /static/:filename` 服务嵌入字节 —— 运行期无文件系统依赖，在每个部署语境中行为一致。

<a id="html-minification-at-render-time"></a>

## 渲染期 HTML 最小化

渲染出的 HTML 响应在渲染管线（`internal/service/post/post.go`）内用 `tdewolff/minify` 的 HTML 最小化器最小化：一个 `minify.Minifier` 在 `NewService` 中与 goldmark 实例、bluemonday 策略一起一次性构建，跨 goroutine 复用（它是并发安全的）。存入渲染缓存并用于 ETag 哈希的是最小化后的 HTML，因此 ETag 与服务的字节永远一致。最小化从外壳剥除空白、注释与冗余标签；已经净化过的正文可以安全最小化。周边的渲染缓存机制见 [`caching.zh.md`](./caching.zh.md)。
