// Package v1 provides REST API v1 handlers.
package v1

import (
	"context"
	"net/http"
	"time"

	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/internal/service/auth"
	"markpost/pkg/apierr"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

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

// LoginGitHub returns a handler for GitHub OAuth login.
func LoginGitHub(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Code  string `json:"code" binding:"required"`
			State string `json:"state" binding:"required"`
		}
		if !bindJSON(c, &body) {
			return
		}

		u, tokens, err := authSvc.LoginWithGitHub(c.Request.Context(), body.Code)
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
		token := c.GetHeader("Authorization")
		if token != "" && len(token) > 7 {
			token = token[7:]
			if err := authSvc.Logout(c.Request.Context(), token); err != nil {
				apierr.RespondError(c, err)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "auth.logout_success",
					Other: "Logged out successfully",
				},
			}),
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
		u, ok := ExtractUser(c)
		if !ok {
			err := service.NewServiceErrorWrap(service.ErrFailedGetUser, "failed to get user from context", nil)
			apierr.RespondError(c, err)
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

		c.JSON(http.StatusOK, gin.H{
			"message": ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "error.password_changed_success",
					Other: "Password changed successfully",
				},
			}),
		})
	}
}

// QueryPostKey returns a handler for querying user's post key.
func QueryPostKey(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := ExtractUser(c)
		if !ok {
			err := service.NewServiceErrorWrap(service.ErrFailedGetUser, "failed to get user from context", nil)
			apierr.RespondError(c, err)
			return
		}

		postKey, createdAt, err := authSvc.QueryPostKey(c.Request.Context(), u.ID)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"post_key": postKey, "created_at": createdAt})
	}
}

func writeAuthResponse(c *gin.Context, u *user.User, tokens *auth.JWTTokenPair) {
	c.JSON(http.StatusOK, gin.H{
		"token":         tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    int64(time.Until(tokens.ExpiresAt).Seconds()),
		"user": gin.H{
			"id":         u.ID,
			"email":      u.Email,
			"username":   u.Username,
			"name":       u.Name,
			"avatar_url": u.AvatarURL,
			"role":       u.Role,
		},
	})
}

func writeRefreshResponse(c *gin.Context, _ *user.User, tokens *auth.JWTTokenPair) {
	c.JSON(http.StatusOK, gin.H{
		"token":         tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    int64(time.Until(tokens.ExpiresAt).Seconds()),
	})
}
