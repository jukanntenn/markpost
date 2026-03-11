# Markpost API 规范文档

## 概述

Markpost 是一个类似 pastebin 的 Markdown 博客服务，支持 OAuth 和 JWT 认证。

- **版本**: 1.0
- **基础路径**: `/api`
- **默认主机**: `localhost:7330`
- **认证方式**: Bearer Token (JWT)

## API 分组

### 1. 认证相关 (Auth)

#### 1.1 密码登录
- **路径**: `POST /api/auth/login`
- **描述**: 使用用户名和密码进行身份验证
- **请求体**:
  ```json
  {
    "username": "string (required)",
    "password": "string (required)"
  }
  ```
- **响应**: `AuthResponse`

#### 1.2 刷新令牌
- **路径**: `POST /api/auth/refresh`
- **描述**: 使用 refresh token 获取新的 access token
- **请求体**:
  ```json
  {
    "refresh_token": "string (required)"
  }
  ```
- **响应**: `AuthResponse`

#### 1.3 修改密码
- **路径**: `POST /api/auth/change-password`
- **描述**: 修改已认证用户的密码
- **认证**: 需要 Bearer Token
- **请求体**:
  ```json
  {
    "current_password": "string",
    "new_password": "string (required, min: 6)"
  }
  ```
- **响应**: `MessageResponse`

#### 1.4 查询 Post Key
- **路径**: `GET /api/post_key`
- **描述**: 查询用户的 post key，用于创建文章
- **认证**: 需要 Bearer Token
- **响应**: `PostKeyResponse`

### 2. OAuth 相关

#### 2.1 获取 GitHub OAuth URL
- **路径**: `GET /api/oauth/url`
- **描述**: 生成 GitHub OAuth 授权 URL
- **响应**:
  ```json
  {
    "url": "string"
  }
  ```

#### 2.2 GitHub 登录
- **路径**: `POST /api/oauth/login`
- **描述**: 处理 GitHub OAuth 回调并认证用户
- **查询参数**:
  - `state`: string (required) - OAuth state 参数
- **请求体**:
  ```json
  {
    "code": "string (required)"
  }
  ```
- **响应**: `AuthResponse`

### 3. 文章相关 (Posts)

#### 3.1 创建文章
- **路径**: `POST /{post_key}`
- **描述**: 使用 post key 创建新的 Markdown 文章
- **路径参数**:
  - `post_key`: string (required) - 用于认证的 post key
- **请求体**:
  ```json
  {
    "title": "string (required)",
    "body": "string (required)"
  }
  ```
- **响应**: `CreatePostResponse` (201 Created)

#### 3.2 渲染文章
- **路径**: `GET /{id}`
- **描述**: 渲染文章为 HTML 页面
- **路径参数**:
  - `id`: string (required) - 文章 QID
- **查询参数**:
  - `format`: string (optional) - 响应格式，`raw` 返回原始 Markdown
- **响应**:
  - 默认: HTML 内容 (text/html)
  - format=raw: Markdown 内容 (text/markdown)

#### 3.3 获取用户文章列表
- **路径**: `GET /api/posts`
- **描述**: 获取用户的文章分页列表
- **认证**: 需要 Bearer Token
- **查询参数**:
  - `page`: integer (optional, min: 1, default: 1)
  - `limit`: integer (optional, min: 1, max: 100, default: 20)
- **响应**: `PostsListResponse`

### 4. 投递渠道相关 (Delivery Channels)

#### 4.1 列出投递渠道
- **路径**: `GET /api/delivery/channels`
- **描述**: 获取用户的所有投递渠道
- **认证**: 需要 Bearer Token
- **响应**: `DeliveryChannelListResponse`

#### 4.2 创建投递渠道
- **路径**: `POST /api/delivery/channels`
- **描述**: 创建新的投递渠道
- **认证**: 需要 Bearer Token
- **请求体**:
  ```json
  {
    "kind": "string (required)",
    "name": "string",
    "enabled": "boolean (default: true)",
    "webhook_url": "string (required)",
    "keywords": "string"
  }
  ```
- **响应**: `DeliveryChannelResponse`

#### 4.3 更新投递渠道
- **路径**: `PUT /api/delivery/channels/:id`
- **描述**: 更新投递渠道信息
- **认证**: 需要 Bearer Token
- **路径参数**:
  - `id`: integer (required)
- **请求体**:
  ```json
  {
    "name": "string (optional)",
    "enabled": "boolean (optional)",
    "webhook_url": "string (optional)",
    "keywords": "string (optional)"
  }
  ```
- **响应**: `DeliveryChannelResponse`

#### 4.4 删除投递渠道
- **路径**: `DELETE /api/delivery/channels/:id`
- **描述**: 删除投递渠道
- **认证**: 需要 Bearer Token
- **路径参数**:
  - `id`: integer (required)
- **响应**: `{ "ok": true }`

### 5. 管理员相关 (Admin)

所有管理员接口都需要 Bearer Token 认证且用户角色为 `admin`。

#### 5.1 用户管理

##### 5.1.1 列出所有用户
- **路径**: `GET /api/admin/users`
- **描述**: 获取所有用户的分页列表
- **查询参数**:
  - `page`: integer (optional, default: 1)
  - `limit`: integer (optional, default: 10)
- **响应**: `ListUsersResponse`

##### 5.1.2 创建用户
- **路径**: `POST /api/admin/users`
- **描述**: 创建新用户
- **请求体**:
  ```json
  {
    "username": "string (required, min: 1, max: 50)",
    "password": "string (required, min: 6)"
  }
  ```
- **响应**: `{ "user": AdminUserResponse }`

##### 5.1.3 更新用户角色
- **路径**: `PUT /api/admin/users/:id/role`
- **描述**: 更新用户角色
- **路径参数**:
  - `id`: integer (required)
- **请求体**:
  ```json
  {
    "role": "string (required, enum: [admin, user])"
  }
  ```
- **响应**: `{ "user": AdminUserResponse }`

##### 5.1.4 删除用户
- **路径**: `DELETE /api/admin/users/:id`
- **描述**: 删除用户
- **路径参数**:
  - `id`: integer (required)
- **响应**: `{ "ok": true }`

##### 5.1.5 重置用户密码
- **路径**: `POST /api/admin/users/:id/reset-password`
- **描述**: 重置用户密码
- **路径参数**:
  - `id`: integer (required)
- **请求体**:
  ```json
  {
    "password": "string (optional)",
    "generate_random": "boolean (optional)"
  }
  ```
- **响应**: `{ "password": "string" }`

#### 5.2 文章管理

##### 5.2.1 列出所有文章
- **路径**: `GET /api/admin/posts`
- **描述**: 获取所有文章的分页列表
- **查询参数**:
  - `search`: string (optional) - 搜索关键词
  - `page`: integer (optional, min: 1, default: 1)
  - `limit`: integer (optional, min: 1, max: 100, default: 10)
- **响应**: `AdminPostsListResponse`

##### 5.2.2 更新文章
- **路径**: `PUT /api/admin/posts/:id`
- **描述**: 更新文章内容
- **路径参数**:
  - `id`: integer (required)
- **请求体**:
  ```json
  {
    "title": "string (optional)",
    "body": "string (required)"
  }
  ```
- **响应**: `{ "post": AdminPostResponse }`

##### 5.2.3 删除文章
- **路径**: `DELETE /api/admin/posts/:id`
- **描述**: 删除文章
- **路径参数**:
  - `id`: integer (required)
- **响应**: `{ "ok": true }`

#### 5.3 投递渠道管理

##### 5.3.1 列出所有投递渠道
- **路径**: `GET /api/admin/channels`
- **描述**: 获取所有用户的投递渠道
- **响应**: `{ "channels": [AdminDeliveryChannelResponse] }`

##### 5.3.2 更新投递渠道
- **路径**: `PUT /api/admin/channels/:id`
- **描述**: 更新任意用户的投递渠道
- **路径参数**:
  - `id`: integer (required)
- **请求体**:
  ```json
  {
    "name": "string (optional)",
    "enabled": "boolean (optional)",
    "webhook_url": "string (optional)",
    "keywords": "string (optional)"
  }
  ```
- **响应**: `{ "channel": AdminDeliveryChannelResponse }`

##### 5.3.3 删除投递渠道
- **路径**: `DELETE /api/admin/channels/:id`
- **描述**: 删除任意用户的投递渠道
- **路径参数**:
  - `id`: integer (required)
- **响应**: `{ "ok": true }`

### 6. 健康检查

#### 6.1 健康检查
- **路径**: `GET /health`
- **描述**: 检查 API 健康状态
- **响应**:
  ```json
  {
    "status": "string"
  }
  ```

## 数据模型

### AuthResponse
```json
{
  "access_token": "string",
  "refresh_token": "string",
  "user": {
    "id": "integer",
    "username": "string"
  }
}
```

### MessageResponse
```json
{
  "message": "string"
}
```

### PostKeyResponse
```json
{
  "post_key": "string",
  "created_at": "string (RFC3339)"
}
```

### CreatePostResponse
```json
{
  "id": "integer"
}
```

### PostsListResponse
```json
{
  "posts": [
    {
      "id": "integer",
      "qid": "string",
      "title": "string",
      "created_at": "string"
    }
  ],
  "pagination": {
    "page": "integer",
    "limit": "integer",
    "total": "integer",
    "total_pages": "integer"
  }
}
```

### DeliveryChannelResponse
```json
{
  "id": "integer",
  "kind": "string",
  "name": "string",
  "enabled": "boolean",
  "webhook_url": "string",
  "keywords": "string",
  "created_at": "string (RFC3339)",
  "updated_at": "string (RFC3339)"
}
```

### AdminUserResponse
```json
{
  "id": "integer",
  "username": "string",
  "role": "string",
  "github_id": "integer (nullable)",
  "created_at": "string (RFC3339)",
  "updated_at": "string (RFC3339)"
}
```

### AdminPostResponse
```json
{
  "id": "integer",
  "qid": "string",
  "title": "string",
  "body": "string",
  "user_id": "integer",
  "user": {
    "id": "integer",
    "username": "string"
  },
  "created_at": "string (RFC3339)",
  "updated_at": "string (RFC3339)"
}
```

### AdminDeliveryChannelResponse
```json
{
  "id": "integer",
  "user_id": "integer",
  "username": "string",
  "kind": "string",
  "name": "string",
  "enabled": "boolean",
  "webhook_url": "string",
  "keywords": "string",
  "created_at": "string (RFC3339)",
  "updated_at": "string (RFC3339)"
}
```

## 错误响应

所有错误响应遵循统一格式：

```json
{
  "code": "string",
  "message": "string",
  "details": [
    {
      "code": "string",
      "description": "string"
    }
  ]
}
```

### 常见错误码

- `400` - Bad Request: 请求参数错误
- `401` - Unauthorized: 未认证或认证失败
- `403` - Forbidden: 权限不足
- `404` - Not Found: 资源不存在
- `500` - Internal Server Error: 服务器内部错误

## 认证

### Bearer Token 认证

对于需要认证的接口，在请求头中添加：

```
Authorization: Bearer <access_token>
```

### Post Key 认证

创建文章接口使用 post key 进行认证：

```
POST /{post_key}
```

## Swagger 文档访问

启动服务后，可通过以下地址访问 Swagger UI：

```
http://localhost:7330/swagger/index.html
```

## 开发说明

### 生成 Swagger 文档

在 `backend/` 目录下执行：

```bash
swag init -g main.go -o docs --parseDependency --parseInternal
```

### 注释规范

使用 swaggo 注释规范，示例：

```go
// CreatePost godoc
// @Summary      Create a new post
// @Description  Create a new markdown post using post key
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        post_key  path    string         true  "Post key for authentication"
// @Param        request   body    PostRequest    true  "Post content"
// @Success      201  {object}  CreatePostResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /{post_key} [post]
func CreatePost(postSvc PostServiceInterface) gin.HandlerFunc {
    // ...
}
```

## 待完善项

### 当前 Swagger 文档缺失的接口

以下接口在代码中已实现，但未添加 Swagger 注释：

1. **Delivery Channels 相关**:
   - `GET /api/delivery/channels` - 列出投递渠道
   - `POST /api/delivery/channels` - 创建投递渠道
   - `PUT /api/delivery/channels/:id` - 更新投递渠道
   - `DELETE /api/delivery/channels/:id` - 删除投递渠道

2. **Admin 相关**:
   - `GET /api/admin/users` - 列出所有用户
   - `POST /api/admin/users` - 创建用户
   - `PUT /api/admin/users/:id/role` - 更新用户角色
   - `DELETE /api/admin/users/:id` - 删除用户
   - `POST /api/admin/users/:id/reset-password` - 重置用户密码
   - `GET /api/admin/posts` - 列出所有文章
   - `PUT /api/admin/posts/:id` - 更新文章
   - `DELETE /api/admin/posts/:id` - 删除文章
   - `GET /api/admin/channels` - 列出所有投递渠道
   - `PUT /api/admin/channels/:id` - 更新投递渠道
   - `DELETE /api/admin/channels/:id` - 删除投递渠道

### 建议改进

1. **添加缺失的 Swagger 注释**: 为所有接口添加完整的 godoc 注释
2. **统一错误响应模型**: 定义标准的错误响应结构体
3. **添加请求示例**: 在 Swagger 注释中添加 `@Example` 注释
4. **完善响应模型**: 为所有响应定义明确的结构体，避免使用 `map[string]interface{}`
5. **添加认证说明**: 在需要 admin 权限的接口上明确标注
6. **添加分页模型**: 统一分页响应的结构
