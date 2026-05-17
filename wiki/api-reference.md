# API Reference

Base path: `/api/v1`

All authenticated endpoints require `Authorization: Bearer <access_token>` header.

## Authentication

### POST /api/v1/auth/login

Login with username and password.

**Request:**

```json
{
  "username": "string (required)",
  "password": "string (required)"
}
```

**Response (200):**

```json
{
  "token": "access_token",
  "refresh_token": "refresh_token",
  "expires_in": 86400,
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "user",
    "name": "Display Name",
    "avatar_url": "https://...",
    "role": "user"
  }
}
```

### POST /api/v1/auth/refresh

Refresh an expired access token.

**Request:**

```json
{
  "refresh_token": "string (required)"
}
```

**Response (200):**

```json
{
  "token": "new_access_token",
  "refresh_token": "new_refresh_token",
  "expires_in": 86400
}
```

### POST /api/v1/auth/logout

Logout and blacklist the current access token.

**Headers:** `Authorization: Bearer <token>`

**Response (200):**

```json
{
  "message": "Logged out successfully"
}
```

### POST /api/v1/auth/change-password

Change the current user's password.

**Headers:** `Authorization: Bearer <token>`

**Request:**

```json
{
  "current_password": "string",
  "new_password": "string (required, min 6 chars)"
}
```

**Response (200):**

```json
{
  "message": "Password changed successfully"
}
```

## OAuth

### GET /api/v1/oauth/url

Get the GitHub OAuth authorization URL.

**Response (200):**

```json
{
  "url": "https://github.com/login/oauth/authorize?..."
}
```

### POST /api/v1/oauth/login

Complete GitHub OAuth login.

**Request:**

```json
{
  "code": "string (required)",
  "state": "string (required)"
}
```

**Response (200):** Same as `POST /api/v1/auth/login`

## Health Check

### GET /api/v1/health

**Response (200):**

```json
{
  "status": "ok"
}
```

## Post Key

### GET /api/v1/post_key

Get the current user's post key.

**Headers:** `Authorization: Bearer <token>`

**Response (200):**

```json
{
  "post_key": "mpk-abc123...",
  "created_at": "2024-01-01T00:00:00Z"
}
```

## Posts

### POST /:post_key

Create a new post. This is a public endpoint — authentication is via the post key in the URL path.

**Request:**

```json
{
  "title": "string (required)",
  "body": "string (required)"
}
```

**Response (201):**

```json
{
  "id": "p-abc123..."
}
```

### GET /:id

View a post. Public endpoint — no authentication required.

- `:id` is the post's QID (e.g., `p-abc123`)
- Returns rendered HTML by default
- Add `?format=raw` to get raw Markdown

**Response (200, HTML):** Rendered HTML page using the `post.html` template

**Response (200, raw):** Raw Markdown with `Content-Type: text/markdown`

**Response (404):** `Not Found` if the post doesn't exist

### GET /api/v1/posts

List the current user's posts with pagination.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number (min 1) |
| limit | int | 20 | Items per page (min 1, max 100) |

**Response (200):**

```json
{
  "posts": [
    {
      "id": 1,
      "qid": "p-abc123",
      "title": "My Post",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

## Delivery Channels

### GET /api/v1/delivery/channels

List the current user's delivery channels.

**Headers:** `Authorization: Bearer <token>`

**Response (200):**

```json
{
  "channels": [
    {
      "id": 1,
      "kind": "feishu",
      "name": "My Channel",
      "enabled": true,
      "webhook_url": "https://...",
      "keywords": "",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### POST /api/v1/delivery/channels

Create a delivery channel.

**Headers:** `Authorization: Bearer <token>`

**Request:**

```json
{
  "kind": "string (required)",
  "name": "string (required)",
  "webhook_url": "string (required)",
  "keywords": "string (optional)"
}
```

**Response (201):**

```json
{
  "channel": { ... }
}
```

### PUT /api/v1/delivery/channels/:id

Update a delivery channel.

**Headers:** `Authorization: Bearer <token>`

**Request:**

```json
{
  "kind": "string (optional)",
  "name": "string (optional)",
  "webhook_url": "string (optional)",
  "keywords": "string (optional)",
  "enabled": "bool (optional)"
}
```

**Response (200):**

```json
{
  "channel": { ... }
}
```

### DELETE /api/v1/delivery/channels/:id

Delete a delivery channel.

**Headers:** `Authorization: Bearer <token>`

**Response (200):**

```json
{
  "message": "Channel deleted successfully"
}
```

## Admin

All admin endpoints require `Authorization: Bearer <token>` and the user must have `role: "admin"`.

### GET /api/v1/admin/users

List all users.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| limit | int | 20 | Items per page (max 100) |

**Response (200):**

```json
{
  "users": [
    {
      "id": 1,
      "username": "user",
      "email": "user@example.com",
      "role": "user",
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 42
}
```

### GET /api/v1/admin/posts

List all posts with optional search.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| search | string | "" | Search term |
| page | int | 1 | Page number |
| limit | int | 20 | Items per page (max 100) |

**Response (200):**

```json
{
  "posts": [
    {
      "id": "p-abc123",
      "qid": "p-abc123",
      "title": "My Post",
      "user_id": 1,
      "username": "user",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 100
}
```

### GET /api/v1/admin/channels

List all delivery channels.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| limit | int | 20 | Items per page (max 100) |

**Response (200):**

```json
{
  "channels": [
    {
      "id": 1,
      "name": "My Channel",
      "type": "feishu",
      "enabled": true,
      "user_id": 1,
      "webhook_url": "https://...",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 10
}
```

## Error Responses

All error responses follow this format:

```json
{
  "code": "error_code",
  "message": "Human-readable message"
}
```

For validation errors:

```json
{
  "code": "validation",
  "message": "Request validation failed",
  "errors": [
    {
      "field": "title",
      "code": "required",
      "message": "This field is required"
    }
  ]
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `invalid_credentials` | 401 | Wrong username or password |
| `unauthorized` | 401 | Not authenticated |
| `invalid_token` | 401 | Invalid or expired JWT |
| `missing_authorization_header` | 401 | No Authorization header |
| `forbidden` | 403 | Insufficient permissions |
| `invalid_post_key` | 403 | Invalid post key |
| `user_disabled` | 403 | Account deactivated |
| `not_found` | 404 | Resource not found |
| `validation` | 400 | Request validation failed |
| `invalid_request` | 400 | Malformed request |
| `internal` | 500 | Unexpected server error |
