# Frontend Architecture

本文档定义前端架构：App Router 结构、组件组织、状态管理、API client、路由保护、Provider stack。构建配置（纯静态导出）见 [build.md](./build.md)；路由守卫见 [routes.md](./routes.md)；i18n 见 [i18n.md](./i18n.md)。

## Overview

前端是 Next.js 16 应用，使用 App Router + React 19，**纯静态导出**（`output: "export"`，无服务端运行时）。客户端状态用 Zustand，服务端状态用 TanStack Query。

详见 [build.md](./build.md) 的纯静态导出设计与能力边界。

## App Router Structure

页面用路由组（route groups）组织：

```
src/app/
├── layout.tsx                     根 layout（字体、providers）
├── page.tsx                       着陆页（→ /dashboard）
├── globals.css                    全局样式 + CSS 变量
├── (auth)/                        auth 路由组
│   ├── layout.tsx                 auth layout（居中、无侧边栏）+ PublicRoute 守卫
│   ├── login/page.tsx             登录页
│   └── auth/callback/page.tsx     OAuth 回调页
└── (dashboard)/                   dashboard 路由组
    ├── layout.tsx                 dashboard layout（侧边栏 + header）+ ProtectedRoute 守卫
    ├── dashboard/page.tsx         仪表盘
    ├── posts/page.tsx             文章列表
    ├── settings/page.tsx          设置
    └── admin/                     admin 区（嵌套 AdminRoute 守卫）
        ├── layout.tsx             admin layout
        ├── page.tsx               管理概览
        ├── users/page.tsx         用户管理
        ├── posts/page.tsx         文章管理
        └── delivery/
            ├── channels/page.tsx  渠道管理
            └── history/page.tsx   投递历史
```

路由组 `(auth)` 和 `(dashboard)` 各有独立 layout，不共享。

> **已移除**：`src/app/health/route.ts`（API Route，纯静态不支持）；`src/proxy.ts`（SSR 代理，纯静态不支持）。详见 [build.md](./build.md) §3。

## Component Organization

```
src/components/
├── ui/          shadcn/ui 原语（Button、Input、Dialog 等）
├── auth/        认证相关组件（AuthGate、PublicRoute、ProtectedRoute、AdminRoute、route-configs）
├── layout/      布局组件（Sidebar、Header、DashboardLayout、AdminLayout）
├── login/       登录页专用组件（LoginPage、LoginCallbackPage）
├── dashboard/   dashboard 组件
├── admin/       admin 组件
├── posts/       文章相关组件
├── settings/    设置组件
└── providers/   Context providers（QueryProvider）
```

## State Management

### Server State — TanStack Query

API 数据获取由 TanStack Query 管理。`QueryProvider` 包裹应用，提供 query client。

### Client State — Zustand（认证状态）

认证状态用 Zustand + `persist` 中间件管理，持久化到 localStorage：

```typescript
// src/stores/auth.ts
export const useAuthStore = create<AuthState>()(
  persist(
    (set, _get) => ({
      token: null,
      refreshToken: null,
      user: null,
      _hasHydrated: false,
      // ...
    }),
    {
      name: "markpost_auth",
      partialize: ({ token, refreshToken, user }) => ({ token, refreshToken, user }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    }
  )
);
```

状态持久化到 localStorage（key = `markpost_auth`）。水合状态用 `_hasHydrated` 跟踪（防闪烁，见 [routes.md](./routes.md) 水合处理）。

token 存储的安全考量见 [auth.md](../auth.md) §6。

## API Client

API client 在 `src/lib/api/base.ts`，提供泛型 `request<T>()` 函数：

1. 从 Zustand store 读 access token
2. 设置 `Authorization: Bearer <token>` header
3. **携带 `Accept-Language: <当前 locale>` header**（后端据此返回对应语言错误消息，见 [i18n.md](./i18n.md)）
4. 发送请求（**相对路径** `/api/v1/...`，由反向代理转发到后端）
5. 401 时自动尝试 token refresh（单飞去重）
6. refresh 成功 → 用新 token 重试原请求
7. refresh 失败 → logout → 重定向登录

> **直连后端**（不再有 SSR 代理）：前端发相对路径，部署时由 Nginx/Caddy 反代到 Go 后端。详见 [build.md](./build.md) §4。

自动刷新机制详见 [auth.md](../auth.md) §6.3。

## Route Protection

路由保护由客户端守卫组件实现。详见 [routes.md](./routes.md)。

要点：客户端守卫**仅控制 UX**，安全保障在后端 API 层（JWT + Admin 中间件）。

## Provider Stack

根 layout 包裹以下 providers（外到内）：

1. `LocaleProvider` — next-intl locale context（**纯客户端自举**，不接收 serverLocale/serverMessages，见 [i18n.md](./i18n.md)）
2. `QueryProvider` — TanStack Query client
3. `ThemeProvider` — next-themes（dark/light/system）
4. `ToastProvider` — Toast 通知 context

> root layout **不调用服务端 i18n API**（`getLocale()` / `getMessages()`）。纯静态导出下这些不可用。`LocaleProvider` 完全自举：初始 locale = `en`，hydration 后从 localStorage 读取并动态加载 messages。
