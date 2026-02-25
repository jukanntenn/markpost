# Quality Guidelines

> Code quality standards for backend development.

---

## Overview

This project follows Go best practices with emphasis on:
- Clean separation of concerns (handlers → services → repositories)
- Interface-based design for testability
- Comprehensive unit testing with table-driven tests
- Self-documenting code without excessive comments

---

## Forbidden Patterns

### 1. Business Logic in Handlers

Handlers should only parse requests and format responses:

```go
// Bad
func CreatePost(postSvc PostServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Business logic in handler
        if len(req.Title) > 100 {
            c.JSON(400, gin.H{"error": "title too long"})
            return
        }
        // ...
    }
}

// Good
func CreatePost(postSvc PostServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req PostRequest
        if !bindJSON(c, &req) {
            return
        }
        id, err := postSvc.CreatePost(req.Title, req.Body, user.ID)
        // ...
    }
}
```

### 2. Database Calls in Handlers

Always go through the service layer:

```go
// Bad
func GetPosts(repo PostRepoInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        posts, err := repo.GetPostsByUserID(userID, 0, 20)
        // ...
    }
}

// Good
func PostsList(postSvc PostServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        posts, total, err := postSvc.GetUserPosts(user.ID, query.Page, query.Limit)
        // ...
    }
}
```

### 3. Ignoring Errors

```go
// Bad
db.Create(&post)

// Good
if err := db.Create(&post).Error; err != nil {
    return fmt.Errorf("Post.Create: %w", err)
}
```

### 4. Global Mutable State

```go
// Bad
var currentUser *User

func SetUser(u *User) {
    currentUser = u
}

// Good - use context
func AuthMiddleware(c *gin.Context) {
    c.Set("user", user)
    c.Next()
}
```

---

## Required Patterns

### 1. Interface-Based Design

Define interfaces in repositories for testability:

```go
// repositories/post.go:13-22
type PostRepoInterface interface {
    CreatePost(title, body string, userID int) (*models.Post, error)
    CreatePosts(posts []models.Post) (int, error)
    GetPostByQID(qid string) (*models.Post, error)
    CountPostsByUserID(userID int) (int64, error)
    GetPostsByUserID(userID int, offset int, limit int) ([]models.Post, error)
    // ...
}
```

### 2. Constructor Functions

Use `New*` functions for initialization:

```go
// repositories/post.go:28-30
func NewPostRepo(database *models.Database) PostRepoInterface {
    return &PostRepo{database: database}
}

// services/post.go:20-31
func NewPostService(postRepo repositories.PostRepoInterface, delivery DeliveryEnqueuer) *PostService {
    // ...
}
```

### 3. Error Wrapping

Always wrap errors with context:

```go
// Good
if err != nil {
    return nil, fmt.Errorf("GetPost: %w", err)
}
```

### 4. Service Layer Error Handling

Use ServiceError for business errors:

```go
// services/post.go:51-57
if err == models.ErrNotFound {
    return "", "", NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("post with qid %s not found", qid), err)
}
return "", "", NewServiceErrorWrap(ErrInternal, fmt.Sprintf("get post with qid %s failed", qid), err)
```

---

## Testing Requirements

### Test File Organization

Tests are placed alongside source files:

```
backend/
├── handlers/
│   ├── post.go
│   └── post_test.go
├── services/
│   ├── post.go
│   └── post_test.go
└── repositories/
    ├── post.go
    └── post_test.go
```

### Test Structure

Use table-driven tests with subtests:

```go
// handlers/post_test.go:89-95
func TestCreatePost(t *testing.T) {
    t.Run("Missing title", testCreatePost_MissingTitle)
    t.Run("Missing body", testCreatePost_MissingBody)
    t.Run("Success", testCreatePost_Success)
    t.Run("Failed get user", testCreatePost_FailedGetUser)
    t.Run("Internal error", testCreatePost_InternalError)
}
```

### Stub/Mock Pattern

Use stubs for testing handlers:

```go
// handlers/post_test.go:31-83
type stubPostService struct {
    createPostFunc    func(title, body string, userID int) (string, error)
    // ...
    called            int
    lastTitle         string
    lastBody          string
}

func (s *stubPostService) CreatePost(title, body string, userID int) (string, error) {
    s.called++
    s.lastTitle = title
    s.lastBody = body
    if s.createPostFunc != nil {
        return s.createPostFunc(title, body, userID)
    }
    return "", nil
}
```

### Test Helpers

Use helper functions for common setup:

```go
// handlers/common_test.go
func newTestI18nRouter(t *testing.T) *gin.Engine {
    // Setup router with i18n support
}
```

---

## Code Review Checklist

- [ ] Handler only parses request and calls service
- [ ] Service contains business logic, not database details
- [ ] Repository abstracts all database operations
- [ ] Errors are wrapped with context using `fmt.Errorf("...: %w", err)`
- [ ] Business errors use ServiceError
- [ ] Tests cover both success and error cases
- [ ] No business logic in tests (use stubs/mocks)
- [ ] No global mutable state
- [ ] No hardcoded credentials or secrets
- [ ] Swagger comments on handler functions

---

## Swagger Documentation

Document all API endpoints:

```go
// handlers/post.go:28-40
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
// @Router       /{post_key} [post]
func CreatePost(postSvc PostServiceInterface) gin.HandlerFunc {
    // ...
}
```

Generate docs with: `swag init -g main.go -o docs --parseDependency --parseInternal`
