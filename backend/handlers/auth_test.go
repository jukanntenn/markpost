package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"

	apperrors "markpost/errors"
	"markpost/models"
	"markpost/services"
	"markpost/utils"
)

type stubGitHubAuthURLGenerator struct {
	url    string
	err    error
	called int
}

func (s *stubGitHubAuthURLGenerator) GenerateGitHubAuthURL(ctx context.Context) (string, error) {
	s.called++
	return s.url, s.err
}

type stubAuthService struct {
	loginWithGitHubFunc   func(ctx context.Context, code string) (*models.User, *services.JWTTokenPair, error)
	loginWithPasswordFunc func(username, password string) (*models.User, *services.JWTTokenPair, error)
	refreshTokenFunc      func(refreshToken string) (*models.User, *services.JWTTokenPair, error)
	changePasswordFunc    func(userID int, current, new string) error
	queryPostKeyFunc      func(userID int) (string, time.Time, error)
	called                int
	lastCode              string
	lastUsername          string
	lastPassword          string
	lastRefreshToken      string
	lastUserID            int
}

func (s *stubAuthService) LoginWithGitHub(ctx context.Context, code string) (*models.User, *services.JWTTokenPair, error) {
	s.called++
	s.lastCode = code
	if s.loginWithGitHubFunc != nil {
		return s.loginWithGitHubFunc(ctx, code)
	}
	return nil, nil, nil
}

func (s *stubAuthService) LoginWithPassword(username, password string) (*models.User, *services.JWTTokenPair, error) {
	s.called++
	s.lastUsername = username
	s.lastPassword = password
	if s.loginWithPasswordFunc != nil {
		return s.loginWithPasswordFunc(username, password)
	}
	return nil, nil, nil
}

func (s *stubAuthService) RefreshToken(refreshToken string) (*models.User, *services.JWTTokenPair, error) {
	s.called++
	s.lastRefreshToken = refreshToken
	if s.refreshTokenFunc != nil {
		return s.refreshTokenFunc(refreshToken)
	}
	return nil, nil, nil
}

func (s *stubAuthService) ChangePassword(userID int, current, new string) error {
	s.called++
	s.lastUserID = userID
	if s.changePasswordFunc != nil {
		return s.changePasswordFunc(userID, current, new)
	}
	return nil
}

func (s *stubAuthService) QueryPostKey(userID int) (string, time.Time, error) {
	s.called++
	s.lastUserID = userID
	if s.queryPostKeyFunc != nil {
		return s.queryPostKeyFunc(userID)
	}
	return "", time.Time{}, nil
}

func (s *stubAuthService) GenerateGitHubAuthURL(ctx context.Context) (string, error) {
	return "", nil
}

func (s *stubAuthService) GetAllUsers(page, limit int) ([]models.User, int64, error) {
	return nil, 0, nil
}

func (s *stubAuthService) InitializeFirstAdmin(initialUsername string) error {
	return nil
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newTestI18nRouter(t *testing.T) *gin.Engine {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get caller path")
	}
	localesPath := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "locales"))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginI18n.Localize(ginI18n.WithBundle(&ginI18n.BundleCfg{
		RootPath:         localesPath,
		AcceptLanguage:   []language.Tag{language.English, language.Chinese},
		DefaultLanguage:  language.English,
		UnmarshalFunc:    toml.Unmarshal,
		FormatBundleFile: "toml",
		Loader:           utils.ActiveLocaleLoader{},
	})))
	return r
}

func TestGenerateGitHubOAuthURL_Success(t *testing.T) {
	svc := &stubGitHubAuthURLGenerator{
		url: "https://github.com/login/oauth/authorize?state=abc",
	}

	r := newTestI18nRouter(t)
	r.GET("/api/oauth/url", GenerateGitHubOAuthURL(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/oauth/url", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if svc.called != 1 {
		t.Fatalf("expected GenerateGitHubAuthURL called once, got %d", svc.called)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if got, _ := body["url"].(string); got != svc.url {
		t.Fatalf("expected url %q, got %q", svc.url, got)
	}
}

func TestGenerateGitHubOAuthURL_InternalError_ZH(t *testing.T) {
	svc := &stubGitHubAuthURLGenerator{
		err: services.NewServiceErrorWrap(services.ErrInternal, "failed to generate state", errors.New("rand read failed")),
	}

	r := newTestI18nRouter(t)
	r.GET("/api/oauth/url", GenerateGitHubOAuthURL(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/oauth/url", nil)
	req.Header.Set("Accept-Language", "zh")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body.Code != string(services.ErrInternal) {
		t.Fatalf("expected code %q, got %q", string(services.ErrInternal), body.Code)
	}
	if body.Message != "服务器内部错误" {
		t.Fatalf("expected zh message %q, got %q", "服务器内部错误", body.Message)
	}
}

func TestGenerateGitHubOAuthURL_InternalError_EN(t *testing.T) {
	svc := &stubGitHubAuthURLGenerator{
		err: services.NewServiceErrorWrap(services.ErrInternal, "failed to generate state", errors.New("rand read failed")),
	}

	r := newTestI18nRouter(t)
	r.GET("/api/oauth/url", GenerateGitHubOAuthURL(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/oauth/url", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body.Code != string(services.ErrInternal) {
		t.Fatalf("expected code %q, got %q", string(services.ErrInternal), body.Code)
	}
	if body.Message != "Internal server error" {
		t.Fatalf("expected en message %q, got %q", "Internal server error", body.Message)
	}
}

func TestGenerateGitHubOAuthURL_InternalError_DefaultEN(t *testing.T) {
	svc := &stubGitHubAuthURLGenerator{
		err: services.NewServiceErrorWrap(services.ErrInternal, "failed to generate state", errors.New("rand read failed")),
	}

	r := newTestI18nRouter(t)
	r.GET("/api/oauth/url", GenerateGitHubOAuthURL(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/oauth/url", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body.Code != string(services.ErrInternal) {
		t.Fatalf("expected code %q, got %q", string(services.ErrInternal), body.Code)
	}
	if body.Message != "Internal server error" {
		t.Fatalf("expected default en message %q, got %q", "Internal server error", body.Message)
	}
}

type authSuccessResponse struct {
	User struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type validationErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Errors  []apperrors.FieldError `json:"errors"`
}

func TestLoginGitHub(t *testing.T) {
	t.Run("Missing state parameter", testLoginGitHub_MissingState)
	t.Run("Missing code field", testLoginGitHub_MissingCode)
	t.Run("Success", testLoginGitHub_Success)
	t.Run("OAuth exchange failed", func(t *testing.T) {
		t.Run("Chinese", testLoginGitHub_Unauthorized_ZH)
		t.Run("English", testLoginGitHub_Unauthorized_EN)
		t.Run("Default English", testLoginGitHub_Unauthorized_DefaultEN)
	})
	t.Run("Token generation failed", func(t *testing.T) {
		t.Run("Chinese", testLoginGitHub_InternalError_ZH)
		t.Run("English", testLoginGitHub_InternalError_EN)
		t.Run("Default English", testLoginGitHub_InternalError_DefaultEN)
	})
}

func testLoginGitHub_MissingState(t *testing.T) {
	svc := &stubAuthService{}
	r := newTestI18nRouter(t)
	r.POST("/api/oauth/login", LoginGitHub(svc))

	body := `{"code": "test_code"}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var resp validationErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v, body=%s", err, rec.Body.String())
	}

	if resp.Code != string(services.ErrValidation) {
		t.Fatalf("expected code %q, got %q", string(services.ErrValidation), resp.Code)
	}

	if len(resp.Errors) == 0 {
		t.Fatal("expected validation errors")
	}

	if resp.Errors[0].Code != string(services.ErrRequired) {
		t.Fatalf("expected detail code %q, got %q", string(services.ErrRequired), resp.Errors[0].Code)
	}

	if resp.Errors[0].Field != "state" {
		t.Fatalf("expected field 'state', got %q", resp.Errors[0].Field)
	}
}

func testLoginGitHub_MissingCode(t *testing.T) {
	svc := &stubAuthService{}
	r := newTestI18nRouter(t)
	r.POST("/api/oauth/login", LoginGitHub(svc))

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/login?state=test_state", strings.NewReader(body))
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

	if resp.Errors[0].Code != string(services.ErrRequired) {
		t.Fatalf("expected detail code %q, got %q", string(services.ErrRequired), resp.Errors[0].Code)
	}

	if resp.Errors[0].Field != "code" {
		t.Fatalf("expected field 'code', got %q", resp.Errors[0].Field)
	}
}

func testLoginGitHub_Success(t *testing.T) {
	expectedUser := &models.User{
		ID:       123,
		Username: "testuser",
	}
	expectedTokens := &services.JWTTokenPair{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
	}

	svc := &stubAuthService{
		loginWithGitHubFunc: func(ctx context.Context, code string) (*models.User, *services.JWTTokenPair, error) {
			if code != "github_auth_code" {
				t.Fatalf("expected code 'github_auth_code', got %q", code)
			}
			return expectedUser, expectedTokens, nil
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/api/oauth/login", LoginGitHub(svc))

	body := `{"code": "github_auth_code"}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/login?state=random_state", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp authSuccessResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.User.ID != expectedUser.ID {
		t.Fatalf("expected user ID %d, got %d", expectedUser.ID, resp.User.ID)
	}

	if resp.User.Username != expectedUser.Username {
		t.Fatalf("expected username %q, got %q", expectedUser.Username, resp.User.Username)
	}

	if resp.AccessToken != expectedTokens.AccessToken {
		t.Fatalf("expected access_token %q, got %q", expectedTokens.AccessToken, resp.AccessToken)
	}

	if resp.RefreshToken != expectedTokens.RefreshToken {
		t.Fatalf("expected refresh_token %q, got %q", expectedTokens.RefreshToken, resp.RefreshToken)
	}

	if svc.called != 1 {
		t.Fatalf("expected LoginWithGitHub called once, got %d", svc.called)
	}
}

func testLoginGitHub_Unauthorized_ZH(t *testing.T) {
	svc := &stubAuthService{
		loginWithGitHubFunc: func(ctx context.Context, code string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrUnauthorized,
				"oauth exchange failed",
				errors.New("invalid code"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/api/oauth/login", LoginGitHub(svc))

	body := `{"code": "invalid_code"}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/login?state=test_state", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrUnauthorized) {
		t.Fatalf("expected code %q, got %q", string(services.ErrUnauthorized), resp.Code)
	}

	if resp.Message != "未授权" {
		t.Fatalf("expected zh message %q, got %q", "未授权", resp.Message)
	}
}

func testLoginGitHub_Unauthorized_EN(t *testing.T) {
	svc := &stubAuthService{
		loginWithGitHubFunc: func(ctx context.Context, code string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrUnauthorized,
				"oauth exchange failed",
				errors.New("invalid code"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/api/oauth/login", LoginGitHub(svc))

	body := `{"code": "invalid_code"}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/login?state=test_state", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrUnauthorized) {
		t.Fatalf("expected code %q, got %q", string(services.ErrUnauthorized), resp.Code)
	}

	if resp.Message != "Unauthorized" {
		t.Fatalf("expected en message %q, got %q", "Unauthorized", resp.Message)
	}
}

func testLoginGitHub_Unauthorized_DefaultEN(t *testing.T) {
	svc := &stubAuthService{
		loginWithGitHubFunc: func(ctx context.Context, code string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrUnauthorized,
				"oauth exchange failed",
				errors.New("invalid code"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/api/oauth/login", LoginGitHub(svc))

	body := `{"code": "invalid_code"}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/login?state=test_state", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrUnauthorized) {
		t.Fatalf("expected code %q, got %q", string(services.ErrUnauthorized), resp.Code)
	}

	if resp.Message != "Unauthorized" {
		t.Fatalf("expected default en message %q, got %q", "Unauthorized", resp.Message)
	}
}

func testLoginGitHub_InternalError_ZH(t *testing.T) {
	svc := &stubAuthService{
		loginWithGitHubFunc: func(ctx context.Context, code string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrInternal,
				"generate access/refresh token pair failed",
				errors.New("JWT signing error"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/api/oauth/login", LoginGitHub(svc))

	body := `{"code": "valid_code"}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/login?state=test_state", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh")
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

	if resp.Message != "服务器内部错误" {
		t.Fatalf("expected zh message %q, got %q", "服务器内部错误", resp.Message)
	}
}

func testLoginGitHub_InternalError_EN(t *testing.T) {
	svc := &stubAuthService{
		loginWithGitHubFunc: func(ctx context.Context, code string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrInternal,
				"generate access/refresh token pair failed",
				errors.New("JWT signing error"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/api/oauth/login", LoginGitHub(svc))

	body := `{"code": "valid_code"}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/login?state=test_state", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
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

	if resp.Message != "Internal server error" {
		t.Fatalf("expected en message %q, got %q", "Internal server error", resp.Message)
	}
}

func testLoginGitHub_InternalError_DefaultEN(t *testing.T) {
	svc := &stubAuthService{
		loginWithGitHubFunc: func(ctx context.Context, code string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrInternal,
				"generate access/refresh token pair failed",
				errors.New("JWT signing error"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/api/oauth/login", LoginGitHub(svc))

	body := `{"code": "valid_code"}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/login?state=test_state", strings.NewReader(body))
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

	if resp.Message != "Internal server error" {
		t.Fatalf("expected default en message %q, got %q", "Internal server error", resp.Message)
	}
}

func TestLoginWithPassword(t *testing.T) {
	t.Run("Missing username", testLoginWithPassword_MissingUsername)
	t.Run("Missing password", testLoginWithPassword_MissingPassword)
	t.Run("Success", testLoginWithPassword_Success)
	t.Run("Invalid credentials", func(t *testing.T) {
		t.Run("Chinese", testLoginWithPassword_InvalidCredentials_ZH)
		t.Run("English", testLoginWithPassword_InvalidCredentials_EN)
	})
	t.Run("Internal error", func(t *testing.T) {
		t.Run("Chinese", testLoginWithPassword_InternalError_ZH)
		t.Run("English", testLoginWithPassword_InternalError_EN)
	})
}

func testLoginWithPassword_MissingUsername(t *testing.T) {
	svc := &stubAuthService{}
	r := newTestI18nRouter(t)
	r.POST("/auth/login", LoginWithPassword(svc))

	body := `{"password": "testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
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

	if resp.Errors[0].Field != "username" {
		t.Fatalf("expected field 'username', got %q", resp.Errors[0].Field)
	}
}

func testLoginWithPassword_MissingPassword(t *testing.T) {
	svc := &stubAuthService{}
	r := newTestI18nRouter(t)
	r.POST("/auth/login", LoginWithPassword(svc))

	body := `{"username": "testuser"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
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

	if resp.Errors[0].Field != "password" {
		t.Fatalf("expected field 'password', got %q", resp.Errors[0].Field)
	}
}

func testLoginWithPassword_Success(t *testing.T) {
	expectedUser := &models.User{
		ID:       456,
		Username: "testuser",
	}
	expectedTokens := &services.JWTTokenPair{
		AccessToken:  "new_access_token",
		RefreshToken: "new_refresh_token",
	}

	svc := &stubAuthService{
		loginWithPasswordFunc: func(username, password string) (*models.User, *services.JWTTokenPair, error) {
			if username != "testuser" {
				t.Fatalf("expected username 'testuser', got %q", username)
			}
			if password != "correctpass" {
				t.Fatalf("expected password 'correctpass', got %q", password)
			}
			return expectedUser, expectedTokens, nil
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/auth/login", LoginWithPassword(svc))

	body := `{"username": "testuser", "password": "correctpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp authSuccessResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.User.ID != expectedUser.ID {
		t.Fatalf("expected user ID %d, got %d", expectedUser.ID, resp.User.ID)
	}

	if resp.User.Username != expectedUser.Username {
		t.Fatalf("expected username %q, got %q", expectedUser.Username, resp.User.Username)
	}

	if resp.AccessToken != expectedTokens.AccessToken {
		t.Fatalf("expected access_token %q, got %q", expectedTokens.AccessToken, resp.AccessToken)
	}

	if resp.RefreshToken != expectedTokens.RefreshToken {
		t.Fatalf("expected refresh_token %q, got %q", expectedTokens.RefreshToken, resp.RefreshToken)
	}

	if svc.called != 1 {
		t.Fatalf("expected LoginWithPassword called once, got %d", svc.called)
	}
}

func testLoginWithPassword_InvalidCredentials_ZH(t *testing.T) {
	svc := &stubAuthService{
		loginWithPasswordFunc: func(username, password string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrInvalidCredentials,
				"invalid credentials",
				errors.New("username or password incorrect"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/auth/login", LoginWithPassword(svc))

	body := `{"username": "testuser", "password": "wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrInvalidCredentials) {
		t.Fatalf("expected code %q, got %q", string(services.ErrInvalidCredentials), resp.Code)
	}

	if resp.Message != "用户名或密码错误" {
		t.Fatalf("expected zh message %q, got %q", "用户名或密码错误", resp.Message)
	}
}

func testLoginWithPassword_InvalidCredentials_EN(t *testing.T) {
	svc := &stubAuthService{
		loginWithPasswordFunc: func(username, password string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrInvalidCredentials,
				"invalid credentials",
				errors.New("username or password incorrect"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/auth/login", LoginWithPassword(svc))

	body := `{"username": "testuser", "password": "wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrInvalidCredentials) {
		t.Fatalf("expected code %q, got %q", string(services.ErrInvalidCredentials), resp.Code)
	}

	if resp.Message != "Invalid username or password" {
		t.Fatalf("expected en message %q, got %q", "Invalid username or password", resp.Message)
	}
}

func testLoginWithPassword_InternalError_ZH(t *testing.T) {
	svc := &stubAuthService{
		loginWithPasswordFunc: func(username, password string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrInternal,
				"database error",
				errors.New("connection failed"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/auth/login", LoginWithPassword(svc))

	body := `{"username": "testuser", "password": "testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh")
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

	if resp.Message != "服务器内部错误" {
		t.Fatalf("expected zh message %q, got %q", "服务器内部错误", resp.Message)
	}
}

func testLoginWithPassword_InternalError_EN(t *testing.T) {
	svc := &stubAuthService{
		loginWithPasswordFunc: func(username, password string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrInternal,
				"database error",
				errors.New("connection failed"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/auth/login", LoginWithPassword(svc))

	body := `{"username": "testuser", "password": "testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
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

	if resp.Message != "Internal server error" {
		t.Fatalf("expected en message %q, got %q", "Internal server error", resp.Message)
	}
}

func TestRefreshToken(t *testing.T) {
	t.Run("Missing refresh token", testRefreshToken_MissingRefreshToken)
	t.Run("Success", testRefreshToken_Success)
	t.Run("Invalid token", func(t *testing.T) {
		t.Run("Chinese", testRefreshToken_InvalidToken_ZH)
		t.Run("English", testRefreshToken_InvalidToken_EN)
	})
	t.Run("Internal error", func(t *testing.T) {
		t.Run("Chinese", testRefreshToken_InternalError_ZH)
		t.Run("English", testRefreshToken_InternalError_EN)
	})
}

func testRefreshToken_MissingRefreshToken(t *testing.T) {
	svc := &stubAuthService{}
	r := newTestI18nRouter(t)
	r.POST("/auth/refresh", RefreshToken(svc))

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
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

	if resp.Errors[0].Field != "refresh_token" {
		t.Fatalf("expected field 'refresh_token', got %q", resp.Errors[0].Field)
	}
}

func testRefreshToken_Success(t *testing.T) {
	expectedUser := &models.User{
		ID:       789,
		Username: "refresheduser",
	}
	expectedTokens := &services.JWTTokenPair{
		AccessToken:  "refreshed_access_token",
		RefreshToken: "refreshed_refresh_token",
	}

	svc := &stubAuthService{
		refreshTokenFunc: func(refreshToken string) (*models.User, *services.JWTTokenPair, error) {
			if refreshToken != "valid_refresh_token" {
				t.Fatalf("expected refresh_token 'valid_refresh_token', got %q", refreshToken)
			}
			return expectedUser, expectedTokens, nil
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/auth/refresh", RefreshToken(svc))

	body := `{"refresh_token": "valid_refresh_token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp authSuccessResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.User.ID != expectedUser.ID {
		t.Fatalf("expected user ID %d, got %d", expectedUser.ID, resp.User.ID)
	}

	if resp.User.Username != expectedUser.Username {
		t.Fatalf("expected username %q, got %q", expectedUser.Username, resp.User.Username)
	}

	if resp.AccessToken != expectedTokens.AccessToken {
		t.Fatalf("expected access_token %q, got %q", expectedTokens.AccessToken, resp.AccessToken)
	}

	if resp.RefreshToken != expectedTokens.RefreshToken {
		t.Fatalf("expected refresh_token %q, got %q", expectedTokens.RefreshToken, resp.RefreshToken)
	}

	if svc.called != 1 {
		t.Fatalf("expected RefreshToken called once, got %d", svc.called)
	}
}

func testRefreshToken_InvalidToken_ZH(t *testing.T) {
	svc := &stubAuthService{
		refreshTokenFunc: func(refreshToken string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrUnauthorized,
				"invalid refresh token",
				errors.New("token expired"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/auth/refresh", RefreshToken(svc))

	body := `{"refresh_token": "expired_token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrUnauthorized) {
		t.Fatalf("expected code %q, got %q", string(services.ErrUnauthorized), resp.Code)
	}

	if resp.Message != "未授权" {
		t.Fatalf("expected zh message %q, got %q", "未授权", resp.Message)
	}
}

func testRefreshToken_InvalidToken_EN(t *testing.T) {
	svc := &stubAuthService{
		refreshTokenFunc: func(refreshToken string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrUnauthorized,
				"invalid refresh token",
				errors.New("token expired"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/auth/refresh", RefreshToken(svc))

	body := `{"refresh_token": "expired_token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrUnauthorized) {
		t.Fatalf("expected code %q, got %q", string(services.ErrUnauthorized), resp.Code)
	}

	if resp.Message != "Unauthorized" {
		t.Fatalf("expected en message %q, got %q", "Unauthorized", resp.Message)
	}
}

func testRefreshToken_InternalError_ZH(t *testing.T) {
	svc := &stubAuthService{
		refreshTokenFunc: func(refreshToken string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrInternal,
				"database error",
				errors.New("connection failed"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/auth/refresh", RefreshToken(svc))

	body := `{"refresh_token": "some_token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh")
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

	if resp.Message != "服务器内部错误" {
		t.Fatalf("expected zh message %q, got %q", "服务器内部错误", resp.Message)
	}
}

func testRefreshToken_InternalError_EN(t *testing.T) {
	svc := &stubAuthService{
		refreshTokenFunc: func(refreshToken string) (*models.User, *services.JWTTokenPair, error) {
			return nil, nil, services.NewServiceErrorWrap(
				services.ErrInternal,
				"database error",
				errors.New("connection failed"),
			)
		},
	}

	r := newTestI18nRouter(t)
	r.POST("/auth/refresh", RefreshToken(svc))

	body := `{"refresh_token": "some_token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
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

	if resp.Message != "Internal server error" {
		t.Fatalf("expected en message %q, got %q", "Internal server error", resp.Message)
	}
}

type messageResponse struct {
	Message string `json:"message"`
}

func TestChangePassword(t *testing.T) {
	t.Run("Missing new password", testChangePassword_MissingNewPassword)
	t.Run("New password too short", testChangePassword_NewPasswordTooShort)
	t.Run("Success", testChangePassword_Success)
	t.Run("Invalid current password", func(t *testing.T) {
		t.Run("Chinese", testChangePassword_InvalidCurrentPassword_ZH)
		t.Run("English", testChangePassword_InvalidCurrentPassword_EN)
	})
	t.Run("Failed get user", testChangePassword_FailedGetUser)
	t.Run("User not found", testChangePassword_UserNotFound)
	t.Run("Internal error", testChangePassword_InternalError)
}

func testChangePassword_MissingNewPassword(t *testing.T) {
	svc := &stubAuthService{}
	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.POST("/auth/change-password", ChangePassword(svc))

	body := `{"current_password": "oldpass"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", strings.NewReader(body))
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

	if resp.Errors[0].Field != "new_password" {
		t.Fatalf("expected field 'new_password', got %q", resp.Errors[0].Field)
	}

	if resp.Errors[0].Code != string(services.ErrRequired) {
		t.Fatalf("expected detail code %q, got %q", string(services.ErrRequired), resp.Errors[0].Code)
	}
}

func testChangePassword_NewPasswordTooShort(t *testing.T) {
	svc := &stubAuthService{}
	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.POST("/auth/change-password", ChangePassword(svc))

	body := `{"current_password": "oldpass", "new_password": "12345"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", strings.NewReader(body))
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

	if resp.Errors[0].Field != "new_password" {
		t.Fatalf("expected field 'new_password', got %q", resp.Errors[0].Field)
	}

	if resp.Errors[0].Code != string(services.ErrMinLength) {
		t.Fatalf("expected detail code %q, got %q", string(services.ErrMinLength), resp.Errors[0].Code)
	}
}

func testChangePassword_Success(t *testing.T) {
	svc := &stubAuthService{
		changePasswordFunc: func(userID int, current, new string) error {
			if userID != 123 {
				t.Fatalf("expected userID 123, got %d", userID)
			}
			if current != "oldpass" {
				t.Fatalf("expected current password 'oldpass', got %q", current)
			}
			if new != "newpass123" {
				t.Fatalf("expected new password 'newpass123', got %q", new)
			}
			return nil
		},
	}

	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.POST("/auth/change-password", ChangePassword(svc))

	body := `{"current_password": "oldpass", "new_password": "newpass123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp messageResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Message != "Password changed successfully" {
		t.Fatalf("expected message 'Password changed successfully', got %q", resp.Message)
	}

	if svc.called != 1 {
		t.Fatalf("expected ChangePassword called once, got %d", svc.called)
	}
}

func testChangePassword_InvalidCurrentPassword_ZH(t *testing.T) {
	svc := &stubAuthService{
		changePasswordFunc: func(userID int, current, new string) error {
			return services.NewServiceErrorWrap(
				services.ErrInvalidPassword,
				"current password incorrect",
				errors.New("password mismatch"),
			)
		},
	}

	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.POST("/auth/change-password", ChangePassword(svc))

	body := `{"current_password": "wrongpass", "new_password": "newpass123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrInvalidPassword) {
		t.Fatalf("expected code %q, got %q", string(services.ErrInvalidPassword), resp.Code)
	}

	if resp.Message != "当前密码不正确" {
		t.Fatalf("expected zh message %q, got %q", "当前密码不正确", resp.Message)
	}
}

func testChangePassword_InvalidCurrentPassword_EN(t *testing.T) {
	svc := &stubAuthService{
		changePasswordFunc: func(userID int, current, new string) error {
			return services.NewServiceErrorWrap(
				services.ErrInvalidPassword,
				"current password incorrect",
				errors.New("password mismatch"),
			)
		},
	}

	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.POST("/auth/change-password", ChangePassword(svc))

	body := `{"current_password": "wrongpass", "new_password": "newpass123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrInvalidPassword) {
		t.Fatalf("expected code %q, got %q", string(services.ErrInvalidPassword), resp.Code)
	}

	if resp.Message != "Current password is incorrect" {
		t.Fatalf("expected en message %q, got %q", "Current password is incorrect", resp.Message)
	}
}

func testChangePassword_FailedGetUser(t *testing.T) {
	svc := &stubAuthService{}
	r := newTestI18nRouter(t)
	r.POST("/auth/change-password", ChangePassword(svc))

	body := `{"current_password": "oldpass", "new_password": "newpass123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", strings.NewReader(body))
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

func testChangePassword_UserNotFound(t *testing.T) {
	svc := &stubAuthService{
		changePasswordFunc: func(userID int, current, new string) error {
			return services.NewServiceError(
				services.ErrNotFound,
				"user not found",
			)
		},
	}

	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.POST("/auth/change-password", ChangePassword(svc))

	body := `{"current_password": "oldpass", "new_password": "newpass123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrNotFound) {
		t.Fatalf("expected code %q, got %q", string(services.ErrNotFound), resp.Code)
	}
}

func testChangePassword_InternalError(t *testing.T) {
	svc := &stubAuthService{
		changePasswordFunc: func(userID int, current, new string) error {
			return services.NewServiceErrorWrap(
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

	r.POST("/auth/change-password", ChangePassword(svc))

	body := `{"current_password": "oldpass", "new_password": "newpass123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
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

	if resp.Message != "Internal server error" {
		t.Fatalf("expected en message %q, got %q", "Internal server error", resp.Message)
	}
}

type postKeyResponse struct {
	PostKey   string    `json:"post_key"`
	CreatedAt time.Time `json:"created_at"`
}

func TestQueryPostKey(t *testing.T) {
	t.Run("Success", testQueryPostKey_Success)
	t.Run("User not found", testQueryPostKey_UserNotFound)
	t.Run("Failed get user", testQueryPostKey_FailedGetUser)
	t.Run("Internal error", testQueryPostKey_InternalError)
}

func testQueryPostKey_Success(t *testing.T) {
	expectedPostKey := "test-post-key-123"
	expectedCreatedAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	svc := &stubAuthService{
		queryPostKeyFunc: func(userID int) (string, time.Time, error) {
			if userID != 123 {
				t.Fatalf("expected userID 123, got %d", userID)
			}
			return expectedPostKey, expectedCreatedAt, nil
		},
	}

	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.GET("/post_key", QueryPostKey(svc))

	req := httptest.NewRequest(http.MethodGet, "/post_key", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp postKeyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.PostKey != expectedPostKey {
		t.Fatalf("expected post_key %q, got %q", expectedPostKey, resp.PostKey)
	}

	if !resp.CreatedAt.Equal(expectedCreatedAt) {
		t.Fatalf("expected created_at %v, got %v", expectedCreatedAt, resp.CreatedAt)
	}

	if svc.called != 1 {
		t.Fatalf("expected QueryPostKey called once, got %d", svc.called)
	}
}

func testQueryPostKey_UserNotFound(t *testing.T) {
	svc := &stubAuthService{
		queryPostKeyFunc: func(userID int) (string, time.Time, error) {
			return "", time.Time{}, services.NewServiceError(
				services.ErrNotFound,
				"post key not found",
			)
		},
	}

	r := newTestI18nRouter(t)

	testUser := &models.User{ID: 123, Username: "testuser"}
	r.Use(func(c *gin.Context) {
		c.Set("user", testUser)
		c.Next()
	})

	r.GET("/post_key", QueryPostKey(svc))

	req := httptest.NewRequest(http.MethodGet, "/post_key", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != string(services.ErrNotFound) {
		t.Fatalf("expected code %q, got %q", string(services.ErrNotFound), resp.Code)
	}
}

func testQueryPostKey_FailedGetUser(t *testing.T) {
	svc := &stubAuthService{}
	r := newTestI18nRouter(t)
	r.GET("/post_key", QueryPostKey(svc))

	req := httptest.NewRequest(http.MethodGet, "/post_key", nil)
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

func testQueryPostKey_InternalError(t *testing.T) {
	svc := &stubAuthService{
		queryPostKeyFunc: func(userID int) (string, time.Time, error) {
			return "", time.Time{}, services.NewServiceErrorWrap(
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

	r.GET("/post_key", QueryPostKey(svc))

	req := httptest.NewRequest(http.MethodGet, "/post_key", nil)
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
