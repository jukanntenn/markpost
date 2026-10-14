package auth

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"markpost/internal/infra"
)

// recordingMetrics captures the auth counters; all three are plain increments.
type recordingMetrics struct {
	loginSuccesses atomic.Int64
	loginFailures  atomic.Int64
	tokenRefreshes atomic.Int64
}

func (r *recordingMetrics) IncLoginSuccess(context.Context) { r.loginSuccesses.Add(1) }
func (r *recordingMetrics) IncLoginFailure(context.Context) { r.loginFailures.Add(1) }
func (r *recordingMetrics) IncTokenRefresh(context.Context) { r.tokenRefreshes.Add(1) }

// setupAuthServiceWithMetrics builds a DB-backed service wired to a recording
// recorder, mirroring setupAuthService.
func setupAuthServiceWithMetrics(t *testing.T) (*Service, *recordingMetrics) {
	t.Helper()
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	tokenRepo := infra.NewTokenRepository(db)
	jwtSvc := NewJWTService("test-access-secret-key-min-32-chars!!", "test-refresh-secret-key-min-32-chars!!", time.Hour, time.Hour*24)
	met := &recordingMetrics{}
	svc := NewService(userRepo, tokenRepo, nil, jwtSvc, "markpost", "testpassword", WithMetrics(met))
	return svc, met
}

func TestLoginMetrics_EmailFlow(t *testing.T) {
	t.Run("success counts one login success", func(t *testing.T) {
		svc, met := setupAuthServiceWithMetrics(t)
		ctx := context.Background()
		_, _ = svc.users.Create(ctx, "test@example.com", "testuser", "correctpassword")

		if _, _, err := svc.LoginWithEmail(ctx, "testuser", "correctpassword"); err != nil {
			t.Fatalf("login: %v", err)
		}
		if got := met.loginSuccesses.Load(); got != 1 {
			t.Errorf("login successes = %d, want 1", got)
		}
		if got := met.loginFailures.Load(); got != 0 {
			t.Errorf("login failures = %d, want 0", got)
		}
	})

	t.Run("invalid credentials count one login failure", func(t *testing.T) {
		svc, met := setupAuthServiceWithMetrics(t)
		ctx := context.Background()
		_, _ = svc.users.Create(ctx, "test@example.com", "testuser", "correctpassword")

		if _, _, err := svc.LoginWithEmail(ctx, "testuser", "wrongpassword"); err == nil {
			t.Fatal("expected error for wrong password")
		}
		if got := met.loginFailures.Load(); got != 1 {
			t.Errorf("login failures = %d, want 1", got)
		}
		if got := met.loginSuccesses.Load(); got != 0 {
			t.Errorf("login successes = %d, want 0", got)
		}
	})

	t.Run("locked account counts as login failure, with correct password", func(t *testing.T) {
		svc, met := setupAuthServiceWithMetrics(t)
		ctx := context.Background()
		_, _ = svc.users.Create(ctx, "test@example.com", "testuser", "correctpassword")

		// Burn the 5 attempts that trip the lock (C2.1 层 B).
		for i := 0; i < 5; i++ {
			if _, _, err := svc.LoginWithEmail(ctx, "testuser", "wrongpassword"); err == nil {
				t.Fatalf("warm-up login %d unexpectedly succeeded", i)
			}
		}
		if got := met.loginFailures.Load(); got != 5 {
			t.Fatalf("after 5 wrong passwords: failures = %d, want 5", got)
		}

		// The locked rejection — even with the correct password — is a
		// failed login from the caller's perspective.
		if _, _, err := svc.LoginWithEmail(ctx, "testuser", "correctpassword"); err == nil {
			t.Fatal("expected account_locked for the post-lock attempt")
		}
		if got := met.loginFailures.Load(); got != 6 {
			t.Errorf("locked rejection must count as failure: failures = %d, want 6", got)
		}
		if got := met.loginSuccesses.Load(); got != 0 {
			t.Errorf("locked rejection must not count success: successes = %d, want 0", got)
		}
	})
}

func TestLoginMetrics_GitHubFlow(t *testing.T) {
	t.Run("success counts one login success", func(t *testing.T) {
		svc := setupGitHubAuthService(t, newGitHubMockMux())
		met := &recordingMetrics{}
		svc.metrics = met
		ctx := ctxWithGitHubTransport(newGitHubMockMux())

		if _, _, err := svc.LoginWithGitHub(ctx, "test-code", oauthStateForTest(t, svc)); err != nil {
			t.Fatalf("LoginWithGitHub: %v", err)
		}
		if got := met.loginSuccesses.Load(); got != 1 {
			t.Errorf("login successes = %d, want 1", got)
		}
		if got := met.loginFailures.Load(); got != 0 {
			t.Errorf("login failures = %d, want 0", got)
		}
	})

	t.Run("invalid state counts one login failure", func(t *testing.T) {
		svc := setupGitHubAuthService(t, newGitHubMockMux())
		met := &recordingMetrics{}
		svc.metrics = met
		ctx := ctxWithGitHubTransport(newGitHubMockMux())

		if _, _, err := svc.LoginWithGitHub(ctx, "test-code", "never-issued-state"); err == nil {
			t.Fatal("expected error for invalid state")
		}
		if got := met.loginFailures.Load(); got != 1 {
			t.Errorf("login failures = %d, want 1", got)
		}
		if got := met.loginSuccesses.Load(); got != 0 {
			t.Errorf("login successes = %d, want 0", got)
		}
	})
}

func TestRefreshMetrics(t *testing.T) {
	t.Run("completed refresh counts one token refresh", func(t *testing.T) {
		svc, met := setupAuthServiceWithMetrics(t)
		ctx := context.Background()
		_, _ = svc.users.Create(ctx, "test@example.com", "testuser", "password")
		_, pair, _ := svc.LoginWithEmail(ctx, "testuser", "password")

		// The login itself counted one success; the refresh must not.
		if got := met.loginSuccesses.Load(); got != 1 {
			t.Fatalf("pre-refresh login successes = %d, want 1", got)
		}

		if _, _, err := svc.RefreshToken(ctx, pair.RefreshToken); err != nil {
			t.Fatalf("refresh: %v", err)
		}
		if got := met.tokenRefreshes.Load(); got != 1 {
			t.Errorf("token refreshes = %d, want 1", got)
		}
		if got := met.loginSuccesses.Load(); got != 1 {
			t.Errorf("a refresh must not count as a login: successes = %d, want 1", got)
		}
	})

	t.Run("rejected refresh counts nothing", func(t *testing.T) {
		svc, met := setupAuthServiceWithMetrics(t)
		ctx := context.Background()

		if _, _, err := svc.RefreshToken(ctx, "invalid-token"); err == nil {
			t.Fatal("expected error for invalid refresh token")
		}
		if got := met.tokenRefreshes.Load(); got != 0 {
			t.Errorf("rejected refresh counted: token refreshes = %d, want 0", got)
		}
		if got := met.loginFailures.Load(); got != 0 {
			t.Errorf("rejected refresh must not count as login failure: failures = %d, want 0", got)
		}
	})
}
