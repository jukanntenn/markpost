package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"markpost/internal/infra"
	"markpost/internal/service"

	"golang.org/x/oauth2"
)

type interceptTransport struct {
	inner http.RoundTripper
	mux   *http.ServeMux
}

func (t *interceptTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Host, "github.com") {
		rec := httptest.NewRecorder()
		t.mux.ServeHTTP(rec, req)
		return rec.Result(), nil
	}
	return t.inner.RoundTrip(req)
}

func newGitHubMockMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-github-token",
			"token_type":   "bearer",
		})
	})

	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         12345,
			"login":      "testuser",
			"avatar_url": "https://example.com/avatar.png",
		})
	})

	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"email": "primary@example.com", "primary": true, "verified": true},
			{"email": "secondary@example.com", "primary": false, "verified": true},
		})
	})

	return mux
}

func setupGitHubAuthService(t *testing.T, mux *http.ServeMux) *Service {
	t.Helper()

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	tokenRepo := infra.NewTokenRepository(db)
	jwtSvc := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour*24)

	oauthCfg := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  ts.URL + "/login/oauth/authorize",
			TokenURL: ts.URL + "/login/oauth/access_token",
		},
		RedirectURL: "http://localhost/callback",
		Scopes:      []string{"user:email"},
	}

	return NewService(userRepo, tokenRepo, oauthCfg, jwtSvc, "markpost")
}

func ctxWithGitHubTransport(mux *http.ServeMux) context.Context {
	ctx := context.Background()
	return context.WithValue(ctx, oauth2.HTTPClient, &http.Client{
		Transport: &interceptTransport{
			inner: http.DefaultTransport,
			mux:   mux,
		},
	})
}

func TestGenerateGitHubAuthURL(t *testing.T) {
	svc := setupGitHubAuthService(t, newGitHubMockMux())

	url, err := svc.GenerateGitHubAuthURL(context.Background())
	if err != nil {
		t.Fatalf("GenerateGitHubAuthURL error: %v", err)
	}
	if url == "" {
		t.Fatal("expected non-empty URL")
	}
}

func TestLoginWithGitHub(t *testing.T) {
	t.Run("creates new user and returns tokens", func(t *testing.T) {
		svc := setupGitHubAuthService(t, newGitHubMockMux())
		ctx := ctxWithGitHubTransport(newGitHubMockMux())

		u, tokens, err := svc.LoginWithGitHub(ctx, "test-code")
		if err != nil {
			t.Fatalf("LoginWithGitHub error: %v", err)
		}
		if u == nil {
			t.Fatal("expected user, got nil")
		}
		if u.Username != "testuser" {
			t.Errorf("expected username 'testuser', got %q", u.Username)
		}
		if tokens == nil {
			t.Fatal("expected tokens, got nil")
		}
		if tokens.AccessToken == "" {
			t.Error("expected access token")
		}
	})

	t.Run("returns existing user on second login", func(t *testing.T) {
		mux := newGitHubMockMux()
		svc := setupGitHubAuthService(t, mux)
		ctx := ctxWithGitHubTransport(mux)

		u1, _, _ := svc.LoginWithGitHub(ctx, "test-code")

		// Delete existing refresh tokens to avoid unique constraint on token hash
		_ = svc.tokens.DeleteRefreshTokensByUserID(ctx, u1.ID)

		u2, _, err := svc.LoginWithGitHub(ctx, "test-code")
		if err != nil {
			t.Fatalf("LoginWithGitHub error: %v", err)
		}
		if u1.ID != u2.ID {
			t.Errorf("expected same user ID, got %d and %d", u1.ID, u2.ID)
		}
	})
}

func TestLoginWithGitHub_InvalidCode(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, "bad verification code")
	})

	svc := setupGitHubAuthService(t, mux)
	ctx := ctxWithGitHubTransport(mux)

	_, _, err := svc.LoginWithGitHub(ctx, "invalid-code")
	if err == nil {
		t.Fatal("expected error for invalid code")
	}
	se, ok := service.AsServiceError(err)
	if !ok {
		t.Fatalf("expected service error, got %T", err)
	}
	if se.Code != service.ErrUnauthorized {
		t.Errorf("expected code %q, got %q", service.ErrUnauthorized, se.Code)
	}
}

func TestGetGitHubUser_InvalidData(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-token",
			"token_type":   "bearer",
		})
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    0,
			"login": "",
		})
	})

	svc := setupGitHubAuthService(t, mux)
	ctx := ctxWithGitHubTransport(mux)

	_, _, err := svc.LoginWithGitHub(ctx, "test-code")
	if err == nil {
		t.Fatal("expected error for invalid GitHub user data")
	}
}

func TestGetGitHubUserEmails_NoPrimary(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-token",
			"token_type":   "bearer",
		})
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         99999,
			"login":      "noprimuser",
			"avatar_url": "https://example.com/avatar.png",
		})
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"email": "verified@example.com", "primary": false, "verified": true},
			{"email": "unverified@example.com", "primary": false, "verified": false},
		})
	})

	svc := setupGitHubAuthService(t, mux)
	ctx := ctxWithGitHubTransport(mux)

	u, _, err := svc.LoginWithGitHub(ctx, "test-code")
	if err != nil {
		t.Fatalf("LoginWithGitHub error: %v", err)
	}
	if u.Email != "verified@example.com" {
		t.Errorf("expected email 'verified@example.com', got %q", u.Email)
	}
}

func TestGetGitHubUserEmails_FetchError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-token",
			"token_type":   "bearer",
		})
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         88888,
			"login":      "erruser",
			"avatar_url": "https://example.com/avatar.png",
		})
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, "rate limited")
	})

	svc := setupGitHubAuthService(t, mux)
	ctx := ctxWithGitHubTransport(mux)

	_, _, err := svc.LoginWithGitHub(ctx, "test-code")
	if err == nil {
		t.Fatal("expected error when GitHub emails API fails")
	}
}

func TestGetGitHubUser_FetchError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-token",
			"token_type":   "bearer",
		})
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "internal error")
	})

	svc := setupGitHubAuthService(t, mux)
	ctx := ctxWithGitHubTransport(mux)

	_, _, err := svc.LoginWithGitHub(ctx, "test-code")
	if err == nil {
		t.Fatal("expected error when GitHub user API fails")
	}
}
