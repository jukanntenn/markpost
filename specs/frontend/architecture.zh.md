# 前端架构

[English](architecture.md) | 中文

本文档定义前端架构：App Router 结构、组件组织、状态管理、API client、路由保护、Provider stack。构建配置（纯静态导出）见 [build.zh.md](./build.zh.md)；路由守卫见 [routes.zh.md](./routes.zh.md)；i18n 见 [i18n.zh.md](./i18n.zh.md)。

<a id="overview"></a>

## 概览

前端是基于 App Router + React 19 的 Next.js 16 应用，**纯静态导出**（`output: "export"`，无服务端运行时）。客户端状态用 Zustand，服务端状态用 TanStack Query。

纯静态导出设计与能力边界见 [build.zh.md](./build.zh.md)。

<a id="app-router-structure"></a>

## App Router 结构

页面用路由组（route groups）组织：

```
src/app/
├── layout.tsx                     root layout (fonts, providers)
├── page.tsx                       landing page (CTA → /dashboard)
├── globals.css                    global styles + CSS variables
├── (auth)/                        auth route group
│   ├── layout.tsx                 auth layout (centered, no sidebar) + PublicRoute guard
│   ├── login/page.tsx             login page
│   └── auth/callback/page.tsx     OAuth callback page
└── (dashboard)/                   dashboard route group
    ├── layout.tsx                 dashboard layout (sidebar + header) + ProtectedRoute guard
    ├── dashboard/page.tsx         dashboard
    ├── posts/page.tsx             post list
    ├── settings/page.tsx          settings
    └── admin/                     admin area (nested AdminRoute guard)
        ├── layout.tsx             admin layout
        ├── page.tsx               admin overview
        ├── users/page.tsx         user management
        ├── posts/page.tsx         post management
        └── delivery/
            ├── channels/page.tsx  channel management
            └── history/page.tsx   delivery history
```

路由组 `(auth)` 与 `(dashboard)` 各有独立 layout，互不共享。

> 纯静态导出不含 API Route 与 SSR 代理（无 `route.ts` / `proxy.ts` 文件）。健康检查由后端 `/api/v1/health` 端点提供，`/api/*` 反代由 Caddy 完成（开发环境为 `next.config.ts` rewrites）。详见 [build.zh.md](./build.zh.md) §3。

<a id="component-organization"></a>

## 组件组织

```
src/components/
├── ui/          shadcn/ui primitives (Button, Input, Dialog, ...)
├── auth/        auth components (AuthGate, PublicRoute, ProtectedRoute, AdminRoute, route-configs)
├── layout/      layout components (Sidebar, Header, DashboardLayout, AdminLayout)
├── login/       login-page components (LoginPage, LoginCallbackPage)
├── dashboard/   dashboard components
├── admin/       admin components
├── posts/       post components
├── settings/    settings components
└── providers/   context providers (QueryProvider)
```

<a id="state-management"></a>

## 状态管理

<a id="server-state--tanstack-query"></a>

### 服务端状态 —— TanStack Query

API 数据获取由 TanStack Query 管理。`QueryProvider` 包裹应用并提供 query client。

<a id="client-state--zustand-authentication-state"></a>

### 客户端状态 —— Zustand（认证状态）

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
      partialize: ({ token, refreshToken, user }) => ({
        token,
        refreshToken,
        user,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    },
  ),
);
```

状态持久化到 localStorage（key = `markpost_auth`）。水合状态用 `_hasHydrated` 标志跟踪（防闪烁，见 [routes.zh.md](./routes.zh.md) 的水合处理）。

token 存储的安全考量见 [auth.zh.md](../auth.zh.md) §6。

<a id="api-client"></a>

## API client

API client 位于 `src/lib/api/base.ts`，提供泛型 `request<T>()` 函数：

1. 从 Zustand store 读取 access token
2. 设置 `Authorization: Bearer <token>` header
3. **携带 `Accept-Language: <当前 locale>`** —— 后端据此返回对应语言的错误消息（见 [i18n.zh.md](./i18n.zh.md)）
4. 发送请求（**相对路径** `/api/v1/...`，由反向代理转发到后端）
5. 401 时自动尝试刷新令牌（单飞去重）
6. 刷新成功 → 用新 token 重试原请求
7. 刷新失败 → logout → 重定向登录

> **直连后端**（无 SSR 代理）：前端发相对路径，部署时由 Nginx/Caddy 反代到 Go 后端。详见 [build.zh.md](./build.zh.md) §4。

自动刷新机制见 [auth.zh.md](../auth.zh.md) §6.3。

<a id="route-protection"></a>

## 路由保护

路由保护由客户端守卫组件实现，见 [routes.zh.md](./routes.zh.md)。

要点：客户端守卫**仅控制 UX**；安全保障位于后端 API 层（JWT + Admin 中间件）。

<a id="provider-stack"></a>

## Provider stack

根 layout 包裹以下 providers（外到内）：

1. `LocaleProvider` —— next-intl locale context（**纯客户端自举**，不接收 serverLocale/serverMessages，见 [i18n.zh.md](./i18n.zh.md)）
2. `QueryProvider` —— TanStack Query client
3. `ThemeProvider` —— next-themes（dark/light/system）
4. `ToastProvider` —— Toast 通知 context

> 根 layout 不调用服务端 i18n API（`getLocale()` / `getMessages()`）—— 它们是 server-only 的，纯静态导出下不可用。`LocaleProvider` 完全在客户端自举：初始 locale 为 `en`，水合后从 localStorage 读取并动态加载 messages chunk。
