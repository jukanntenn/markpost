package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"markpost/internal/domain/post"
	"markpost/internal/service"
	"markpost/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type mockPostService struct {
	posts map[string]*post.Post
}

func newMockPostService() *mockPostService {
	return &mockPostService{
		posts: make(map[string]*post.Post),
	}
}

func newPostTestEngine() *gin.Engine {
	return testutil.NewTestEngine(testutil.TestEngineConfig{
		Validators: []testutil.ValidatorRegistration{
			{Tag: "titlesize", Fn: func(fl validator.FieldLevel) bool {
				return len(fl.Field().String()) <= 255
			}},
			{Tag: "bodysize", Fn: func(fl validator.FieldLevel) bool {
				return len(fl.Field().String()) <= 100000
			}},
		},
	})
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
	router := newPostTestEngine()

	router.POST("/posts", withUser(1), CreatePost(mockSvc))

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
	router := newPostTestEngine()

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

func TestRenderPost_NotFound(t *testing.T) {
	mockSvc := newMockPostService()
	router := newPostTestEngine()

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
	router := newPostTestEngine()

	// Create a post first
	_, _ = mockSvc.CreatePost(context.Background(), "Test Title", "Test Body", 1)

	router.GET("/posts", withUser(1), PostsList(mockSvc))

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
