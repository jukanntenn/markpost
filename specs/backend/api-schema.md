# API Schema

English | [中文](api-schema.zh.md)

This document is the endpoint reference for markpost's REST API (request/response fields per route). The design rules (URL naming, HTTP method semantics, status codes, list format, auth model, rate limiting) live in [api-design.md](../api-design.md); the error response format lives in [error-handling.md](./error-handling.md); the authentication flows live in [auth.md](../auth.md).

Base path: `/api/v1`

## Conventions

- Authenticated requests require the header: `Authorization: Bearer <jwt_token>`
- Error responses use a uniform JSON structure, see [error-handling.md](./error-handling.md)
- List responses wrap an object: `{ items, total, page, limit, total_pages }`
- Status codes: 200 query / 201 create / 204 delete / 400 malformed / 422 validation / 401 unauthenticated / 403 forbidden / 404 not found / 429 rate-limited

---

## Health Check

| Method | Path      | Auth | Description                                                                                                                  |
| ------ | --------- | ---- | ---------------------------------------------------------------------------------------------------------------------------- |
| GET    | `/health` | —    | Liveness: the process is up. Always 200 while serving                                                                        |
| GET    | `/ready`  | —    | Readiness: a driver-level database round trip. 503 when the process is up but unfit to serve (DB unreachable/pool exhausted) |

**Response** (`/health`): `{ "status": "ok" }` · (`/ready`): `{ "status": "ready" }` or 503 `{ "status": "unavailable" }`

---

## OAuth

| Method | Path           | Auth | Description                                                      |
| ------ | -------------- | ---- | ---------------------------------------------------------------- |
| GET    | `/oauth/url`   | —    | Get the GitHub OAuth authorization URL (PKCE challenge included) |
| POST   | `/oauth/login` | —    | Log in with a GitHub OAuth code + state                          |

### GET /oauth/url

**Response**: `{ url, state }`

`url` is the full GitHub authorization URL (with state and the PKCE code_challenge). The state and verifier are stored server-side in ristretto (TTL 10min).

### POST /oauth/login

**Request body**: `code`, `state`

**Response**: `{ user, token, refresh_token, expires_in }`

The server validates the state (ristretto, one-time consumption) + the PKCE exchange + fetches the GitHub user + issues the token pair. Details in [auth.md](../auth.md) §3.

---

## Auth

| Method | Path                    | Auth | Description              |
| ------ | ----------------------- | ---- | ------------------------ |
| POST   | `/auth/login`           | —    | Username/password login  |
| POST   | `/auth/refresh`         | —    | Refresh the access token |
| POST   | `/auth/logout`          | JWT  | Log out                  |
| POST   | `/auth/change-password` | JWT  | Change password          |

### POST /auth/login

**Request body**: `username`, `password`

**Response**: `{ user, token, refresh_token, expires_in }`

### POST /auth/refresh

**Request body**: `refresh_token`

**Response**: `{ token, refresh_token, expires_in }`

One-time rotation: the old refresh token is revoked (`revoked=true`) and a new pair is issued. Reuse detection in [auth.md](../auth.md) §2.3.

### POST /auth/logout

**Response**: 204 No Content

Logout both blacklists the access token and revokes the user's refresh tokens (`revoked=true`).

### POST /auth/change-password

**Request body**: `current_password`, `new_password`

Password policy: minimum 8 characters, maximum 72 characters, no complexity requirements. Details in [auth.md](../auth.md) §4.

**Response**: `{ message }`

---

## Post Key

| Method | Path        | Auth | Description                       |
| ------ | ----------- | ---- | --------------------------------- |
| GET    | `/post-key` | JWT  | Query the current user's Post Key |

**Response**: `{ post_key, created_at }`

---

---

## Me

| Method | Path            | Auth | Description                             |
| ------ | --------------- | ---- | --------------------------------------- |
| GET    | `/me/retention` | JWT  | The caller's effective retention policy |

**Response**: `{ posts_days, history_days }` — each `0` (keep forever) or a whole-day count; an explicit per-user override drives both numbers, an inherit resolves each table's own global window.

## Posts

| Method | Path         | Auth | Description                    |
| ------ | ------------ | ---- | ------------------------------ |
| GET    | `/posts`     | JWT  | List the current user's posts  |
| DELETE | `/posts/:id` | JWT  | Delete the current user's post |

### GET /posts

**Query params**: `page`, `limit` (default 20, max 100)

**Response**: `{ items: [{ id, qid, title, created_at }], total, page, limit, total_pages }`

### DELETE /posts/:id

**Response**: 204 No Content

---

## Delivery Channels

| Method | Path                     | Auth | Description                                                    |
| ------ | ------------------------ | ---- | -------------------------------------------------------------- |
| GET    | `/delivery/channels`     | JWT  | List the current user's delivery channels                      |
| POST   | `/delivery/channels`     | JWT  | Create a delivery channel                                      |
| PATCH  | `/delivery/channels/:id` | JWT  | Partially update a delivery channel (omitted fields unchanged) |
| DELETE | `/delivery/channels/:id` | JWT  | Delete a delivery channel                                      |

### POST /delivery/channels

**Response**: 201 `{ channel: { id, kind, name, enabled, webhook_url, keywords, created_at, updated_at } }`

**Request body**: `kind`, `name`, `webhook_url`, `keywords`

`keywords` is an optional filter expression (filters by post title). Syntax: `,`/`|` = OR, `&` = AND, `!` = NOT, `()` groups; empty = always deliver. Malformed expressions return 422. See [keyword-filter.md](./keyword-filter.md).

### PATCH /delivery/channels/:id

**Request body** (partial update): `kind`, `name`, `webhook_url`, `keywords`, `enabled`

`keywords` is a partial-update field: omitted = unchanged, empty string = cleared (cleared → matches everything). The expression is validated the same way; malformed expressions return 422.

Omitted fields keep their current values (PATCH semantics). See [api-design.md](../api-design.md) §2.

**Response**: `{ channel: { ... } }`

### DELETE /delivery/channels/:id

**Response**: 204 No Content

---

## Delivery History

| Method | Path                | Auth | Description                             |
| ------ | ------------------- | ---- | --------------------------------------- |
| GET    | `/delivery/history` | JWT  | Get the current user's delivery history |

**Query params**: `page`, `limit`

**Response**: `{ items: [...], total, page, limit, total_pages }`

---

## Admin

All admin endpoints require JWT authentication + the Admin role.

| Method | Path                          | Auth      | Description                                           |
| ------ | ----------------------------- | --------- | ----------------------------------------------------- |
| GET    | `/admin/users`                | JWT+Admin | List all users                                        |
| GET    | `/admin/users/:id`            | JWT+Admin | User detail                                           |
| POST   | `/admin/users`                | JWT+Admin | Create a user                                         |
| PATCH  | `/admin/users/:id/role`       | JWT+Admin | Set role                                              |
| PATCH  | `/admin/users/:id/active`     | JWT+Admin | Enable/disable                                        |
| PATCH  | `/admin/users/:id/vip`        | JWT+Admin | Set VIP honorific                                     |
| PATCH  | `/admin/users/:id/retention`  | JWT+Admin | Set one user's history retention policy               |
| POST   | `/admin/users/retention/bulk` | JWT+Admin | Bulk-set retention for explicit ids or every VIP user |
| POST   | `/admin/retention/impact`     | JWT+Admin | Preview deletion impact of a candidate policy         |
| GET    | `/admin/retention/defaults`   | JWT+Admin | Global retention fallback windows                     |
| GET    | `/admin/settings`             | JWT+Admin | List runtime settings                                 |
| PUT    | `/admin/settings/:key`        | JWT+Admin | Upsert one runtime setting                            |
| GET    | `/admin/posts`                | JWT+Admin | List all posts                                        |
| DELETE | `/admin/posts/:id`            | JWT+Admin | Delete any post                                       |
| GET    | `/admin/delivery/channels`    | JWT+Admin | List all delivery channels                            |
| GET    | `/admin/delivery/history`     | JWT+Admin | List all delivery history                             |

> The delivery domain's admin endpoints nest under `/admin/delivery/` to reflect resource ownership; the users / posts admin endpoints sit directly under `/admin/`. See [api-design.md](../api-design.md) §1.1.

### GET /admin/users

**Query params**: `page`, `limit`

**Response**: `{ items: [{ id, username, email, role, is_active, created_at }], total, page, limit, total_pages }`

### GET /admin/posts

**Query params**: `page`, `limit`, `search`

**Response**: `{ items: [{ id, qid, title, user_id, username, created_at }], total, page, limit, total_pages }`

### DELETE /admin/posts/:id

**Response**: 204 No Content

### GET /admin/delivery/channels

**Query params**: `page`, `limit`

**Response**: `{ items: [{ id, name, kind, enabled, user_id, webhook_url, created_at }], total, page, limit, total_pages }`

### GET /admin/delivery/history

**Query params**: `page`, `limit`

**Response**: `{ items: [...], total, page, limit, total_pages }`

---

## Root-level Endpoints (outside /api/v1)

These endpoints sit outside the `/api/v1` prefix; they are the post-by-email-style external interface. See [api-design.md](../api-design.md) §1.4.

| Method | Path         | Auth               | Description                    |
| ------ | ------------ | ------------------ | ------------------------------ |
| POST   | `/:post_key` | PostKey middleware | Create a post via the Post Key |
| GET    | `/:id`       | —                  | Render a post                  |

### POST /:post_key

Authentication: the `post_key` in the URL path is verified by middleware (resolved to a user).

**Request body**: `title`, `body` (Markdown)

**Response**: 201 `{ id }`

### GET /:id

**Query params**: `format` — pass `raw` to return Markdown; otherwise an HTML page is returned.

---

## References

- [api-design.md](../api-design.md) — API design rules (URL/methods/status codes/lists/auth/rate limiting)
- [error-handling.md](./error-handling.md) — error response format
- [auth.md](../auth.md) — authentication flows
- [keyword-filter.md](./keyword-filter.md) — keyword filter expression syntax
