# Backend Quality

> Testing patterns, logging, Swagger documentation, and code review standards.

## Forbidden Patterns

1. **Business logic in handlers** — handlers parse requests and call services only
2. **Database calls in handlers** — always go through the service layer
3. **Ignoring errors** — always check and handle `err`
4. **Global mutable state** — use `gin.Context` for request-scoped data

## Required Patterns

### Interface-Based Design

Define interfaces in repositories for testability:

```go
type PostRepoInterface interface {
    CreatePost(title, body string, userID int) (*models.Post, error)
    GetPostByQID(qid string) (*models.Post, error)
    // ...
}
```

### Constructor Functions

Use `New*` functions for all initialization:

```go
func NewPostRepo(database *models.Database) PostRepoInterface { ... }
func NewPostService(postRepo PostRepoInterface) *PostService { ... }
```

### Error Wrapping

Always wrap errors with context in the service layer:

```go
if err != nil { return nil, fmt.Errorf("GetPost: %w", err) }
```

## Testing

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
func TestCreatePost(t *testing.T) {
    t.Run("Missing title", testCreatePost_MissingTitle)
    t.Run("Missing body", testCreatePost_MissingBody)
    t.Run("Success", testCreatePost_Success)
    t.Run("Failed get user", testCreatePost_FailedGetUser)
    t.Run("Internal error", testCreatePost_InternalError)
}
```

### Stub/Mock Pattern

Use struct stubs for testing handlers and services:

```go
type stubPostService struct {
    createPostFunc func(title, body string, userID int) (string, error)
    called         int
    lastTitle      string
    lastBody       string
}

func (s *stubPostService) CreatePost(title, body string, userID int) (string, error) {
    s.called++
    s.lastTitle = title
    s.lastBody = body
    if s.createPostFunc != nil { return s.createPostFunc(title, body, userID) }
    return "", nil
}
```

### Error Testing

- In services: assert `services.AsServiceError(err)` returns the correct code
- In handlers: assert HTTP status code and `ErrorResponse.Code` match expectations

## Swagger Documentation

Document all API endpoints with godoc annotations:

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
// @Router       /{post_key} [post]
```

Generate: `swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal`

## Logging

### Setup

- **Library**: Go standard `log` package
- **Format**: Plain text with timestamp
- **Output**: stdout/stderr

### Log Levels

| Function | When to Use |
|----------|-------------|
| `log.Printf` | Info-level events (startup, configuration) |
| `log.Fatalf` | Fatal errors that should terminate the application |
| `log.Printf` with "error"/"failed" | Error conditions |

### What to Log

| DO | DO NOT |
|----|--------|
| Application startup and configuration | Passwords or password hashes |
| Initialization of critical components | JWT tokens or API keys |
| Admin user creation | User data / PII |
| Server listen address | Request bodies with sensitive data |
| Unexpected errors with context | |

### Common Mistakes

1. **Using `fmt.Println` instead of `log`** — always use `log.Println`/`log.Printf`
2. **Not including context** — `log.Printf("Failed to initialize first admin: %v", err)` not `log.Printf("Error: %v", err)`
3. **Logging sensitive information** — never log passwords, tokens, or user data
4. **Using `log.Fatal` in library code** — return errors to callers instead of terminating

## Code Review Checklist

- [ ] Handler only parses request and calls service
- [ ] Service contains business logic, not database details
- [ ] Repository abstracts all database operations
- [ ] Errors are wrapped with context using `fmt.Errorf("...: %w", err)`
- [ ] Business errors use `ServiceError`
- [ ] Tests cover both success and error cases
- [ ] No business logic in tests (use stubs/mocks)
- [ ] No global mutable state
- [ ] No hardcoded credentials or secrets
- [ ] Swagger comments on handler functions
