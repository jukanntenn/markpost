# Admin Management System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a complete admin management system for managing Users, Posts, and Delivery Channels with full CRUD operations.

**Architecture:** Nested routes (`/admin/users`, `/admin/posts`, `/admin/channels`) with shared components. Backend adds admin-specific endpoints protected by `RequireAdmin()` middleware. Frontend uses SWR for server state and React hooks for UI state.

**Tech Stack:** Go (Gin/GORM), React (TypeScript), SWR, shadcn/ui, Tailwind CSS v4

---

## Phase 1: Backend API

### Task 1: Create Admin Handlers File

**Files:**
- Create: `backend/handlers/admin.go`
- Test: `backend/handlers/admin_test.go`

**Step 1: Write the failing test**

Create `backend/handlers/admin_test.go`:

```go
package handlers

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/gin-gonic/gin"
    "markpost/models"
    "markpost/services"
)

// Mock service for testing
type mockAdminService struct {
    users []models.User
}

func (m *mockAdminService) GetAllUsers(page, limit int) ([]models.User, int64, error) {
    return m.users, int64(len(m.users)), nil
}

func (m *mockAdminService) UpdateUserRole(id int, role models.Role) (*models.User, error) {
    for i, u := range m.users {
        if u.ID == id {
            m.users[i].Role = role
            return &m.users[i], nil
        }
    }
    return nil, services.ErrNotFound
}

func TestUpdateUserRole(t *testing.T) {
    gin.SetMode(gin.TestMode)
    mockSvc := &mockAdminService{
        users: []models.User{
            {ID: 1, Username: "test", Role: models.RoleUser},
        },
    }
    handler := NewAdminHandler(mockSvc)

    t.Run("valid role change", func(t *testing.T) {
        w := httptest.NewRecorder()
        c, _ := gin.CreateTestContext(w)

        c.Request = httptest.NewRequest("PUT", "/api/admin/users/1/role", nil)
        c.Request.Header.Set("Content-Type", "application/json")
        c.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

        // Simulate authenticated admin user
        c.Set("user", &models.User{ID: 1, Role: models.RoleAdmin})

        // This will fail because handler doesn't exist yet
        handler.UpdateUserRole(c)

        if w.Code != http.StatusOK {
            t.Errorf("Expected status 200, got %d", w.Code)
        }
    })
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./handlers -run TestUpdateUserRole -v`
Expected: FAIL with "undefined: NewAdminHandler"

**Step 3: Write minimal implementation**

Create `backend/handlers/admin.go`:

```go
package handlers

import (
    "net/http"
    "strconv"

    apperrors "markpost/errors"
    "markpost/models"
    "markpost/services"

    "github.com/gin-gonic/gin"
)

type AdminServiceInterface interface {
    GetAllUsers(page, limit int) ([]models.User, int64, error)
    UpdateUserRole(id int, role models.Role) (*models.User, error)
    DeleteUser(id int) error
    GetAllPosts(query map[string]any, offset, limit int) ([]models.Post, int64, error)
    UpdatePost(id int, title, body string) (*models.Post, error)
    DeletePost(id int) error
    GetAllDeliveryChannels(query map[string]any) ([]models.DeliveryChannel, error)
    UpdateDeliveryChannel(id int, name *string, webhookURL *string, keywords *string, enabled *bool) (*models.DeliveryChannel, error)
    DeleteDeliveryChannel(id int) error
}

type AdminHandler struct {
    svc AdminServiceInterface
}

func NewAdminHandler(svc AdminServiceInterface) *AdminHandler {
    return &AdminHandler{svc: svc}
}

// UpdateUserRoleRequest represents request to change user role
type UpdateUserRoleRequest struct {
    Role models.Role `json:"role" binding:"required,oneof=admin user"`
}

// UpdatePostRequest represents request to update post
type UpdatePostRequest struct {
    Title string `json:"title"`
    Body  string `json:"body" binding:"required"`
}

// UpdateUserRole changes a user's role
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil || id <= 0 {
        apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "invalid user id", []services.ServiceError{
            {Code: services.ErrFieldViolation, Description: "id"},
        }))
        return
    }

    var req UpdateUserRoleRequest
    if !bindJSON(c, &req) {
        return
    }

    user, err := h.svc.UpdateUserRole(id, req.Role)
    if err != nil {
        apperrors.RespondError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{"user": user})
}

// DeleteUser deletes a user (cascade deletes their posts/channels)
func (h *AdminHandler) DeleteUser(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil || id <= 0 {
        apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "invalid user id", []services.ServiceError{
            {Code: services.ErrFieldViolation, Description: "id"},
        }))
        return
    }

    if err := h.svc.DeleteUser(id); err != nil {
        apperrors.RespondError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ListAllPosts returns all posts with optional filter
func (h *AdminHandler) ListAllPosts(c *gin.Context) {
    search := c.Query("search")
    pageStr := c.DefaultQuery("page", "1")
    limitStr := c.DefaultQuery("limit", "10")

    page, err := strconv.Atoi(pageStr)
    if err != nil || page < 1 {
        page = 1
    }

    limit, err := strconv.Atoi(limitStr)
    if err != nil || limit < 1 {
        limit = 10
    }

    query := make(map[string]any)
    if search != "" {
        query["title LIKE ?"] = "%" + search + "%"
    }

    posts, total, err := h.svc.GetAllPosts(query, (page-1)*limit, limit)
    if err != nil {
        apperrors.RespondError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "posts":     posts,
        "total":     total,
        "page":      page,
        "page_size": limit,
    })
}

// UpdatePost updates post content
func (h *AdminHandler) UpdatePost(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil || id <= 0 {
        apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "invalid post id", []services.ServiceError{
            {Code: services.ErrFieldViolation, Description: "id"},
        }))
        return
    }

    var req UpdatePostRequest
    if !bindJSON(c, &req) {
        return
    }

    post, err := h.svc.UpdatePost(id, req.Title, req.Body)
    if err != nil {
        apperrors.RespondError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{"post": post})
}

// DeletePost deletes a post
func (h *AdminHandler) DeletePost(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil || id <= 0 {
        apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "invalid post id", []services.ServiceError{
            {Code: services.ErrFieldViolation, Description: "id"},
        }))
        return
    }

    if err := h.svc.DeletePost(id); err != nil {
        apperrors.RespondError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ListAllDeliveryChannels returns all delivery channels
func (h *AdminHandler) ListAllDeliveryChannels(c *gin.Context) {
    channels, err := h.svc.GetAllDeliveryChannels(nil)
    if err != nil {
        apperrors.RespondError(c, err)
        return
    }

    resp := make([]DeliveryChannelResponse, 0, len(channels))
    for _, ch := range channels {
        resp = append(resp, DeliveryChannelResponse{
            ID:         ch.ID,
            Kind:       string(ch.Kind),
            Name:       ch.Name,
            Enabled:    ch.Enabled,
            WebhookURL: ch.WebhookURL,
            Keywords:   ch.Keywords,
            CreatedAt:  ch.CreatedAt.Format(timeFormatRFC3339),
            UpdatedAt:  ch.UpdatedAt.Format(timeFormatRFC3339),
        })
    }

    c.JSON(http.StatusOK, gin.H{"channels": resp})
}

// UpdateDeliveryChannel updates a delivery channel
func (h *AdminHandler) UpdateDeliveryChannel(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil || id <= 0 {
        apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "invalid channel id", []services.ServiceError{
            {Code: services.ErrFieldViolation, Description: "id"},
        }))
        return
    }

    var req updateDeliveryChannelRequest
    if !bindJSON(c, &req) {
        return
    }

    channel, err := h.svc.UpdateDeliveryChannel(id, req.Name, req.WebhookURL, req.Keywords, req.Enabled)
    if err != nil {
        apperrors.RespondError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "channel": DeliveryChannelResponse{
            ID:         channel.ID,
            Kind:       string(channel.Kind),
            Name:       channel.Name,
            Enabled:    channel.Enabled,
            WebhookURL: channel.WebhookURL,
            Keywords:   channel.Keywords,
            CreatedAt:  channel.CreatedAt.Format(timeFormatRFC3339),
            UpdatedAt:  channel.UpdatedAt.Format(timeFormatRFC3339),
        },
    })
}

// DeleteDeliveryChannel deletes a delivery channel
func (h *AdminHandler) DeleteDeliveryChannel(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil || id <= 0 {
        apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "invalid channel id", []services.ServiceError{
            {Code: services.ErrFieldViolation, Description: "id"},
        }))
        return
    }

    if err := h.svc.DeleteDeliveryChannel(id); err != nil {
        apperrors.RespondError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{"ok": true})
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./handlers -run TestUpdateUserRole -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/handlers/admin.go backend/handlers/admin_test.go
git commit -m "feat(handlers): add admin handlers for user/post/channel management"
```

---

### Task 2: Add Admin Service Methods

**Files:**
- Modify: `backend/services/auth.go` (or create new `backend/services/admin.go`)
- Test: `backend/services/admin_test.go` (if creating new file)

**Step 1: Write the failing test**

Create `backend/services/admin_test.go`:

```go
package services

import (
    "testing"
    "markpost/models"
)

func TestUpdateUserRole(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    user := &models.User{Username: "test", Password: "hash", PostKey: "key", Role: models.RoleUser}
    user.Create(db)

    svc := NewAuthService(db, nil, nil)

    updated, err := svc.UpdateUserRole(user.ID, models.RoleAdmin)
    if err != nil {
        t.Fatalf("UpdateUserRole failed: %v", err)
    }

    if updated.Role != models.RoleAdmin {
        t.Errorf("expected role admin, got %s", updated.Role)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./services -run TestUpdateUserRole -v`
Expected: FAIL with "undefined: NewAuthService.UpdateUserRole"

**Step 3: Write minimal implementation**

Add to `backend/services/auth.go` (or create new `admin.go`):

```go
// UpdateUserRole changes a user's role
func (s *AuthService) UpdateUserRole(id int, role models.Role) (*models.User, error) {
    user, err := models.GetUser(s.database, map[string]any{"id": id})
    if err != nil {
        return nil, ErrNotFound
    }

    user.Role = role
    if err := user.Update(s.database); err != nil {
        return nil, NewServiceErrorWrap(ErrFailedUpdateUser, "failed to update user role", err)
    }

    return user, nil
}

// DeleteUser deletes a user (cascade deletes their posts/channels via GORM)
func (s *AuthService) DeleteUser(id int) error {
    _, err := models.DeleteUsers(s.database, map[string]any{"id": id})
    if err != nil {
        return NewServiceErrorWrap(ErrFailedDeleteUser, "failed to delete user", err)
    }
    return nil
}

// GetAllPosts returns all posts with filter (admin bypasses user filtering)
func (s *AuthService) GetAllPosts(query map[string]any, offset, limit int) ([]models.Post, int64, error) {
    posts, err := models.GetPosts(s.database, query, offset, limit)
    if err != nil {
        return nil, 0, NewServiceErrorWrap(ErrFailedGetPosts, "failed to get posts", err)
    }

    total, err := models.CountPosts(s.database, query)
    if err != nil {
        return nil, 0, NewServiceErrorWrap(ErrFailedGetPosts, "failed to count posts", err)
    }

    return posts, total, nil
}

// UpdatePost updates post content (admin can edit any post)
func (s *AuthService) UpdatePost(id int, title, body string) (*models.Post, error) {
    post, err := models.GetPost(s.database, map[string]any{"id": id})
    if err != nil {
        return nil, ErrNotFound
    }

    post.Title = title
    post.Body = body
    if err := post.Update(s.database); err != nil {
        return nil, NewServiceErrorWrap(ErrFailedUpdatePost, "failed to update post", err)
    }

    return post, nil
}

// DeletePost deletes a post
func (s *AuthService) DeletePost(id int) error {
    _, err := models.DeletePosts(s.database, map[string]any{"id": id})
    if err != nil {
        return NewServiceErrorWrap(ErrFailedDeletePost, "failed to delete post", err)
    }
    return nil
}

// GetAllDeliveryChannels returns all channels (admin can see all)
func (s *AuthService) GetAllDeliveryChannels(query map[string]any) ([]models.DeliveryChannel, error) {
    channels, err := models.GetDeliveryChannels(s.database, query)
    if err != nil {
        return nil, NewServiceErrorWrap(ErrFailedGetChannels, "failed to get channels", err)
    }
    return channels, nil
}

// UpdateDeliveryChannelAdmin updates any channel (admin bypass)
func (s *AuthService) UpdateDeliveryChannelAdmin(id int, name *string, webhookURL *string, keywords *string, enabled *bool) (*models.DeliveryChannel, error) {
    channel, err := models.GetDeliveryChannel(s.database, map[string]any{"id": id})
    if err != nil {
        return nil, ErrNotFound
    }

    if name != nil {
        channel.Name = *name
    }
    if webhookURL != nil {
        channel.WebhookURL = *webhookURL
    }
    if keywords != nil {
        channel.Keywords = *keywords
    }
    if enabled != nil {
        channel.Enabled = *enabled
    }

    if err := channel.Update(s.database); err != nil {
        return nil, NewServiceErrorWrap(ErrFailedUpdateChannel, "failed to update channel", err)
    }

    return channel, nil
}

// DeleteDeliveryChannelAdmin deletes any channel (admin bypass)
func (s *AuthService) DeleteDeliveryChannelAdmin(id int) error {
    _, err := models.DeleteDeliveryChannels(s.database, map[string]any{"id": id})
    if err != nil {
        return NewServiceErrorWrap(ErrFailedDeleteChannel, "failed to delete channel", err)
    }
    return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./services -run TestUpdateUserRole -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/services/auth.go backend/services/admin_test.go
git commit -m "feat(services): add admin service methods for CRUD operations"
```

---

### Task 3: Register Admin Routes

**Files:**
- Modify: `backend/routes.go`

**Step 1: Update routes**

Modify the admin group in `backend/routes.go` (around line 53-59):

```go
admin := api.Group("/admin")
admin.Use(middlewares.Auth(jwtSvc, userRepo))
admin.Use(middlewares.RequireAdmin())
{
    userHandler := handlers.NewUserHandler(authSvc)
    admin.GET("/users", userHandler.ListAllUsers)

    adminHandler := handlers.NewAdminHandler(authSvc)
    // User management
    admin.PUT("/users/:id/role", adminHandler.UpdateUserRole)
    admin.DELETE("/users/:id", adminHandler.DeleteUser)

    // Post management
    admin.GET("/posts", adminHandler.ListAllPosts)
    admin.PUT("/posts/:id", adminHandler.UpdatePost)
    admin.DELETE("/posts/:id", adminHandler.DeletePost)

    // Delivery channel management
    admin.GET("/channels", adminHandler.ListAllDeliveryChannels)
    admin.PUT("/channels/:id", adminHandler.UpdateDeliveryChannel)
    admin.DELETE("/channels/:id", adminHandler.DeleteDeliveryChannel)
}
```

**Step 2: Run tests**

Run: `cd backend && go test ./... -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add backend/routes.go
git commit -m "feat(routes): register admin API endpoints"
```

---

### Task 4: Regenerate Swagger Documentation

**Files:**
- Modify: `backend/docs/` (auto-generated)

**Step 1: Add Swagger annotations to admin.go**

Add to `backend/handlers/admin.go` before each handler:

```go
// UpdateUserRole godoc
// @Summary      Update user role
// @Description  Update a user's role (admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "User ID"
// @Param        request body UpdateUserRoleRequest true "Role"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Router       /api/admin/users/{id}/role [put]
// @Security     BearerAuth
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
    // ... existing code
}

// Add similar annotations for all other admin handlers
```

**Step 2: Generate docs**

Run: `cd backend && swag init -g main.go -o docs --parseDependency --parseInternal`

**Step 3: Commit**

```bash
git add backend/docs/
git commit -m "docs: add swagger docs for admin endpoints"
```

---

## Phase 2: Frontend Foundation

### Task 5: Create Admin Dashboard Layout

**Files:**
- Create: `frontend/src/pages/AdminDashboard.tsx`
- Create: `frontend/src/components/admin/Sidebar.tsx`

**Step 1: Create Sidebar component**

Create `frontend/src/components/admin/Sidebar.tsx`:

```tsx
import { NavLink } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

interface NavItem {
  path: string;
  key: string;
}

const navItems: NavItem[] = [
  { path: "/admin/users", key: "users" },
  { path: "/admin/posts", key: "posts" },
  { path: "/admin/channels", key: "channels" },
];

export function AdminSidebar() {
  const { t } = useTranslation();

  return (
    <aside className="w-64 border-r bg-muted/40 p-4">
      <nav className="space-y-1">
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) =>
              cn(
                "block rounded-md px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground"
              )
            }
          >
            {t(`admin.nav.${item.key}`)}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
```

**Step 2: Create Dashboard layout**

Create `frontend/src/pages/AdminDashboard.tsx`:

```tsx
import { Outlet } from "react-router-dom";
import { AdminSidebar } from "@/components/admin/Sidebar";

export function AdminDashboard() {
  return (
    <div className="flex min-h-[calc(100vh-4rem)]">
      <AdminSidebar />
      <main className="flex-1 p-6">
        <Outlet />
      </main>
    </div>
  );
}
```

**Step 3: Commit**

```bash
git add frontend/src/pages/AdminDashboard.tsx frontend/src/components/admin/Sidebar.tsx
git commit -m "feat(admin): create dashboard layout with sidebar navigation"
```

---

### Task 6: Add i18n Translations

**Files:**
- Modify: `frontend/src/i18n/locales/en.json`
- Modify: `frontend/src/i18n/locales/zh.json`

**Step 1: Add English translations**

Add to `frontend/src/i18n/locales/en.json` in the "admin" section (around line 162):

```json
"admin": {
  "nav": {
    "users": "Users",
    "posts": "Posts",
    "channels": "Channels"
  },
  "title": "Admin",
  "users": {
    "title": "User Management",
    "actions": "Actions",
    "changeRole": "Change Role",
    "delete": "Delete User",
    "deleteConfirm": "Are you sure you want to delete this user? Their posts and channels will also be deleted.",
    "roleChanged": "User role updated",
    "roleChangeFailed": "Failed to update user role",
    "userDeleted": "User deleted",
    "deleteFailed": "Failed to delete user"
  },
  "posts": {
    "title": "Post Management",
    "search": "Search by title...",
    "table": {
      "id": "ID",
      "title": "Title",
      "user": "User",
      "createdAt": "Created At",
      "actions": "Actions"
    },
    "actions": {
      "edit": "Edit",
      "delete": "Delete"
    },
    "editDialog": {
      "title": "Edit Post",
      "titleLabel": "Title",
      "bodyLabel": "Body",
      "cancel": "Cancel",
      "save": "Save",
      "saving": "Saving..."
    },
    "errors": {
      "updateFailed": "Failed to update post",
      "deleteFailed": "Failed to delete post",
      "deleteConfirm": "Are you sure you want to delete this post?"
    },
    "postUpdated": "Post updated",
    "postDeleted": "Post deleted"
  },
  "channels": {
    "title": "Channel Management",
    "table": {
      "id": "ID",
      "user": "User",
      "name": "Name",
      "kind": "Type",
      "enabled": "Enabled",
      "actions": "Actions"
    },
    "actions": {
      "edit": "Edit",
      "delete": "Delete"
    },
    "editDialog": {
      "title": "Edit Channel",
      "nameLabel": "Name",
      "webhookLabel": "Webhook URL",
      "keywordsLabel": "Keywords",
      "enabledLabel": "Enabled",
      "cancel": "Cancel",
      "save": "Save",
      "saving": "Saving..."
    },
    "errors": {
      "updateFailed": "Failed to update channel",
      "deleteFailed": "Failed to delete channel",
      "deleteConfirm": "Are you sure you want to delete this channel?"
    },
    "channelUpdated": "Channel updated",
    "channelDeleted": "Channel deleted"
  },
  "errors": {
    "forbidden": "Access denied",
    "notFound": "Resource not found",
    "default": "Something went wrong"
  },
  "loading": "Loading...",
  "noData": "No data found",
  "previous": "Previous",
  "next": "Next",
  "page": "{{current}} / {{total}}",
  "total": "Total: {{count}}",
  "id": "id",
  "username": "Username",
  "role": "Role",
  "roleAdmin": "Admin",
  "roleUser": "User",
  "githubId": "GitHub ID",
  "createdAt": "Created At",
  "updatedAt": "Updated At",
  "error": "Failed to load data",
  "noUsers": "No users found"
}
```

**Step 2: Add Chinese translations**

Add similar keys to `frontend/src/i18n/locales/zh.json` with Chinese values.

**Step 3: Commit**

```bash
git add frontend/src/i18n/locales/
git commit -m "feat(i18n): add admin panel translations"
```

---

### Task 7: Update App Routing

**Files:**
- Modify: `frontend/src/App.tsx`

**Step 1: Update routes**

Modify the admin route in `frontend/src/App.tsx` (around line 116-122):

```tsx
const AdminDashboard = React.lazy(() => import("./pages/AdminDashboard"));
const AdminUsers = React.lazy(() => import("./pages/AdminUsers"));
const AdminPosts = React.lazy(() => import("./pages/AdminPosts")); // To be created
const AdminChannels = React.lazy(() => import("./pages/AdminChannels")); // To be created

// In Routes section, replace the admin route with:
<Route
  path="admin"
  element={
    <AdminRoute>
      <AdminDashboard />
    </AdminRoute>
  }
>
  <Route index element={<Navigate to="users" replace />} />
  <Route path="users" element={<AdminUsers />} />
  <Route path="posts" element={<AdminPosts />} />
  <Route path="channels" element={<AdminChannels />} />
</Route>
```

**Step 2: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "feat(routing): add nested admin routes"
```

---

### Task 8: Create Admin SWR Hooks

**Files:**
- Create: `frontend/src/hooks/swr/useAdminPosts.ts`
- Create: `frontend/src/hooks/swr/useAdminChannels.ts`
- Create: `frontend/src/hooks/swr/useAdminActions.ts`

**Step 1: Create useAdminPosts hook**

Create `frontend/src/hooks/swr/useAdminPosts.ts`:

```tsx
import useSWR from "swr";
import { authFetcher } from "../swr/fetcher";

export interface AdminPost {
  id: number;
  qid: string;
  title: string;
  body: string;
  user_id: number;
  created_at: string;
  updated_at: string;
  user?: {
    id: number;
    username: string;
  };
}

export interface AdminPostsResponse {
  posts: AdminPost[];
  total: number;
  page: number;
  page_size: number;
}

export function useAdminPosts(page: number, limit: number = 10, search: string = "") {
  const params = new URLSearchParams({
    page: String(page),
    limit: String(limit),
  });
  if (search) params.set("search", search);

  return useSWR<AdminPostsResponse>(
    page ? `/api/admin/posts?${params.toString()}` : null,
    authFetcher,
    {
      refreshWhenHidden: false,
      revalidateOnFocus: false,
    }
  );
}
```

**Step 2: Create useAdminChannels hook**

Create `frontend/src/hooks/swr/useAdminChannels.ts`:

```tsx
import useSWR from "swr";
import { authFetcher } from "../swr/fetcher";

export interface AdminDeliveryChannel {
  id: number;
  user_id: number;
  kind: string;
  name: string;
  enabled: boolean;
  webhook_url: string;
  keywords: string;
  created_at: string;
  updated_at: string;
  user?: {
    id: number;
    username: string;
  };
}

export interface AdminChannelsResponse {
  channels: AdminDeliveryChannel[];
}

export function useAdminChannels() {
  return useSWR<AdminChannelsResponse>("/api/admin/channels", authFetcher, {
    refreshWhenHidden: false,
    revalidateOnFocus: false,
  });
}
```

**Step 3: Create useAdminActions hook**

Create `frontend/src/hooks/swr/useAdminActions.ts`:

```tsx
import useSWRMutation from "swr/mutation";
import { auth } from "../../utils/api";
import { withAuthRefresh } from "../swr/middleware";

// User actions
export interface UpdateRoleArgs {
  id: number;
  role: string;
}

const sendUpdateRole = async (_url: string, { arg }: { arg: UpdateRoleArgs }) => {
  const res = await withAuthRefresh(() =>
    auth.put(`/api/admin/users/${arg.id}/role`, { role: arg.role })
  );
  return res.data;
};

export function useUpdateRole() {
  return useSWRMutation("/api/admin/users", sendUpdateRole);
}

export interface DeleteUserArgs {
  id: number;
}

const sendDeleteUser = async (_url: string, { arg }: { arg: DeleteUserArgs }) => {
  const res = await withAuthRefresh(() =>
    auth.delete(`/api/admin/users/${arg.id}`)
  );
  return res.data;
};

export function useDeleteUser() {
  return useSWRMutation("/api/admin/users", sendDeleteUser);
}

// Post actions
export interface UpdatePostArgs {
  id: number;
  title: string;
  body: string;
}

const sendUpdatePost = async (_url: string, { arg }: { arg: UpdatePostArgs }) => {
  const res = await withAuthRefresh(() =>
    auth.put(`/api/admin/posts/${arg.id}`, { title: arg.title, body: arg.body })
  );
  return res.data;
};

export function useUpdatePost() {
  return useSWRMutation("/api/admin/posts", sendUpdatePost);
}

const sendDeletePost = async (_url: string, { arg }: { arg: DeleteUserArgs }) => {
  const res = await withAuthRefresh(() =>
    auth.delete(`/api/admin/posts/${arg.id}`)
  );
  return res.data;
};

export function useDeletePost() {
  return useSWRMutation("/api/admin/posts", sendDeletePost);
}

// Channel actions
export interface UpdateChannelArgs {
  id: number;
  name?: string;
  webhook_url?: string;
  keywords?: string;
  enabled?: boolean;
}

const sendUpdateChannel = async (_url: string, { arg }: { arg: UpdateChannelArgs }) => {
  const res = await withAuthRefresh(() =>
    auth.put(`/api/admin/channels/${arg.id}`, {
      name: arg.name,
      webhook_url: arg.webhook_url,
      keywords: arg.keywords,
      enabled: arg.enabled,
    })
  );
  return res.data;
};

export function useUpdateChannel() {
  return useSWRMutation("/api/admin/channels", sendUpdateChannel);
}

const sendDeleteChannel = async (_url: string, { arg }: { arg: DeleteUserArgs }) => {
  const res = await withAuthRefresh(() =>
    auth.delete(`/api/admin/channels/${arg.id}`)
  );
  return res.data;
};

export function useDeleteChannel() {
  return useSWRMutation("/api/admin/channels", sendDeleteChannel);
}
```

**Step 4: Commit**

```bash
git add frontend/src/hooks/swr/useAdminPosts.ts frontend/src/hooks/swr/useAdminChannels.ts frontend/src/hooks/swr/useAdminActions.ts
git commit -m "feat(hooks): add admin SWR hooks for CRUD operations"
```

---

## Phase 3: Frontend Features

### Task 9: Create Shared Admin Components

**Files:**
- Create: `frontend/src/components/admin/ConfirmDialog.tsx`
- Create: `frontend/src/components/admin/RoleChangeDialog.tsx`
- Create: `frontend/src/components/admin/PostEditDialog.tsx`
- Create: `frontend/src/components/admin/ChannelEditDialog.tsx`

**Step 1: Create ConfirmDialog**

Create `frontend/src/components/admin/ConfirmDialog.tsx`:

```tsx
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useTranslation } from "react-i18next";

interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  onConfirm: () => void;
  isLoading?: boolean;
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  onConfirm,
  isLoading = false,
}: ConfirmDialogProps) {
  const { t } = useTranslation();

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isLoading}>{t("common.cancel")}</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm} disabled={isLoading}>
            {isLoading ? t("common.loading") : t("common.confirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
```

**Step 2: Create RoleChangeDialog**

Create `frontend/src/components/admin/RoleChangeDialog.tsx`:

```tsx
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useUpdateRole } from "@/hooks/swr/useAdminActions";
import type { User } from "@/hooks/swr/useUsers";

interface RoleChangeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: User | null;
  onSuccess: () => void;
}

export function RoleChangeDialog({
  open,
  onOpenChange,
  user,
  onSuccess,
}: RoleChangeDialogProps) {
  const { t } = useTranslation();
  const { trigger, isMutating } = useUpdateRole();
  const [selectedRole, setSelectedRole] = useState<"admin" | "user">("user");

  const handleSubmit = async () => {
    if (!user) return;

    try {
      await trigger({ id: user.id, role: selectedRole });
      toast.success(t("admin.users.roleChanged"));
      onSuccess();
      onOpenChange(false);
    } catch {
      toast.error(t("admin.users.roleChangeFailed"));
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("admin.users.changeRole")}</DialogTitle>
          <DialogDescription>
            {t("admin.users.user")}: {user?.username}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Button
            type="button"
            variant={selectedRole === "admin" ? "default" : "outline"}
            className="w-full justify-start"
            onClick={() => setSelectedRole("admin")}
          >
            <Badge variant="destructive" className="mr-2">
              {t("admin.roleAdmin")}
            </Badge>
          </Button>
          <Button
            type="button"
            variant={selectedRole === "user" ? "default" : "outline"}
            className="w-full justify-start"
            onClick={() => setSelectedRole("user")}
          >
            <Badge variant="secondary" className="mr-2">
              {t("admin.roleUser")}
            </Badge>
          </Button>
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isMutating}
          >
            {t("common.cancel")}
          </Button>
          <Button type="button" onClick={handleSubmit} disabled={isMutating}>
            {isMutating ? t("common.saving") : t("common.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

**Step 3: Create PostEditDialog and ChannelEditDialog**

Follow similar patterns for PostEditDialog and ChannelEditDialog. Reference existing components like `CreateTestPostModal.tsx` for patterns.

**Step 4: Commit**

```bash
git add frontend/src/components/admin/
git commit -m "feat(admin): add shared dialog components"
```

---

### Task 10: Refactor Admin to AdminUsers

**Files:**
- Create: `frontend/src/pages/AdminUsers.tsx`
- Delete: `frontend/src/pages/Admin.tsx`

**Step 1: Create AdminUsers**

Create `frontend/src/pages/AdminUsers.tsx` by copying and modifying existing `Admin.tsx` to add role change and delete actions:

```tsx
import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Loader2Icon, MoreHorizontalIcon, PencilIcon, TrashIcon } from "lucide-react";
import { toast } from "sonner";
import { useUsers, type User } from "../hooks/swr/useUsers";
import { useUpdateRole, useDeleteUser } from "../hooks/swr/useAdminActions";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { RoleChangeDialog } from "@/components/admin/RoleChangeDialog";
import { ConfirmDialog } from "@/components/admin/ConfirmDialog";

const AdminUsers: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const limit = 10;
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [roleDialogOpen, setRoleDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [userToDelete, setUserToDelete] = useState<User | null>(null);

  const { data, error, isLoading, mutate } = useUsers(page, limit);
  const { trigger: updateRole, isMutating: isUpdatingRole } = useUpdateRole();
  const { trigger: deleteUser, isMutating: isDeleting } = useDeleteUser();

  const handleRoleChange = (user: User) => {
    setSelectedUser(user);
    setRoleDialogOpen(true);
  };

  const handleDeleteClick = (user: User) => {
    setUserToDelete(user);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!userToDelete) return;

    try {
      await deleteUser({ id: userToDelete.id });
      toast.success(t("admin.users.userDeleted"));
      mutate();
      setDeleteDialogOpen(false);
      setUserToDelete(null);
    } catch {
      toast.error(t("admin.users.deleteFailed"));
    }
  };

  // ... rest of component similar to existing Admin.tsx with added actions

  return (
    <div className="mx-auto max-w-6xl">
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.users.title")}</CardTitle>
        </CardHeader>
        <CardContent>
          {/* Table with actions column */}
        </CardContent>
      </Card>

      <RoleChangeDialog
        open={roleDialogOpen}
        onOpenChange={setRoleDialogOpen}
        user={selectedUser}
        onSuccess={() => mutate()}
      />

      <ConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title={t("admin.users.delete")}
        description={userToDelete ? t("admin.users.deleteConfirm") : ""}
        onConfirm={handleDeleteConfirm}
        isLoading={isDeleting}
      />
    </div>
  );
};

export default AdminUsers;
```

**Step 2: Delete old Admin.tsx**

Run: `rm frontend/src/pages/Admin.tsx`

**Step 3: Commit**

```bash
git add frontend/src/pages/AdminUsers.tsx
git commit -m "refactor(admin): rename Admin to AdminUsers with actions"
```

---

### Task 11: Create AdminPosts Page

**Files:**
- Create: `frontend/src/pages/AdminPosts.tsx`

**Step 1: Create AdminPosts component**

Create `frontend/src/pages/AdminPosts.tsx` with full CRUD functionality:

```tsx
import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Loader2Icon, SearchIcon, EditIcon, TrashIcon } from "lucide-react";
import { toast } from "sonner";
import { useAdminPosts, type AdminPost } from "../hooks/swr/useAdminPosts";
import { useUpdatePost, useDeletePost } from "../hooks/swr/useAdminActions";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { PostEditDialog } from "@/components/admin/PostEditDialog";
import { ConfirmDialog } from "@/components/admin/ConfirmDialog";

const AdminPosts: React.FC = () => {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [editingPost, setEditingPost] = useState<AdminPost | null>(null);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [postToDelete, setPostToDelete] = useState<AdminPost | null>(null);

  const { data, error, isLoading, mutate } = useAdminPosts(page, 10, search);
  const { trigger: updatePost, isMutating: isUpdating } = useUpdatePost();
  const { trigger: deletePost, isMutating: isDeleting } = useDeletePost();

  const handleEdit = (post: AdminPost) => {
    setEditingPost(post);
    setEditDialogOpen(true);
  };

  const handleDeleteClick = (post: AdminPost) => {
    setPostToDelete(post);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!postToDelete) return;

    try {
      await deletePost({ id: postToDelete.id });
      toast.success(t("admin.posts.postDeleted"));
      mutate();
      setDeleteDialogOpen(false);
      setPostToDelete(null);
    } catch {
      toast.error(t("admin.posts.errors.deleteFailed"));
    }
  };

  const totalPages = data ? Math.ceil(data.total / data.page_size) : 1;

  return (
    <div className="mx-auto max-w-6xl">
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.posts.title")}</CardTitle>
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <SearchIcon className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder={t("admin.posts.search")}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setPage(1);
                }}
                className="pl-10"
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {/* Table with posts data and actions */}
        </CardContent>
      </Card>

      <PostEditDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        post={editingPost}
        onSuccess={() => mutate()}
      />

      <ConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title={t("admin.posts.actions.delete")}
        description={t("admin.posts.errors.deleteConfirm")}
        onConfirm={handleDeleteConfirm}
        isLoading={isDeleting}
      />
    </div>
  );
};

export default AdminPosts;
```

**Step 2: Commit**

```bash
git add frontend/src/pages/AdminPosts.tsx
git commit -m "feat(admin): create AdminPosts page with CRUD"
```

---

### Task 12: Create AdminChannels Page

**Files:**
- Create: `frontend/src/pages/AdminChannels.tsx`

**Step 1: Create AdminChannels component**

Follow similar pattern to AdminPosts but for delivery channels.

**Step 2: Commit**

```bash
git add frontend/src/pages/AdminChannels.tsx
git commit -m "feat(admin): create AdminChannels page with CRUD"
```

---

## Phase 4: Testing & Polish

### Task 13: Add Common Translations

**Files:**
- Modify: `frontend/src/i18n/locales/en.json`
- Modify: `frontend/src/i18n/locales/zh.json`

**Step 1: Add missing common keys**

Add to `frontend/src/i18n/locales/en.json`:

```json
"common": {
  "cancel": "Cancel",
  "confirm": "Confirm",
  "save": "Save",
  "saving": "Saving...",
  "loading": "Loading...",
  "delete": "Delete",
  "edit": "Edit",
  "actions": "Actions"
}
```

**Step 2: Add Chinese equivalents**

**Step 3: Commit**

```bash
git add frontend/src/i18n/locales/
git commit -m "feat(i18n): add common action translations"
```

---

### Task 14: Manual Testing Checklist

**Step 1: Run through checklist**

- [ ] Admin can view all users, change roles, delete users
- [ ] Admin can view all posts, edit, delete
- [ ] Admin can view all channels, edit, delete
- [ ] Non-admin gets 403 on admin endpoints
- [ ] Delete user cascades to their posts/channels
- [ ] Pagination works for all lists
- [ ] Filters work (search by username, post title)
- [ ] Mobile responsive tables/cards

**Step 2: Document any issues found**

Create follow-up tasks for bugs found.

---

### Task 15: Update Project Documentation

**Files:**
- Modify: `README.md`
- Modify: `docs/plans/2026-03-01-admin-management-system-design.md`

**Step 1: Update README**

Add section about admin panel access and features.

**Step 2: Mark design as implemented**

Update design document status.

**Step 3: Final commit**

```bash
git add README.md docs/plans/2026-03-01-admin-management-system-design.md
git commit -m "docs: update documentation for admin management system"
```

---

## Summary

**Total Tasks:** 15

**Files Created:**
- Backend: 2 handlers files, 1 service file
- Frontend: 8 components/pages, 3 hooks files, 2 translation files

**Files Modified:**
- Backend: routes.go, services files
- Frontend: App.tsx, i18n files, README

**Estimated Completion:** 4-5 days following TDD and frequent commits.
