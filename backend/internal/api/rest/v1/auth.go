// Package v1 provides REST API v1 handlers.
package v1

import (
	"context"
	"net/http"
	"time"

	"markpost/internal/domain/user"
	"markpost/internal/middleware"
	"markpost/internal/service/auth"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

func newUserInfo(u user.User) UserInfo {
	return UserInfo{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
		Role:      string(u.Role),
	}
}

// AuthResponse represents an authentication response.
type AuthResponse struct {
	User         UserInfo `json:"user"`
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"`
}

// RefreshTokenResponse represents a token refresh response.
type RefreshTokenResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// UserInfo represents user information in responses.
type UserInfo struct {
	ID        int     `json:"id"`
	Email     string  `json:"email"`
	Username  string  `json:"username"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url"`
	Role      string  `json:"role"`
}

// PostKeyResponse represents a post key response.
type PostKeyResponse struct {
	PostKey   string    `json:"post_key"`
	CreatedAt time.Time `json:"created_at"`
}

// GitHubAuthURLGenerator generates GitHub OAuth authorization URLs.
type GitHubAuthURLGenerator interface {
	GenerateGitHubAuthURL(ctx context.Context) (string, error)
}

// GenerateGitHubOAuthURL returns a handler that generates GitHub OAuth URL.
func GenerateGitHubOAuthURL(authSvc GitHubAuthURLGenerator) gin.HandlerFunc {
	return func(c *gin.Context) {
		url, err := authSvc.GenerateGitHubAuthURL(c.Request.Context())
		if err != nil {
			apierr.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": url})
	}
}

// AuthService provides authentication operations.
type AuthService interface {
	LoginWithGitHub(ctx context.Context, code string) (*user.User, *auth.JWTTokenPair, error)
	LoginWithEmail(ctx context.Context, username, password string) (*user.User, *auth.JWTTokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*user.User, *auth.JWTTokenPair, error)
	Logout(ctx context.Context, accessToken string) error
	ChangePassword(ctx context.Context, userID int, current, newPassword string) error
	QueryPostKey(ctx context.Context, userID int) (string, time.Time, error)
}

// GitHubLoginRequest represents a GitHub OAuth login request.
type GitHubLoginRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// LoginGitHub returns a handler for GitHub OAuth login.
func LoginGitHub(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GitHubLoginRequest
		if !bindJSON(c, &req) {
			return
		}

		u, tokens, err := authSvc.LoginWithGitHub(c.Request.Context(), req.Code)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}
		writeAuthResponse(c, u, tokens)
	}
}

// UsernameLoginRequest represents a username login request.
type UsernameLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginWithUsername returns a handler for username/password login.
func LoginWithUsername(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UsernameLoginRequest
		if !bindJSON(c, &req) {
			return
		}

		u, tokens, err := authSvc.LoginWithEmail(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}
		writeAuthResponse(c, u, tokens)
	}
}

// RefreshTokenRequest represents a token refresh request.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken returns a handler for refreshing access tokens.
func RefreshToken(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RefreshTokenRequest
		if !bindJSON(c, &req) {
			return
		}

		u, tokens, err := authSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}
		writeRefreshResponse(c, u, tokens)
	}
}

// Logout returns a handler for user logout.
func Logout(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token, ok := middleware.ExtractAccessToken(c); ok {
			if err := authSvc.Logout(c.Request.Context(), token); err != nil {
				apierr.RespondError(c, err)
				return
			}
		}

		c.JSON(http.StatusOK, MessageResponse{
			Message: getI18nMessage(c, "Logged out successfully", "auth.logout_success"),
		})
	}
}

// PasswordChangeRequest represents a password change request.
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword returns a handler for changing user password.
func ChangePassword(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := requireUser(c)
		if !ok {
			return
		}

		var req PasswordChangeRequest
		if !bindJSON(c, &req) {
			return
		}

		if err := authSvc.ChangePassword(c.Request.Context(), u.ID, req.CurrentPassword, req.NewPassword); err != nil {
			apierr.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, MessageResponse{
			Message: getI18nMessage(c, "Password changed successfully", "error.password_changed_success"),
		})
	}
}

// QueryPostKey returns a handler for querying user's post key.
func QueryPostKey(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := requireUser(c)
		if !ok {
			return
		}

		postKey, createdAt, err := authSvc.QueryPostKey(c.Request.Context(), u.ID)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, PostKeyResponse{PostKey: postKey, CreatedAt: createdAt})
	}
}

func writeAuthResponse(c *gin.Context, u *user.User, tokens *auth.JWTTokenPair) {
	c.JSON(http.StatusOK, AuthResponse{
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresInSeconds(),
		User:         newUserInfo(*u),
	})
}

func writeRefreshResponse(c *gin.Context, _ *user.User, tokens *auth.JWTTokenPair) {
	c.JSON(http.StatusOK, RefreshTokenResponse{
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresInSeconds(),
	})
}
