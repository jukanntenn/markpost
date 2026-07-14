package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"markpost/internal/domain/user"
	"markpost/internal/infra"
	"markpost/internal/service"
	"markpost/internal/service/auth"

	"github.com/gin-gonic/gin"
)

func setupRealAuthService(t *testing.T) (*auth.Service, user.Repository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	tokenRepo := infra.NewTokenRepository(db)
	jwtSvc := auth.NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour*24)
	svc := auth.NewService(userRepo, tokenRepo, nil, jwtSvc, "markpost")
	return svc, userRepo
}

func TestLoginWithUsername_Success(t *testing.T) {
	svc, userRepo := setupRealAuthService(t)
	ctx := t.Context()
	_, _ = userRepo.Create(ctx, "test@example.com", "testuser", "correctpassword")

	router := newTestEngine()
	router.POST("/login", LoginWithUsername(svc))

	body := UsernameLoginRequest{Username: "testuser", Password: "correctpassword"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected token in response")
	}
	if resp.User.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", resp.User.Username)
	}
}

func TestLoginWithUsername_InvalidCredentials(t *testing.T) {
	svc, _ := setupRealAuthService(t)

	router := newTestEngine()
	router.POST("/login", LoginWithUsername(svc))

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

func TestLoginWithUsername_InvalidBody(t *testing.T) {
	svc, _ := setupRealAuthService(t)

	router := newTestEngine()
	router.POST("/login", LoginWithUsername(svc))

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRefreshToken_Success(t *testing.T) {
	svc, userRepo := setupRealAuthService(t)
	ctx := t.Context()
	_, _ = userRepo.Create(ctx, "test@example.com", "testuser", "password")

	// Login to get a refresh token
	loginRouter := newTestEngine()
	loginRouter.POST("/login", LoginWithUsername(svc))
	loginBody, _ := json.Marshal(UsernameLoginRequest{Username: "testuser", Password: "password"})
	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	loginRouter.ServeHTTP(loginW, loginReq)

	var loginResp AuthResponse
	_ = json.Unmarshal(loginW.Body.Bytes(), &loginResp)

	// Now refresh
	router := newTestEngine()
	router.POST("/refresh", RefreshToken(svc))

	refreshBody, _ := json.Marshal(RefreshTokenRequest{RefreshToken: loginResp.RefreshToken})
	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(refreshBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp RefreshTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected token in response")
	}
	if resp.RefreshToken == "" {
		t.Error("expected refresh_token in response")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc, _ := setupRealAuthService(t)

	router := newTestEngine()
	router.POST("/refresh", RefreshToken(svc))

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

func TestQueryPostKey_Success(t *testing.T) {
	svc, userRepo := setupRealAuthService(t)
	ctx := t.Context()
	created, _ := userRepo.Create(ctx, "test@example.com", "testuser", "password")
	_ = userRepo.SetRole(ctx, created.ID, user.RoleUser)

	token, _ := auth.NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour*24).GenerateAccessToken(time.Now(), created.ID, "test@example.com", "testuser", "user")

	router := newTestEngine()
	router.GET("/post_key", func(c *gin.Context) {
		c.Set("user", &user.User{ID: created.ID, Email: "test@example.com", Username: "testuser", Role: user.RoleUser})
		c.Set("access_token", token)
		c.Next()
	}, QueryPostKey(svc))

	req := httptest.NewRequest(http.MethodGet, "/post_key", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp PostKeyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.PostKey == "" {
		t.Error("expected post_key in response")
	}
}

func TestChangePassword_Success(t *testing.T) {
	svc, userRepo := setupRealAuthService(t)
	ctx := t.Context()
	created, _ := userRepo.Create(ctx, "test@example.com", "testuser", "oldpassword")

	router := newTestEngine()
	router.POST("/change-password", func(c *gin.Context) {
		c.Set("user", &user.User{ID: created.ID, Email: "test@example.com", Username: "testuser", Role: user.RoleUser})
		c.Next()
	}, ChangePassword(svc))

	body := PasswordChangeRequest{CurrentPassword: "oldpassword", NewPassword: "newpassword123"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	svc, userRepo := setupRealAuthService(t)
	ctx := t.Context()
	created, _ := userRepo.Create(ctx, "test@example.com", "testuser", "oldpassword")

	router := newTestEngine()
	router.POST("/change-password", func(c *gin.Context) {
		c.Set("user", &user.User{ID: created.ID, Email: "test@example.com", Username: "testuser", Role: user.RoleUser})
		c.Next()
	}, ChangePassword(svc))

	body := PasswordChangeRequest{CurrentPassword: "wrongpassword", NewPassword: "newpassword123"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLogout_Success(t *testing.T) {
	svc, _ := setupRealAuthService(t)

	router := newTestEngine()
	router.POST("/logout", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1})
		c.Set("access_token", "test-token")
		c.Next()
	}, Logout(svc))

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestLogout_NoAccessToken(t *testing.T) {
	svc, _ := setupRealAuthService(t)

	router := newTestEngine()
	router.POST("/logout", func(c *gin.Context) {
		c.Set("user", &user.User{ID: 1})
		c.Next()
	}, Logout(svc))

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

type mockGitHubAuthURLGenerator struct {
	url   string
	state string
	err   error
}

func (m *mockGitHubAuthURLGenerator) GenerateGitHubAuthURL(_ context.Context) (string, string, error) {
	return m.url, m.state, m.err
}

func TestGenerateGitHubOAuthURL_Success(t *testing.T) {
	mock := &mockGitHubAuthURLGenerator{url: "https://github.com/login/oauth/authorize?client_id=test"}

	router := newTestEngine()
	router.GET("/auth/github", GenerateGitHubOAuthURL(mock))

	req := httptest.NewRequest(http.MethodGet, "/auth/github", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["url"] == nil {
		t.Error("expected url in response")
	}
}

func TestGenerateGitHubOAuthURL_Error(t *testing.T) {
	mock := &mockGitHubAuthURLGenerator{err: errors.New("oauth not configured")}

	router := newTestEngine()
	router.GET("/auth/github", GenerateGitHubOAuthURL(mock))

	req := httptest.NewRequest(http.MethodGet, "/auth/github", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

type mockAuthServiceForGitHub struct {
	u      *user.User
	tokens *auth.JWTTokenPair
	err    error
}

func (m *mockAuthServiceForGitHub) LoginWithGitHub(_ context.Context, _, _ string) (*user.User, *auth.JWTTokenPair, error) {
	return m.u, m.tokens, m.err
}
func (m *mockAuthServiceForGitHub) LoginWithEmail(_ context.Context, _, _ string) (*user.User, *auth.JWTTokenPair, error) {
	return nil, nil, nil
}
func (m *mockAuthServiceForGitHub) RefreshToken(_ context.Context, _ string) (*user.User, *auth.JWTTokenPair, error) {
	return nil, nil, nil
}
func (m *mockAuthServiceForGitHub) Logout(_ context.Context, _ string) error { return nil }
func (m *mockAuthServiceForGitHub) ChangePassword(_ context.Context, _ int, _, _ string) error {
	return nil
}
func (m *mockAuthServiceForGitHub) QueryPostKey(_ context.Context, _ int) (string, time.Time, error) {
	return "", time.Time{}, nil
}

func TestLoginGitHub_Success(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour*24)
	pair, _ := jwtSvc.GenerateTokenPair(1, "gh@example.com", "ghuser", "user")

	mock := &mockAuthServiceForGitHub{
		u:      &user.User{ID: 1, Email: "gh@example.com", Username: "ghuser", Role: user.RoleUser},
		tokens: pair,
	}

	router := newTestEngine()
	router.POST("/auth/github/callback", LoginGitHub(mock))

	body := GitHubLoginRequest{Code: "test-code", State: "test-state"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/auth/github/callback", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.User.Username != "ghuser" {
		t.Errorf("expected username 'ghuser', got %q", resp.User.Username)
	}
}

func TestLoginGitHub_InvalidBody(t *testing.T) {
	mock := &mockAuthServiceForGitHub{}

	router := newTestEngine()
	router.POST("/auth/github/callback", LoginGitHub(mock))

	req := httptest.NewRequest(http.MethodPost, "/auth/github/callback", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLoginGitHub_AuthError(t *testing.T) {
	mock := &mockAuthServiceForGitHub{
		err: service.New(service.ErrUnauthorized, "oauth exchange failed"),
	}

	router := newTestEngine()
	router.POST("/auth/github/callback", LoginGitHub(mock))

	body := GitHubLoginRequest{Code: "bad-code", State: "test-state"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/auth/github/callback", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
