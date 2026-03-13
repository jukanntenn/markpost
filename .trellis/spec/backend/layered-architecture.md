# Layered Architecture

> Backend architecture: Handler → Service → Repository → Domain

---

## Overview

```
┌─────────────────────────────────────────┐
│  Handler Layer (HTTP)                   │
│  - Parse/validate requests              │
│  - Extract context                      │
│  - Format responses                     │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Service Layer (Business Logic)         │
│  - Business rules                       │
│  - Orchestration                        │
│  - Error wrapping                       │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Repository Layer (Data Access)         │
│  - Database operations                  │
│  - Query building                       │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Domain Layer (Entities)                │
│  - GORM models                          │
│  - Domain errors                        │
└─────────────────────────────────────────┘
```

---

## Layer Responsibilities

### Handler Layer

**Location**: `internal/api/rest/v1/`

- Parse and validate incoming requests
- Extract user context from `gin.Context`
- Call service methods
- Format and return HTTP responses

```go
func CreatePost(postSvc PostService) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req PostRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            RespondError(c, err)
            return
        }
        id, err := postSvc.CreatePost(req.Title, req.Body, user.ID)
        if err != nil {
            RespondError(c, err)
            return
        }
        c.JSON(http.StatusOK, gin.H{"id": id})
    }
}
```

### Service Layer

**Location**: `internal/service/`

- Implement business rules
- Orchestrate repository calls
- Wrap errors with domain errors

```go
func (s *PostService) CreatePost(title, body string, userID int) (string, error) {
    post, err := s.repo.CreatePost(title, body, userID)
    if err != nil {
        return "", fmt.Errorf("create post: %w", err)
    }
    return post.QID, nil
}
```

### Repository Layer

**Location**: `internal/infra/database/`

- Abstract database operations
- Define interfaces for testability
- Execute queries

```go
type PostRepository interface {
    CreatePost(title, body string, userID int) (*post.Post, error)
    GetPostByQID(qid string) (*post.Post, error)
}

type postRepo struct {
    db *gorm.DB
}

func (r *postRepo) CreatePost(title, body string, userID int) (*post.Post, error) {
    p := &post.Post{Title: title, Body: body, UserID: userID}
    return p, r.db.Create(p).Error
}
```

### Domain Layer

**Location**: `internal/domain/`

- Define GORM models
- Domain errors

```go
type Post struct {
    ID     int    `gorm:"primaryKey"`
    QID    string `gorm:"unique"`
    Title  string
    Body   string
    UserID int
}
```

---

## Error Handling Flow

```
Domain Layer
    ↓ Returns: error (raw)
Repository Layer
    ↓ Returns: error (raw)
Service Layer
    ↓ Wraps with context: fmt.Errorf("...: %w", err)
Handler Layer
    ↓ Uses: RespondError(c, err)
HTTP Response
```

---

## Dependency Injection

```go
// Repository
func NewPostRepository(db *gorm.DB) PostRepository {
    return &postRepo{db: db}
}

// Service
func NewPostService(repo PostRepository) *PostService {
    return &PostService{repo: repo}
}

// Handler
func CreatePost(svc *PostService) gin.HandlerFunc { ... }
```

---

## Layer Violations to Avoid

| Violation | Correct |
|-----------|---------|
| Handler accessing DB directly | Handler → Service → Repository |
| Service returning raw errors | Service wraps with context |
| Repository importing services | Repository returns raw errors |
| Domain layer with HTTP logic | Domain is pure data |

---

## Testing Strategy

| Layer | Test Approach |
|-------|---------------|
| Domain | Unit tests with in-memory DB |
| Repository | Integration tests with test DB |
| Service | Unit tests with mock repositories |
| Handler | HTTP tests with mock services |
