package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/internal/service/auth"

	"github.com/gin-gonic/gin"
)

type mockAuthService struct {
	users map[int]*user.User
}

func newMockAuthService() *mockAuthService {
	return &mockAuthService{
		users: make(map[int]*user.User),
	}
}

func (m *mockAuthService) GenerateGitHubAuthURL(_ context.Context) (string, error) {
	return "https://github.com/login/oauth/authorize?...", nil
}

func (m *mockAuthService) LoginWithGitHub(_ context.Context, _ string) (*user.User, *auth.JWTTokenPair, error) {
	return nil, nil, service.NewServiceError(service.ErrUnauthorized, "not implemented")
}

func (m *mockAuthService) LoginWithEmail(_ context.Context, username, password string) (*user.User, *auth.JWTTokenPair, error) {
	if username == "testuser" && password == "correctpassword" {
		u := &user.User{ID: 1, Email: "test@example.com", Username: username, Role: user.RoleUser}
		tokens := &auth.JWTTokenPair{AccessToken: "test-access", RefreshToken: "test-refresh"}
		return u, tokens, nil
	}
	return nil, nil, service.NewServiceError(service.ErrInvalidCredentials, "invalid credentials")
}

func (m *mockAuthService) RefreshToken(_ context.Context, refreshToken string) (*user.User, *auth.JWTTokenPair, error) {
	if refreshToken == "valid-refresh-token" {
		u := &user.User{ID: 1, Email: "test@example.com", Username: "testuser", Role: user.RoleUser}
		tokens := &auth.JWTTokenPair{AccessToken: "new-access", RefreshToken: "new-refresh"}
		return u, tokens, nil
	}
	return nil, nil, service.NewServiceError(service.ErrUnauthorized, "invalid refresh token")
}

func (m *mockAuthService) Logout(_ context.Context, _ string) error {
	return nil
}

func (m *mockAuthService) ChangePassword(_ context.Context, userID int, _, _ string) error {
	if userID == 1 {
		return nil
	}
	return service.NewServiceError(service.ErrNotFound, "user not found")
}

func (m *mockAuthService) QueryPostKey(_ context.Context, userID int) (string, time.Time, error) {
	if userID == 1 {
		return "test-post-key", time.Now(), nil
	}
	return "", time.Time{}, service.NewServiceError(service.ErrNotFound, "user not found")
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestLoginWithEmail_Success(t *testing.T) {
	mockSvc := newMockAuthService()
	router := setupTestRouter()

	router.POST("/login", func(c *gin.Context) {
		var req UsernameLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		u, tokens, err := mockSvc.LoginWithEmail(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			if se, ok := service.AsServiceError(err); ok && se.Code == service.ErrInvalidCredentials {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user":          gin.H{"id": u.ID, "email": u.Email, "username": u.Username, "role": string(u.Role)},
			"access_token":  tokens.AccessToken,
			"refresh_token": tokens.RefreshToken,
		})
	})

	body := UsernameLoginRequest{Username: "testuser", Password: "correctpassword"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["access_token"] == nil {
		t.Error("expected access_token in response")
	}
}

func TestLoginWithEmail_InvalidCredentials(t *testing.T) {
	mockSvc := newMockAuthService()
	router := setupTestRouter()

	router.POST("/login", func(c *gin.Context) {
		var req UsernameLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_, _, err := mockSvc.LoginWithEmail(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			if se, ok := service.AsServiceError(err); ok && se.Code == service.ErrInvalidCredentials {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	body := UsernameLoginRequest{Username: "wronguser", Password: "wrongpass"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRefreshToken_Success(t *testing.T) {
	mockSvc := newMockAuthService()
	router := setupTestRouter()

	router.POST("/refresh", func(c *gin.Context) {
		var req RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		u, tokens, err := mockSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user":          gin.H{"id": u.ID, "email": u.Email, "username": u.Username, "role": string(u.Role)},
			"access_token":  tokens.AccessToken,
			"refresh_token": tokens.RefreshToken,
		})
	})

	body := RefreshTokenRequest{RefreshToken: "valid-refresh-token"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["access_token"] == nil {
		t.Error("expected access_token in response")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	mockSvc := newMockAuthService()
	router := setupTestRouter()

	router.POST("/refresh", func(c *gin.Context) {
		var req RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_, _, err := mockSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	body := RefreshTokenRequest{RefreshToken: "invalid-token"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestChangePassword_Success(t *testing.T) {
	mockSvc := newMockAuthService()
	router := setupTestRouter()

	router.POST("/change-password", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Email: "test@example.com", Username: "testuser", Role: user.RoleUser})
		c.Next()
	}, func(c *gin.Context) {
		u, _ := c.Get("user")
		userObj := u.(*user.User)

		var req PasswordChangeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := mockSvc.ChangePassword(c.Request.Context(), userObj.ID, req.CurrentPassword, req.NewPassword); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "password changed"})
	})

	body := PasswordChangeRequest{CurrentPassword: "old", NewPassword: "newpassword123"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestQueryPostKey_Success(t *testing.T) {
	mockSvc := newMockAuthService()
	router := setupTestRouter()

	router.GET("/post-key", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1, Email: "test@example.com", Username: "testuser", Role: user.RoleUser})
		c.Next()
	}, func(c *gin.Context) {
		u, _ := c.Get("user")
		userObj := u.(*user.User)

		postKey, _, err := mockSvc.QueryPostKey(c.Request.Context(), userObj.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"post_key": postKey})
	})

	req := httptest.NewRequest(http.MethodGet, "/post-key", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["post_key"] == nil {
		t.Error("expected post_key in response")
	}
}
