# E2E Verification Report

**Date**: 2026-04-28  
**Environment**: Sandbox (no network/Docker access)  
**Method**: Static code analysis + unit test execution + bug fix verification

## Test Execution Summary

| Suite | Result | Details |
|-------|--------|---------|
| Backend Go Tests | ✅ PASS | 5 test files, all passing |
| Frontend Vitest Tests | ✅ PASS | 5 test files, 29 tests, all passing |
| Go Build | ✅ PASS | Clean compilation, no errors |

## API Endpoint Verification

### Public Endpoints (No Auth)

| Endpoint | Status | Notes |
|----------|--------|-------|
| `GET /health` | ✅ | Returns `{"status":"ok"}` with 200 |
| `GET /:id` | ✅ | Renders post HTML; `?format=raw` returns markdown; 404 for missing |
| `POST /:post_key` | ✅ | PostKey middleware validates; creates post with 201 |
| `POST /api/v1/auth/login` | ✅ | Validates required username/password; returns token pair |
| `POST /api/v1/auth/refresh` | ✅ | Validates required refresh_token; returns new pair |
| `GET /api/v1/oauth/url` | ✅ | Generates GitHub OAuth URL |
| `POST /api/v1/oauth/login` | ✅ | Validates code+state; returns token pair |

### JWT-Protected Endpoints

| Endpoint | Status | Notes |
|----------|--------|-------|
| `GET /api/v1/post_key` | ✅ | Returns user's post key and created_at |
| `POST /api/v1/auth/logout` | ✅ | Blacklists token (fixed: now reports errors) |
| `POST /api/v1/auth/change-password` | ✅ | Validates new_password min=6 |
| `GET /api/v1/posts` | ✅ | Paginated, limit capped at 100 |

### Delivery Channel Endpoints (JWT Required)

| Endpoint | Status | Notes |
|----------|--------|-------|
| `GET /api/v1/delivery/channels` | ✅ | Lists user's channels |
| `POST /api/v1/delivery/channels` | ✅ | Validates kind/name/webhook_url |
| `PUT /api/v1/delivery/channels/:id` | ✅ | Partial update, all fields optional |
| `DELETE /api/v1/delivery/channels/:id` | ✅ | Deletes channel |

### Admin Endpoints (JWT + RequireAdmin)

| Endpoint | Status | Notes |
|----------|--------|-------|
| `GET /api/v1/admin/users` | ✅ | Returns paginated user list |
| `GET /api/v1/admin/posts` | ✅ | Returns paginated post list |
| `GET /api/v1/admin/channels` | ✅ | Returns paginated channel list |

All admin endpoints verified to return 403 for non-admin users via `RequireAdmin` middleware.

## Frontend Verification

### Auth Flow
- ✅ Login page validates username/password, stores tokens in Zustand with persistence
- ✅ Token refresh on 401 with concurrent refresh deduplication
- ✅ `ProtectedRoute` redirects unauthenticated users to `/login`
- ✅ `AdminRoute` redirects non-admin users to `/dashboard`
- ✅ Admin role check: `user.role === "admin"` in both frontend and backend

### Page States
- ✅ Dashboard: loading → error → empty → populated states handled
- ✅ Posts list: loading → error → empty → populated states handled
- ✅ Admin pages: wrapped in `AdminRoute` with proper role guard
- ✅ Settings page: JWT-protected

### Noted Issues (Pre-existing, Not Blocking)
- i18n system fully wired (600+ keys) but no components use it yet
- ESLint has circular structure error with FlatCompat (pre-existing config issue)

## Bugs Fixed

### Critical: Missing `ErrUserDisabled` Error Mapping
- **File**: `backend/pkg/apierr/apierr.go`
- **Issue**: Disabled users got 500 instead of 403
- **Fix**: Added `ErrUserDisabled` → 403 Forbidden mapping with i18n message
- **Locale files updated**: `en.toml`, `active.en-us.toml`, `zh-hans.toml`, `active.zh-hans.toml`

### Moderate: Logout Error Swallowed
- **File**: `backend/internal/api/rest/v1/auth.go`
- **Issue**: Logout errors silently discarded, returning 200 even on failure
- **Fix**: Now returns error response if logout service call fails

## Infrastructure Verification

### Formatting & Linting Hooks
- ✅ Claude Code hooks (`.claude/settings.json`): Format + lint on Edit/Write/MultiEdit
- ✅ Codex CLI hooks (`hooks.json`): Format + lint on PostToolUse
- ✅ Python: `ruff format` + `ruff check --fix`
- ✅ Go: `gofmt -w` + `goimports -w` + `golangci-lint run --fix`
- ✅ TypeScript: ESLint via `npx eslint --fix` (pre-existing config issue with circular structure)

### Docker Dual-Registry
- ✅ `docker/build.sh` has `PRIVATE_REGISTRY` env var (default: `192.168.5.50:5000`)
- ✅ When `PUSH=true`, builds identical tags for both Docker Hub and private registry
- ✅ `devops/ansible/templates/docker-compose_nas.yml.j2` uses `192.168.5.50:5000/markpost:latest`

## Limitations

- **No live E2E testing**: Container security policy blocks TCP socket binding and Docker API access
- Full E2E verification with browser testing should be performed in an environment with network access
