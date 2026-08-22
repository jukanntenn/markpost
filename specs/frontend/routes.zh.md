# 前端路由与访问控制

[English](routes.md) | 中文

本文档定义前端路由表、守卫架构、安全边界。前端是纯静态客户端（见 [build.zh.md](./build.zh.md)），守卫是客户端守卫。

<a id="route-table"></a>

## 路由表

| 路径                       | 路由组            | 守卫           | 不满足条件时                               | 页面                         |
| -------------------------- | ----------------- | -------------- | ------------------------------------------ | ---------------------------- |
| `/`                        | —                 | —              | —                                          | 着陆页（见下）               |
| `/login`                   | (auth)            | PublicRoute    | 已认证 → `/dashboard`                      | 登录页（密码 + GitHub 按钮） |
| `/auth/callback`           | (auth)            | PublicRoute    | 已认证 → `/dashboard`                      | OAuth 回调页                 |
| `/dashboard`               | (dashboard)       | ProtectedRoute | 未认证 → `/login`                          | 仪表盘                       |
| `/posts`                   | (dashboard)       | ProtectedRoute | 未认证 → `/login`                          | 文章列表                     |
| `/settings`                | (dashboard)       | ProtectedRoute | 未认证 → `/login`                          | 设置                         |
| `/admin`                   | (dashboard)/admin | AdminRoute     | 未认证 → `/login`；非 admin → `/dashboard` | 管理概览                     |
| `/admin/users`             | (dashboard)/admin | AdminRoute     | 同上                                       | 用户管理                     |
| `/admin/posts`             | (dashboard)/admin | AdminRoute     | 同上                                       | 文章管理                     |
| `/admin/delivery/channels` | (dashboard)/admin | AdminRoute     | 同上                                       | 渠道管理                     |
| `/admin/delivery/history`  | (dashboard)/admin | AdminRoute     | 同上                                       | 投递历史                     |

OAuth 流程用同页重定向，唯一的回调路由就是这个 `/auth/callback` —— 不带 provider 段（见 [auth.zh.md](../auth.zh.md) §3.6）。健康检查由后端 `/api/v1/health` 端点提供 —— 纯静态导出不含 API Route（[build.zh.md](./build.zh.md) §3）。

<a id="landing-page-"></a>

## 着陆页（`/`）

着陆页是纯静态营销页（`components/landing/`），无守卫、无数据请求。行为约定：

- 未登录：Masthead 右上为描边样式的「登录」；Hero 与 Colophon 的主 CTA 为「开始使用」→ `/login`。
- 已登录（`useAuthReady` 水合后判定，不强制跳转、无闪烁重定向）：按钮文案变为「打开控制台」→ `/dashboard`。
- 页面结构：Masthead（§00）→ Hero 对开页（§01）→ 原理（§02）→ 产物（§03）→ 投递（§04）→ 开源（§05）→ Colophon 页脚（§06）。每节 = 一个主张 + 一件物证 + 可验证的事实；§03 的文章页物证复刻 `backend/templates/post.html` 的冷色 slate 样式，与 Ember 暖色纸面刻意保持材质差异。
- 文案全部走 `landing.*` 命名空间（en / zh-Hans / zh-Hant / ja），物证中的示例文章（`landing.sample.*`）在 hero、§03、§04 之间共享同一篇，保持叙事连贯。

<a id="guard-architecture"></a>

## 守卫架构

<a id="declarative-guards"></a>

### 声明式守卫

守卫架构是声明式的、可组合的 —— 一个执行器之上的三个组件：

```
AuthGate (the executor, consumes useAuthGuard)
├── PublicRoute     — the (auth) group
├── ProtectedRoute  — the (dashboard) group
└── AdminRoute      — nested under (dashboard)/admin
```

<a id="route-configsts-guard-configuration"></a>

### route-configs.ts（守卫配置）

守卫配置是纯函数，声明每类路由的判定逻辑：

```typescript
export const publicRoute = {
  shouldShow: (isAuth: boolean) => !isAuth,
  redirectPath: "/dashboard",
  showSpinnerWhen: (isAuth: boolean) => !isAuth,
};

export const protectedRoute = {
  shouldShow: (isAuth: boolean) => isAuth,
  redirectPath: "/login",
};

export const adminRoute = {
  shouldShow: (isAuth: boolean, isAdmin: boolean) => isAuth && isAdmin,
  redirectPath: "/dashboard",
};
```

<a id="authgate-the-executor"></a>

### AuthGate（执行器）

```tsx
function AuthGate({ shouldShow, showSpinnerWhen, redirectPath, children }) {
  const { hasHydrated, isAuthenticated, isAdmin } = useAuthGuard({
    shouldRedirect: (isAuth, isAdm) => !shouldShow(isAuth, isAdm),
    redirectPath,
  });

  if (!hasHydrated) return <PageSpinner />;
  if (!shouldShow(isAuthenticated, isAdmin)) {
    return showSpinnerWhen?.(isAuthenticated, isAdmin) ? <PageSpinner /> : null;
  }
  return <>{children}</>;
}
```

<a id="applied-at-the-layout-level"></a>

### 布局层级应用

守卫在路由组的 `layout.tsx` 中应用（布局层级守卫），admin 在 dashboard 内嵌套：

```
app/(auth)/layout.tsx          → <PublicRoute>{children}</PublicRoute>
app/(dashboard)/layout.tsx     → <ProtectedRoute><DashboardLayout>{children}</DashboardLayout></ProtectedRoute>
app/(dashboard)/admin/layout.tsx → <AdminRoute><AdminLayout>{children}</AdminLayout></AdminRoute>
```

<a id="guard-behavior"></a>

### 守卫行为

**PublicRoute**（(auth) 组：login、callback）：

- 水合中 → 渲染 PageSpinner
- 已认证 → `router.replace("/dashboard")`
- 未认证 → 渲染 children

**ProtectedRoute**（(dashboard) 组）：

- 水合中 → 渲染 PageSpinner
- 未认证（水合后）→ `router.replace("/login")`
- 已认证 → 渲染 children

**AdminRoute**（(dashboard)/admin 嵌套）：

- 未认证 → `router.replace("/login")`
- 已认证但非 admin → `router.replace("/dashboard")`
- admin → 渲染 children

<a id="hydration-handling"></a>

### 水合处理

Zustand persist 从 localStorage 恢复是异步的。用 `_hasHydrated` 标志防止水合前把默认空状态（`token=null`）误判为「未认证」导致闪烁跳转：

```typescript
onRehydrateStorage: () => (state) => {
  state?.setHasHydrated(true);
};
```

守卫在水合完成前渲染 PageSpinner，水合后根据真实认证状态决定渲染 / 重定向。

<a id="security-boundary"></a>

## 安全边界

**客户端守卫仅控制 UX（用户体验），不提供安全保障。**

纯静态导出（`output: "export"`）不能用 Next.js 的 middleware / Proxy / Server Component 做服务端路由保护 —— 这些都是服务端运行时能力，纯静态前端不可用（见 [build.zh.md](./build.zh.md) §7）。客户端守卫是**唯一选择**。

> Next.js 官方文档（`authentication.mdx:1447`）："client-side UI restrictions alone are not sufficient for security." —— 这句警告的语境是**有 Server Actions / API Routes 的全栈 Next.js 应用**（客户端 return null 不能阻止用户直接调用 Server Action）。

**对 markpost 不适用**，因为：

- 前端纯静态，**零 Server Actions / API Routes**（这些都在后端 Go）
- 所有数据访问通过后端 REST API，后端有 **JWT 认证 + Admin 中间件**做权威校验
- 客户端守卫被绕过（篡改 localStorage）→ 前端可能渲染页面骨架，但**所有数据请求被后端 401/403 拒绝** → 页面是空的，无数据泄露

**安全保障在后端 API 层**（见 [auth.zh.md](../auth.zh.md)、[api-design.zh.md](../api-design.zh.md) §5）。

<a id="oauth-callback-page"></a>

## OAuth 回调页

`/auth/callback` 页面（(auth) 路由组，PublicRoute 守卫）处理 OAuth 回调。完整流程见 [auth.zh.md](../auth.zh.md) §3、§7。

职责：从 URL query 读 code + state → 前端二次校验 state（vs sessionStorage）→ POST `/api/v1/oauth/login` → setAuth → `router.replace('/dashboard')`。所有失败路径 `router.replace('/login')`。

<a id="references"></a>

## 参考

- [auth.zh.md](../auth.zh.md) —— 认证流程、OAuth callback 逻辑
- [build.zh.md](./build.zh.md) —— 纯静态导出、能力边界
- [architecture.zh.md](./architecture.zh.md) —— 前端架构、Provider stack
