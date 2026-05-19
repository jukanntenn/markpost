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

// Service handles authentication operations including OAuth, JWT token management,
// and user session handling.
type Service struct {
	users  user.Repository      // User data repository
	tokens user.TokenRepository // Token storage and blacklist management
	oauth  *oauth2.Config       // OAuth2 configuration for GitHub
	jwt    *JWTService          // JWT token generation and validation
	issuer string               // Token issuer identifier
}

// NewService creates a new Service instance with the provided dependencies.
func NewService(users user.Repository, tokens user.TokenRepository, oauth *oauth2.Config, jwt *JWTService, issuer string) *Service {
	return &Service{
		users:  users,
		tokens: tokens,
		oauth:  oauth,
		jwt:    jwt,
		issuer: issuer,
	}
}

// GenerateGitHubAuthURL generates a GitHub OAuth authorization URL for user authentication.
func (s *Service) GenerateGitHubAuthURL(_ context.Context) (string, error) {
	state, err := utils.GenerateState()
	if err != nil {
		return "", service.NewServiceErrorWrap(service.ErrInternal, "failed to generate state", err)
	}

	return s.oauth.AuthCodeURL(state), nil
}

// LoginWithGitHub authenticates a user using GitHub OAuth code and returns user info with JWT tokens.
func (s *Service) LoginWithGitHub(ctx context.Context, code string) (*user.User, *JWTTokenPair, error) {
	token, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		return nil, nil, service.NewServiceErrorWrap(service.ErrUnauthorized, "oauth exchange failed", err)
	}

	githubUser, err := s.getGitHubUser(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	u, err := s.users.GetOrCreateFromGitHub(ctx, githubUser)
	if err != nil {
		return nil, nil, service.NewServiceErrorWrap(service.ErrInternal, "create user failed", err)
	}

	return s.completeLogin(ctx, u)
}

func (s *Service) getGitHubUser(ctx context.Context, token *oauth2.Token) (*user.GitHubUser, error) {
	client := s.oauth.Client(ctx, token)

	var githubUser user.GitHubUser
	if err := httputil.FetchAndDecodeJSON(client, "https://api.github.com/user", &githubUser); err != nil {
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "failed to get GitHub user", err)
	}

	if githubUser.ID == 0 || githubUser.Login == "" {
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "invalid GitHub user data", fmt.Errorf("ID=%d, Login='%s'", githubUser.ID, githubUser.Login))
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
		return "", service.NewServiceErrorWrap(service.ErrInternal, "failed to get GitHub user emails", err)
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
		return nil, nil, service.NewServiceErrorWrap(service.ErrInvalidCredentials, "invalid username or password", err)
	}

	return s.completeLogin(ctx, u)
}

// RefreshToken validates a refresh token and issues a new token pair for the user.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*user.User, *JWTTokenPair, error) {
	tokenData, err := s.validateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, nil, err
	}

	u, err := s.getUserByID(ctx, tokenData.UserID)
	if err != nil {
		return nil, nil, err
	}

	_ = s.revokeRefreshToken(ctx, refreshToken)

	return s.completeLogin(ctx, u)
}

// Logout invalidates the provided access token by adding it to the blacklist.
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

	return nil
}

func (s *Service) generateAndPersistTokenPair(ctx context.Context, u *user.User) (*JWTTokenPair, error) {
	pair, err := s.jwt.GenerateTokenPair(u.ID, u.Email, u.Username, string(u.Role))
	if err != nil {
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "generate token pair failed", err)
	}

	tokenHash := utils.HashToken(pair.RefreshToken)
	expiresAt := time.Now().Add(s.jwt.refreshTokenExpire)
	if err := s.tokens.StoreRefreshToken(ctx, u.ID, tokenHash, expiresAt); err != nil {
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "store refresh token failed", err)
	}

	return pair, nil
}

func (s *Service) completeLogin(ctx context.Context, u *user.User) (*user.User, *JWTTokenPair, error) {
	if !u.IsActive {
		return nil, nil, service.NewServiceError(service.ErrUserDisabled, "user account is disabled")
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

func (s *Service) validateRefreshToken(ctx context.Context, refreshToken string) (*user.RefreshToken, error) {
	tokenHash := utils.HashToken(refreshToken)

	tokenData, err := s.tokens.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, service.NewServiceError(service.ErrInvalidToken, "invalid refresh token")
		}
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "failed to validate refresh token", err)
	}

	if tokenData.IsExpired() {
		_ = s.revokeRefreshToken(ctx, refreshToken)
		return nil, service.NewServiceError(service.ErrInvalidToken, "refresh token expired")
	}

	return tokenData, nil
}

func (s *Service) getUserByID(ctx context.Context, userID int) (*user.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, service.WrapNotFoundOrInternal(err, fmt.Sprintf("user %d not found", userID), fmt.Sprintf("get user %d failed", userID))
	}
	return u, nil
}

func (s *Service) revokeRefreshToken(ctx context.Context, refreshToken string) error {
	tokenHash := utils.HashToken(refreshToken)
	return s.tokens.DeleteRefreshToken(ctx, tokenHash)
}

func (s *Service) verifyCurrentPassword(u *user.User, current string) error {
	if u.Password == "" {
		return nil
	}
	ok, err := utils.CheckPassword(current, u.Password)
	if err != nil {
		return service.NewServiceErrorWrap(service.ErrInternal, "validate current password failed", err)
	}
	if !ok {
		return service.NewServiceError(service.ErrInvalidPassword, "invalid current password")
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
		return service.NewServiceErrorWrap(service.ErrInternal, "set password failed", err)
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
		return service.NewServiceError(service.ErrNotFound, fmt.Sprintf("initial admin user '%s' not found", initialUsername))
	}

	if u.IsAdmin() {
		return nil
	}

	if err := s.users.SetRole(ctx, u.ID, user.RoleAdmin); err != nil {
		return service.NewServiceErrorWrap(service.ErrInternal, fmt.Sprintf("failed to promote user '%s' to admin", initialUsername), err)
	}

	return nil
}
