# Frontend Routes & Access Control

## Route Table

| Path | Page | Guard | Condition | Redirect |
|------|------|-------|-----------|----------|
| `/` | — | — | always | → `/dashboard` |
| `/login` | Login | PublicRoute | authenticated | → `/dashboard` |
| `/auth` | OAuth Callback | PublicRoute | authenticated | → `/dashboard` |
| `/dashboard` | Dashboard | ProtectedRoute | not authenticated | → `/login` |
| `/posts` | Posts List | ProtectedRoute | not authenticated | → `/login` |
| `/settings` | Settings | ProtectedRoute | not authenticated | → `/login` |
| `/admin` | Admin Overview | AdminRoute | not authenticated | → `/login` |
| `/admin` | Admin Overview | AdminRoute | authenticated, not admin | → `/dashboard` |
| `/admin/users` | Admin Users | AdminRoute | not authenticated | → `/login` |
| `/admin/users` | Admin Users | AdminRoute | authenticated, not admin | → `/dashboard` |
| `/admin/posts` | Admin Posts | AdminRoute | not authenticated | → `/login` |
| `/admin/posts` | Admin Posts | AdminRoute | authenticated, not admin | → `/dashboard` |
| `/admin/channels` | Admin Channels | AdminRoute | not authenticated | → `/login` |
| `/admin/channels` | Admin Channels | AdminRoute | authenticated, not admin | → `/dashboard` |
| `/auth/github/callback` | OAuth Callback | — | — | — |

## Guard Behavior

### PublicRoute

Used in `(auth)` route group. Wraps login and OAuth callback pages.

- Hydrating → render nothing
- Authenticated → `router.replace("/dashboard")`
- Not authenticated → render children

### ProtectedRoute

Used in `(dashboard)` route group. Wraps all dashboard pages.

- Hydrating or not authenticated → render spinner
- Not authenticated (after hydration) → `router.replace("/login")`
- Authenticated → render children

### AdminRoute

Used in `(dashboard)/admin` sub-layout. Nested inside ProtectedRoute.

- Not authenticated → `router.replace("/login")`
- Authenticated but not admin (`role !== "admin"`) → `router.replace("/dashboard")`
- Admin → render children

## Special Routes

### `/auth/github/callback`

GitHub OAuth redirect target. No route guard. Handles the OAuth code exchange and token storage directly.

### `/auth`

Alternate OAuth callback inside the `(auth)` group, wrapped by `PublicRoute`. Shares the same `LoginCallbackPage` component as `/auth/github/callback`.
