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

func TestLoginWithEmail_Success(t *testing.T) {
	mockSvc := newMockAuthService()
	router := newTestEngine()
	router.POST("/login", LoginWithUsername(mockSvc))

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

func TestLoginWithEmail_InvalidCredentials(t *testing.T) {
	mockSvc := newMockAuthService()
	router := newTestEngine()
	router.POST("/login", LoginWithUsername(mockSvc))

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
	router := newTestEngine()
	router.POST("/refresh", RefreshToken(mockSvc))

	body := RefreshTokenRequest{RefreshToken: "valid-refresh-token"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(jsonBody))
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
	mockSvc := newMockAuthService()
	router := newTestEngine()
	router.POST("/refresh", RefreshToken(mockSvc))

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
