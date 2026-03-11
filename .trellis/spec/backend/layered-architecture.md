# Layered Architecture

> How the backend is organized into layers: models → repositories → services → handlers

---

## Overview

This project follows a **strict layered architecture** with clear separation of concerns:

```
┌─────────────────────────────────────────┐
│           HTTP Request                  │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Handlers Layer (HTTP Layer)            │
│  - Parse requests                       │
│  - Validate input                       │
│  - Extract context                      │
│  - Format responses                     │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Services Layer (Business Logic)        │
│  - Business rules                       │
│  - Orchestration                        │
│  - Data transformation                  │
│  - Error wrapping                       │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Repositories Layer (Data Access)       │
│  - Database operations                  │
│  - Query building                       │
│  - Data persistence                     │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Models Layer (Data Structures)         │
│  - GORM models                          │
│  - Database schemas                     │
│  - Domain errors                        │
│  - Basic CRUD methods                   │
└─────────────────────────────────────────┘
```

---

## Layer Responsibilities

### 1. Models Layer

**Location**: `backend/models/`

**Purpose**: Define data structures and database schemas

**Responsibilities**:
- Define GORM models with struct tags
- Implement model-level CRUD methods (for simple operations)
- Define domain errors (e.g., `ErrNotFound`)
- Encapsulate basic database operations

**Key Files**:
- `models/post.go` - Post model and operations
- `models/user.go` - User model and operations
- `models/delivery_channel.go` - DeliveryChannel model
- `models/database.go` - Database connection wrapper
- `models/errors.go` - Domain error definitions

**Example**:

```go
// models/post.go:11-20
type Post struct {
    ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
    QID       string    `json:"qid" gorm:"unique;not null;column:qid"`
    Title     string    `json:"title" gorm:"not null"`
    Body      string    `json:"body" gorm:"not null;type:text"`
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
    UserID    int       `json:"user_id" gorm:"index;not null;column:user_id"`
    User      User      `json:"user" gorm:"constraint:OnDelete:CASCADE"`
}

// Model-level CRUD method
func (model *Post) Create(database *Database) error {
    db := database.DB()
    if err := db.Create(model).Error; err != nil {
        return fmt.Errorf("Post.Create: %w", err)
    }
    return nil
}

// Query function
func GetPost(database *Database, query map[string]any) (*Post, error) {
    db := database.DB()
    var post Post
    err := db.Take(&post, query).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("GetPost: %w", err)
    }
    return &post, nil
}
```

**Domain Errors**:

```go
// models/errors.go:5
var ErrNotFound = errors.New("record not found")
```

**Rules**:
- Models should be pure data structures with GORM tags
- Basic CRUD methods are allowed (Create, Update, Delete)
- Query functions should be simple and reusable
- Return `ErrNotFound` for missing records
- **Never** include business logic

---

### 2. Repositories Layer

**Location**: `backend/repositories/`

**Purpose**: Abstract database operations and provide data access interfaces

**Responsibilities**:
- Define repository interfaces for testability
- Implement data access methods
- Build and execute database queries
- Handle database transactions
- Call model methods or use GORM directly

**Key Files**:
- `repositories/post.go` - PostRepo interface and implementation
- `repositories/user.go` - UserRepo interface and implementation
- `repositories/delivery_channel.go` - DeliveryChannelRepo

**Example**:

```go
// repositories/post.go:14-28
type PostRepoInterface interface {
    CreatePost(title, body string, userID int) (*models.Post, error)
    CreatePosts(posts []models.Post) (int, error)
    GetPostByQID(qid string) (*models.Post, error)
    GetPostByID(id int) (*models.Post, error)
    CountPostsByUserID(userID int) (int64, error)
    GetPostsByUserID(userID int, offset int, limit int) ([]models.Post, error)
    ListAllPosts(search string, offset int, limit int) ([]models.Post, error)
    CountAllPosts(search string) (int64, error)
    UpdatePostByID(id int, title string, body string) (*models.Post, error)
    DeletePostByID(id int) (int64, error)
    PruneExpiredPosts(retentionDays int, batchSize int) error
    CountExpiredPosts(retentionDays int) (int64, error)
}

// Implementation
type PostRepo struct {
    database *models.Database
}

func NewPostRepo(database *models.Database) PostRepoInterface {
    return &PostRepo{database: database}
}

func (r *PostRepo) CreatePost(title, body string, userID int) (*models.Post, error) {
    qid, err := gonanoid.New()
    if err != nil {
        return nil, err
    }

    post := models.Post{
        QID:    qid,
        Title:  title,
        Body:   body,
        UserID: userID,
    }
    err = post.Create(r.database)
    if err != nil {
        return nil, err
    }

    return &post, nil
}

func (r *PostRepo) GetPostByQID(qid string) (*models.Post, error) {
    post, err := models.GetPost(r.database, map[string]any{"qid": qid})
    if err == nil {
        return post, nil
    }
    return nil, err
}
```

**Rules**:
- Always define interfaces for dependency injection and testing
- Return raw errors (let Service layer wrap them)
- Use model methods when available
- Handle GORM-specific logic (e.g., `gorm.ErrRecordNotFound`)
- **Never** include business logic or validation
- **Never** return `ServiceError`

---

### 3. Services Layer

**Location**: `backend/services/`

**Purpose**: Implement business logic and orchestrate operations

**Responsibilities**:
- Implement business rules and validation
- Orchestrate multiple repository calls
- Transform data between layers
- Handle cross-cutting concerns (e.g., delivery dispatch)
- Wrap all errors with `ServiceError`
- Coordinate transactions across repositories

**Key Files**:
- `services/post.go` - PostService
- `services/auth.go` - AuthService
- `services/admin.go` - AdminService
- `services/delivery_channels.go` - DeliveryChannelService
- `services/errors.go` - ServiceError definitions

**Example**:

```go
// services/post.go:14-18
type PostService struct {
    postRepo repositories.PostRepoInterface
    md       goldmark.Markdown
    delivery DeliveryEnqueuer
}

func NewPostService(postRepo repositories.PostRepoInterface, delivery DeliveryEnqueuer) *PostService {
    md := goldmark.New(
        goldmark.WithRendererOptions(
            html.WithUnsafe(),
        ),
    )
    return &PostService{
        postRepo: postRepo,
        md:       md,
        delivery: delivery,
    }
}

func (s *PostService) CreatePost(title, body string, userID int) (string, error) {
    post, err := s.postRepo.CreatePost(title, body, userID)
    if err != nil {
        return "", NewServiceErrorWrap(ErrInternal, "create post failed", err)
    }
    if s.delivery != nil {
        s.delivery.Enqueue(DeliveryJob{
            UserID:  userID,
            PostQID: post.QID,
            Title:   post.Title,
            Body:    post.Body,
        })
    }
    return post.QID, nil
}

func (s *PostService) RenderPostHTML(qid string) (string, string, error) {
    post, err := s.postRepo.GetPostByQID(qid)
    if err != nil {
        if err == models.ErrNotFound {
            return "", "", NewServiceErrorWrap(ErrNotFound,
                fmt.Sprintf("post with qid %s not found", qid), err)
        }
        return "", "", NewServiceErrorWrap(ErrInternal,
            fmt.Sprintf("get post with qid %s failed", qid), err)
    }

    var buf bytes.Buffer
    if err := s.md.Convert([]byte(post.Body), &buf); err != nil {
        return "", "", NewServiceErrorWrap(ErrInternal,
            fmt.Sprintf("convert post with qid %s failed", qid), err)
    }

    return post.Title, buf.String(), nil
}
```

**Business Validation Example**:

```go
// services/delivery_channels.go:83-116
func validateChannel(kind models.DeliveryChannelKind, webhookURL string) *ServiceError {
    if strings.TrimSpace(webhookURL) == "" {
        return NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
            {Code: ErrRequired, Description: "webhook_url"},
        })
    }

    u, err := url.Parse(webhookURL)
    if err != nil || u.Scheme == "" || u.Host == "" {
        return NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
            {Code: ErrFieldViolation, Description: "webhook_url"},
        })
    }
    if u.Scheme != "https" {
        return NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
            {Code: ErrFieldViolation, Description: "webhook_url"},
        })
    }

    switch kind {
    case models.DeliveryChannelKindFeishu:
        host := strings.ToLower(u.Hostname())
        if host == "open.feishu.cn" || strings.HasSuffix(host, ".feishu.cn") || strings.HasSuffix(host, ".larksuite.com") {
            return nil
        }
        return NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
            {Code: ErrFieldViolation, Description: "webhook_url"},
        })
    default:
        return NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
            {Code: ErrFieldViolation, Description: "kind"},
        })
    }
}
```

**Rules**:
- Always wrap errors with `ServiceError`
- Check for specific error types (e.g., `models.ErrNotFound`)
- Implement business validation logic
- Orchestrate multiple repository calls
- **Never** access database directly (use repositories)
- **Never** handle HTTP concerns (that's handler's job)

---

### 4. Handlers Layer

**Location**: `backend/handlers/`

**Purpose**: Handle HTTP requests and responses

**Responsibilities**:
- Parse and validate incoming requests
- Extract user context from `gin.Context`
- Call service methods
- Format and return HTTP responses
- Define request/response DTOs
- Use `apperrors.RespondError` for error responses

**Key Files**:
- `handlers/post.go` - Post handlers
- `handlers/auth.go` - Auth handlers
- `handlers/admin.go` - Admin handlers
- `handlers/delivery_channels.go` - DeliveryChannel handlers
- `handlers/common.go` - Shared utilities

**Example**:

```go
// handlers/post.go:16-26
type PostRequest struct {
    Title string `json:"title" binding:"required,titlesize"`
    Body  string `json:"body" binding:"required,bodysize"`
}

type PostServiceInterface interface {
    CreatePost(title, body string, userID int) (string, error)
    RenderPostHTML(qid string) (string, string, error)
    GetPostMarkdown(qid string) (string, string, error)
    GetUserPosts(userID int, page, limit int) ([]models.Post, int64, error)
}

// handlers/post.go:40-62
func CreatePost(postSvc PostServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        user, ok := ExtractUser(c)
        if !ok {
            err := services.NewServiceErrorWrap(services.ErrFailedGetUser,
                "failed to get user from context", nil)
            apperrors.RespondError(c, err)
            return
        }

        var req PostRequest
        if !bindJSON(c, &req) {
            return
        }

        id, err := postSvc.CreatePost(req.Title, req.Body, user.ID)
        if err != nil {
            apperrors.RespondError(c, err)
            return
        }

        c.JSON(http.StatusOK, gin.H{"id": id})
    }
}
```

**Request Validation**:

```go
// handlers/common.go:25-93
func bindJSON(c *gin.Context, req interface{}) bool {
    if err := c.ShouldBindJSON(req); err != nil {
        return writeBindingError(c, req, err)
    }
    return true
}

func writeBindingError(c *gin.Context, req interface{}, err error) bool {
    var causes []services.ServiceError
    if ve, ok := err.(validator.ValidationErrors); ok {
        t := reflect.TypeOf(req)
        if t.Kind() == reflect.Ptr {
            t = t.Elem()
        }
        for _, fe := range ve {
            fieldName := fe.Field()
            jsonField := fieldName
            if t.Kind() == reflect.Struct {
                if sf, ok := t.FieldByName(fieldName); ok {
                    tag := sf.Tag.Get("json")
                    if tag == "" {
                        tag = sf.Tag.Get("form")
                    }
                    if tag != "" {
                        parts := strings.Split(tag, ",")
                        if parts[0] != "" {
                            jsonField = parts[0]
                        }
                    }
                }
            }
            tag := fe.Tag()
            var code services.ErrCode
            switch tag {
            case "required":
                code = services.ErrRequired
            case "min":
                code = services.ErrMinLength
            default:
                code = services.ErrFieldViolation
            }
            causes = append(causes, services.ServiceError{
                Code:        code,
                Description: jsonField,
            })
        }
    }
    if len(causes) == 0 {
        causes = append(causes, services.ServiceError{
            Code:        services.ErrFieldViolation,
            Description: "",
        })
    }

    errResp := &services.ServiceError{
        Code:        services.ErrValidation,
        Description: "request validation failed",
        Details:     causes,
    }
    apperrors.RespondError(c, errResp)
    return false
}
```

**Rules**:
- Always define service interfaces for dependency injection
- Use `bindJSON` and `bindQuery` for request binding
- Use `ExtractUser` to get authenticated user
- Always use `apperrors.RespondError` for error responses
- **Never** implement business logic (call service methods)
- **Never** access database directly (use services)

---

## Data Flow Example

### Complete Request Flow: Create Post

```
1. HTTP Request
   POST /{post_key}
   {
     "title": "My Post",
     "body": "Content here"
   }

2. Handler Layer (handlers/post.go:40-62)
   ├── Extract user from context
   ├── Validate request (bindJSON)
   ├── Call service: postSvc.CreatePost(title, body, user.ID)
   └── Return HTTP response

3. Service Layer (services/post.go:33-47)
   ├── Call repository: postRepo.CreatePost(title, body, userID)
   ├── Wrap error if any: NewServiceErrorWrap(ErrInternal, ...)
   ├── Enqueue delivery job (business logic)
   └── Return post QID

4. Repository Layer (repositories/post.go:38-56)
   ├── Generate QID (gonanoid.New)
   ├── Create Post model instance
   ├── Call model method: post.Create(r.database)
   └── Return post or error

5. Model Layer (models/post.go:22-29)
   ├── Execute GORM operation: db.Create(model)
   ├── Wrap GORM error if any
   └── Return nil or error

6. Response Flow (reverse)
   Model → Repository → Service → Handler → HTTP Response
   {
     "id": "abc123"
   }
```

---

## Dependency Injection Pattern

### Constructor Injection

Each layer receives dependencies through constructors:

```go
// Models - receives database
func NewDatabase(dsn string) (*Database, error)

// Repository - receives database
func NewPostRepo(database *models.Database) PostRepoInterface {
    return &PostRepo{database: database}
}

// Service - receives repositories
func NewPostService(postRepo repositories.PostRepoInterface, delivery DeliveryEnqueuer) *PostService {
    return &PostService{
        postRepo: postRepo,
        md:       goldmark.New(goldmark.WithRendererOptions(html.WithUnsafe())),
        delivery: delivery,
    }
}

// Handler - receives services
func CreatePost(postSvc PostServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        // handler implementation
    }
}
```

### Interface-Based Design

Always use interfaces for testability:

```go
// Repository Interface
type PostRepoInterface interface {
    CreatePost(title, body string, userID int) (*models.Post, error)
    GetPostByQID(qid string) (*models.Post, error)
    // ... more methods
}

// Service Interface
type PostServiceInterface interface {
    CreatePost(title, body string, userID int) (string, error)
    RenderPostHTML(qid string) (string, string, error)
    // ... more methods
}
```

---

## Error Handling Flow

```
Model Layer
    ↓ Returns: error (e.g., models.ErrNotFound)
Repository Layer
    ↓ Returns: error (raw GORM errors or model errors)
Service Layer
    ↓ Wraps with: ServiceError (NewServiceErrorWrap)
Handler Layer
    ↓ Uses: apperrors.RespondError(c, err)
HTTP Response
    {
      "code": "not_found",
      "message": "Not Found"
    }
```

See `error-handling.md` for complete error handling guidelines.

---

## Testing Strategy

### Unit Testing Each Layer

#### Models Layer Testing

```go
func TestPost_Create(t *testing.T) {
    db, _ := models.NewTestDatabase()
    post := &models.Post{
        QID:    "test123",
        Title:  "Test",
        Body:   "Content",
        UserID: 1,
    }
    err := post.Create(db)
    assert.NoError(t, err)
}
```

#### Repository Layer Testing

```go
func TestPostRepo_CreatePost(t *testing.T) {
    db, _ := models.NewTestDatabase()
    repo := repositories.NewPostRepo(db)

    post, err := repo.CreatePost("Title", "Body", 1)
    assert.NoError(t, err)
    assert.NotEmpty(t, post.QID)
}
```

#### Service Layer Testing

```go
func TestPostService_CreatePost(t *testing.T) {
    db, _ := models.NewTestDatabase()
    repo := repositories.NewPostRepo(db)
    svc := services.NewPostService(repo, nil)

    qid, err := svc.CreatePost("Title", "Body", 1)
    assert.NoError(t, err)
    assert.NotEmpty(t, qid)
}
```

#### Handler Layer Testing

```go
func TestCreatePost(t *testing.T) {
    db, _ := models.NewTestDatabase()
    repo := repositories.NewPostRepo(db)
    svc := services.NewPostService(repo, nil)
    handler := handlers.CreatePost(svc)

    // Use httptest to test HTTP handler
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    // ... setup request and context

    handler(c)
    assert.Equal(t, http.StatusOK, w.Code)
}
```

---

## Common Patterns

### Pattern 1: Simple CRUD

```
Handler → Service → Repository → Model
```

Example: CreatePost, GetPost, DeletePost

### Pattern 2: Business Logic

```
Handler → Service (validation + orchestration) → Repository → Model
```

Example: CreateUser (with unique username check), UpdateDeliveryChannel (with webhook validation)

### Pattern 3: Cross-Cutting Concerns

```
Handler → Service (orchestration) → Multiple Repositories + External Services
```

Example: CreatePost with delivery dispatch

```go
func (s *PostService) CreatePost(title, body string, userID int) (string, error) {
    // 1. Create post via repository
    post, err := s.postRepo.CreatePost(title, body, userID)
    if err != nil {
        return "", NewServiceErrorWrap(ErrInternal, "create post failed", err)
    }

    // 2. Enqueue delivery job (cross-cutting concern)
    if s.delivery != nil {
        s.delivery.Enqueue(DeliveryJob{
            UserID:  userID,
            PostQID: post.QID,
            Title:   post.Title,
            Body:    post.Body,
        })
    }

    return post.QID, nil
}
```

---

## Layer Violations to Avoid

### ❌ Bad: Handler accessing database directly

```go
// handlers/post.go
func CreatePost(c *gin.Context) {
    db := c.MustGet("db").(*models.Database)
    var post models.Post
    c.ShouldBindJSON(&post)
    db.DB().Create(&post)  // ❌ Violation!
}
```

### ✅ Good: Handler calling service

```go
func CreatePost(postSvc PostServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req PostRequest
        c.ShouldBindJSON(&req)
        id, err := postSvc.CreatePost(req.Title, req.Body, user.ID)  // ✅ Correct
        // ...
    }
}
```

### ❌ Bad: Service returning raw error

```go
func (s *PostService) CreatePost(title, body string, userID int) (string, error) {
    post, err := s.postRepo.CreatePost(title, body, userID)
    if err != nil {
        return "", err  // ❌ Violation! Should wrap with ServiceError
    }
    return post.QID, nil
}
```

### ✅ Good: Service wrapping error

```go
func (s *PostService) CreatePost(title, body string, userID int) (string, error) {
    post, err := s.postRepo.CreatePost(title, body, userID)
    if err != nil {
        return "", NewServiceErrorWrap(ErrInternal, "create post failed", err)  // ✅ Correct
    }
    return post.QID, nil
}
```

### ❌ Bad: Repository returning ServiceError

```go
func (r *PostRepo) CreatePost(title, body string, userID int) (*models.Post, error) {
    // ...
    if err != nil {
        return nil, services.NewServiceError(...)  // ❌ Violation! Repository should not know about ServiceError
    }
}
```

### ✅ Good: Repository returning raw error

```go
func (r *PostRepo) CreatePost(title, body string, userID int) (*models.Post, error) {
    // ...
    if err != nil {
        return nil, err  // ✅ Correct - let Service layer wrap it
    }
}
```

---

## Summary

### Layer Checklist

| Layer | Returns | Error Handling | Dependencies |
|-------|---------|----------------|--------------|
| **Models** | Model instances, errors | Raw errors (e.g., `ErrNotFound`) | Database |
| **Repositories** | Model instances, errors | Raw errors (pass through) | Database |
| **Services** | Data, errors | Wrapped with `ServiceError` | Repositories |
| **Handlers** | HTTP responses | Use `apperrors.RespondError` | Services |

### Key Principles

1. **Single Responsibility**: Each layer has one clear purpose
2. **Dependency Injection**: Dependencies passed through constructors
3. **Interface-Based**: Use interfaces for testability
4. **Error Propagation**: Models → raw errors → Services → ServiceError → Handlers → HTTP
5. **No Layer Skipping**: Handler → Service → Repository → Model (never skip a layer)
6. **Business Logic in Services**: All business rules belong in the service layer

### When to Add Code

| Scenario | Where to Add |
|----------|--------------|
| New entity | Define in `models/` |
| New query method | Add to `repositories/` |
| New business rule | Implement in `services/` |
| New API endpoint | Create in `handlers/` |
| New validation | Service layer (business) or Handler layer (request) |

---

**Language**: All documentation is written in **English**.
