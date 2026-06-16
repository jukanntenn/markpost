package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"markpost/internal/domain/post"
	"markpost/internal/infra"
	"markpost/internal/service"
	postsvc "markpost/internal/service/post"
)

type mockPostService struct {
	posts map[string]*post.Post
}

func newMockPostService() *mockPostService {
	return &mockPostService{
		posts: make(map[string]*post.Post),
	}
}

func (m *mockPostService) CreatePost(_ context.Context, title, body string, userID int) (string, error) {
	qid := "test-qid"
	m.posts[qid] = &post.Post{
		ID:        1,
		QID:       qid,
		Title:     title,
		Body:      body,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	return qid, nil
}

func (m *mockPostService) RenderPostHTML(_ context.Context, qid string) (string, string, error) {
	if p, ok := m.posts[qid]; ok {
		return p.Title, "<h1>" + p.Title + "</h1><p>" + p.Body + "</p>", nil
	}
	return "", "", service.NewServiceError(service.ErrNotFound, "post not found")
}

func (m *mockPostService) GetPostMarkdown(_ context.Context, qid string) (string, string, error) {
	if p, ok := m.posts[qid]; ok {
		return p.Title, p.Body, nil
	}
	return "", "", service.NewServiceError(service.ErrNotFound, "post not found")
}

func (m *mockPostService) GetUserPosts(_ context.Context, userID int, _, _ int) ([]post.Post, int64, error) {
	var result []post.Post
	for _, p := range m.posts {
		if p.UserID == userID {
			result = append(result, *p)
		}
	}
	return result, int64(len(result)), nil
}

func TestCreatePost_Success(t *testing.T) {
	mockSvc := newMockPostService()
	router := newTestEngine(withValidators(postValidators...))

	router.POST("/posts", withTestUser(1), CreatePost(mockSvc))

	body := PostRequest{Title: "Test Title", Body: "Test Body"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["id"] == nil {
		t.Error("expected id in response")
	}
}

func TestRenderPost_Success(t *testing.T) {
	mockSvc := newMockPostService()
	router := newTestEngine(withValidators(postValidators...))

	// Create a post first
	_, _ = mockSvc.CreatePost(context.Background(), "Test Title", "Test Body", 1)

	router.GET("/posts/:id", RenderPost(mockSvc))

	// Use format=raw to avoid HTML template rendering
	req := httptest.NewRequest(http.MethodGet, "/posts/test-qid?format=raw", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response content type is markdown
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/markdown; charset=utf-8" {
		t.Errorf("expected content type text/markdown; charset=utf-8, got %s", contentType)
	}
}

func TestRenderPost_HTMLIsSanitized(t *testing.T) {
	db := infra.SetupTestDB(t)
	repo := infra.NewPostRepository(db)
	svc := postsvc.NewService(repo, nil)

	body := "<script>alert(1)</script>\n\n| a | b |\n|---|---|\n| 1 | 2 |\n"
	qid, err := svc.CreatePost(context.Background(), "Sanitized", body, 1)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	router := newTestEngine()
	router.LoadHTMLGlob("../../../../templates/*")
	router.GET("/posts/:id", RenderPost(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts/"+qid, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusOK, w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html content type, got %q", ct)
	}

	html := w.Body.String()
	if strings.Contains(html, "<script") {
		t.Errorf("rendered response must not contain <script>\nbody: %s", html)
	}
	if !strings.Contains(html, "<table>") {
		t.Errorf("expected rendered GFM table in response\nbody: %s", html)
	}
	if !strings.Contains(html, `<h1 class="post-title">Sanitized</h1>`) {
		t.Errorf("expected rendered title in response\nbody: %s", html)
	}
}

func TestRenderPost_NotFound(t *testing.T) {
	mockSvc := newMockPostService()
	router := newTestEngine(withValidators(postValidators...))

	router.GET("/posts/:id", RenderPost(mockSvc))

	// Use format=raw to avoid i18n dependency in tests
	req := httptest.NewRequest(http.MethodGet, "/posts/nonexistent?format=raw", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestPostsList_Success(t *testing.T) {
	mockSvc := newMockPostService()
	router := newTestEngine(withValidators(postValidators...))

	// Create a post first
	_, _ = mockSvc.CreatePost(context.Background(), "Test Title", "Test Body", 1)

	router.GET("/posts", withTestUser(1), PostsList(mockSvc))

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["posts"] == nil {
		t.Error("expected posts in response")
	}
}

func TestCreatePost_InvalidBody(t *testing.T) {
	mockSvc := newMockPostService()
	router := newTestEngine(withValidators(postValidators...))

	router.POST("/posts", withTestUser(1), CreatePost(mockSvc))

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreatePost_ServiceError(t *testing.T) {
	errMock := &errorPostService{err: service.NewServiceError(service.ErrValidation, "title too long")}
	router := newTestEngine(withValidators(postValidators...))

	router.POST("/posts", withTestUser(1), CreatePost(errMock))

	body := PostRequest{Title: "Test", Body: "Body"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

type errorPostService struct {
	err error
}

func (m *errorPostService) CreatePost(_ context.Context, _, _ string, _ int) (string, error) {
	return "", m.err
}
func (m *errorPostService) RenderPostHTML(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}
func (m *errorPostService) GetPostMarkdown(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}
func (m *errorPostService) GetUserPosts(_ context.Context, _ int, _, _ int) ([]post.Post, int64, error) {
	return nil, 0, nil
}

func TestPostsList_PaginationError(t *testing.T) {
	mockSvc := newMockPostService()
	router := newTestEngine()

	router.GET("/posts", withTestUser(1), PostsList(mockSvc))

	req := httptest.NewRequest(http.MethodGet, "/posts?limit=999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
