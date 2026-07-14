// Package auth provides authentication services including OAuth, JWT token management,
// and user session handling.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

// Service handles authentication operations including OAuth, JWT token management,
// and user session handling.
type Service struct {
	users     user.Repository      // User data repository
	tokens    user.TokenRepository // Token storage and blacklist management
	oauth     *oauth2.Config       // OAuth2 configuration for GitHub
	jwt       *JWTService          // JWT token generation and validation
	issuer    string               // Token issuer identifier
	stateStore OAuthStateStore    // OAuth state→verifier store (PKCE + CSRF)
}

// NewService creates a new Service instance with the provided dependencies.
func NewService(users user.Repository, tokens user.TokenRepository, oauth *oauth2.Config, jwt *JWTService, issuer string) *Service {
	return &Service{
		users:      users,
		tokens:     tokens,
		oauth:      oauth,
		jwt:        jwt,
		issuer:     issuer,
		stateStore: noopOAuthStateStore{},
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

	return s.completeLogin(ctx, u)
}

func (s *Service) getGitHubUser(ctx context.Context, token *oauth2.Token) (*user.GitHubUser, error) {
	client := s.oauth.Client(ctx, token)

	var githubUser user.GitHubUser
	if err := httputil.FetchAndDecodeJSON(client, "https://api.github.com/user", &githubUser); err != nil {
		return nil, service.Wrap(ErrGitHubUserFetch, "failed to get GitHub user", err)
	}

	if githubUser.ID == 0 || githubUser.Login == "" {
		return nil, service.Wrap(ErrGitHubUserFetch, "invalid GitHub user data", fmt.Errorf("ID=%d, Login='%s'", githubUser.ID, githubUser.Login))
	}

	email, err := s.getGitHubUserEmails(client)
	if err != nil {
		return nil, err
	}
	if email != "" {
		githubUser.Email = email
	}

	return &githubUser, nil
}

func (s *Service) getGitHubUserEmails(client *http.Client) (string, error) {
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := httputil.FetchAndDecodeJSON(client, "https://api.github.com/user/emails", &emails); err != nil {
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

// LoginWithEmail authenticates a user with email and password, returning user info with JWT tokens.
func (s *Service) LoginWithEmail(ctx context.Context, username, password string) (*user.User, *JWTTokenPair, error) {
	u, err := s.users.ValidatePassword(ctx, username, password)
	if err != nil {
		return nil, nil, service.Wrap(ErrInvalidCredentials, "invalid username or password", err)
	}

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
	pair, err := s.jwt.GenerateTokenPair(u.ID, u.Email, u.Username, string(u.Role))
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

// ChangePassword updates a user's password after validating the current password.
func (s *Service) ChangePassword(ctx context.Context, userID int, current, newPassword string) error {
	u, err := s.getUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.verifyCurrentPassword(u, current); err != nil {
		return err
	}

	if err := s.users.SetPassword(ctx, userID, newPassword); err != nil {
		return service.Wrap(service.ErrInternal, "set password failed", err)
	}

	return nil
}

// QueryPostKey retrieves the post key and creation time for a user.
func (s *Service) QueryPostKey(ctx context.Context, userID int) (string, time.Time, error) {
	u, err := s.getUserByID(ctx, userID)
	if err != nil {
		return "", time.Time{}, err
	}
	return u.PostKey, u.CreatedAt, nil
}

// InitializeFirstAdmin promotes the specified user to admin role if not already an admin.
func (s *Service) InitializeFirstAdmin(ctx context.Context, initialUsername string) error {
	u, err := s.users.GetByUsername(ctx, initialUsername)
	if err != nil {
		return service.New(service.ErrNotFound, fmt.Sprintf("initial admin user '%s' not found", initialUsername))
	}

	if u.IsAdmin() {
		return nil
	}

	if err := s.users.SetRole(ctx, u.ID, user.RoleAdmin); err != nil {
		return service.Wrap(service.ErrInternal, fmt.Sprintf("failed to promote user '%s' to admin", initialUsername), err)
	}

	return nil
}
