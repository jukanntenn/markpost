# 后端测试

[English](testing.md) | 中文

<a id="test-file-placement"></a>

## 测试文件放置

测试文件与其被测源文件同目录放置，遵循 Go 惯例：

```
internal/service/post/post.go
internal/service/post/post_test.go
internal/api/rest/v1/auth.go
internal/api/rest/v1/auth_test.go
```

<a id="test-database"></a>

## 测试数据库

`infra` 包提供 `SetupTestDB(t)`：启动（或复用）一个真实的 PostgreSQL testcontainer，应用全部内嵌迁移，并返回一个已连接的 `*gorm.DB`，其清理会在测试之间清空所有数据（`internal/infra/testdb.go`）：

```go
func TestSomething(t *testing.T) {
    db := infra.SetupTestDB(t)
    // db is a *gorm.DB against the shared postgres container
}
```

任何调用 `infra.SetupTestDB` 的包都把它的 `TestMain` 交给 `infra.RunTestMain` 路由（见 `internal/infra/main_test.go`）：共享容器存活期长于单个测试，由该包自行终止它。需要运行中的 Docker 守护进程；在没有守护进程的环境设置 `TESTCONTAINERS_SKIP=1` 跳过依赖容器的测试。

<a id="mock-repositories"></a>

## Mock 仓储

Service 测试使用实现 domain 接口的手写 mock 仓储：

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

<a id="test-patterns"></a>

## 测试模式

<a id="table-driven-tests"></a>

### 表驱动测试

测试用 `t.Run` 子测试组织：

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

<a id="handler-tests"></a>

### Handler 测试

Handler 测试搭建 Gin 测试上下文并校验 HTTP 响应：

```go
func TestLoginWithUsername(t *testing.T) {
    // Create router with mock service
    // Send request via httptest.NewRecorder
    // Assert status code and response body
}
```

<a id="running-tests"></a>

## 运行测试

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
