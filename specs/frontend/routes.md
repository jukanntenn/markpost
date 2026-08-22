# Frontend Routes & Access Control

English | [中文](routes.zh.md)

This document defines the frontend route table, the guard architecture, and the security boundary. The frontend is a pure static client (see [build.md](./build.md)); the guards are client-side guards.

## Route Table

| Path                       | Route group       | Guard          | When the condition fails                             | Page                             |
| -------------------------- | ----------------- | -------------- | ---------------------------------------------------- | -------------------------------- |
| `/`                        | —                 | —              | —                                                    | landing page (below)             |
| `/login`                   | (auth)            | PublicRoute    | authenticated → `/dashboard`                         | login (password + GitHub button) |
| `/auth/callback`           | (auth)            | PublicRoute    | authenticated → `/dashboard`                         | OAuth callback                   |
| `/dashboard`               | (dashboard)       | ProtectedRoute | unauthenticated → `/login`                           | dashboard                        |
| `/posts`                   | (dashboard)       | ProtectedRoute | unauthenticated → `/login`                           | post list                        |
| `/settings`                | (dashboard)       | ProtectedRoute | unauthenticated → `/login`                           | settings                         |
| `/admin`                   | (dashboard)/admin | AdminRoute     | unauthenticated → `/login`; non-admin → `/dashboard` | admin overview                   |
| `/admin/users`             | (dashboard)/admin | AdminRoute     | same as above                                        | user management                  |
| `/admin/posts`             | (dashboard)/admin | AdminRoute     | same as above                                        | post management                  |
| `/admin/delivery/channels` | (dashboard)/admin | AdminRoute     | same as above                                        | channel management               |
| `/admin/delivery/history`  | (dashboard)/admin | AdminRoute     | same as above                                        | delivery history                 |

The OAuth flow uses a same-page redirect with this single `/auth/callback` route — no provider segment (see [auth.md](../auth.md) §3.6). Health checks come from the backend `/api/v1/health` endpoint — a static export ships no API Routes ([build.md](./build.md) §3).

## Landing Page (`/`)

The landing page is a purely static marketing page (`components/landing/`) with no guard and no data requests. Behavior:

- Signed out: the Masthead shows an outlined "Sign in" at the top right; the primary CTA in the Hero and the Colophon is "Get started" → `/login`.
- Signed in (decided once `useAuthReady` has hydrated; no forced redirect, no flashing redirect): the buttons read "Open the console" → `/dashboard`.
- Page structure: Masthead (§00) → Hero spread (§01) → Principles (§02) → Artifacts (§03) → Delivery (§04) → Open source (§05) → Colophon footer (§06). Each section = one claim + one exhibit + a verifiable fact; the §03 post-page exhibit recreates the cool slate styling of `backend/templates/post.html`, deliberately keeping a material contrast with the warm Ember paper.
- All copy lives in the `landing.*` namespace (en / zh-Hans / zh-Hant / ja); the sample post in the exhibits (`landing.sample.*`) is one shared post across the hero, §03, and §04, keeping the narrative coherent.

## Guard Architecture

### Declarative guards

The guard architecture is declarative and composable — three components over one executor:

```
AuthGate (the executor, consumes useAuthGuard)
├── PublicRoute     — the (auth) group
├── ProtectedRoute  — the (dashboard) group
└── AdminRoute      — nested under (dashboard)/admin
```

### route-configs.ts (guard configuration)

Guard configuration lives in pure functions that declare each route class's decision logic:

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

### AuthGate (the executor)

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

### Applied at the layout level

Guards are applied in the route groups' `layout.tsx` files (layout-level guarding), with admin nested inside dashboard:

```
app/(auth)/layout.tsx          → <PublicRoute>{children}</PublicRoute>
app/(dashboard)/layout.tsx     → <ProtectedRoute><DashboardLayout>{children}</DashboardLayout></ProtectedRoute>
app/(dashboard)/admin/layout.tsx → <AdminRoute><AdminLayout>{children}</AdminLayout></AdminRoute>
```

### Guard behavior

**PublicRoute** (the (auth) group: login, callback):

- hydrating → render PageSpinner
- authenticated → `router.replace("/dashboard")`
- unauthenticated → render children

**ProtectedRoute** (the (dashboard) group):

- hydrating → render PageSpinner
- unauthenticated (after hydration) → `router.replace("/login")`
- authenticated → render children

**AdminRoute** (nested under (dashboard)/admin):

- unauthenticated → `router.replace("/login")`
- authenticated but not admin → `router.replace("/dashboard")`
- admin → render children

### Hydration handling

Zustand persist restores from localStorage asynchronously. The `_hasHydrated` flag keeps guards from misreading the default empty state (`token=null`) as "unauthenticated" before hydration, which would cause a flashing redirect:

```typescript
onRehydrateStorage: () => (state) => {
  state?.setHasHydrated(true);
};
```

Guards render PageSpinner until hydration completes, then decide render versus redirect from the real authentication state.

## Security Boundary

**Client-side guards control UX only — they provide no security.**

Pure static export (`output: "export"`) rules out Next.js middleware / Proxy / Server Components for server-side route protection — those are server-runtime capabilities, unavailable to a purely static frontend (see [build.md](./build.md) §7). Client-side guards are the **only option**.

> The Next.js documentation (`authentication.mdx:1447`): "client-side UI restrictions alone are not sufficient for security." — the context of that warning is a full-stack Next.js application with Server Actions / API Routes (a client-side `return null` cannot stop a user from calling a Server Action directly).

**It does not apply to markpost**, because:

- The frontend is purely static, with **zero Server Actions / API Routes** (those all live in the Go backend)
- Every data access goes through the backend REST API, where **JWT authentication + admin middleware** enforce the authoritative check
- With the client guards bypassed (tampered localStorage), the frontend may render a page skeleton, but **every data request is rejected by the backend with 401/403** — the page stays empty, with no data leak

**Security lives in the backend API layer** (see [auth.md](../auth.md), [api-design.md](../api-design.md) §5).

## OAuth Callback Page

The `/auth/callback` page ((auth) route group, PublicRoute guard) handles the OAuth callback. The complete flow: [auth.md](../auth.md) §3, §7.

Responsibilities: read code + state from the URL query → re-validate state on the client (against sessionStorage) → POST `/api/v1/oauth/login` → setAuth → `router.replace('/dashboard')`. Every failure path goes `router.replace('/login')`.

## References

- [auth.md](../auth.md) — authentication flows, OAuth callback logic
- [build.md](./build.md) — pure static export, capability boundary
- [architecture.md](./architecture.md) — frontend architecture, provider stack
