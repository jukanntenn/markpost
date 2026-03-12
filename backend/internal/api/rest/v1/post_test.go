package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/service"

	"github.com/gin-gonic/gin"
)

type mockPostService struct {
	posts map[string]*post.Post
}

func newMockPostService() *mockPostService {
	return &mockPostService{
		posts: make(map[string]*post.Post),
	}
}

func (m *mockPostService) CreatePost(ctx context.Context, title, body string, userID int) (string, error) {
	qid := "test-qid-123"
	m.posts[qid] = &post.Post{
		ID:     1,
		QID:    qid,
		Title:  title,
		Body:   body,
		UserID: userID,
	}
	return qid, nil
}

func (m *mockPostService) RenderPostHTML(ctx context.Context, qid string) (string, string, error) {
	p, ok := m.posts[qid]
	if !ok {
		return "", "", service.NewServiceError(service.ErrNotFound, "post not found")
	}
	return p.Title, "<h1>" + p.Title + "</h1><p>" + p.Body + "</p>", nil
}

func (m *mockPostService) GetPostMarkdown(ctx context.Context, qid string) (string, string, error) {
	p, ok := m.posts[qid]
	if !ok {
		return "", "", service.NewServiceError(service.ErrNotFound, "post not found")
	}
	return p.Title, p.Body, nil
}

func (m *mockPostService) GetUserPosts(ctx context.Context, userID int, page, limit int) ([]post.Post, int64, error) {
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
	router := setupTestRouter()

	router.POST("/posts", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Username: "testuser", Role: user.RoleUser})
		c.Next()
	}, func(c *gin.Context) {
		var req struct {
			Title string `json:"title" binding:"required"`
			Body  string `json:"body" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		u, _ := c.Get("user")
		userObj := u.(*user.User)

		qid, err := mockSvc.CreatePost(c.Request.Context(), req.Title, req.Body, userObj.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": qid})
	})

	body := struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}{Title: "Test Title", Body: "Test Body"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"] == nil {
		t.Error("expected id in response")
	}
}

func TestCreatePost_MissingTitle(t *testing.T) {
	mockSvc := newMockPostService()
	router := setupTestRouter()

	router.POST("/posts", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Username: "testuser", Role: user.RoleUser})
		c.Next()
	}, func(c *gin.Context) {
		var req struct {
			Title string `json:"title" binding:"required"`
			Body  string `json:"body" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		u, _ := c.Get("user")
		userObj := u.(*user.User)

		qid, err := mockSvc.CreatePost(c.Request.Context(), req.Title, req.Body, userObj.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": qid})
	})

	body := struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}{Title: "", Body: "Test Body"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPostsList_Success(t *testing.T) {
	mockSvc := newMockPostService()
	router := setupTestRouter()

	mockSvc.CreatePost(context.Background(), "Post 1", "Body 1", 1)
	mockSvc.CreatePost(context.Background(), "Post 2", "Body 2", 1)

	router.GET("/posts", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Username: "testuser", Role: user.RoleUser})
		c.Next()
	}, func(c *gin.Context) {
		u, _ := c.Get("user")
		userObj := u.(*user.User)

		type queryParams struct {
			Page  int `form:"page" binding:"omitempty,min=1"`
			Limit int `form:"limit" binding:"omitempty,min=1"`
		}
		var query queryParams
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		query.Page = defaultInt(query.Page, 1)
		query.Limit = defaultInt(query.Limit, 20)
		if query.Limit > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}

		posts, total, err := mockSvc.GetUserPosts(c.Request.Context(), userObj.ID, query.Page, query.Limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		items := make([]gin.H, 0, len(posts))
		for _, p := range posts {
			items = append(items, gin.H{
				"id":         p.ID,
				"qid":        p.QID,
				"title":      p.Title,
				"created_at": p.CreatedAt,
			})
		}
		totalPages := (total + int64(query.Limit) - 1) / int64(query.Limit)

		c.JSON(http.StatusOK, gin.H{
			"posts": items,
			"pagination": gin.H{
				"page":        query.Page,
				"limit":       query.Limit,
				"total":       total,
				"total_pages": totalPages,
			},
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["posts"] == nil {
		t.Error("expected posts in response")
	}

	pagination, ok := resp["pagination"].(map[string]interface{})
	if !ok {
		t.Error("expected pagination in response")
	} else {
		if pagination["page"] == nil {
			t.Error("expected page in pagination")
		}
		if pagination["total"] == nil {
			t.Error("expected total in pagination")
		}
	}
}

func TestPostsList_InvalidLimit(t *testing.T) {
	router := setupTestRouter()

	router.GET("/posts", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Username: "testuser", Role: user.RoleUser})
		c.Next()
	}, func(c *gin.Context) {
		type queryParams struct {
			Page  int `form:"page" binding:"omitempty,min=1"`
			Limit int `form:"limit" binding:"omitempty,min=1"`
		}
		var query queryParams
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		query.Limit = defaultInt(query.Limit, 20)
		if query.Limit > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest(http.MethodGet, "/posts?limit=200", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRenderPost_HTML(t *testing.T) {
	mockSvc := newMockPostService()
	router := setupTestRouter()

	mockSvc.CreatePost(context.Background(), "Test Post", "Test Body", 1)

	router.GET("/posts/:id", func(c *gin.Context) {
		qid := c.Param("id")

		title, html, err := mockSvc.RenderPostHTML(c.Request.Context(), qid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"title": title, "html": html})
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/test-qid-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRenderPost_NotFound(t *testing.T) {
	mockSvc := newMockPostService()
	router := setupTestRouter()

	router.GET("/posts/:id", func(c *gin.Context) {
		qid := c.Param("id")

		_, _, err := mockSvc.RenderPostHTML(c.Request.Context(), qid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}
