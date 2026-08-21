# Backend Testing

## Test File Placement

Test files are placed alongside the source files they test, following Go convention:

```
internal/service/post/post.go
internal/service/post/post_test.go
internal/api/rest/v1/auth.go
internal/api/rest/v1/auth_test.go
```

## Test Database

The `infra` package provides `SetupTestDB(t)`, which starts (or reuses) a real PostgreSQL testcontainer, applies all embedded migrations, and returns a connected `*gorm.DB` whose cleanup truncates all data between tests (`internal/infra/testdb.go`):

```go
func TestSomething(t *testing.T) {
    db := infra.SetupTestDB(t)
    // db is a *gorm.DB against the shared postgres container
}
```

Any package calling `infra.SetupTestDB` routes its `TestMain` through `infra.RunTestMain` (see `internal/infra/main_test.go`): the shared container outlives individual tests, and the package terminates it itself. A running Docker daemon is required; set `TESTCONTAINERS_SKIP=1` to skip container-backed tests where none exists.

## Mock Repositories

Service tests use hand-written mock repositories that implement the domain interfaces:

```go
type mockPostRepository struct {
    posts   map[string]*post.Post
    idPosts map[int]*post.Post
    nextID  int
}

func newMockPostRepository() *mockPostRepository {
    return &mockPostRepository{
        posts:   make(map[string]*post.Post),
        idPosts: make(map[int]*post.Post),
        nextID:  1,
    }
}

// Implement all interface methods...
func (m *mockPostRepository) GetByQID(_ context.Context, qid string) (*post.Post, error) {
    p, ok := m.posts[qid]
    if !ok {
        return nil, post.ErrNotFound
    }
    return p, nil
}
```

## Test Patterns

### Table-Driven Tests

Tests are organized using subtests with `t.Run`:

```go
func TestService_CreatePost(t *testing.T) {
    mockRepo := newMockPostRepository()
    svc := NewService(mockRepo, nil)
    ctx := context.Background()

    t.Run("creates post successfully", func(t *testing.T) {
        qid, err := svc.CreatePost(ctx, "Test Title", "Test Body", 1)
        if err != nil {
            t.Fatalf("expected no error, got: %v", err)
        }
        if qid == "" {
            t.Error("expected qid, got empty")
        }
    })

    t.Run("returns error for non-existent post", func(t *testing.T) {
        _, _, err := svc.GetPostMarkdown(ctx, "nonexistent")
        if err == nil {
            t.Fatal("expected error for non-existent post")
        }
    })
}
```

### Handler Tests

Handler tests set up a Gin test context and verify HTTP responses:

```go
func TestLoginWithUsername(t *testing.T) {
    // Create router with mock service
    // Send request via httptest.NewRecorder
    // Assert status code and response body
}
```

## Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/service/post/...

# Verbose output
go test -v ./...

# Run a specific test
go test -run TestService_CreatePost ./internal/service/post/
```
