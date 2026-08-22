# API Design Specification

English | [中文](api-design.zh.md)

This document defines the design rules of the markpost REST API, closely following the style of the [GitHub REST API](https://docs.github.com/rest). The endpoint catalog (request/response fields per route) lives in [backend/api-schema.md](./backend/api-schema.md); the error response format lives in [backend/error-handling.md](./backend/error-handling.md).

## 1. URL design principles

### 1.1 Resource naming (GitHub-aligned)

| Pattern              | Rule                | Examples                                                                                                                                      |
| -------------------- | ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Collection           | plural nouns        | `/posts`, `/delivery/channels`, `/admin/users`                                                                                                |
| Singleton / service  | singular noun       | `/post-key`, `/health`, `/oauth`, `/auth`                                                                                                     |
| Nesting              | expresses ownership | `/delivery/channels/:id`, `/admin/delivery/channels`                                                                                          |
| admin namespace      | `/admin` prefix     | the delivery domain's admin endpoints nest under `/admin/delivery/*`, expressing resource ownership                                           |
| Functional endpoints | verb namespaces     | `/auth/*` (login/refresh/logout/change-password), `/oauth/*` (url/login) — authentication is inherently an action, not forced into a resource |

### 1.2 kebab-case (uniform)

Every path segment is kebab-case (matching the overwhelming majority of GitHub endpoints):

- ✅ `/post-key`, `/change-password`, `/delivery-history`
- ❌ `/post_key` (underscores)

### 1.3 Versioning

URL path versioning: `/api/v1`. Every REST API sits under this prefix. A version bump uses `/api/v2`.

### 1.4 Root-level routes (outside /api/v1)

External endpoints in the post-by-email style stay at the root level, serving external tools such as curl and Telegram bots:

| Method | Path         | Purpose                                                                      |
| ------ | ------------ | ---------------------------------------------------------------------------- |
| POST   | `/:post_key` | external delivery creates a post; the post_key authenticates in the URL path |
| GET    | `/:id`       | render a post (returns HTML, not JSON, or Markdown with `?format=raw`)       |

These endpoints do not return JSON (GET returns an HTML page) and are not part of the REST API collection.

---

## 2. HTTP method semantics (GitHub-aligned)

| Method | Semantics                                     | Success status                  | Examples                                      |
| ------ | --------------------------------------------- | ------------------------------- | --------------------------------------------- |
| GET    | query / read                                  | 200                             | lists, details                                |
| POST   | create / action                               | **201 Created**                 | create a channel, create a post, login, OAuth |
| PATCH  | **partial update** (omitted fields unchanged) | 200                             | update a channel                              |
| DELETE | delete                                        | **204 No Content** (empty body) | delete a channel, delete a post               |

> **PATCH, not PUT**: delivery channel updates use PATCH for partial updates, aligned with GitHub. The canonical semantics of PUT are full replacement; PATCH is the accurate verb for a partial update.

> **204 has no body**: a successful DELETE returns 204 No Content (GitHub-aligned) with an empty body.

---

## 3. Error responses (see error-handling.md)

### 3.1 400 vs 422 distinction (GitHub-aligned)

| Scenario                                                                    | Status  | ErrCode             | Meaning                                                     |
| --------------------------------------------------------------------------- | ------- | ------------------- | ----------------------------------------------------------- |
| Malformed request (JSON deserialization failure, empty body, type mismatch) | **400** | `ErrInvalidRequest` | the server cannot parse the request content                 |
| Field validation failure (required / min_length / ...)                      | **422** | `ErrValidation`     | the request parses, but field values violate business rules |

**GitHub 422 evidence** (from `rest-api-docs-md`; all three trigger scenarios return 422):

- Semantic parameter conflict: `type` combined with `visibility`/`affiliation` → 422 (`user/repos/get.md`)
- Missing required semantic qualifier: search without `is:issue` → 422 (`search/issues/get.md`)
- Business precondition unmet: repository with ≥10000 commits → 422 (`stats/code_frequency/get.md`)

The common thread across the three: **the request parses correctly (it is not malformed), but the server cannot process it semantically**. That matches RFC 4918's definition of 422.

`ErrValidation` maps to HTTP **422** (the `HTTP` field of the ErrCode struct in error-handling.md).

### 3.2 Unified error body (GitHub style)

See [error-handling.md](./backend/error-handling.md). In short:

```json
{ "code": "invalid_credentials", "message": "Invalid username or password" }
```

Validation errors carry field-level detail in `errors[]`:

```json
{
  "code": "validation",
  "message": "Request validation failed",
  "errors": [
    {
      "field": "new_password",
      "code": "min_length",
      "message": "new_password must be at least 8 characters"
    }
  ]
}
```

`code` is machine-readable (frontend logic); `message` is human-readable (rendered after i18n).

---

## 4. List response format

List endpoints return a **wrapper object** (the frontend needs total to render pagination info; GitHub search endpoints use a wrapper object for the same reason):

```json
{
  "items": [...],
  "total": 42,
  "page": 1,
  "limit": 20,
  "total_pages": 3
}
```

- The field name is uniformly `items` (invariant across resources, so frontend types stay uniform)
- `total` + `total_pages` feed the frontend pagination UI

**Pagination parameters**: `limit` (default 20, cap 100) + `page` (default 1), offset = `(page-1) * limit`.

> `limit` is kept (not forced to GitHub's `per_page`). `limit` is the more general name, and the frontend is adapted to it.

---

## 5. Authentication model (dual track)

markpost has two authentication modes covering distinct route classes:

| Route class                                                        | Authentication                    | Middleware        | Rate-limit dimension               |
| ------------------------------------------------------------------ | --------------------------------- | ----------------- | ---------------------------------- |
| Public read (`GET /:id`)                                           | none                              | —                 | L1 per-IP                          |
| Public write (`POST /:post_key`)                                   | PostKey (the key in the URL path) | PostKey           | L2 per-user_id (10/min + 1000/day) |
| Authenticated API (`/api/v1/*` protected)                          | JWT Bearer                        | AuthWithBlacklist | L3 per-user_id (30/min)            |
| Public API (`/oauth/*`, `/auth/login`, `/auth/refresh`, `/health`) | none                              | —                 | L1 per-IP                          |

Both authentication modes set `user_id` in the gin context, so rate limiting is uniform along the user_id dimension.

See [auth.md](./auth.md) for the authentication flows.

---

## 6. Rate limiting (GitHub-style quota exposure)

### 6.1 Three-tier token buckets (tollbooth)

| Tier | Scope                     | Dimension   | Rate                                                | Burst     |
| ---- | ------------------------- | ----------- | --------------------------------------------------- | --------- |
| L1   | public read               | per-IP      | 100/s                                               | 200       |
| L2   | public write (post_key)   | per-user_id | 10/min + 1000/day dual limit (two chained limiters) | 20 / 1000 |
| L3   | authenticated write (JWT) | per-user_id | 30/min                                              | 60        |

Dimension choice: the read path has only the IP (with CDN origin pulls, the IP is the sole identifier); the write path uses user_id (neither rotating credentials nor changing IPs escapes the limit).

### 6.2 Response headers

Rate-limit information is exposed to the frontend (aligned with GitHub exposing `X-RateLimit-*`):

- `RateLimit-Limit` / `RateLimit-Remaining` / `RateLimit-Reset`
- CORS `expose_headers` is configured to expose these headers
- **No `Retry-After` is sent** (tollbooth does not provide one; clients judge retry timing from the remaining quota)

### 6.3 IP resolution

Everything goes through gin's `ClientIP` (trusted-proxy resolution), bypassing tollbooth's built-in IP extraction. This avoids double extraction and keeps a single trusted-proxy configuration source.

> Source: tollbooth's `LimitByKeys` does no IP extraction itself (`tollbooth.go:45-48`) — whatever key is passed is what gets limited. gin's `ClientIP` resolves XFF/X-Real-IP correctly through `SetTrustedProxies`.

---

## 7. Complete endpoint list

```
/api/v1
├── GET    /health                              public, health check
├── /oauth
│   ├── GET    /url                             public, get the GitHub authorization URL (with PKCE challenge)
│   └── POST   /login                           public, {code, state} → token pair
├── /auth
│   ├── POST   /login                           public, {username, password} → token pair
│   ├── POST   /refresh                         public, {refresh_token} → new token pair
│   ├── POST   /logout                          JWT, logout (blacklist access + revoke refresh)
│   └── POST   /change-password                 JWT, {current, new}
├── GET    /post-key                            JWT, current user's post key
├── GET    /posts                               JWT, post list → {items, total, ...}
├── DELETE /posts/:id                           JWT, delete a post → 204
├── /delivery
│   ├── GET    /channels                        JWT, channel list → {items, total, ...}
│   ├── POST   /channels                        JWT, create a channel → 201
│   ├── PATCH  /channels/:id                    JWT, partially update a channel
│   ├── DELETE /channels/:id                    JWT, delete a channel → 204
│   └── GET    /history                         JWT, delivery history → {items, total, ...}
├── /admin (JWT + Admin)
│   ├── GET    /users                           all users → {items, total, ...}
│   ├── GET    /posts                           all posts → {items, total, ...}
│   ├── DELETE /posts/:id                       delete any post → 204
│   └── /delivery
│       ├── GET    /channels                    all channels → {items, total, ...}
│       └── GET    /history                     all delivery history → {items, total, ...}

Root level (outside /api/v1)
├── POST   /:post_key                           PostKey auth, external delivery creates a post → 201 {id}
└── GET    /:id                                 public, render a post (HTML / ?format=raw)
```

Per-endpoint request/response field detail lives in [api-schema.md](./backend/api-schema.md).

---

## References

- [backend/api-schema.md](./backend/api-schema.md) — endpoint reference (request/response fields per route)
- [backend/error-handling.md](./backend/error-handling.md) — error response format, the ErrCode struct
- [auth.md](./auth.md) — authentication flows (JWT, OAuth, refresh, password)
- [backend/rate-limiting.md](./backend/rate-limiting.md) — rate limiting design detail
