# API Schema

本文档是 markpost REST API 的端点参考（每路由的请求/响应字段）。设计规范（URL 命名、HTTP 方法语义、状态码、列表格式、认证模型、限流）见 [api-design.md](../api-design.md)；错误响应格式见 [error-handling.md](./error-handling.md)；认证流程见 [auth.md](../auth.md)。

Base path: `/api/v1`

## Conventions

- 认证请求需 header: `Authorization: Bearer <jwt_token>`
- 错误响应统一 JSON 结构，见 [error-handling.md](./error-handling.md)
- 列表响应包裹对象：`{ items, total, page, limit, total_pages }`
- 状态码：200 查询 / 201 创建 / 204 删除 / 400 格式错 / 422 校验错 / 401 未认证 / 403 无权限 / 404 不存在 / 429 限流

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
| GET | `/oauth/url` | — | 获取 GitHub OAuth 授权 URL（含 PKCE challenge） |
| POST | `/oauth/login` | — | 使用 GitHub OAuth code + state 登录 |

### GET /oauth/url

**Response**: `{ url, state }`

`url` 是完整的 GitHub 授权 URL（含 state 和 PKCE code_challenge）。state 和 verifier 已存在后端 ristretto（TTL 10min）。

### POST /oauth/login

**Request body**: `code`, `state`

**Response**: `{ user, token, refresh_token, expires_in }`

后端校验 state（ristretto，一次性消费）+ PKCE Exchange + 获取 GitHub 用户 + 签发 token 对。详见 [auth.md](../auth.md) §3。

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

一次性轮转：旧 refresh token 吊销（revoked=true）+ 签发新对。重用检测见 [auth.md](../auth.md) §2.3。

### POST /auth/logout

**Response**: 204 No Content

登出同时黑名单 access token + 吊销该用户 refresh token（revoked=true）。

### POST /auth/change-password

**Request body**: `current_password`, `new_password`

密码策略：最小 8 字符，最大 72 字符，不强制复杂度。详见 [auth.md](../auth.md) §4。

**Response**: `{ message }`

---

## Post Key

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/post-key` | JWT | 查询当前用户的 Post Key |

**Response**: `{ post_key, created_at }`

---

## Posts

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/posts` | JWT | 获取当前用户的文章列表 |
| DELETE | `/posts/:id` | JWT | 删除当前用户的文章 |

### GET /posts

**Query params**: `page`, `limit`（默认 20，上限 100）

**Response**: `{ items: [{ id, qid, title, created_at }], total, page, limit, total_pages }`

### DELETE /posts/:id

**Response**: 204 No Content

---

## Delivery Channels

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/delivery/channels` | JWT | 获取当前用户的投递渠道列表 |
| POST | `/delivery/channels` | JWT | 创建投递渠道 |
| PATCH | `/delivery/channels/:id` | JWT | 部分更新投递渠道（省略字段=不变） |
| DELETE | `/delivery/channels/:id` | JWT | 删除投递渠道 |

### POST /delivery/channels

**Response**: 201 `{ channel: { id, kind, name, enabled, webhook_url, keywords, created_at, updated_at } }`

**Request body**: `kind`, `name`, `webhook_url`, `keywords`

`keywords` 是可选的过滤表达式（按文章标题过滤）。语法：`,`/`|` = OR，`&` = AND，`!` = NOT，`()` 分组；空 = 总是投递。格式错误返回 422。详见 [keyword-filter.md](./keyword-filter.md)。

### PATCH /delivery/channels/:id

**Request body**（部分更新）: `kind`, `name`, `webhook_url`, `keywords`, `enabled`

`keywords` 是部分更新字段：省略 = 不变，传空字符串 = 清除（清除 → 匹配一切）。表达式同样校验，格式错误返回 422。

省略的字段保持原值（PATCH 语义）。详见 [api-design.md](../api-design.md) §2。

**Response**: `{ channel: { ... } }`

### DELETE /delivery/channels/:id

**Response**: 204 No Content

---

## Delivery History

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/delivery/history` | JWT | 获取当前用户的投递历史 |

**Query params**: `page`, `limit`

**Response**: `{ items: [...], total, page, limit, total_pages }`

---

## Admin

所有管理员端点要求 JWT 认证 + Admin 角色。

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/admin/users` | JWT+Admin | 获取全部用户列表 |
| GET | `/admin/posts` | JWT+Admin | 获取全部文章列表 |
| DELETE | `/admin/posts/:id` | JWT+Admin | 删除任意文章 |
| GET | `/admin/delivery/channels` | JWT+Admin | 获取全部投递渠道 |
| GET | `/admin/delivery/history` | JWT+Admin | 获取全部投递历史 |

> delivery 域的 admin 端点嵌套在 `/admin/delivery/` 下，体现资源归属；users / posts 的 admin 端点直接在 `/admin/` 下。详见 [api-design.md](../api-design.md) §1.1。

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

## 根级端点（/api/v1 之外）

以下端点不在 `/api/v1` 前缀下，是 post-by-email 风格的外部接口。详见 [api-design.md](../api-design.md) §1.4。

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/:post_key` | PostKey 中间件 | 通过 Post Key 创建文章 |
| GET | `/:id` | — | 渲染文章 |

### POST /:post_key

认证方式：URL 路径中的 `post_key` 由中间件校验（解析到 user）。

**Request body**: `title`, `body`（Markdown）

**Response**: 201 `{ id }`

### GET /:id

**Query params**: `format` — 传 `raw` 返回 Markdown，否则返回 HTML 页面。

---

## 参考

- [api-design.md](../api-design.md) — API 设计规范（URL/方法/状态码/列表/认证/限流）
- [error-handling.md](./error-handling.md) — 错误响应格式
- [auth.md](../auth.md) — 认证流程
- [keyword-filter.md](./keyword-filter.md) — 关键词过滤表达式语法
