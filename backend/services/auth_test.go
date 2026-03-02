package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"markpost/models"
	"markpost/repositories"
	"markpost/utils"

	"golang.org/x/oauth2"
)

func newOAuthMockServer(userJSON string, tokenStatus int) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if tokenStatus != http.StatusOK {
			w.WriteHeader(tokenStatus)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-access","token_type":"Bearer"}`))
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(userJSON))
	})
	return httptest.NewServer(mux)
}

type rewriteTransport struct {
	base      http.RoundTripper
	serverURL string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "api.github.com" && req.URL.Path == "/user" {
		u, _ := url.Parse(rt.serverURL)
		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host
		req.URL.Path = "/user"
	}
	return rt.base.RoundTrip(req)
}

type stubUserRepo struct {
	user      *models.User
	getErr    error
	updateErr error
}

func (s *stubUserRepo) GetUserByPostKey(postKey string) (*models.User, error) {
	return nil, models.ErrNotFound
}
func (s *stubUserRepo) GetUserByID(id int) (*models.User, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.user, nil
}
func (s *stubUserRepo) GetUserByGitHubID(githubID int64) (*models.User, error) {
	return nil, models.ErrNotFound
}
func (s *stubUserRepo) GetUserByUsername(username string) (*models.User, error) {
	return nil, models.ErrNotFound
}
func (s *stubUserRepo) CreateUserFromGitHub(githubUser *models.GitHubUser) (*models.User, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubUserRepo) GetOrCreateUserFromGitHub(githubUser *models.GitHubUser) (*models.User, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubUserRepo) CreateUser(username, password string) (*models.User, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubUserRepo) ValidateUserPassword(username, password string) (*models.User, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubUserRepo) SetUserPassword(userID int, password string) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	hashed, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	if s.user != nil {
		s.user.Password = hashed
	}
	return nil
}

func (s *stubUserRepo) SetUserRole(userID int, role models.Role) error {
	if s.user != nil && s.user.ID == userID {
		s.user.Role = role
	}
	return nil
}

func (s *stubUserRepo) DeleteUserByID(userID int) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (s *stubUserRepo) GetAllUsers(offset, limit int) ([]models.User, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *stubUserRepo) CountUsers() (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func setupAuthTestDatabase(t *testing.T) *models.Database {
	t.Helper()

	database, err := models.NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}

	return database
}

func newTestOAuthConfig(authURL, tokenURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{},
		Endpoint:     oauth2.Endpoint{AuthURL: authURL, TokenURL: tokenURL},
	}
}

func TestAuthService_GenerateGitHubAuthURL(t *testing.T) {
	oauthCfg := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{},
		Endpoint:     oauth2.Endpoint{AuthURL: "http://example.com/auth", TokenURL: "http://example.com/token"},
	}
	jwtSvc := newTestJWTService()
	users := &stubUserRepo{}
	svc := NewAuthService(users, oauthCfg, jwtSvc)

	u, err := svc.GenerateGitHubAuthURL(context.Background())
	if err != nil {
		t.Fatalf("GenerateGitHubAuthURL error: %v", err)
	}
	if u == "" {
		t.Fatalf("empty auth url")
	}

	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("parse url error: %v", err)
	}
	q := parsed.Query()
	if got := q.Get("client_id"); got != oauthCfg.ClientID {
		t.Fatalf("client_id mismatch: %s", got)
	}
	if got := q.Get("redirect_uri"); got != oauthCfg.RedirectURL {
		t.Fatalf("redirect_uri mismatch: %s", got)
	}
}

func TestAuthService_LoginWithGitHub(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		database := setupAuthTestDatabase(t)

		userRepo := repositories.NewUserRepo(database)

		server := newOAuthMockServer(`{"id":1234,"login":"alice"}`, http.StatusOK)
		defer server.Close()

		oauthCfg := newTestOAuthConfig(server.URL+"/auth", server.URL+"/token")
		jwtSvc := newTestJWTService()
		svc := NewAuthService(userRepo, oauthCfg, jwtSvc)

		orig := http.DefaultTransport
		http.DefaultTransport = &rewriteTransport{base: orig, serverURL: server.URL}
		defer func() { http.DefaultTransport = orig }()

		user, tokens, err := svc.LoginWithGitHub(context.Background(), "dummy-code")
		if err != nil {
			t.Fatalf("LoginWithGitHub error: %v", err)
		}
		if user == nil || user.ID == 0 || user.Username != "alice" || user.GitHubID == nil || *user.GitHubID != 1234 {
			t.Fatalf("unexpected user: %+v", user)
		}
		if tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
			t.Fatalf("unexpected tokens: %+v", tokens)
		}
	})

	t.Run("exchange failed -> ErrUnauthorized", func(t *testing.T) {
		db := setupAuthTestDatabase(t)

		userRepo := repositories.NewUserRepo(db)

		server := newOAuthMockServer(`{"id":1234,"login":"alice"}`, http.StatusBadRequest)
		defer server.Close()

		oauthCfg := newTestOAuthConfig(server.URL+"/auth", server.URL+"/token")
		jwtSvc := newTestJWTService()
		svc := NewAuthService(userRepo, oauthCfg, jwtSvc)

		_, _, err := svc.LoginWithGitHub(context.Background(), "bad-code")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrUnauthorized {
			t.Fatalf("expected ServiceErrorUnauthorized, got: %#v", err)
		}
	})

	t.Run("fetch GitHub user failed -> ErrInternal", func(t *testing.T) {
		database := setupAuthTestDatabase(t)
		userRepo := repositories.NewUserRepo(database)

		server := newOAuthMockServer(`{"id":0,"login":""}`, http.StatusOK)
		defer server.Close()

		oauthCfg := newTestOAuthConfig(server.URL+"/auth", server.URL+"/token")
		jwtSvc := newTestJWTService()
		svc := NewAuthService(userRepo, oauthCfg, jwtSvc)

		orig := http.DefaultTransport
		http.DefaultTransport = &rewriteTransport{base: orig, serverURL: server.URL}
		defer func() { http.DefaultTransport = orig }()

		_, _, err := svc.LoginWithGitHub(context.Background(), "code")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrInternal {
			t.Fatalf("expected ServiceErrorInternal, got: %#v", err)
		}
	})
}

func TestAuthService_LoginWithPassword(t *testing.T) {
	database := setupAuthTestDatabase(t)

	repo := repositories.NewUserRepo(database)
	user, err := repo.CreateUser("carol", "pass123")
	if err != nil || user == nil {
		t.Fatalf("seed user error: %v", err)
	}

	jwtSvc := newTestJWTService()
	oauthCfg := &oauth2.Config{}
	svc := NewAuthService(repo, oauthCfg, jwtSvc)

	u, tokens, err := svc.LoginWithPassword("carol", "pass123")
	if err != nil {
		t.Fatalf("LoginWithPassword error: %v", err)
	}
	if u == nil || tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("unexpected login result: %v %v", u, tokens)
	}

	_, _, err = svc.LoginWithPassword("carol", "bad")
	if err == nil {
		t.Fatalf("expected error for invalid credentials")
	}
	if se, ok := err.(*ServiceError); !ok || se.Code != ErrInvalidCredentials {
		t.Fatalf("expected ServiceErrorInvalidCredentials, got: %#v", err)
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		database := setupAuthTestDatabase(t)

		repo := repositories.NewUserRepo(database)
		user, err := repo.CreateUser("rita", "pass123")
		if err != nil || user == nil {
			t.Fatalf("seed user error: %v", err)
		}

		jwtSvc := newTestJWTService()
		oauthCfg := &oauth2.Config{}
		svc := NewAuthService(repo, oauthCfg, jwtSvc)

		token, err := jwtSvc.GenerateRefreshToken(user.ID, string(user.Role))
		if err != nil || token == "" {
			t.Fatalf("generate token error: %v", err)
		}

		u, tokens, err := svc.RefreshToken(token)
		if err != nil {
			t.Fatalf("RefreshToken error: %v", err)
		}
		if u == nil || tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" || u.ID != user.ID {
			t.Fatalf("unexpected refresh result: %v %v", u, tokens)
		}
	})

	t.Run("invalid token -> ErrUnauthorized", func(t *testing.T) {
		database := setupAuthTestDatabase(t)
		repo := repositories.NewUserRepo(database)
		jwtSvc := newTestJWTService()
		oauthCfg := &oauth2.Config{}
		svc := NewAuthService(repo, oauthCfg, jwtSvc)

		_, _, err := svc.RefreshToken("invalid.token")
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrUnauthorized {
			t.Fatalf("expected ServiceErrorUnauthorized, got: %#v", err)
		}
	})

	t.Run("user not found -> ErrUnauthorized", func(t *testing.T) {
		database := setupAuthTestDatabase(t)

		repo := repositories.NewUserRepo(database)
		jwtSvc := newTestJWTService()
		oauthCfg := &oauth2.Config{}
		svc := NewAuthService(repo, oauthCfg, jwtSvc)

		token, err := jwtSvc.GenerateRefreshToken(99999, "user")
		if err != nil || token == "" {
			t.Fatalf("generate token error: %v", err)
		}

		_, _, err = svc.RefreshToken(token)
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrUnauthorized {
			t.Fatalf("expected ServiceErrorUnauthorized, got: %#v", err)
		}
	})
}

func TestAuthService_ChangePassword(t *testing.T) {
	t.Run("user not found -> ErrNotFound", func(t *testing.T) {
		database := setupAuthTestDatabase(t)

		repo := repositories.NewUserRepo(database)
		jwtSvc := newTestJWTService()
		oauthCfg := &oauth2.Config{}
		svc := NewAuthService(repo, oauthCfg, jwtSvc)

		err := svc.ChangePassword(999999, "x", "y")
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrNotFound {
			t.Fatalf("expected ServiceErrorNotFound, got: %#v", err)
		}
	})

	t.Run("invalid current password -> ErrInvalidPassword", func(t *testing.T) {
		database := setupAuthTestDatabase(t)

		repo := repositories.NewUserRepo(database)
		user, _ := repo.CreateUser("dora", "old")
		jwtSvc := newTestJWTService()
		oauthCfg := &oauth2.Config{}
		svc := NewAuthService(repo, oauthCfg, jwtSvc)

		err := svc.ChangePassword(user.ID, "wrong", "new")
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrInvalidPassword {
			t.Fatalf("expected ServiceErrorInvalidPassword, got: %#v", err)
		}
	})

	t.Run("same password allowed", func(t *testing.T) {
		database := setupAuthTestDatabase(t)

		repo := repositories.NewUserRepo(database)
		user, _ := repo.CreateUser("ed", "old")
		jwtSvc := newTestJWTService()
		oauthCfg := &oauth2.Config{}
		svc := NewAuthService(repo, oauthCfg, jwtSvc)

		err := svc.ChangePassword(user.ID, "old", "old")
		if err != nil {
			t.Fatalf("expected no error, got: %#v", err)
		}
		updated, err := repo.GetUserByID(user.ID)
		if err != nil {
			t.Fatalf("GetUserByID error: %v", err)
		}
		if ok, err := utils.CheckPassword("old", updated.Password); err != nil || !ok {
			t.Fatalf("expected password to remain valid after same-password change, ok=%v err=%v", ok, err)
		}
	})

	t.Run("update password failed -> ErrInternal (stub)", func(t *testing.T) {
		hashed, _ := utils.HashPassword("old")
		stub := &stubUserRepo{user: &models.User{ID: 1, Username: "stub", Password: hashed}, updateErr: fmt.Errorf("update failed")}
		jwtSvc := newTestJWTService()
		oauthCfg := &oauth2.Config{}
		svc := NewAuthService(stub, oauthCfg, jwtSvc)
		err := svc.ChangePassword(1, "old", "new")
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrInternal {
			t.Fatalf("expected ServiceErrorInternal, got: %#v", err)
		}
	})

	t.Run("success and DB value updated", func(t *testing.T) {
		database := setupAuthTestDatabase(t)

		repo := repositories.NewUserRepo(database)
		user, _ := repo.CreateUser("ellen", "oldpw")
		jwtSvc := newTestJWTService()
		oauthCfg := &oauth2.Config{}
		svc := NewAuthService(repo, oauthCfg, jwtSvc)

		if err := svc.ChangePassword(user.ID, "oldpw", "newpw"); err != nil {
			t.Fatalf("ChangePassword error: %v", err)
		}
		u2, err := repo.GetUserByID(user.ID)
		if err != nil {
			t.Fatalf("GetUserByID error: %v", err)
		}
		if _, err := utils.CheckPassword("newpw", u2.Password); err != nil {
			t.Fatalf("password not updated correctly")
		}
	})
}

func TestAuthService_QueryPostKey(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		database := setupAuthTestDatabase(t)

		repo := repositories.NewUserRepo(database)
		jwtSvc := newTestJWTService()
		oauthCfg := &oauth2.Config{}
		svc := NewAuthService(repo, oauthCfg, jwtSvc)
		user, _ := repo.CreateUser("quinn", "pw")

		postKey, createdAt, err := svc.QueryPostKey(user.ID)
		if err != nil {
			t.Fatalf("QueryPostKey error: %v", err)
		}
		if postKey == "" || createdAt.IsZero() || postKey != user.PostKey {
			t.Fatalf("unexpected post key or createdAt: %s %v", postKey, createdAt)
		}
	})

	t.Run("not found -> ErrNotFound", func(t *testing.T) {
		database := setupAuthTestDatabase(t)

		repo := repositories.NewUserRepo(database)
		jwtSvc := newTestJWTService()
		oauthCfg := &oauth2.Config{}
		svc := NewAuthService(repo, oauthCfg, jwtSvc)

		_, _, err := svc.QueryPostKey(99999)
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrNotFound {
			t.Fatalf("expected ServiceErrorNotFound, got: %#v", err)
		}
	})

	t.Run("other error -> ErrInternal (stub)", func(t *testing.T) {
		stub := &stubUserRepo{getErr: fmt.Errorf("db error")}
		jwtSvc := newTestJWTService()
		oauthCfg := &oauth2.Config{}
		svc := NewAuthService(stub, oauthCfg, jwtSvc)
		_, _, err := svc.QueryPostKey(1)
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrInternal {
			t.Fatalf("expected ServiceErrorInternal, got: %#v", err)
		}
	})
}
