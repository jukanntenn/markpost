# Admin Management System Design

**Date:** 2026-03-01
**Status:** Approved
**Author:** AI Design Session

## Overview

A complete admin management system for managing all existing data in the markpost application: Users, Posts, and Delivery Channels.

## Requirements

### User Management (Basic)
- View list of all users (existing)
- Change user role (admin/user)
- Delete users

### Post Management (Standard)
- View all posts with filtering
- Edit post content (title, body)
- Delete posts

### Delivery Channel Management (Standard)
- View all channels with user info
- Edit channel settings
- Delete channels

## Architecture

### Routing Structure
```
/admin          → AdminDashboard (layout with sidebar/nav)
  /users        → AdminUsers.tsx
  /posts        → AdminPosts.tsx (new)
  /channels     → AdminChannels.tsx (new)
```

### Frontend Components

#### Page Components
| Component | Location | Purpose |
|-----------|----------|---------|
| `AdminDashboard.tsx` | `pages/AdminDashboard.tsx` | Layout with sidebar navigation |
| `AdminUsers.tsx` | `pages/AdminUsers.tsx` | User list with actions |
| `AdminPosts.tsx` | `pages/AdminPosts.tsx` | Posts list with edit/delete |
| `AdminChannels.tsx` | `pages/AdminChannels.tsx` | Channels list with edit/delete |

#### Shared Components
| Component | Purpose |
|-----------|---------|
| `AdminTable.tsx` | Reusable table with pagination, sort, filter |
| `ConfirmDialog.tsx` | Delete confirmation dialog |
| `PostEditDialog.tsx` | Edit post title/body |
| `ChannelEditDialog.tsx` | Edit channel settings |
| `RoleChangeDialog.tsx` | Change user role |

#### New SWR Hooks
- `hooks/swr/useAdminPosts.ts`
- `hooks/swr/useAdminChannels.ts`
- `hooks/swr/useAdminActions.ts`

### Backend API

#### New Admin Endpoints
```
# User Management (admin only)
PUT    /api/admin/users/:id/role     - Change user role
DELETE /api/admin/users/:id          - Delete user

# Post Management (admin only)
GET    /api/admin/posts              - List all posts (with filter)
PUT    /api/admin/posts/:id          - Update post content
DELETE /api/admin/posts/:id          - Delete post

# Delivery Channel Management (admin only)
GET    /api/admin/channels           - List all channels
PUT    /api/admin/channels/:id        - Update channel
DELETE /api/admin/channels/:id       - Delete channel
```

#### Request/Response Types
```go
// PUT /api/admin/users/:id/role
type UpdateRoleRequest struct {
    Role string `json:"role" binding:"required,oneof=admin user"`
}

// PUT /api/admin/posts/:id
type UpdatePostRequest struct {
    Title string `json:"title"`
    Body  string `json:"body" binding:"required"`
}

// GET /api/admin/posts?search=xxx&page=1&limit=10
type AdminPostsListRequest struct {
    Search string `form:"search"`
    Page   int    `form:"page" binding:"min=1"`
    Limit  int    `form:"limit" binding:"min=1,max=100"`
}
```

## Data Flow

### Read Flow (List Views)
```
Component → SWR Hook → API GET → Handler → Service → Repository → DB
              ↓                                        ↓
         Cache/Revalidate                        GORM Query
```

### Update Flow
```
EditDialog → Form Submit → API PUT → Handler → Service → Repository → DB
     ↓              ↓              ↓
  Optimistic    Validation    Error Handling
   Update
```

### Delete Flow
```
ConfirmDialog → Confirm → API DELETE → Handler → Service → Repository → DB
       ↓              ↓              ↓
   showDialog    Cascade     Success: refresh list
                 delete
```

## Error Handling

| Scenario | Handler Action | UI Response |
|----------|----------------|-------------|
| Invalid input | Return 400 | Inline validation error |
| Not found | Return 404 | Toast "Resource not found" |
| Unauthorized | Return 401 | Redirect to login |
| Forbidden (non-admin) | Return 403 | Toast "Access denied" |
| Server error | Return 500 | Toast "Operation failed" |
| Network error | N/A | Toast "Network error" |

### i18n Keys to Add
```json
{
  "admin": {
    "errors": {
      "forbidden": "Access denied",
      "notFound": "Resource not found",
      "deleteFailed": "Failed to delete",
      "updateFailed": "Failed to update",
      "default": "Something went wrong"
    }
  }
}
```

## Testing Strategy

### Unit Tests (Backend)
- `handlers/admin_test.go` - Test all admin endpoints
- Test permission checks (admin-only, non-admin rejected)
- Test cascade deletes work correctly

### Unit Tests (Frontend)
- Component tests for dialogs (EditDialog, ConfirmDialog)
- Hook tests for admin SWR hooks
- Mock API responses with MSW

### Integration Tests
- Create admin user → login → perform CRUD operations
- Verify non-admin cannot access admin endpoints
- Verify cascade deletes (user → posts/channels deleted)

### Manual Testing Checklist
- [ ] Admin can view all users, change roles, delete users
- [ ] Admin can view all posts, edit, delete
- [ ] Admin can view all channels, edit, delete
- [ ] Non-admin gets 403 on admin endpoints
- [ ] Delete user cascades to their posts/channels
- [ ] Pagination works for all lists
- [ ] Filters work (search by username, post title)
- [ ] Mobile responsive tables/cards

## Implementation Order

### Phase 1: Backend (Day 1-2)
1. Add admin handlers (`handlers/admin.go`)
2. Add admin routes to `routes.go`
3. Write backend tests
4. Regenerate Swagger docs

### Phase 2: Frontend - Foundation (Day 2-3)
1. Create AdminDashboard layout with navigation
2. Refactor existing Admin.tsx → AdminUsers.tsx
3. Create shared components (AdminTable, dialogs)
4. Add i18n translations

### Phase 3: Frontend - Features (Day 3-4)
1. Implement AdminPosts page with CRUD
2. Implement AdminChannels page with CRUD
3. Add role change to AdminUsers
4. Add delete to AdminUsers

### Phase 4: Testing & Polish (Day 4-5)
1. Write frontend tests
2. Manual testing checklist
3. Fix bugs, responsive adjustments
4. Documentation update

## Summary

**Deliverables:**
- 3 new admin pages (Users, Posts, Channels)
- 9 new admin API endpoints
- Shared admin components
- Full CRUD for all entities
- Mobile responsive design

**Estimated Time:** 4-5 days
