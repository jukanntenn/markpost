# Frontend Routes & Access Control

本文档定义前端路由表、权限守卫架构、安全边界。前端是纯静态客户端（见 [build.md](./build.md)），守卫是客户端守卫。

## Route Table

| Path | 路由组 | Guard | 不满足条件时 | 页面 |
|------|--------|-------|------------|------|
| `/` | — | — | → `/dashboard` | 着陆页（重定向） |
| `/login` | (auth) | PublicRoute | 已认证 → `/dashboard` | 登录页（密码 + GitHub 按钮） |
| `/auth/callback` | (auth) | PublicRoute | 已认证 → `/dashboard` | OAuth 回调页 |
| `/dashboard` | (dashboard) | ProtectedRoute | 未认证 → `/login` | 仪表盘 |
| `/posts` | (dashboard) | ProtectedRoute | 未认证 → `/login` | 文章列表 |
| `/settings` | (dashboard) | ProtectedRoute | 未认证 → `/login` | 设置 |
| `/admin` | (dashboard)/admin | AdminRoute | 未认证 → `/login`；非 admin → `/dashboard` | 管理概览 |
| `/admin/users` | (dashboard)/admin | AdminRoute | 同上 | 用户管理 |
| `/admin/posts` | (dashboard)/admin | AdminRoute | 同上 | 文章管理 |
| `/admin/delivery/channels` | (dashboard)/admin | AdminRoute | 同上 | 渠道管理 |
| `/admin/delivery/history` | (dashboard)/admin | AdminRoute | 同上 | 投递历史 |

### 路由变更说明

| 变更 | 现状 | 目标 | 原因 |
|------|------|------|------|
| OAuth callback | `/auth/github/callback` + `/auth`（两个） | `/auth/callback`（一个） | OAuth 改同页重定向，单一 callback，不带 provider（见 [auth.md](../auth.md) §3.6） |
| health API route | `/health/route.ts` | 移除 | 纯静态导出不支持 API Route（见 [build.md](./build.md) §3） |

---

## Guard Architecture

### 声明式守卫

守卫架构是声明式的、可组合的，三层组件：

```
AuthGate（执行器，消费 useAuthGuard）
├── PublicRoute     — (auth) 组
├── ProtectedRoute  — (dashboard) 组
└── AdminRoute      — (dashboard)/admin 嵌套
```

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

### 布局层级应用

守卫在路由组的 `layout.tsx` 中应用（布局层级守卫），admin 在 dashboard 内嵌套：

```
app/(auth)/layout.tsx          → <PublicRoute>{children}</PublicRoute>
app/(dashboard)/layout.tsx     → <ProtectedRoute><DashboardLayout>{children}</DashboardLayout></ProtectedRoute>
app/(dashboard)/admin/layout.tsx → <AdminRoute><AdminLayout>{children}</AdminLayout></AdminRoute>
```

### Guard 行为

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

### 水合处理

Zustand persist 从 localStorage 恢复是异步的。用 `_hasHydrated` 标志防止水合前用默认空状态（`token=null`）误判"未认证"导致闪烁跳转：

```typescript
onRehydrateStorage: () => (state) => {
  state?.setHasHydrated(true);
}
```

守卫在水合完成前渲染 PageSpinner，水合后根据真实认证状态决定渲染 / 重定向。

---

## Security Boundary

**客户端守卫仅控制 UX（用户体验），不提供安全保障。**

纯静态导出（`output: "export"`）不能用 Next.js 的 middleware / Proxy / Server Component 做服务端路由保护——这些都是服务端运行时能力，纯静态前端不可用（见 [build.md](./build.md) §7）。客户端守卫是**唯一选择**。

> Next.js 官方文档（`authentication.mdx:1447`）："client-side UI restrictions alone are not sufficient for security."——这句警告的语境是**有 Server Actions / API Routes 的全栈 Next.js 应用**（客户端 return null 不能阻止用户直接调用 Server Action）。

**对 markpost 不适用**，因为：
- 前端纯静态，**零 Server Actions / API Routes**（这些都在后端 Go）
- 所有数据访问通过后端 REST API，后端有 **JWT 认证 + Admin 中间件**做权威校验
- 客户端守卫被绕过（篡改 localStorage）→ 前端可能渲染页面骨架，但**所有数据请求被后端 401/403 拒绝** → 页面是空的，无数据泄露

**安全保障在后端 API 层**（见 [auth.md](../auth.md)、[api-design.md](../api-design.md) §5）。

---

## OAuth Callback Page

`/auth/callback` 页面（(auth) 路由组，PublicRoute 守卫）处理 OAuth 回调。完整流程见 [auth.md](../auth.md) §3、§7。

职责：从 URL query 读 code + state → 前端二次校验 state（vs sessionStorage）→ POST `/api/v1/oauth/login` → setAuth → `router.replace('/dashboard')`。所有失败路径 `router.replace('/login')`。

---

## 参考

- [auth.md](../auth.md) — 认证流程、OAuth callback 逻辑
- [build.md](./build.md) — 纯静态导出、能力边界
- [architecture.md](./architecture.md) — 前端架构、Provider stack
