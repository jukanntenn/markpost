# Frontend Architecture

## Overview

The frontend is a Next.js 16 application using the App Router with React 19. It follows a component-based architecture with Zustand for client state and TanStack Query for server state.

## App Router Structure

Pages are organized using route groups:

```
src/app/
├── layout.tsx                     — Root layout (fonts, providers)
├── page.tsx                       — Landing page
├── globals.css                    — Global styles and CSS variables
├── health/route.ts                — Health check API route
├── auth/                           — OAuth callback (outside route groups)
│   └── github/callback/page.tsx   — GitHub OAuth redirect target
├── (auth)/                        — Auth route group
│   ├── layout.tsx                 — Auth layout (centered, no sidebar)
│   ├── auth/page.tsx              — OAuth callback page
│   └── login/page.tsx             — Login page
└── (dashboard)/                   — Dashboard route group
    ├── layout.tsx                 — Dashboard layout (sidebar + header)
    ├── dashboard/page.tsx         — Main dashboard
    ├── posts/page.tsx             — Post list
    ├── settings/page.tsx          — User settings
    └── admin/                     — Admin section
        ├── layout.tsx             — Admin layout
        ├── page.tsx               — Admin dashboard
        ├── users/page.tsx         — User management
        ├── posts/page.tsx         — Post management
        └── channels/page.tsx      — Channel management
```

Route groups `(auth)` and `(dashboard)` share no layout — each has its own.

## Component Organization

```
src/components/
├── ui/          — shadcn/ui primitives (Button, Input, Dialog, etc.)
├── auth/        — Auth-related components
├── layout/      — Layout components (Sidebar, Header)
├── login/       — Login page-specific components
├── dashboard/   — Dashboard components
├── admin/       — Admin components
├── posts/       — Post-related components
├── settings/    — User settings components
└── providers/   — Context providers (LocaleProvider, QueryProvider)
```

## State Management

### Server State — TanStack Query

API data fetching is managed by TanStack Query. The `QueryProvider` wraps the app and provides the query client.

### Client State — Zustand

Authentication state is managed via Zustand with persistence:

```typescript
// src/stores/auth.ts
export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      refreshToken: null,
      user: null,
      // ...
    }),
    {
      name: "markpost_auth",
      partialize: (state) => ({
        token: state.token,
        refreshToken: state.refreshToken,
        user: state.user,
      }),
    }
  )
);
```

State is persisted to localStorage under the `markpost_auth` key. Hydration state is tracked via `_hasHydrated`.

### Toast State

A separate Zustand store (`src/stores/toast.ts`) manages toast notifications.

## API Client

The API client is in `src/lib/api/base.ts`. It provides a generic `request<T>()` function that:

1. Reads the auth token from the Zustand store
2. Sets `Authorization: Bearer <token>` header
3. Sends the request to the backend
4. On 401, automatically attempts token refresh
5. Retries the request with the new token after refresh
6. Redirects to login if refresh fails

The client always sends relative paths (e.g. `/api/v1/posts`); the server-side proxy (`src/proxy.ts`) forwards them to the backend via `BACKEND_URL`.

## Route Protection

Protected routes check authentication state in the dashboard layout. The auth store's `isAuthenticated()` method returns true only when both `token` and `user` are present.

Admin routes additionally check `isAdmin()` which verifies `user.role === "admin"`.

## Provider Stack

The root layout wraps the app in these providers (outer to inner):

1. `LocaleProvider` — next-intl locale context
2. `QueryProvider` — TanStack Query client
3. `ThemeProvider` — next-themes (dark/light/system)
4. `ToastProvider` — Toast notification context
