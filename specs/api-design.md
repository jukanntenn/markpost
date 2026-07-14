# API Design Specification

本文档定义 markpost REST API 的设计规范，深度借鉴 [GitHub REST API](https://docs.github.com/rest) 的设计风格。端点清单（每个路由的请求/响应字段）见 [backend/api-schema.md](./backend/api-schema.md)；错误响应格式见 [backend/error-handling.md](./backend/error-handling.md)。

## 一、URL 设计原则

### 1.1 资源命名（对齐 GitHub）

| 模式 | 规则 | 示例 |
|------|------|------|
| 集合 | 复数名词 | `/posts`、`/delivery/channels`、`/admin/users` |
| 单例 / 服务 | 单数名词 | `/post-key`、`/health`、`/oauth`、`/auth` |
| 嵌套 | 表达从属关系 | `/delivery/channels/:id`、`/admin/delivery/channels` |
| admin 命名空间 | `/admin` 前缀 | delivery 域的 admin 端点用 `/admin/delivery/*` 嵌套，体现资源归属 |
| 功能端点 | 动词命名空间 | `/auth/*`（login/refresh/logout/change-password）、`/oauth/*`（url/login）——认证本质是动作，不强行资源化 |

### 1.2 kebab-case（统一）

所有路径段统一 kebab-case（与 GitHub 绝大多数端点一致）：

- ✅ `/post-key`、`/change-password`、`/delivery-history`
- ❌ `/post_key`（下划线）

### 1.3 版本化

URL path 版本化：`/api/v1`。所有 REST API 在此前缀下。版本升级时用 `/api/v2`。

### 1.4 根级路由（`/api/v1` 之外）

post-by-email 风格的外部接口保留在根级，给 curl / Telegram bot 等外部工具用：

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/:post_key` | 外部投递创建文章，post_key 在 URL path 中认证 |
| GET | `/:id` | 渲染文章（返回 HTML 非 JSON，或 `?format=raw` 返回 Markdown） |

这些端点不返回 JSON（GET 返回 HTML 页面），本就不属于 REST API 集合。

---

## 二、HTTP 方法语义（对齐 GitHub）

| 方法 | 语义 | 成功状态码 | 用例 |
|------|------|----------|------|
| GET | 查询 / 读取 | 200 | 列表、详情 |
| POST | 创建 / 动作 | **201 Created** | 创建渠道、创建文章、登录、OAuth |
| PATCH | **部分更新**（省略字段 = 不变） | 200 | 更新渠道 |
| DELETE | 删除 | **204 No Content**（无 body） | 删除渠道、删除文章 |

> **PATCH 而非 PUT**：现状 delivery 渠道更新用 PUT 做部分更新。对齐 GitHub 改为 PATCH。PUT 的规范语义是"整体替换"，部分更新用 PATCH 更准确。

> **204 无 body**：DELETE 成功返回 204 No Content（对齐 GitHub）。不再返回 `{ message: "..." }`。

---

## 三、错误响应（引用 error-handling.md）

### 3.1 400 vs 422 区分（对齐 GitHub）

| 场景 | 状态码 | ErrCode | 说明 |
|------|--------|---------|------|
| 请求格式错误（JSON 反序列化失败、空 body、类型不匹配） | **400** | `ErrInvalidRequest` | 服务器无法解析请求内容 |
| 字段校验失败（required / min_length / ...） | **422** | `ErrValidation` | 请求能解析但字段值不满足业务规则 |

**GitHub 422 证据**（来自 `rest-api-docs-md`，三种触发场景都返回 422）：
- 参数语义冲突：`type` 参数与 `visibility`/`affiliation` 同时用 → 422（`user/repos/get.md`）
- 缺必需语义限定符：search 缺 `is:issue` → 422（`search/issues/get.md`）
- 业务前置条件不满足：仓库 ≥10000 commits → 422（`stats/code_frequency/get.md`）

三种场景的共同点：**请求能被正确解析（不是格式错误），但服务器无法按语义处理**。这对应 RFC 4918 对 422 的定义。

`ErrValidation` 的 HTTP 状态码从 400 改为 **422**（error-handling.md 里 ErrCode struct 的 `HTTP` 字段值）。

### 3.2 统一错误体（GitHub 风格）

详见 [error-handling.md](./backend/error-handling.md)。简述：

```json
{ "code": "invalid_credentials", "message": "Invalid username or password" }
```

校验错误带 `errors[]` 字段级详情：

```json
{
  "code": "validation",
  "message": "Request validation failed",
  "errors": [
    { "field": "new_password", "code": "min_length", "message": "new_password must be at least 8 characters" }
  ]
}
```

`code` 是 machine-readable（前端做逻辑），`message` 是 human-readable（i18n 后展示）。

---

## 四、列表响应格式

列表端点返回**包裹对象**（因前端需要 total 显示分页信息，GitHub search 端点同理用包裹对象）：

```json
{
  "items": [...],
  "total": 42,
  "page": 1,
  "limit": 20,
  "total_pages": 3
}
```

- 字段名统一 `items`（不按资源名变化，前端类型统一）
- `total` + `total_pages` 供前端分页 UI

**分页参数**：`limit`（默认 20，上限 100）+ `page`（默认 1），offset = `(page-1) * limit`。

> 保留 `limit`（不强行改 GitHub 的 `per_page`）。`limit` 是更通用的命名，前端已适配。

---

## 五、认证模型（双轨）

markpost 有两种认证方式，对应不同路径类：

| 路径类 | 认证方式 | 中间件 | 限流维度 |
|--------|---------|--------|---------|
| 公共读（`GET /:id`） | 无 | — | L1 per-IP |
| 公共写（`POST /:post_key`） | PostKey（URL path 中的 key） | PostKey | L2 per-user_id（10/min + 1000/day） |
| 认证 API（`/api/v1/*` 受保护） | JWT Bearer | AuthWithBlacklist | L3 per-user_id（30/min） |
| 公共 API（`/oauth/*`、`/auth/login`、`/auth/refresh`、`/health`） | 无 | — | L1 per-IP |

两种认证都设置 `user_id` 到 gin context，限流统一按 user_id 维度。

认证流程详见 [auth.md](./auth.md)。

---

## 六、限流（对齐 GitHub 暴露额度的做法）

### 6.1 三层 token bucket（tollbooth）

| 层 | 作用域 | 维度 | 速率 | burst |
|----|--------|------|------|-------|
| L1 | 公共读 | per-IP | 100/s | 200 |
| L2 | 公共写（post_key） | per-user_id | 10/min + 1000/day 双限（链式两个 limiter） | 20 / 1000 |
| L3 | 认证写（JWT） | per-user_id | 30/min | 60 |

维度选择：读路径只有 IP（CDN 回源场景 IP 是唯一标识）；写路径用 user_id（轮换凭证 / 换 IP 都不能逃限）。

### 6.2 响应头

暴露限流信息给前端（对齐 GitHub 暴露 `X-RateLimit-*` 的做法）：

- `RateLimit-Limit` / `RateLimit-Remaining` / `RateLimit-Reset`
- CORS `expose_headers` 已配置暴露这些头
- **不发 `Retry-After`**（tollbooth 不提供，客户端从剩余额度判断重试时机）

### 6.3 IP 解析

统一走 gin `ClientIP`（trusted proxies 解析），bypass tollbooth 自带的 IP 提取逻辑。避免双重提取，单一可信代理配置来源。

> 依据：tollbooth 的 `LimitByKeys` 自身不做 IP 提取（`tollbooth.go:45-48`），传什么 key 就按什么限流。gin `ClientIP` 通过 `SetTrustedProxies` 正确解析 XFF/X-Real-IP。

---

## 七、完整端点清单

```
/api/v1
├── GET    /health                              公开，健康检查
├── /oauth
│   ├── GET    /url                             公开，获取 GitHub 授权 URL（含 PKCE challenge）
│   └── POST   /login                           公开，{code, state} → token 对
├── /auth
│   ├── POST   /login                           公开，{username, password} → token 对
│   ├── POST   /refresh                         公开，{refresh_token} → 新 token 对
│   ├── POST   /logout                          JWT，登出（黑名单 access + 吊销 refresh）
│   └── POST   /change-password                 JWT，{current, new}
├── GET    /post-key                            JWT，查询当前用户 post key
├── GET    /posts                               JWT，文章列表 → {items, total, ...}
├── DELETE /posts/:id                           JWT，删除文章 → 204
├── /delivery
│   ├── GET    /channels                        JWT，渠道列表 → {items, total, ...}
│   ├── POST   /channels                        JWT，创建渠道 → 201
│   ├── PATCH  /channels/:id                    JWT，部分更新渠道
│   ├── DELETE /channels/:id                    JWT，删除渠道 → 204
│   └── GET    /history                         JWT，投递历史 → {items, total, ...}
├── /admin（JWT + Admin）
│   ├── GET    /users                           全部用户 → {items, total, ...}
│   ├── GET    /posts                           全部文章 → {items, total, ...}
│   ├── DELETE /posts/:id                       删除任意文章 → 204
│   └── /delivery
│       ├── GET    /channels                    全部渠道 → {items, total, ...}
│       └── GET    /history                     全部投递历史 → {items, total, ...}

根级（/api/v1 之外）
├── POST   /:post_key                           PostKey 认证，外部投递创建 → 201 {id}
└── GET    /:id                                 公开，渲染文章（HTML / ?format=raw）
```

每个端点的请求/响应字段详情见 [api-schema.md](./backend/api-schema.md)。

---

## 参考

- [backend/api-schema.md](./backend/api-schema.md) — 端点参考（每路由的请求/响应字段）
- [backend/error-handling.md](./backend/error-handling.md) — 错误响应格式、ErrCode struct
- [auth.md](./auth.md) — 认证流程（JWT、OAuth、refresh、密码）
- [backend/performance-optimization.md](./backend/performance-optimization.md) — 限流设计细节
