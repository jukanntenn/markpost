# Frontend Build Specification

本文档定义前端的构建配置：纯静态导出（`output: "export"`）、Turbopack 打包器、移除的服务端依赖、API 地址策略、构建产物结构。前端架构见 [architecture.md](./architecture.md)。

## 一、构建配置

### 1.1 纯静态导出

`next.config.ts`：

```ts
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export",
};

export default nextConfig;
```

`output: "export"` 产出纯静态 `out/` 目录（HTML/CSS/JS），**无 Node.js 运行时**。

> 依据（Next.js v16.1.6 源码）：`output` 选项合法值为 `'standalone' | 'export'`（`config-shared.ts:1258-1266`）；`'export'` 产出 `out` 目录（`build/index.ts:950`，默认目录名 `'out'`）。

### 1.2 为什么纯静态导出

markpost 的所有业务逻辑在后端 Go，前端不需要 API Routes / SSR 代理 / Server Actions。纯静态导出的优势：

| 优势 | 说明 |
|------|------|
| 零 Node 内存 | 不需要常驻 Node 进程（省 50-150MB+），内存留给 Go 后端 + PostgreSQL（符合 2C/2G 硬件约束） |
| 部署极简 | 静态文件丢到任意静态服务器 / CDN，`nginx root /var/www/out;` 即可 |
| 冷启动零延迟 | CDN 边缘直接返回，无 Node 冷启动 |
| 镜像复用 | 同一份 `out/` 部署到任何同源拓扑（与 docker/build-spec 的镜像复用一致） |

---

## 二、Turbopack

### 2.1 默认 bundler（Next.js 16）

Next.js 16 起，**Turbopack 已稳定且默认启用**，`next dev` 和 `next build` 都默认用它。无需 `--turbopack` flag。

> 依据（Next.js v16 源码文档 `version-16.mdx:92`）："Starting with Next.js 16, Turbopack is stable and used by default with `next dev` and `next build`. Previously you had to enable Turbopack using `--turbopack`. This is no longer necessary."

### 2.2 配置规则

- **不在 `next.config.ts` 里配 `webpack` 字段**——Next.js 16 下配了 webpack 字段会导致 Turbopack 构建直接失败（`version-16.mdx:108`，"the build will fail to prevent misconfiguration issues"）
- **不需要 `turbopack` 配置块**——零配置开箱即用（JS/TS via SWC、CSS via Lightning CSS、CommonJS/ESM）
- 若未来需要自定义 loader（如自定义文件类型），用 `turbopack.rules`（Turbopack 原生），而非 webpack loader
- 用回 Webpack 需显式 `--webpack` opt-out（本项目不用）

### 2.3 Turbopack 与纯静态导出兼容

Turbopack 与 `output: "export"` 完全兼容。Turbopack 文档未将静态导出列为限制。

---

## 三、移除的服务端依赖

纯静态导出不能用任何服务端运行时能力。以下文件移除：

| 文件 | 移除原因 | 源码依据 |
|------|---------|---------|
| `src/proxy.ts` | middleware（`NextResponse.rewrite` 转发 `/api/*` 到后端），纯静态导出不支持 middleware | `export/index.ts` 构建报错 |
| `src/app/health/route.ts` | API Route（Route Handler），纯静态导出不支持 | `export/index.ts:304-317` |
| `src/i18n/request.ts` | `getRequestConfig` + `cookies()` 是服务端 API，纯静态导出不用 | next-intl 服务端装配 |

i18n 改为纯客户端装配（`NextIntlClientProvider` 直接传 locale + messages），详见 [i18n.md](./i18n.md)。

---

## 四、API 地址策略

### 4.1 相对路径 + 反向代理

前端 API client 发**相对路径** `/api/v1/...`，部署时由反向代理（Nginx/Caddy）转发到 Go 后端。

```
浏览器 → Nginx → 静态文件（out/）+ /api/* 转发到 Go 后端
```

构建产物 `out/` 与后端地址**完全解耦**——同一份镜像部署到任何同源拓扑。

### 4.2 不用 NEXT_PUBLIC_ 环境变量

Next.js 官方文档明确（`environment-variables.mdx:152`）：`NEXT_PUBLIC_` 变量在 `next build` 时 inline 进 JS bundle，**构建后冻死，运行时不可变**。这意味着换后端地址必须重新构建，与 Docker 镜像复用冲突。所以不用 `NEXT_PUBLIC_API_BASE_URL`。

相对路径方案让后端地址由部署时的反代配置决定，与构建产物解耦。

---

## 五、构建产物

### 5.1 产物目录

`out/`（Next.js 默认，`build/index.ts:950`）。

### 5.2 路由策略：全静态预渲染

每个路由预渲染为一个 HTML 文件（`/login` → `out/login.html`，`/dashboard` → `out/dashboard.html`）。用户访问每个路由都有对应 HTML，首屏快。所有页面是 `'use client'` 客户端交互。

### 5.3 字体：next/font 自托管

`next/font/google` 的 built-in self-hosting 在纯静态导出下**完全可用**——构建时下载字体文件嵌入 `out/`，运行时不请求 Google Fonts。

> 依据（Next.js 文档 `fonts.mdx`）："built-in self-hosting for any font file... removes external network requests for improved privacy and performance."

---

## 六、package.json 脚本

```json
{
  "scripts": {
    "dev": "next dev --port ${FRONTEND_PORT:-3034}",
    "build": "next build",
    "serve": "npx serve out -p ${FRONTEND_PORT:-3034}",
    "lint": "eslint .",
    "test": "vitest",
    "test:run": "vitest run"
  }
}
```

| 脚本 | 说明 | 变化 |
|------|------|------|
| `dev` | 开发服务器（Turbopack 默认） | 去掉冗余 `--turbopack`（已是默认） |
| `build` | 生产构建（产出 `out/`） | 不变（Turbopack 默认） |
| `serve` | 本地预览静态产物 | **新增** |
| `lint` | ESLint | 不变 |
| `test` / `test:run` | Vitest | 不变 |
| ~~`start`~~ | ~~`next start`~~ | **移除**（纯静态不需要 Node 运行时） |

---

## 七、能力边界

纯静态导出下的特性可用性：

| 能力 | 可用 | 说明 |
|------|------|------|
| 静态 HTML/CSS/JS | ✅ | `out/` 全静态 |
| `next/font` 自托管字体 | ✅ | 构建时嵌入 |
| 客户端路由（`next/link`、`router`） | ✅ | SPA 式导航 |
| 客户端数据获取（fetch） | ✅ | 直连后端 API（相对路径 + 反代） |
| Turbopack 打包 | ✅ | 默认 bundler |
| middleware / Proxy | ❌ | 已移除（纯静态不支持） |
| API Routes（Route Handler） | ❌ | 已移除 |
| Server Actions | ❌ | 不使用 |
| Server Components（数据获取） | ❌ | 仅静态渲染，无运行时 |
| ISR（增量静态再生成） | ❌ | 全静态 |
| `NEXT_PUBLIC_` 运行时变量 | ⚠️ | 构建时冻死，不用 |

---

## 八、部署形态

静态文件 `out/` + 反向代理（Nginx/Caddy/CDN）：

- 静态文件由反代 / CDN 直接服务
- `/api/*` 由反代转发到 Go 后端
- **无 Node.js 运行时**，节约内存（符合 [performance-optimization.md](../backend/performance-optimization.md) 的 2C/2G 硬件约束）

详见 [cloudflare.md](../backend/cloudflare.md) 的三种部署模式（SaaS / self-hosted / homelab）和 [docker/build-specification.md](../docker/build-specification.md)。

---

## 参考

- [architecture.md](./architecture.md) — 前端架构、Provider stack、API client
- [i18n.md](./i18n.md) — i18n 纯客户端装配
- [routes.md](./routes.md) — 路由结构
