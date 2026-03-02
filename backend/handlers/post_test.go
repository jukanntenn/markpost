package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"markpost/models"
	"markpost/services"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("titlesize", func(fl validator.FieldLevel) bool {
			return true
		})
		v.RegisterValidation("bodysize", func(fl validator.FieldLevel) bool {
			return true
		})
	}
}

type stubPostService struct {
	createPostFunc      func(title, body string, userID int) (string, error)
	renderPostHTMLFunc  func(qid string) (string, string, error)
	getPostMarkdownFunc func(qid string) (string, string, error)
	getUserPostsFunc    func(userID int, page, limit int) ([]models.Post, int64, error)
	called              int
	lastTitle           string
	lastBody            string
	lastUserID          int
	lastQID             string
	lastPage            int
	lastLimit           int
}

func (s *stubPostService) CreatePost(title, body string, userID int) (string, error) {
	s.called++
	s.lastTitle = title
	s.lastBody = body
	s.lastUserID = userID
	if s.createPostFunc != nil {
		return s.createPostFunc(title, body, userID)
	}
	return "", nil
}

func (s *stubPostService) RenderPostHTML(qid string) (string, string, error) {
	s.called++
	s.lastQID = qid
	if s.renderPostHTMLFunc != nil {
		return s.renderPostHTMLFunc(qid)
	}
	return "", "", nil
}

func (s *stubPostService) GetPostMarkdown(qid string) (string, string, error) {
	s.called++
	s.lastQID = qid
	if s.getPostMarkdownFunc != nil {
		return s.getPostMarkdownFunc(qid)
	}
	return "", "", nil
}

func (s *stubPostService) GetUserPosts(userID int, page, limit int) ([]models.Post, int64, error) {
	s.called++
	s.lastUserID = userID
	s.lastPage = page
	s.lastLimit = limit
	if s.getUserPostsFunc != nil {
		return s.getUserPostsFunc(userID, page, limit)
	}
	return nil, 0, nil
}

type createPostResponse struct {
	ID string `json:"id"`
}

func TestCreatePost(t *testing.T) {
	t.Run("Missing title", testCreatePost_MissingTitle)
	t.Run("Missing body", testCreatePost_MissingBody)
	t.Run("Success", testCreatePost_Success)
	t.Run("Failed get user", testCreatePost_FailedGetUser)
	t.Run("Internal error", testCreatePost_InternalError)
}

func testCreatePost_MissingTitle(t *testing.T) {
	svc := &stubPostService{}
	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.POST("/test_post_key", CreatePost(svc))

	body := `{"body": "Test post content"}`
	req := httptest.NewRequest(http.MethodPost, "/test_post_key", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var resp validationErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrValidation) {
		t.Fatalf("expected code %q, got %q", string(services.ErrValidation), resp.Code)
	}

	if len(resp.Errors) == 0 {
		t.Fatal("expected validation errors")
	}

	if resp.Errors[0].Field != "title" {
		t.Fatalf("expected field 'title', got %q", resp.Errors[0].Field)
	}
}

func testCreatePost_MissingBody(t *testing.T) {
	svc := &stubPostService{}
	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.POST("/test_post_key", CreatePost(svc))

	body := `{"title": "Test Post"}`
	req := httptest.NewRequest(http.MethodPost, "/test_post_key", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var resp validationErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrValidation) {
		t.Fatalf("expected code %q, got %q", string(services.ErrValidation), resp.Code)
	}

	if len(resp.Errors) == 0 {
		t.Fatal("expected validation errors")
	}

	if resp.Errors[0].Field != "body" {
		t.Fatalf("expected field 'body', got %q", resp.Errors[0].Field)
	}
}

func testCreatePost_Success(t *testing.T) {
	expectedQID := "post-qid-123"

	svc := &stubPostService{
		createPostFunc: func(title, body string, userID int) (string, error) {
			if title != "Test Post" {
				t.Fatalf("expected title 'Test Post', got %q", title)
			}
			if body != "Test post content" {
				t.Fatalf("expected body 'Test post content', got %q", body)
			}
			if userID != 123 {
				t.Fatalf("expected userID 123, got %d", userID)
			}
			return expectedQID, nil
		},
	}

	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.POST("/test_post_key", CreatePost(svc))

	body := `{"title": "Test Post", "body": "Test post content"}`
	req := httptest.NewRequest(http.MethodPost, "/test_post_key", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp createPostResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.ID != expectedQID {
		t.Fatalf("expected id %q, got %q", expectedQID, resp.ID)
	}

	if svc.called != 1 {
		t.Fatalf("expected CreatePost called once, got %d", svc.called)
	}
}

func testCreatePost_FailedGetUser(t *testing.T) {
	svc := &stubPostService{}
	r := newTestI18nRouter(t)
	r.POST("/test_post_key", CreatePost(svc))

	body := `{"title": "Test Post", "body": "Test content"}`
	req := httptest.NewRequest(http.MethodPost, "/test_post_key", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrFailedGetUser) {
		t.Fatalf("expected code %q, got %q", string(services.ErrFailedGetUser), resp.Code)
	}
}

func testCreatePost_InternalError(t *testing.T) {
	svc := &stubPostService{
		createPostFunc: func(title, body string, userID int) (string, error) {
			return "", services.NewServiceErrorWrap(
				services.ErrInternal,
				"database error",
				errors.New("connection failed"),
			)
		},
	}

	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.POST("/test_post_key", CreatePost(svc))

	body := `{"title": "Test Post", "body": "Test content"}`
	req := httptest.NewRequest(http.MethodPost, "/test_post_key", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrInternal) {
		t.Fatalf("expected code %q, got %q", string(services.ErrInternal), resp.Code)
	}
}

func TestRenderPost(t *testing.T) {
	t.Run("Not found", func(t *testing.T) {
		t.Run("Chinese", testRenderPost_NotFound_ZH)
		t.Run("English", testRenderPost_NotFound_EN)
	})
	t.Run("Render error", func(t *testing.T) {
		t.Run("Chinese", testRenderPost_RenderError_ZH)
		t.Run("English", testRenderPost_RenderError_EN)
	})
	t.Run("Raw", func(t *testing.T) {
		t.Run("Success", testRenderPost_Raw_Success)
		t.Run("Not found", testRenderPost_Raw_NotFound)
		t.Run("Render error", testRenderPost_Raw_RenderError)
	})
}

func testRenderPost_NotFound_ZH(t *testing.T) {
	svc := &stubPostService{
		renderPostHTMLFunc: func(qid string) (string, string, error) {
			return "", "", services.NewServiceError(
				services.ErrNotFound,
				"post not found",
			)
		},
	}

	r := newTestI18nRouter(t)
	r.GET("/posts/:id", RenderPost(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts/invalid-qid", nil)
	req.Header.Set("Accept-Language", "zh")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "未找到") {
		t.Fatalf("expected response to contain '未找到', got %q", body)
	}
}

func testRenderPost_NotFound_EN(t *testing.T) {
	svc := &stubPostService{
		renderPostHTMLFunc: func(qid string) (string, string, error) {
			return "", "", services.NewServiceError(
				services.ErrNotFound,
				"post not found",
			)
		},
	}

	r := newTestI18nRouter(t)
	r.GET("/posts/:id", RenderPost(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts/invalid-qid", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Not Found") {
		t.Fatalf("expected response to contain 'Not Found', got %q", body)
	}
}

func testRenderPost_RenderError_ZH(t *testing.T) {
	svc := &stubPostService{
		renderPostHTMLFunc: func(qid string) (string, string, error) {
			return "", "", services.NewServiceErrorWrap(
				services.ErrInternal,
				"markdown conversion failed",
				errors.New("invalid markdown"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.GET("/posts/:id", RenderPost(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts/post-qid-123", nil)
	req.Header.Set("Accept-Language", "zh")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "渲染文章失败") {
		t.Fatalf("expected response to contain '渲染文章失败', got %q", body)
	}
}

func testRenderPost_RenderError_EN(t *testing.T) {
	svc := &stubPostService{
		renderPostHTMLFunc: func(qid string) (string, string, error) {
			return "", "", services.NewServiceErrorWrap(
				services.ErrInternal,
				"markdown conversion failed",
				errors.New("invalid markdown"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.GET("/posts/:id", RenderPost(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts/post-qid-123", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Failed to render post") {
		t.Fatalf("expected response to contain 'Failed to render post', got %q", body)
	}
}

func testRenderPost_Raw_Success(t *testing.T) {
	svc := &stubPostService{
		getPostMarkdownFunc: func(qid string) (string, string, error) {
			if qid != "post-qid-123" {
				t.Fatalf("expected qid %q, got %q", "post-qid-123", qid)
			}
			return "Test Title", "Test body", nil
		},
	}

	r := newTestI18nRouter(t)
	r.GET("/posts/:id", RenderPost(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts/post-qid-123?format=raw", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected content-type to contain %q, got %q", "text/markdown", contentType)
	}

	expectedBody := "# Test Title\n\nTest body"
	if rec.Body.String() != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, rec.Body.String())
	}
}

func testRenderPost_Raw_NotFound(t *testing.T) {
	svc := &stubPostService{
		getPostMarkdownFunc: func(qid string) (string, string, error) {
			return "", "", services.NewServiceError(
				services.ErrNotFound,
				"post not found",
			)
		},
	}

	r := newTestI18nRouter(t)
	r.GET("/posts/:id", RenderPost(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts/invalid-qid?format=raw", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Not Found") {
		t.Fatalf("expected response to contain 'Not Found', got %q", rec.Body.String())
	}
}

func testRenderPost_Raw_RenderError(t *testing.T) {
	svc := &stubPostService{
		getPostMarkdownFunc: func(qid string) (string, string, error) {
			return "", "", services.NewServiceErrorWrap(
				services.ErrInternal,
				"get post failed",
				errors.New("db error"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.GET("/posts/:id", RenderPost(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts/post-qid-123?format=raw", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Failed to render post") {
		t.Fatalf("expected response to contain 'Failed to render post', got %q", rec.Body.String())
	}
}

type postListResponse struct {
	Posts      []postItem `json:"posts"`
	Pagination pagination `json:"pagination"`
}

type postItem struct {
	ID        int       `json:"id"`
	QID       string    `json:"qid"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func TestPostsList(t *testing.T) {
	t.Run("Success", testPostsList_Success)
	t.Run("Default pagination", testPostsList_DefaultPagination)
	t.Run("Invalid page", testPostsList_InvalidPage)
	t.Run("Invalid limit", testPostsList_InvalidLimit)
	t.Run("Limit exceeded", testPostsList_LimitExceeded)
	t.Run("Failed get user", testPostsList_FailedGetUser)
	t.Run("Internal error", testPostsList_InternalError)
}

func testPostsList_Success(t *testing.T) {
	expectedPosts := []models.Post{
		{ID: 1, QID: "post-1", Title: "Post 1", CreatedAt: time.Now()},
		{ID: 2, QID: "post-2", Title: "Post 2", CreatedAt: time.Now()},
	}
	expectedTotal := int64(100)

	svc := &stubPostService{
		getUserPostsFunc: func(userID int, page, limit int) ([]models.Post, int64, error) {
			if userID != 123 {
				t.Fatalf("expected userID 123, got %d", userID)
			}
			if page != 1 {
				t.Fatalf("expected page 1, got %d", page)
			}
			if limit != 10 {
				t.Fatalf("expected limit 10, got %d", limit)
			}
			return expectedPosts, expectedTotal, nil
		},
	}

	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.GET("/posts", PostsList(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts?page=1&limit=10", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp postListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if len(resp.Posts) != len(expectedPosts) {
		t.Fatalf("expected %d posts, got %d", len(expectedPosts), len(resp.Posts))
	}

	if resp.Pagination.Page != 1 {
		t.Fatalf("expected page 1, got %d", resp.Pagination.Page)
	}

	if resp.Pagination.Limit != 10 {
		t.Fatalf("expected limit 10, got %d", resp.Pagination.Limit)
	}

	if resp.Pagination.Total != int(expectedTotal) {
		t.Fatalf("expected total %d, got %d", int(expectedTotal), resp.Pagination.Total)
	}

	if svc.called != 1 {
		t.Fatalf("expected GetUserPosts called once, got %d", svc.called)
	}
}

func testPostsList_DefaultPagination(t *testing.T) {
	svc := &stubPostService{
		getUserPostsFunc: func(userID int, page, limit int) ([]models.Post, int64, error) {
			if page != 1 {
				t.Fatalf("expected default page 1, got %d", page)
			}
			if limit != 20 {
				t.Fatalf("expected default limit 20, got %d", limit)
			}
			return []models.Post{}, int64(0), nil
		},
	}

	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.GET("/posts", PostsList(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp postListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Pagination.Page != 1 {
		t.Fatalf("expected page 1, got %d", resp.Pagination.Page)
	}

	if resp.Pagination.Limit != 20 {
		t.Fatalf("expected limit 20, got %d", resp.Pagination.Limit)
	}
}

func testPostsList_InvalidPage(t *testing.T) {
	svc := &stubPostService{}
	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.GET("/posts", PostsList(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts?page=-1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var resp validationErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrValidation) {
		t.Fatalf("expected code %q, got %q", string(services.ErrValidation), resp.Code)
	}

	if len(resp.Errors) == 0 {
		t.Fatal("expected validation errors")
	}

	if resp.Errors[0].Field != "page" {
		t.Fatalf("expected field 'page', got %q", resp.Errors[0].Field)
	}
}

func testPostsList_InvalidLimit(t *testing.T) {
	svc := &stubPostService{}
	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.GET("/posts", PostsList(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts?limit=-1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var resp validationErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrValidation) {
		t.Fatalf("expected code %q, got %q", string(services.ErrValidation), resp.Code)
	}

	if len(resp.Errors) == 0 {
		t.Fatal("expected validation errors")
	}

	if resp.Errors[0].Field != "limit" {
		t.Fatalf("expected field 'limit', got %q", resp.Errors[0].Field)
	}
}

func testPostsList_LimitExceeded(t *testing.T) {
	svc := &stubPostService{}
	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.GET("/posts", PostsList(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts?limit=101", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrInvalidRequest) {
		t.Fatalf("expected code %q, got %q", string(services.ErrInvalidRequest), resp.Code)
	}
}

func testPostsList_FailedGetUser(t *testing.T) {
	svc := &stubPostService{}
	r := newTestI18nRouter(t)
	r.GET("/posts", PostsList(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrFailedGetUser) {
		t.Fatalf("expected code %q, got %q", string(services.ErrFailedGetUser), resp.Code)
	}
}

func testPostsList_InternalError(t *testing.T) {
	svc := &stubPostService{
		getUserPostsFunc: func(userID int, page, limit int) ([]models.Post, int64, error) {
			return nil, 0, services.NewServiceErrorWrap(
				services.ErrInternal,
				"database error",
				errors.New("connection failed"),
			)
		},
	}

	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.GET("/posts", PostsList(svc))

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrInternal) {
		t.Fatalf("expected code %q, got %q", string(services.ErrInternal), resp.Code)
	}
}
