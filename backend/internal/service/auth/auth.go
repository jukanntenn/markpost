// Package auth provides authentication services including OAuth, JWT token management,
// and user session handling.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"markpost/internal/domain"
	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/pkg/httputil"
	"markpost/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// oauthStateEntry bundles the PKCE verifier and creation time stored against
// an OAuth state value. See auth.md §3.4-3.5.
type oauthStateEntry struct {
	Verifier  string
	CreatedAt time.Time
}

// OAuthStateStore is the port for storing OAuth state→verifier entries with a
// TTL and one-time consumption. Backed by ristretto in the composition root.
type OAuthStateStore interface {
	// Save stores the state→entry mapping with the given TTL.
	Save(state string, entry oauthStateEntry, ttl time.Duration)
	// Consume fetches and deletes the entry for state (one-time use). The
	// boolean is false when state is absent/expired.
	Consume(state string) (oauthStateEntry, bool)
}

// noopOAuthStateStore is the zero-dependency default used when no store is
// wired (e.g. tests that don't exercise the OAuth flow).
type noopOAuthStateStore struct{}

func (noopOAuthStateStore) Save(string, oauthStateEntry, time.Duration) {}
func (noopOAuthStateStore) Consume(string) (oauthStateEntry, bool) {
	return oauthStateEntry{}, false
}

// VIPStrategyChecker is the read port for the GitHub-login VIP grant strategy
// switch (MRFC 2026-08-23-github-login-vip-grant-strategy). Backed by the
// settings repository in the composition root.
type VIPStrategyChecker interface {
	VIPStrategyEnabled(ctx context.Context) (bool, error)
}

// Service handles authentication operations including OAuth, JWT token management,
// and user session handling.
type Service struct {
	users           user.Repository      // User data repository
	tokens          user.TokenRepository // Token storage and blacklist management
	oauth           *oauth2.Config       // OAuth2 configuration for GitHub
	jwt             *JWTService          // JWT token generation and validation
	issuer          string               // Token issuer identifier
	stateStore      OAuthStateStore      // OAuth state→verifier store (PKCE + CSRF)
	vipStrategy     VIPStrategyChecker   // GitHub-login VIP grant switch (nil = never grant)
	userURL         string               // Override for GitHub user API URL (for testing)
	initialPassword string               // Password for the initial admin user (created on first startup)
	attempts        *LoginAttemptTracker // C2.1 层 B：账号级登录失败锁定
}

// NewService creates a new Service instance with the provided dependencies.
func NewService(users user.Repository, tokens user.TokenRepository, oauth *oauth2.Config, jwt *JWTService, issuer string, initialPassword string) *Service {
	return &Service{
		users:           users,
		tokens:          tokens,
		oauth:           oauth,
		jwt:             jwt,
		issuer:          issuer,
		stateStore:      noopOAuthStateStore{},
		initialPassword: initialPassword,
		attempts:        NewLoginAttemptTracker(),
	}
}

// WithOAuthStateStore sets the OAuth state store (ristretto-backed in the
// composition root). Returns the service for chaining.
func (s *Service) WithOAuthStateStore(store OAuthStateStore) *Service {
	if store != nil {
		s.stateStore = store
	}
	return s
}

// WithVIPStrategy wires the GitHub-login VIP grant switch (settings-repository
// backed in the composition root). Nil keeps granting disabled.
func (s *Service) WithVIPStrategy(checker VIPStrategyChecker) *Service {
	if checker != nil {
		s.vipStrategy = checker
	}
	return s
}

// WithUserURL overrides the GitHub user API base URL. When set, getGitHubUser
// and getGitHubUserEmails use this URL instead of api.github.com. Intended
// for E2E testing with a mock OAuth server.
func (s *Service) WithUserURL(url string) *Service {
	s.userURL = strings.TrimRight(url, "/")
	return s
}

// oauthStateTTL is how long a state→verifier entry remains consumable. The
// 10-minute window covers the user completing GitHub authorization. auth.md §3.5.
const oauthStateTTL = 10 * time.Minute

// GenerateGitHubAuthURL generates a GitHub OAuth authorization URL with a PKCE
// code challenge and stores the state→verifier entry for one-time consumption
// on callback. Returns (url, state). See auth.md §3.2.
func (s *Service) GenerateGitHubAuthURL(_ context.Context) (url, state string, err error) {
	state, err = utils.GenerateState()
	if err != nil {
		return "", "", service.Wrap(service.ErrInternal, "failed to generate state", err)
	}

	verifier := oauth2.GenerateVerifier()
	authURL := s.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	s.stateStore.Save(state, oauthStateEntry{Verifier: verifier, CreatedAt: time.Now()}, oauthStateTTL)
	return authURL, state, nil
}

// LoginWithGitHub completes the OAuth flow: validates the state (one-time
// consumption from the state store), exchanges the code with the PKCE verifier,
// fetches the GitHub user, and issues a token pair. See auth.md §3.2 step ⑦.
// A missing/mismatched/expired/already-consumed state returns ErrInvalidState.
func (s *Service) LoginWithGitHub(ctx context.Context, code, state string) (*user.User, *JWTTokenPair, error) {
	if state == "" {
		return nil, nil, service.New(ErrMissingState, "state is required")
	}
	if code == "" {
		return nil, nil, service.New(ErrMissingCode, "code is required")
	}

	entry, ok := s.stateStore.Consume(state)
	if !ok {
		return nil, nil, service.New(ErrInvalidState, "state is invalid, expired, or already used")
	}

	token, err := s.oauth.Exchange(ctx, code, oauth2.VerifierOption(entry.Verifier))
	if err != nil {
		return nil, nil, service.Wrap(ErrOAuthExchangeFailed, "oauth exchange failed", err)
	}

	githubUser, err := s.getGitHubUser(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	u, err := s.users.GetOrCreateFromGitHub(ctx, githubUser)
	if err != nil {
		return nil, nil, service.Wrap(service.ErrInternal, "create user failed", err)
	}

	s.grantVIPForGitHubLogin(ctx, u)

	return s.completeLogin(ctx, u)
}

// grantVIPForGitHubLogin applies the VIP grant strategy: while enabled, any
// GitHub login idempotently sets vip=true; while disabled (or unwired), logins
// leave vip untouched in both directions (MRFC
// 2026-08-23-github-login-vip-grant-strategy). A strategy-read failure fails
// toward not-granting — a wrongly granted vip is harder to walk back than a
// missed one — and never blocks the login itself.
func (s *Service) grantVIPForGitHubLogin(ctx context.Context, u *user.User) {
	if s.vipStrategy == nil {
		return
	}
	enabled, err := s.vipStrategy.VIPStrategyEnabled(ctx)
	if err != nil {
		slog.Error("vip strategy read failed; skipping grant", "error", err, "user_id", u.ID)
		return
	}
	if !enabled || u.VIP {
		return
	}
	if err := s.users.SetUserVIP(ctx, u.ID, true); err != nil {
		slog.Error("vip grant write failed", "error", err, "user_id", u.ID)
		return
	}
	u.VIP = true
}

func (s *Service) getGitHubUser(ctx context.Context, token *oauth2.Token) (*user.GitHubUser, error) {
	client := s.oauth.Client(ctx, token)

	userURL := "https://api.github.com/user"
	if s.userURL != "" {
		userURL = s.userURL + "/user"
	}

	var githubUser user.GitHubUser
	if err := httputil.FetchAndDecodeJSON(ctx, client, userURL, &githubUser); err != nil {
		return nil, service.Wrap(ErrGitHubUserFetch, "failed to get GitHub user", err)
	}

	if githubUser.ID == 0 || githubUser.Login == "" {
		return nil, service.Wrap(ErrGitHubUserFetch, "invalid GitHub user data", fmt.Errorf("ID=%d, Login='%s'", githubUser.ID, githubUser.Login))
	}

	email, err := s.getGitHubUserEmails(ctx, client)
	if err != nil {
		return nil, err
	}
	if email != "" {
		githubUser.Email = email
	}

	return &githubUser, nil
}

func (s *Service) getGitHubUserEmails(ctx context.Context, client *http.Client) (string, error) {
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	emailsURL := "https://api.github.com/user/emails"
	if s.userURL != "" {
		emailsURL = s.userURL + "/user/emails"
	}

	if err := httputil.FetchAndDecodeJSON(ctx, client, emailsURL, &emails); err != nil {
		return "", service.Wrap(ErrGitHubUserFetch, "failed to get GitHub user emails", err)
	}

	var primary string
	var fallback string
	for _, e := range emails {
		if e.Verified {
			if fallback == "" {
				fallback = e.Email
			}
			if e.Primary {
				primary = e.Email
			}
		}
	}
	if primary != "" {
		return primary, nil
	}
	return fallback, nil
}

// accountLockedError builds the account_locked service error carrying the
// remaining lock time: Minutes feeds the i18n template, RetryAfter feeds the
// handler's Retry-After header (Q12 一致性裁决).
func accountLockedError(remaining time.Duration) error {
	seconds := int(math.Ceil(remaining.Seconds()))
	minutes := int(math.Ceil(remaining.Minutes()))
	return service.WithData(ErrAccountLocked, "account is locked", map[string]any{
		"Minutes":    minutes,
		"RetryAfter": seconds,
	})
}

// LoginWithEmail authenticates a user with email and password, returning user info with JWT tokens.
func (s *Service) LoginWithEmail(ctx context.Context, username, password string) (*user.User, *JWTTokenPair, error) {
	// C2.1 层 B：锁定期间（含正确密码）一律返回 account_locked + 剩余时间。
	if locked, remaining := s.attempts.CheckLocked(username); locked {
		return nil, nil, accountLockedError(remaining)
	}

	u, err := s.users.ValidatePassword(ctx, username, password)
	if err != nil {
		// C2.1：记录失败；第 5 次失败触发锁定，本次即返回 account_locked。
		if locked, remaining := s.attempts.RecordFailure(username); locked {
			return nil, nil, accountLockedError(remaining)
		}
		return nil, nil, service.Wrap(ErrInvalidCredentials, "invalid username or password", err)
	}

	s.attempts.Reset(username)
	return s.completeLogin(ctx, u)
}

// RefreshToken validates a refresh token, performs one-time rotation with
// token-theft reuse detection (auth.md §2), and issues a new token pair.
//
// A resubmitted already-revoked token is treated as theft: every refresh
// token for its user is revoked and the request is rejected. A valid (active)
// token is revoked and a fresh pair is issued.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*user.User, *JWTTokenPair, error) {
	tokenHash := utils.HashToken(refreshToken)

	tokenData, err := s.tokens.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// Not active — check whether it was revoked. A revoked token being
			// resubmitted is the reuse/theft signal: revoke everything for the
			// user and reject. We can only detect theft if we can still tie the
			// hash to a user; IsRefreshTokenRevoked reads the row to get it.
			if revoked, rErr := s.tokens.IsRefreshTokenRevoked(ctx, tokenHash); rErr == nil && revoked {
				if rt, gErr := s.tokens.GetRevokedRefreshToken(ctx, tokenHash); gErr == nil {
					_ = s.tokens.RevokeAllByUserID(ctx, rt.UserID)
				}
				slog.Warn("refresh token reuse detected", "token_hash", tokenHash)
				return nil, nil, service.New(ErrInvalidToken, "refresh token reuse detected")
			}
			return nil, nil, service.New(ErrInvalidToken, "invalid refresh token")
		}
		return nil, nil, service.Wrap(service.ErrInternal, "failed to validate refresh token", err)
	}

	if tokenData.IsExpired() {
		_ = s.tokens.RevokeRefreshToken(ctx, tokenHash)
		return nil, nil, service.New(ErrInvalidToken, "refresh token expired")
	}

	u, err := s.getUserByID(ctx, tokenData.UserID)
	if err != nil {
		return nil, nil, err
	}

	// C2.6/K.4：refresh token 的 tv 必须等于用户当前 token_version，否则
	// 视为已失效（改密/强制下线/封禁后旧 refresh 不可再换新 access）。
	if claims, vErr := s.jwt.ValidateRefresh(refreshToken); vErr == nil && claims.TokenVersion != u.TokenVersion {
		_ = s.tokens.RevokeRefreshToken(ctx, tokenHash)
		return nil, nil, service.New(ErrInvalidToken, "refresh token token_version mismatch")
	}

	// One-time rotation: revoke the consumed token, then issue a fresh pair.
	if err := s.tokens.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return nil, nil, service.Wrap(service.ErrInternal, "failed to revoke refresh token", err)
	}

	return s.completeLogin(ctx, u)
}

// Logout invalidates the provided access token by adding it to the blacklist
// AND revokes every active refresh token for the user (auth.md §5). Revoking
// refresh tokens on logout prevents an attacker from using a residual refresh
// token to regain access after the access token expires.
func (s *Service) Logout(ctx context.Context, accessToken string) error {
	if accessToken == "" {
		slog.Debug("logout called with empty token")
		return nil
	}

	claims, err := s.jwt.ValidateAccess(accessToken)
	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		slog.Debug("logout token validation failed", "error", err)
		return nil
	}

	var ttl time.Duration
	if claims != nil && claims.ExpiresAt != nil {
		ttl = time.Until(claims.ExpiresAt.Time)
		if ttl <= 0 {
			slog.Debug("logout token already expired")
			return nil
		}
	} else {
		ttl = 24 * time.Hour
	}

	tokenHash := utils.HashToken(accessToken)
	expiresAt := time.Now().Add(ttl)
	if err := s.tokens.StoreBlacklistedToken(ctx, tokenHash, expiresAt); err != nil {
		return err
	}

	// Revoke all of the user's refresh tokens. Even on an expired access token
	// the claims still carry the user id (golang-jwt returns claims on
	// ErrTokenExpired), so we can revoke.
	if claims != nil {
		if err := s.tokens.RevokeAllByUserID(ctx, claims.UserID); err != nil {
			slog.Error("logout: revoke refresh tokens failed", "error", err, "user_id", claims.UserID)
		}
	}

	return nil
}

func (s *Service) generateAndPersistTokenPair(ctx context.Context, u *user.User) (*JWTTokenPair, error) {
	pair, err := s.jwt.GenerateTokenPair(u.ID, u.Email, u.Username, string(u.Role), u.TokenVersion)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "generate token pair failed", err)
	}

	tokenHash := utils.HashToken(pair.RefreshToken)
	expiresAt := time.Now().Add(s.jwt.refreshTokenExpire)
	if err := s.tokens.StoreRefreshToken(ctx, u.ID, tokenHash, expiresAt); err != nil {
		return nil, service.Wrap(service.ErrInternal, "store refresh token failed", err)
	}

	return pair, nil
}

func (s *Service) completeLogin(ctx context.Context, u *user.User) (*user.User, *JWTTokenPair, error) {
	if !u.IsActive {
		return nil, nil, service.New(ErrUserDisabled, "user account is disabled")
	}

	pair, err := s.generateAndPersistTokenPair(ctx, u)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	if err := s.users.UpdateLastLoginAt(ctx, u.ID, now); err != nil {
		slog.Error("update last login at failed", "error", err, "user_id", u.ID)
	}

	return u, pair, nil
}

func (s *Service) getUserByID(ctx context.Context, userID int) (*user.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, service.WrapNotFoundOrInternal(err, fmt.Sprintf("user %d not found", userID), fmt.Sprintf("get user %d failed", userID))
	}
	return u, nil
}

func (s *Service) verifyCurrentPassword(u *user.User, current string) error {
	if u.Password == "" {
		return nil
	}
	ok, err := utils.CheckPassword(current, u.Password)
	if err != nil {
		return service.Wrap(service.ErrInternal, "validate current password failed", err)
	}
	if !ok {
		return service.New(ErrInvalidPassword, "invalid current password")
	}
	return nil
}

// ChangePassword updates a user's password after validating the current
// password. On success it enforces C2.2: revoke all refresh tokens + bump
// token_version (all existing access tokens die instantly) + return a fresh
// token pair so the client continues seamlessly without re-authentication.
func (s *Service) ChangePassword(ctx context.Context, userID int, current, newPassword string) (*JWTTokenPair, error) {
	if err := utils.ValidatePasswordPolicy(newPassword); err != nil {
		if utils.TooShort(newPassword) {
			return nil, service.New(ErrPasswordTooShort, "password too short")
		}
		return nil, service.New(ErrPasswordTooLong, "password too long")
	}

	u, err := s.getUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := s.verifyCurrentPassword(u, current); err != nil {
		return nil, err
	}

	if err := s.users.SetPassword(ctx, userID, newPassword); err != nil {
		return nil, service.Wrap(service.ErrInternal, "set password failed", err)
	}

	// C2.6 统一失效原语：改密成功 → token_version++；refresh tokens 一并吊销。
	if err := s.users.BumpTokenVersion(ctx, userID); err != nil {
		return nil, service.Wrap(service.ErrInternal, "bump token version failed", err)
	}
	if err := s.tokens.RevokeAllByUserID(ctx, userID); err != nil {
		return nil, service.Wrap(service.ErrInternal, "revoke refresh tokens failed", err)
	}

	// 重新读取用户以拿到最新 token_version，签发新 token 对。
	u, err = s.getUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.generateAndPersistTokenPair(ctx, u)
}

// QueryPostKey retrieves the post key and creation time for a user.
func (s *Service) QueryPostKey(ctx context.Context, userID int) (string, time.Time, error) {
	u, err := s.getUserByID(ctx, userID)
	if err != nil {
		return "", time.Time{}, err
	}
	return u.PostKey, u.CreatedAt, nil
}

// RotatePostKey generates and persists a fresh post key for the user (C2.5).
// The old key stops resolving immediately (GetByPostKey finds nothing). Banned
// users are rejected earlier by the auth middleware's is_active check.
func (s *Service) RotatePostKey(ctx context.Context, userID int) (string, error) {
	key, err := s.users.RotatePostKey(ctx, userID)
	if err != nil {
		return "", service.Wrap(service.ErrInternal, "rotate post key failed", err)
	}
	return key, nil
}

// ListSessions returns the user's own refresh tokens (I.12 用户透明).
func (s *Service) ListSessions(ctx context.Context, userID int) ([]user.RefreshToken, error) {
	tokens, err := s.tokens.ListByUserID(ctx, userID)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "list sessions failed", err)
	}
	// I.10 契约：空列表序列化为 [] 而非 null。
	if tokens == nil {
		tokens = []user.RefreshToken{}
	}
	return tokens, nil
}

// RevokeSession revokes a single refresh token belonging to the user
// (I.12 用户吊销单条会话).
func (s *Service) RevokeSession(ctx context.Context, userID, tokenID int) error {
	if err := s.tokens.RevokeRefreshTokenByID(ctx, tokenID, userID); err != nil {
		return service.WrapNotFoundOrInternal(err, "session not found", "revoke session failed")
	}
	return nil
}

// RevokeAllSessions revokes all of the user's refresh tokens while keeping the
// current access token alive (I.12 "全部下线"; access tokens are independent
// of refresh tokens — the client keeps working until token expiry, then must
// sign in again).
func (s *Service) RevokeAllSessions(ctx context.Context, userID int) error {
	if err := s.tokens.RevokeAllByUserID(ctx, userID); err != nil {
		return service.Wrap(service.ErrInternal, "revoke all sessions failed", err)
	}
	return nil
}

// InitializeFirstAdmin creates (if absent) and promotes the specified user to admin role.
func (s *Service) InitializeFirstAdmin(ctx context.Context, initialUsername string) error {
	// C2.3 统一预检：初始管理员密码同样遵守密码策略。
	if err := utils.ValidatePasswordPolicy(s.initialPassword); err != nil {
		return fmt.Errorf("initial admin password violates policy: %w", err)
	}

	u, err := s.users.GetByUsername(ctx, initialUsername)
	if err != nil {
		created, cerr := s.users.Create(ctx, initialUsername+"@localhost", initialUsername, s.initialPassword)
		if cerr != nil {
			return service.Wrap(service.ErrInternal, fmt.Sprintf("create initial admin '%s'", initialUsername), cerr)
		}
		u = created
	}

	if u.IsAdmin() {
		return nil
	}

	if err := s.users.SetRole(ctx, u.ID, user.RoleAdmin); err != nil {
		return service.Wrap(service.ErrInternal, fmt.Sprintf("promote '%s' to admin", initialUsername), err)
	}

	return nil
}
