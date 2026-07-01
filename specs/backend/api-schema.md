# API Schema

Base path: `/api/v1`

## Conventions

- Authenticated requests require header: `Authorization: Bearer <jwt_token>`
- Error responses follow a unified JSON structure. See [error-handling.md](./error-handling.md)

---

## Health Check

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | — | 服务健康检查 |

**Response**: `{ "status": "ok" }`

---

## OAuth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/oauth/url` | — | 获取 GitHub OAuth 授权 URL |
| POST | `/oauth/login` | — | 使用 GitHub OAuth 授权码登录 |

### POST /oauth/login

**Request body**:

| Field | Required |
|-------|----------|
| `code` | yes |
| `state` | yes |

**Response**: `{ user, token, refresh_token, expires_in }`

---

## Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/login` | — | 用户名密码登录 |
| POST | `/auth/refresh` | — | 刷新访问令牌 |
| POST | `/auth/logout` | JWT | 注销登录 |
| POST | `/auth/change-password` | JWT | 修改密码 |

### POST /auth/login

**Request body**: `username`, `password`

**Response**: `{ user, token, refresh_token, expires_in }`

### POST /auth/refresh

**Request body**: `refresh_token`

**Response**: `{ token, refresh_token, expires_in }`

### POST /auth/logout

**Response**: `{ message }`

### POST /auth/change-password

**Request body**: `current_password`, `new_password`

**Response**: `{ message }`

---

## Post Key

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/post_key` | JWT | 查询当前用户的 Post Key |

**Response**: `{ post_key, created_at }`

---

## Posts

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/posts` | JWT | 获取当前用户的文章列表 |

### GET /posts

**Query params**: `page`, `limit`

**Response**: `{ posts: [{ id, qid, title, created_at }], pagination }`

### Non-API Endpoints

以下端点不在 `/api/v1` 前缀下，不返回 JSON。

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/:post_key` | PostKey 中间件 | 通过 Post Key 创建文章 |
| GET | `/:id` | — | 渲染文章 |

#### POST /:post_key

认证方式：URL 路径中的 `post_key` 由中间件校验。

**Request body**: `title`, `body`（Markdown）

**Response**: `201` `{ id }`

#### GET /:id

**Query params**: `format` — 传 `raw` 返回 Markdown，否则返回 HTML 页面。

---

## Delivery Channels

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/delivery/channels` | JWT | 获取当前用户的投递渠道列表 |
| POST | `/delivery/channels` | JWT | 创建投递渠道 |
| PUT | `/delivery/channels/:id` | JWT | 更新投递渠道 |
| DELETE | `/delivery/channels/:id` | JWT | 删除投递渠道 |

### POST /delivery/channels

**Request body**: `kind`, `name`, `webhook_url`, `keywords`

`keywords` is an optional filter expression over the post title. Syntax: `,`/`|` = OR, `&` = AND, `!` = NOT, `()` group; spaces are keyword content (no quotes needed); empty = always deliver. Malformed expressions are rejected with `400`. See [keyword-filter.md](./keyword-filter.md) for the full grammar.

**Response**: `201` `{ channel: { id, kind, name, enabled, webhook_url, keywords, created_at, updated_at } }`

### PUT /delivery/channels/:id

**Request body**（部分更新）: `kind`, `name`, `webhook_url`, `keywords`, `enabled`

`keywords` is a partial-update field: omit to leave unchanged, pass an empty string to clear it (clears → matches everything). As with POST, the expression is validated and a malformed value is rejected with `400`.

**Response**: `{ channel: { ... } }`

### DELETE /delivery/channels/:id

**Response**: `{ message }`

---

## Admin

所有管理员端点要求 JWT 认证 + Admin 角色。

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/admin/users` | JWT+Admin | 获取全部用户列表 |
| GET | `/admin/posts` | JWT+Admin | 获取全部文章列表 |
| GET | `/admin/channels` | JWT+Admin | 获取全部投递渠道列表 |

### GET /admin/users

**Query params**: `page`, `limit`

**Response**: `{ users: [{ id, username, email, role, is_active, created_at }], pagination }`

### GET /admin/posts

**Query params**: `page`, `limit`, `search`

**Response**: `{ posts: [{ id, qid, title, user_id, username, created_at }], pagination }`

### GET /admin/channels

**Query params**: `page`, `limit`

**Response**: `{ channels: [{ id, name, kind, enabled, user_id, webhook_url, created_at }], pagination }`
