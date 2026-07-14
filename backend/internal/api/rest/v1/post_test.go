package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"markpost/internal/domain/post"
	"markpost/internal/infra"
	"markpost/internal/service"
	postsvc "markpost/internal/service/post"

	"github.com/cespare/xxhash/v2"
)

type mockPostService struct {
	posts map[string]*post.Post
}

func fmtEtag(s string) string {
	return fmt.Sprintf("%016x", xxhash.Sum64String(s))
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

func (m *mockPostService) RenderPostHTML(_ context.Context, qid string) (string, string, string, time.Time, error) {
	if p, ok := m.posts[qid]; ok {
		html := "<h1>" + p.Title + "</h1><p>" + p.Body + "</p>"
		etag := fmtEtag(html)
		return p.Title, html, etag, p.CreatedAt, nil
	}
	return "", "", "", time.Time{}, service.New(service.ErrNotFound, "post not found")
}

func (m *mockPostService) GetPostMarkdown(_ context.Context, qid string) (string, string, string, time.Time, error) {
	if p, ok := m.posts[qid]; ok {
		return p.Title, p.Body, fmtEtag("# " + p.Title + "\n\n" + p.Body), p.CreatedAt, nil
	}
	return "", "", "", time.Time{}, service.New(service.ErrNotFound, "post not found")
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

func (m *mockPostService) DeletePostByQID(_ context.Context, qid string, ownerID int) error {
	p, ok := m.posts[qid]
	if !ok {
		return service.New(service.ErrNotFound, "post not found")
	}
	if ownerID > 0 && p.UserID != ownerID {
		return service.New(service.ErrNotFound, "post not found")
	}
	delete(m.posts, qid)
	return nil
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

func TestRenderPost_CacheHeadersAnd304(t *testing.T) {
	t.Run("raw sets cache headers and 304 on match", func(t *testing.T) {
		mockSvc := newMockPostService()
		router := newTestEngine()
		_, _ = mockSvc.CreatePost(context.Background(), "Title", "Body", 1)
		router.GET("/posts/:id", RenderPost(mockSvc))

		etag := fmtEtag("# Title\n\nBody")

		// First request: 200 with headers
		req := httptest.NewRequest(http.MethodGet, "/posts/test-qid?format=raw", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("first: expected 200, got %d", w.Code)
		}
		for _, h := range []string{"ETag", "Cache-Control", "Cache-Tag", "Vary", "Last-Modified"} {
			if w.Header().Get(h) == "" {
				t.Errorf("first: expected header %q set, headers: %v", h, w.Header())
			}
		}
		if got := w.Header().Get("ETag"); got != `"`+etag+`"` {
			t.Errorf("ETag = %q, want %q", got, `"`+etag+`"`)
		}
		if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=300, s-maxage=3600" {
			t.Errorf("Cache-Control = %q, want %q", cc, "public, max-age=300, s-maxage=3600")
		}

		// Conditional request with matching If-None-Match: 304, no body
		req2 := httptest.NewRequest(http.MethodGet, "/posts/test-qid?format=raw", nil)
		req2.Header.Set("If-None-Match", `"`+etag+`"`)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusNotModified {
			t.Fatalf("second: expected 304, got %d", w2.Code)
		}
		if w2.Body.Len() != 0 {
			t.Errorf("304 must have empty body, got %d bytes", w2.Body.Len())
		}
	})

	t.Run("non-matching If-None-Match yields 200", func(t *testing.T) {
		mockSvc := newMockPostService()
		router := newTestEngine()
		_, _ = mockSvc.CreatePost(context.Background(), "Title", "Body", 1)
		router.GET("/posts/:id", RenderPost(mockSvc))

		req := httptest.NewRequest(http.MethodGet, "/posts/test-qid?format=raw", nil)
		req.Header.Set("If-None-Match", `"deadbeef"`)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200 on mismatch, got %d", w.Code)
		}
	})

	t.Run("wildcard If-None-Match yields 304", func(t *testing.T) {
		mockSvc := newMockPostService()
		router := newTestEngine()
		_, _ = mockSvc.CreatePost(context.Background(), "Title", "Body", 1)
		router.GET("/posts/:id", RenderPost(mockSvc))

		req := httptest.NewRequest(http.MethodGet, "/posts/test-qid?format=raw", nil)
		req.Header.Set("If-None-Match", "*")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotModified {
			t.Errorf("expected 304 on wildcard, got %d", w.Code)
		}
	})
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

func TestDeleteOwnPost(t *testing.T) {
	t.Run("owner deletes own post returns 204", func(t *testing.T) {
		mockSvc := newMockPostService()
		router := newTestEngine()
		_, _ = mockSvc.CreatePost(context.Background(), "T", "B", 7)
		router.DELETE("/posts/:id", withTestUser(7), DeleteOwnPost(mockSvc))

		req := httptest.NewRequest(http.MethodDelete, "/posts/test-qid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d", w.Code)
		}
	})

	t.Run("wrong owner returns 404", func(t *testing.T) {
		mockSvc := newMockPostService()
		router := newTestEngine()
		_, _ = mockSvc.CreatePost(context.Background(), "T", "B", 7)
		router.DELETE("/posts/:id", withTestUser(99), DeleteOwnPost(mockSvc))

		req := httptest.NewRequest(http.MethodDelete, "/posts/test-qid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 for wrong owner, got %d", w.Code)
		}
	})

	t.Run("nonexistent returns 404", func(t *testing.T) {
		mockSvc := newMockPostService()
		router := newTestEngine()
		router.DELETE("/posts/:id", withTestUser(7), DeleteOwnPost(mockSvc))

		req := httptest.NewRequest(http.MethodDelete, "/posts/missing", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 for missing, got %d", w.Code)
		}
	})
}

func TestDeleteAnyPost_AdminDeletesAnyOwner(t *testing.T) {
	mockSvc := newMockPostService()
	router := newTestEngine()
	_, _ = mockSvc.CreatePost(context.Background(), "T", "B", 42)
	router.DELETE("/admin/posts/:id", DeleteAnyPost(mockSvc))

	req := httptest.NewRequest(http.MethodDelete, "/admin/posts/test-qid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("admin delete expected 204, got %d", w.Code)
	}
}

func TestCreatePost_ServiceError(t *testing.T) {
	errMock := &errorPostService{err: service.New(service.ErrValidation, "title too long")}
	router := newTestEngine(withValidators(postValidators...))

	router.POST("/posts", withTestUser(1), CreatePost(errMock))

	body := PostRequest{Title: "Test", Body: "Body"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}
}

type errorPostService struct {
	err error
}

func (m *errorPostService) CreatePost(_ context.Context, _, _ string, _ int) (string, error) {
	return "", m.err
}
func (m *errorPostService) RenderPostHTML(_ context.Context, _ string) (string, string, string, time.Time, error) {
	return "", "", "", time.Time{}, nil
}
func (m *errorPostService) GetPostMarkdown(_ context.Context, _ string) (string, string, string, time.Time, error) {
	return "", "", "", time.Time{}, nil
}
func (m *errorPostService) GetUserPosts(_ context.Context, _ int, _, _ int) ([]post.Post, int64, error) {
	return nil, 0, nil
}
func (m *errorPostService) DeletePostByQID(_ context.Context, _ string, _ int) error {
	return m.err
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
