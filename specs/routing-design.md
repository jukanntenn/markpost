# Routing & Permission Design

This document defines the routing scheme and permission model specific to this
project. It is a companion to `service-routing-spec.md`, which provides the
general architecture rules. This file contains the concrete route tables,
value-prefix definitions, and access-control decisions.

---

## 1. Route Table — Backend

All backend routes are exposed through Next.js rewrites (see
`service-routing-spec.md` §2.2). The backend has **no catch-all routes**.

| Route | Method | Authentication | Description |
|---|---|---|---|
| `/api/v1/health` | GET | None | Health check |
| `/api/v1/auth/login` | POST | None | Username/password login |
| `/api/v1/auth/refresh` | POST | None | Refresh access token |
| `/api/v1/auth/logout` | POST | JWT (blacklist) | Revoke tokens |
| `/api/v1/auth/change-password` | POST | JWT | Change own password |
| `/api/v1/oauth/url` | GET | None | Generate GitHub OAuth URL |
| `/api/v1/oauth/login` | POST | None | GitHub OAuth callback |
| `/api/v1/post_key` | GET | JWT | Query own post key |
| `/api/v1/posts` | GET | JWT | List own posts |
| `/api/v1/delivery/channels` | GET | JWT | List delivery channels |
| `/api/v1/delivery/channels` | POST | JWT | Create delivery channel |
| `/api/v1/admin/users` | GET | JWT + Admin | List all users |
| `/api/v1/admin/posts` | GET | JWT + Admin | List all posts |
| `/api/v1/admin/channels` | GET | JWT + Admin | List all delivery channels |
| `/mpk-:postKey` | POST | Post key | Create a post |
| `/p-:qid` | GET | None | Render a post (HTML or raw markdown) |
| `/swagger/*` | GET | None | API docs (debug mode only) |

---

## 2. Route Table — Frontend

Frontend pages use Next.js App Router. No `basePath`.

| Route | Guard | Description |
|---|---|---|
| `/health` | None | Health check (Next.js API route) |
| `/login` | PublicRoute | Login page — redirects to `/dashboard` if authenticated |
| `/dashboard` | ProtectedRoute | Main dashboard |
| `/posts` | ProtectedRoute | Post management |
| `/settings` | ProtectedRoute | User settings |
| `/admin/users` | AdminRoute | User management |
| `/admin/posts` | AdminRoute | Post management (admin view) |
| `/admin/channels` | AdminRoute | Delivery channel management |

---

## 3. Value-Prefix Definitions

This project uses two value-prefixed identifiers stored in the database.

### 3.1 Post Key — `mpk-`

- **Stored field**: `post_key` on the user entity.
- **Format**: `mpk-` followed by a random alphanumeric string.
- **Route**: `POST /mpk-:postKey` — creates a post. The handler receives the
  full value (e.g. `mpk-a1b2c3d4e5f6`) and uses it to look up the owning user.
- **Purpose**: allows external systems to create posts without JWT, using a
  per-user token that is revocable.

### 3.2 Post QID — `p-`

- **Stored field**: `qid` on the post entity.
- **Format**: `p-` followed by a unique identifier.
- **Route**: `GET /p-:qid` — renders a post. The handler receives the full
  value (e.g. `p-xyz789`) and uses it to look up the post.
- **Purpose**: gives every post a unique, root-level URL that is
  unambiguously distinguishable from frontend routes.

### 3.3 Data Migration

Existing rows that lack the prefix must be updated on application startup. The
migration is idempotent: if a value already starts with the prefix, it is
skipped.

---

## 4. Next.js Rewrites

```ts
async rewrites() {
  const target = process.env.API_PROXY_TARGET;
  if (!target) return [];
  return [
    { source: "/api/:path*", destination: `${target}/api/:path*` },
    { source: "/mpk-:postKey", destination: `${target}/mpk-:postKey` },
    { source: "/p-:qid", destination: `${target}/p-:qid` },
  ];
}
```

The `/api/:path*` rewrite covers all API routes (including
`/api/v1/health`). The value-prefixed rewrites (`/mpk-*`, `/p-*`) cover the
two root-level routes.

---

## 5. Permission Model

### 5.1 Authentication Methods

| Method | Mechanism | Used by |
|---|---|---|
| None | No credentials required | Public endpoints (login, post rendering) |
| JWT (access token) | `Authorization: Bearer <token>` header | Authenticated user actions |
| JWT + blacklist | Same as JWT, plus revocation check | Logout and sensitive actions |
| Post key | Route parameter `mpk-<key>` | External post creation |
| JWT + role | JWT + `role == "admin"` check | Admin-only endpoints |

### 5.2 Role Definitions

| Role | Scope |
|---|---|
| `admin` | Full system access. List/manage all users, posts, and channels. |
| `user` | Manage own resources: own posts, own delivery channels, own password. |

### 5.3 Frontend Route Guards

| Guard | Component | Behavior |
|---|---|---|
| PublicRoute | Wraps `/login` | Redirect to `/dashboard` if a valid session exists |
| ProtectedRoute | Wraps `/dashboard`, `/posts`, `/settings` | Redirect to `/login` if no valid session |
| AdminRoute | Wraps `/admin/*` | Redirect to `/dashboard` if not admin role |

Frontend guards are **UX optimizations**, not security boundaries. Every API
endpoint enforces its own authentication independently. A request that bypasses
the frontend guard will still be rejected by the backend if it lacks valid
credentials.

### 5.4 Swagger Access

Swagger documentation routes are registered **only when `debug = true``. In
production (`debug = false`), the route group does not exist — returning 404
rather than 403, to avoid leaking the endpoint's existence.
