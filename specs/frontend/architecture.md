# Frontend Architecture

English | [中文](architecture.zh.md)

This document defines the frontend architecture: App Router structure, component organization, state management, the API client, route protection, and the provider stack. Build configuration (pure static export) lives in [build.md](./build.md); route guards in [routes.md](./routes.md); i18n in [i18n.md](./i18n.md).

## Overview

The frontend is a Next.js 16 application on App Router + React 19, exported as **pure static output** (`output: "export"`, no server runtime). Client state is Zustand; server state is TanStack Query.

See [build.md](./build.md) for the pure static export design and the capability boundary.

## App Router Structure

Pages are organized with route groups:

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

The `(auth)` and `(dashboard)` route groups each carry their own layout and share none.

> A pure static export ships no API Routes and no SSR proxy (no `route.ts` / `proxy.ts` files). Health checks come from the backend `/api/v1/health` endpoint, and the `/api/*` reverse proxy is Caddy's job (`next.config.ts` rewrites in development). See [build.md](./build.md) §3.

## Component Organization

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

## State Management

### Server State — TanStack Query

API data fetching is managed by TanStack Query. `QueryProvider` wraps the application and provides the query client.

### Client State — Zustand (authentication state)

Authentication state lives in a Zustand store with the `persist` middleware, persisted to localStorage:

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

The store persists to localStorage (key = `markpost_auth`). Hydration is tracked with the `_hasHydrated` flag (flash prevention — see the hydration handling in [routes.md](./routes.md)).

Security considerations for token storage: [auth.md](../auth.md) §6.

## API Client

The API client lives in `src/lib/api/base.ts` and provides a generic `request<T>()` function:

1. read the access token from the Zustand store
2. set the `Authorization: Bearer <token>` header
3. **send `Accept-Language: <current locale>`** — the backend answers with error messages in that language (see [i18n.md](./i18n.md))
4. send the request (**relative path** `/api/v1/...`, forwarded to the backend by the reverse proxy)
5. on 401, attempt a refresh token exchange automatically (single-flight deduplication)
6. refresh succeeds → retry the original request with the new token
7. refresh fails → logout → redirect to login

> **Direct backend calls** (no SSR proxy): the frontend sends relative paths, and in deployment Nginx/Caddy reverse-proxies them to the Go backend. See [build.md](./build.md) §4.

The automatic refresh mechanism: [auth.md](../auth.md) §6.3.

## Route Protection

Route protection is implemented by client-side guard components — see [routes.md](./routes.md).

The essentials: client guards **control UX only**; security lives in the backend API layer (JWT + admin middleware).

## Provider Stack

The root layout wraps the application in these providers (outermost first):

1. `LocaleProvider` — the next-intl locale context (**pure client-side bootstrap**; it receives no serverLocale/serverMessages — see [i18n.md](./i18n.md))
2. `QueryProvider` — the TanStack Query client
3. `ThemeProvider` — next-themes (dark/light/system)
4. `ToastProvider` — the toast notification context

> The root layout calls no server i18n API (`getLocale()` / `getMessages()`) — those are server-only and unavailable under a pure static export. `LocaleProvider` bootstraps entirely on the client: the initial locale is `en`, and after hydration it reads localStorage and dynamically loads the messages chunk.
