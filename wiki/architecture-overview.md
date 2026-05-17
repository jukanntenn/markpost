# Architecture Overview

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.26, Gin (HTTP), GORM (ORM) |
| Frontend | Next.js 16, React 19, TypeScript |
| Database | SQLite (default) or PostgreSQL |
| Authentication | JWT (access + refresh tokens), optional GitHub OAuth |
| Styling | Tailwind CSS 4, shadcn/ui |

## Data Flow

```
Client → Gin Router → Middleware Chain → Handler → Service → Repository → GORM → Database
```

1. **Client** sends an HTTP request
2. **Gin Router** matches the route
3. **Middleware** processes the request (CORS → rate limiting → auth → handler-specific)
4. **Handler** validates input, calls service methods
5. **Service** implements business logic, returns domain models or errors
6. **Repository** performs database operations via GORM
7. **Response** flows back through the stack as JSON or HTML

## Authentication

MarkPost uses a dual-token JWT system:

- **Access Token** — Short-lived (default 24h), sent in `Authorization: Bearer <token>` header
- **Refresh Token** — Long-lived (default 30 days), sent in request body for token renewal

### Login Flows

**Username/Password:**
1. Client sends `POST /api/v1/auth/login` with username and password
2. Server validates credentials, returns both tokens

**GitHub OAuth:**
1. Client fetches OAuth URL from `GET /api/v1/oauth/url`
2. User authorizes on GitHub
3. Client sends authorization code to `POST /api/v1/oauth/login`
4. Server exchanges code for GitHub profile, creates or links user, returns tokens

### Token Lifecycle

- Access tokens are verified on every authenticated request via `middleware.AuthWithBlacklist`
- On logout, the access token is blacklisted (SHA-256 hash stored in `token_blacklist` table)
- Refresh tokens are stored in the database (hashed) and deleted on use (rotation)

## Post Lifecycle

1. **Create** — `POST /:post_key` with title and body in JSON. The `post_key` identifies the user.
2. **Store** — The service generates a unique QID (format: `p-<random>`) and stores the post in the database
3. **Render** — `GET /:id` where `id` is the QID. Returns HTML rendered from Markdown via goldmark. Add `?format=raw` for raw Markdown.
4. **Delivery** — On creation, posts are enqueued for delivery to configured channels (e.g., Feishu webhooks)

## Database Schema

### users

| Column | Type | Description |
|--------|------|-------------|
| id | int (PK) | Auto-increment ID |
| email | string (unique) | User email |
| username | string (unique) | Username |
| name | string | Display name |
| password_hash | string | Bcrypt hash (empty for OAuth-only users) |
| avatar_url | string (nullable) | Profile avatar URL |
| post_key | string (unique) | Unique key for posting via API |
| github_id | int64 (unique, nullable) | GitHub user ID |
| role | string | `admin` or `user` |
| is_active | bool | Account active status |
| is_email_verified | bool | Email verification status |
| last_login_at | timestamp (nullable) | Last login time |
| created_at | timestamp | Creation time |
| updated_at | timestamp | Update time |

### posts

| Column | Type | Description |
|--------|------|-------------|
| id | int (PK) | Auto-increment ID |
| qid | string (unique) | Public identifier (format: `p-<random>`) |
| title | string | Post title |
| body | text | Post body (Markdown) |
| user_id | int (FK → users) | Author |
| created_at | timestamp | Creation time |
| updated_at | timestamp | Update time |

### refresh_tokens

| Column | Type | Description |
|--------|------|-------------|
| id | int64 (PK) | Auto-increment ID |
| user_id | int (FK → users) | Token owner |
| token_hash | string (unique) | SHA-256 hash of refresh token |
| expires_at | timestamp | Expiration time |
| created_at | timestamp | Creation time |

### token_blacklist

| Column | Type | Description |
|--------|------|-------------|
| id | int64 (PK) | Auto-increment ID |
| token_hash | string (unique) | SHA-256 hash of blacklisted access token |
| expires_at | timestamp | When to remove from blacklist |
| created_at | timestamp | When blacklisted |

### channels

| Column | Type | Description |
|--------|------|-------------|
| id | int (PK) | Auto-increment ID |
| user_id | int (FK → users) | Channel owner |
| kind | string (32) | Channel type (e.g., `feishu`) |
| name | string | Display name |
| enabled | bool | Active status |
| webhook_url | text | Webhook endpoint |
| keywords | text | Filter keywords |
| created_at | timestamp | Creation time |
| updated_at | timestamp | Update time |
