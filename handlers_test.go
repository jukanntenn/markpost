package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func setupHandlerTest(t *testing.T) (*gin.Engine, *Database) {
	t.Helper()
	if bundle == nil {
		InitI18n()
	}
	setTestConfig()

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("titlesize", func(fl validator.FieldLevel) bool {
			if config.TitleMaxSize <= 0 {
				return true
			}
			return len([]byte(fl.Field().String())) <= config.TitleMaxSize
		})
		v.RegisterValidation("bodysize", func(fl validator.FieldLevel) bool {
			if config.BodyMaxSize <= 0 {
				return true
			}
			return len([]byte(fl.Field().String())) <= config.BodyMaxSize
		})
	}

	db := setupTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	return r, db
}

func withUser(user *User) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user", user)
	}
}

type testPostRequest struct {
	Title string `json:"title" binding:"required"`
	Body  string `json:"body" binding:"required"`
}

type testPasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func teardownHandlerTest(t *testing.T, db *Database) {
	t.Helper()
	teardownTestDB(t, db)
}

func performRequest(r http.Handler, method, path string, body ...map[string]interface{}) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	if len(body) > 0 && body[0] != nil {
		jsonBytes, _ := json.Marshal(body[0])
		req, _ = http.NewRequest(method, path, bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func bindTestRequest(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindJSON(req); err != nil {
		return err
	}
	return nil
}

func TestGenerateGitHubOAuthURLHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.GET("/api/oauth/url", GenerateGitHubOAuthURLHandler)
		w := performRequest(r, "GET", "/api/oauth/url")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if response["url"] == "" {
			t.Fatalf("expected url in response, got %v", response)
		}
	})
}

func TestLoginGitHubHandler(t *testing.T) {
	t.Run("missing OAuth state header", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.POST("/api/oauth/login", LoginGitHubHandler)
		body := map[string]interface{}{"code": "test-code"}
		w := performRequest(r, "POST", "/api/oauth/login?state=test-state", body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("missing state query param", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.POST("/api/oauth/login", LoginGitHubHandler)
		body := map[string]interface{}{"code": "test-code"}
		req, _ := http.NewRequest("POST", "/api/oauth/login", bytes.NewReader(jsonMustMarshal(body)))
		req.Header.Set("X-Oauth-State", "test-state")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("missing code", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.POST("/api/oauth/login", LoginGitHubHandler)
		req, _ := http.NewRequest("POST", "/api/oauth/login?state=test-state", nil)
		req.Header.Set("X-Oauth-State", "test-state")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("state mismatch", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.POST("/api/oauth/login", LoginGitHubHandler)
		body := map[string]interface{}{"code": "test-code"}
		req, _ := http.NewRequest("POST", "/api/oauth/login?state=state1", bytes.NewReader(jsonMustMarshal(body)))
		req.Header.Set("X-Oauth-State", "state2")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("invalid code -> ErrUnauthorized", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.POST("/api/oauth/login", LoginGitHubHandler)
		body := map[string]interface{}{"code": "invalid-code"}
		req, _ := http.NewRequest("POST", "/api/oauth/login?state=test-state", bytes.NewReader(jsonMustMarshal(body)))
		req.Header.Set("X-Oauth-State", "test-state")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

func TestCreatePostHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		postSvc = NewPostService(db.GetPostRepository())
		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		user, _ := db.GetUserRepository().CreateUser("alice", "password123")
		r.POST("/:post_key", withUser(user), CreatePostHandler)

		body := map[string]interface{}{
			"title": "Test Title",
			"body":  "Test Body",
		}
		w := performRequest(r, "POST", "/test-key", body)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("missing user", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		postSvc = NewPostService(db.GetPostRepository())

		r.POST("/:post_key", CreatePostHandler)
		body := map[string]interface{}{
			"title": "Test Title",
			"body":  "Test Body",
		}
		w := performRequest(r, "POST", "/test-key", body)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("missing title and body", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		postSvc = NewPostService(db.GetPostRepository())
		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		user, _ := db.GetUserRepository().CreateUser("alice", "password123")
		r.POST("/:post_key", withUser(user), CreatePostHandler)

		w := performRequest(r, "POST", "/test-key")

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestRenderPostHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		postSvc = NewPostService(db.GetPostRepository())
		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		user, _ := db.GetUserRepository().CreateUser("alice", "password123")
		postRepo := db.GetPostRepository()
		post, err := postRepo.CreatePostWithUser("Test Title", "## Test\nHello World", user.ID)
		if err != nil {
			t.Fatalf("seed post error: %v", err)
		}

		r.GET("/:id", RenderPostHandler)
		w := performRequest(r, "GET", "/"+post.QID)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		if !strings.Contains(w.Body.String(), "<h2>") {
			t.Fatalf("expected HTML output, got %s", w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		postSvc = NewPostService(db.GetPostRepository())

		r.GET("/:id", RenderPostHandler)
		w := performRequest(r, "GET", "/nonexistent")

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestQueryPostKeyHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		user, _ := db.GetUserRepository().CreateUser("alice", "password123")
		r.GET("/api/post_key", withUser(user), QueryPostKeyHandler)

		w := performRequest(r, "GET", "/api/post_key")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if response["post_key"] == nil || response["created_at"] == nil {
			t.Fatalf("expected post_key and created_at in response, got %v", response)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.GET("/api/post_key", QueryPostKeyHandler)
		w := performRequest(r, "GET", "/api/post_key")

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("user not found -> ErrConflict", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.GET("/api/post_key", withUser(&User{ID: 999999}), QueryPostKeyHandler)

		w := performRequest(r, "GET", "/api/post_key")

		if w.Code != http.StatusConflict {
			t.Fatalf("expected status %d, got %d", http.StatusConflict, w.Code)
		}
	})
}

func TestLoginWithPasswordHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		_, err := db.GetUserRepository().CreateUser("alice", "password123")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		r.POST("/api/auth/login", LoginWithPasswordHandler)
		body := map[string]interface{}{
			"username": "alice",
			"password": "password123",
		}
		w := performRequest(r, "POST", "/api/auth/login", body)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if response["user"] == nil || response["access_token"] == nil {
			t.Fatalf("expected user and tokens in response, got %v", response)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.POST("/api/auth/login", LoginWithPasswordHandler)
		w := performRequest(r, "POST", "/api/auth/login")

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		_, err := db.GetUserRepository().CreateUser("alice", "password123")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		r.POST("/api/auth/login", LoginWithPasswordHandler)
		body := map[string]interface{}{
			"username": "alice",
			"password": "wrongpassword",
		}
		w := performRequest(r, "POST", "/api/auth/login", body)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

func TestRefreshTokenHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		_, err := db.GetUserRepository().CreateUser("alice", "password123")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, tokens, _ := authSvc.LoginWithPassword(context.Background(), "alice", "password123")

		r.POST("/api/auth/refresh", RefreshTokenHandler)
		body := map[string]interface{}{
			"refresh_token": tokens.RefreshToken,
		}
		w := performRequest(r, "POST", "/api/auth/refresh", body)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if response["user"] == nil || response["access_token"] == nil {
			t.Fatalf("expected user and tokens in response, got %v", response)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.POST("/api/auth/refresh", RefreshTokenHandler)
		w := performRequest(r, "POST", "/api/auth/refresh")

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.POST("/api/auth/refresh", RefreshTokenHandler)
		body := map[string]interface{}{
			"refresh_token": "invalid-token",
		}
		w := performRequest(r, "POST", "/api/auth/refresh", body)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

func TestChangePasswordHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		user, _ := db.GetUserRepository().CreateUser("alice", "password123")
		r.POST("/api/auth/change-password", withUser(user), ChangePasswordHandler)

		body := map[string]interface{}{
			"current_password": "password123",
			"new_password":     "newpassword456",
		}
		w := performRequest(r, "POST", "/api/auth/change-password", body)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		r.POST("/api/auth/change-password", ChangePasswordHandler)
		body := map[string]interface{}{
			"current_password": "password123",
			"new_password":     "newpassword456",
		}
		w := performRequest(r, "POST", "/api/auth/change-password", body)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		user, _ := db.GetUserRepository().CreateUser("alice", "password123")
		r.POST("/api/auth/change-password", withUser(user), ChangePasswordHandler)

		w := performRequest(r, "POST", "/api/auth/change-password")

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("invalid current password", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		user, _ := db.GetUserRepository().CreateUser("alice", "password123")
		r.POST("/api/auth/change-password", withUser(user), ChangePasswordHandler)

		body := map[string]interface{}{
			"current_password": "wrongpassword",
			"new_password":     "newpassword456",
		}
		w := performRequest(r, "POST", "/api/auth/change-password", body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("same password", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		user, _ := db.GetUserRepository().CreateUser("alice", "password123")
		r.POST("/api/auth/change-password", withUser(user), ChangePasswordHandler)

		body := map[string]interface{}{
			"current_password": "password123",
			"new_password":     "password123",
		}
		w := performRequest(r, "POST", "/api/auth/change-password", body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestPostsListHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		postSvc = NewPostService(db.GetPostRepository())
		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		user, _ := db.GetUserRepository().CreateUser("alice", "password123")
		postRepo := db.GetPostRepository()
		_, err := postRepo.CreatePostWithUser("Post 1", "Body 1", user.ID)
		if err != nil {
			t.Fatalf("seed post error: %v", err)
		}

		r.GET("/api/posts", withUser(user), PostsListHandler)

		w := performRequest(r, "GET", "/api/posts")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if response["posts"] == nil || response["pagination"] == nil {
			t.Fatalf("expected posts and pagination in response, got %v", response)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		postSvc = NewPostService(db.GetPostRepository())

		r.GET("/api/posts", PostsListHandler)
		w := performRequest(r, "GET", "/api/posts")

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		postSvc = NewPostService(db.GetPostRepository())
		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		user, _ := db.GetUserRepository().CreateUser("alice", "password123")
		postRepo := db.GetPostRepository()
		for i := 0; i < 5; i++ {
			_, err := postRepo.CreatePostWithUser(fmt.Sprintf("Post %d", i), fmt.Sprintf("Body %d", i), user.ID)
			if err != nil {
				t.Fatalf("seed post error: %v", err)
			}
		}

		r.GET("/api/posts", withUser(user), PostsListHandler)

		w := performRequest(r, "GET", "/api/posts?page=1&limit=3")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		posts := response["posts"].([]interface{})
		if len(posts) != 3 {
			t.Fatalf("expected 3 posts, got %d", len(posts))
		}
	})

	t.Run("limit exceeds max", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)

		postSvc = NewPostService(db.GetPostRepository())
		authSvc = NewAuthService(db.GetUserRepository(), oauthConfig)

		user, _ := db.GetUserRepository().CreateUser("alice", "password123")
		r.GET("/api/posts", withUser(user), PostsListHandler)

		w := performRequest(r, "GET", "/api/posts?limit=101")

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestHealthHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, db := setupHandlerTest(t)
		defer teardownHandlerTest(t, db)


		r.GET("/health", HealthHandler)
		w := performRequest(r, "GET", "/health")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if response["status"] != "ok" {
			t.Fatalf("expected status ok, got %v", response)
		}
	})
}

func jsonMustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
