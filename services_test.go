package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// ---- Helpers ----

func setTestConfig() {
	// Minimal JWT config
	config.JWT.SecretKey = "test-secret"
	config.JWT.AccessTokenExpire = time.Hour
	config.JWT.RefreshTokenExpire = 24 * time.Hour

	// GitHub OAuth config
	config.GitHub.ClientID = "test-client-id"
	config.GitHub.ClientSecret = "test-client-secret"
	config.GitHub.RedirectURL = "http://localhost/callback"
	initOAuthConfig()
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

// Minimal stub for UserRepository to trigger specific error paths
type stubUserRepo struct {
	user      *User
	getErr    error
	updateErr error
}

func (s *stubUserRepo) GetUserByPostKey(postKey string) (*User, error) { return nil, sql.ErrNoRows }
func (s *stubUserRepo) GetUserByID(id int) (*User, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.user, nil
}
func (s *stubUserRepo) GetUserByGitHubID(githubID int64) (*User, error)  { return nil, sql.ErrNoRows }
func (s *stubUserRepo) GetUserByUsername(username string) (*User, error) { return nil, sql.ErrNoRows }
func (s *stubUserRepo) CreateUserFromGitHubUser(githubUser *GitHubUser) (*User, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubUserRepo) GetOrCreateUserFromGitHubUser(githubUser *GitHubUser) (*User, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubUserRepo) CreateUserWithPassword(username, password string) (*User, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubUserRepo) ValidateUserPassword(username, password string) (*User, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubUserRepo) UpdatePassword(userID int, hashed string) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if s.user != nil {
		s.user.Password = hashed
	}
	return nil
}

// ---- AuthService tests ----

func TestAuthService_GenerateGitHubAuthURL(t *testing.T) {
	setTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	svc := NewAuthService(db.GetUserRepository(), oauthConfig)
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
	if got := q.Get("client_id"); got != config.GitHub.ClientID {
		t.Fatalf("client_id mismatch: %s", got)
	}
	if got := q.Get("redirect_uri"); got != config.GitHub.RedirectURL {
		t.Fatalf("redirect_uri mismatch: %s", got)
	}
}

func TestAuthService_LoginWithGitHub(t *testing.T) {
	setTestConfig()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		server := newOAuthMockServer(`{"id":1234,"login":"alice"}`, http.StatusOK)
		defer server.Close()

		// Prepare oauth config to use mock token endpoint
		oauthConfig = &oauth2.Config{
			ClientID:     config.GitHub.ClientID,
			ClientSecret: config.GitHub.ClientSecret,
			RedirectURL:  config.GitHub.RedirectURL,
			Endpoint:     oauth2.Endpoint{AuthURL: server.URL + "/auth", TokenURL: server.URL + "/token"},
		}
		svc := NewAuthService(db.GetUserRepository(), oauthConfig)

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

	t.Run("exchange failed -> ErrInternal", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		server := newOAuthMockServer(`{"id":1234,"login":"alice"}`, http.StatusBadRequest)
		defer server.Close()

		oauthConfig = &oauth2.Config{
			ClientID:     config.GitHub.ClientID,
			ClientSecret: config.GitHub.ClientSecret,
			RedirectURL:  config.GitHub.RedirectURL,
			Endpoint:     oauth2.Endpoint{AuthURL: server.URL + "/auth", TokenURL: server.URL + "/token"},
		}
		svc := NewAuthService(db.GetUserRepository(), oauthConfig)

		_, _, err := svc.LoginWithGitHub(context.Background(), "bad-code")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrInternal {
			t.Fatalf("expected ErrInternal, got: %#v", err)
		}
	})

	t.Run("fetch GitHub user failed -> ErrUnauthorized", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		// Return invalid user payload to trigger validation error in getGitHubUser
		server := newOAuthMockServer(`{"id":0,"login":""}`, http.StatusOK)
		defer server.Close()

		oauthConfig = &oauth2.Config{
			ClientID:     config.GitHub.ClientID,
			ClientSecret: config.GitHub.ClientSecret,
			RedirectURL:  config.GitHub.RedirectURL,
			Endpoint:     oauth2.Endpoint{AuthURL: server.URL + "/auth", TokenURL: server.URL + "/token"},
		}
		svc := NewAuthService(db.GetUserRepository(), oauthConfig)

		orig := http.DefaultTransport
		http.DefaultTransport = &rewriteTransport{base: orig, serverURL: server.URL}
		defer func() { http.DefaultTransport = orig }()

		_, _, err := svc.LoginWithGitHub(context.Background(), "code")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrUnauthorized {
			t.Fatalf("expected ErrUnauthorized, got: %#v", err)
		}
	})
}

func TestAuthService_LoginWithPassword(t *testing.T) {
	setTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := db.GetUserRepository()
	user, err := repo.CreateUserWithPassword("carol", "pass123")
	if err != nil || user == nil {
		t.Fatalf("seed user error: %v", err)
	}

	svc := NewAuthService(repo, oauthConfig)
	u, tokens, err := svc.LoginWithPassword(context.Background(), "carol", "pass123")
	if err != nil {
		t.Fatalf("LoginWithPassword error: %v", err)
	}
	if u == nil || tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("unexpected login result: %v %v", u, tokens)
	}

	_, _, err = svc.LoginWithPassword(context.Background(), "carol", "bad")
	if err == nil {
		t.Fatalf("expected error for invalid credentials")
	}
	if se, ok := err.(*ServiceError); !ok || se.Code != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got: %#v", err)
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
    setTestConfig()

    t.Run("success", func(t *testing.T) {
        db := setupTestDB(t)
        defer teardownTestDB(t, db)

        repo := db.GetUserRepository()
        user, err := repo.CreateUserWithPassword("rita", "pass123")
        if err != nil || user == nil {
            t.Fatalf("seed user error: %v", err)
        }

        token, err := generateJWTToken(user.ID, config.JWT.RefreshTokenExpire, config.JWT.SecretKey)
        if err != nil || token == "" {
            t.Fatalf("generate token error: %v", err)
        }

        svc := NewAuthService(repo, oauthConfig)
        u, tokens, err := svc.RefreshToken(context.Background(), token)
        if err != nil {
            t.Fatalf("RefreshToken error: %v", err)
        }
        if u == nil || tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" || u.ID != user.ID {
            t.Fatalf("unexpected refresh result: %v %v", u, tokens)
        }
    })

    t.Run("invalid token -> ErrUnauthorized", func(t *testing.T) {
        db := setupTestDB(t)
        defer teardownTestDB(t, db)

        svc := NewAuthService(db.GetUserRepository(), oauthConfig)
        _, _, err := svc.RefreshToken(context.Background(), "invalid.token")
        if err == nil {
            t.Fatalf("expected error")
        }
        if se, ok := err.(*ServiceError); !ok || se.Code != ErrUnauthorized {
            t.Fatalf("expected ErrUnauthorized, got: %#v", err)
        }
    })

    t.Run("user not found -> ErrNotFound", func(t *testing.T) {
        db := setupTestDB(t)
        defer teardownTestDB(t, db)

        token, err := generateJWTToken(99999, config.JWT.RefreshTokenExpire, config.JWT.SecretKey)
        if err != nil || token == "" {
            t.Fatalf("generate token error: %v", err)
        }

        svc := NewAuthService(db.GetUserRepository(), oauthConfig)
        _, _, err = svc.RefreshToken(context.Background(), token)
        if err == nil {
            t.Fatalf("expected error")
        }
        if se, ok := err.(*ServiceError); !ok || se.Code != ErrNotFound {
            t.Fatalf("expected ErrNotFound, got: %#v", err)
        }
    })
}

func TestAuthService_ChangePassword(t *testing.T) {
	setTestConfig()

	t.Run("user not found -> ErrNotFound", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)
		svc := NewAuthService(db.GetUserRepository(), oauthConfig)
		err := svc.ChangePassword(context.Background(), 999999, "x", "y")
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrNotFound {
			t.Fatalf("expected ErrNotFound, got: %#v", err)
		}
	})

	t.Run("invalid current password -> ErrInvalidCurrentPassword", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)
		repo := db.GetUserRepository()
		u, _ := repo.CreateUserWithPassword("dora", "old")
		svc := NewAuthService(repo, oauthConfig)
		err := svc.ChangePassword(context.Background(), u.ID, "wrong", "new")
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrInvalidCurrentPassword {
			t.Fatalf("expected ErrInvalidCurrentPassword, got: %#v", err)
		}
	})

	t.Run("same password -> ErrSamePassword", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)
		repo := db.GetUserRepository()
		u, _ := repo.CreateUserWithPassword("ed", "old")
		svc := NewAuthService(repo, oauthConfig)
		err := svc.ChangePassword(context.Background(), u.ID, "old", "old")
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrSamePassword {
			t.Fatalf("expected ErrSamePassword, got: %#v", err)
		}
	})

	t.Run("update password failed -> ErrInternal (stub)", func(t *testing.T) {
		// Prepare a user with an initial hashed password
		hashed, _ := HashPassword("old")
		stub := &stubUserRepo{user: &User{ID: 1, Username: "stub", Password: hashed}, updateErr: fmt.Errorf("update failed")}
		svc := NewAuthService(stub, oauthConfig)
		err := svc.ChangePassword(context.Background(), 1, "old", "new")
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrInternal {
			t.Fatalf("expected ErrInternal, got: %#v", err)
		}
	})

	t.Run("success and DB value updated", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)
		repo := db.GetUserRepository()
		u, _ := repo.CreateUserWithPassword("ellen", "oldpw")
		svc := NewAuthService(repo, oauthConfig)
		if err := svc.ChangePassword(context.Background(), u.ID, "oldpw", "newpw"); err != nil {
			t.Fatalf("ChangePassword error: %v", err)
		}
		// Verify DB updated
		u2, err := repo.GetUserByID(u.ID)
		if err != nil {
			t.Fatalf("GetUserByID error: %v", err)
		}
		if err := CheckPassword("newpw", u2.Password); err != nil {
			t.Fatalf("password not updated correctly")
		}
	})
}

func TestAuthService_QueryPostKey(t *testing.T) {
	setTestConfig()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)
		repo := db.GetUserRepository()
		u, _ := repo.CreateUserWithPassword("quinn", "pw")
		svc := NewAuthService(repo, oauthConfig)
		postKey, createdAt, err := svc.QueryPostKey(context.Background(), u.ID)
		if err != nil {
			t.Fatalf("QueryPostKey error: %v", err)
		}
		if postKey == "" || createdAt.IsZero() || postKey != u.PostKey {
			t.Fatalf("unexpected post key or createdAt: %s %v", postKey, createdAt)
		}
	})

	t.Run("not found -> ErrNotFound", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)
		svc := NewAuthService(db.GetUserRepository(), oauthConfig)
		_, _, err := svc.QueryPostKey(context.Background(), 99999)
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrNotFound {
			t.Fatalf("expected ErrNotFound, got: %#v", err)
		}
	})

	t.Run("other error -> ErrInternal (stub)", func(t *testing.T) {
		stub := &stubUserRepo{getErr: fmt.Errorf("db error")}
		svc := NewAuthService(stub, oauthConfig)
		_, _, err := svc.QueryPostKey(context.Background(), 1)
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrInternal {
			t.Fatalf("expected ErrInternal, got: %#v", err)
		}
	})
}

// ---- PostService tests ----

func TestPostService_CreatePost(t *testing.T) {
	setTestConfig()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)
		repoU := db.GetUserRepository()
		u, _ := repoU.CreateUserWithPassword("poster", "pw")

		ps := NewPostService(db.GetPostRepository())
		id, err := ps.CreatePost(context.Background(), "Hello", "World", u.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}
		if id == "" {
			t.Fatalf("empty id")
		}
		p, err := db.GetPostRepository().GetPostByID(id)
		if err != nil || p == nil || p.UserID == nil || *p.UserID != u.ID {
			t.Fatalf("post not persisted or wrong user: %v %+v", err, p)
		}
	})

	t.Run("repo error -> ErrInternal (database closed)", func(t *testing.T) {
		db := setupTestDB(t)
		// close underlying DB to simulate repository failure
		sqlDB, _ := db.GetDB().DB()
		_ = sqlDB.Close()
		ps := NewPostService(db.GetPostRepository())
		_, err := ps.CreatePost(context.Background(), "t", "b", 1)
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrInternal {
			t.Fatalf("expected ErrInternal, got: %#v", err)
		}
	})
}

func TestPostService_RenderPostHTML(t *testing.T) {
	setTestConfig()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)
		repoU := db.GetUserRepository()
		u, _ := repoU.CreateUserWithPassword("md", "pw")
		pr := db.GetPostRepository()
		p, _ := pr.CreatePostWithUser("Title", "Hello\n\nWorld", u.ID)

		ps := NewPostService(pr)
		title, html, err := ps.RenderPostHTML(context.Background(), p.ID)
		if err != nil {
			t.Fatalf("RenderPostHTML error: %v", err)
		}
		if title != "Title" || html == "" || !strings.Contains(html, "<p>") {
			t.Fatalf("unexpected render result: title=%s html=%s", title, html)
		}
	})

	t.Run("not found -> ErrNotFound", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)
		ps := NewPostService(db.GetPostRepository())
		_, _, err := ps.RenderPostHTML(context.Background(), "non-existent")
		if err == nil {
			t.Fatalf("expected error")
		}
		if se, ok := err.(*ServiceError); !ok || se.Code != ErrNotFound {
			t.Fatalf("expected ErrNotFound, got: %#v", err)
		}
	})
}
