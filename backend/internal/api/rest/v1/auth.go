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

type GitHubAuthURLGenerator interface {
	GenerateGitHubAuthURL(ctx context.Context) (string, error)
}

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

type AuthService interface {
	LoginWithGitHub(ctx context.Context, code string) (*user.User, *auth.JWTTokenPair, error)
	LoginWithEmail(ctx context.Context, email, password string) (*user.User, *auth.JWTTokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*user.User, *auth.JWTTokenPair, error)
	Logout(ctx context.Context, accessToken string) error
	ChangePassword(ctx context.Context, userID int, current, new string) error
	QueryPostKey(ctx context.Context, userID int) (string, time.Time, error)
}

func LoginGitHub(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var query struct {
			State string `form:"state" binding:"required"`
		}
		if !bindQuery(c, &query) {
			return
		}
		var body struct {
			Code string `json:"code" binding:"required"`
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

type EmailLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func LoginWithEmail(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req EmailLoginRequest
		if !bindJSON(c, &req) {
			return
		}

		u, tokens, err := authSvc.LoginWithEmail(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}
		writeAuthResponse(c, u, tokens)
	}
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

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

func Logout(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token != "" && len(token) > 7 {
			token = token[7:]
			_ = authSvc.Logout(c.Request.Context(), token)
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

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

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
		"expires_in":    int64(tokens.ExpiresAt.Sub(time.Now()).Seconds()),
		"user": gin.H{
			"id":         u.ID,
			"email":      u.Email,
			"username":   u.Username,
			"name":       u.Name,
			"avatar_url": u.AvatarURL,
		},
	})
}

func writeRefreshResponse(c *gin.Context, u *user.User, tokens *auth.JWTTokenPair) {
	c.JSON(http.StatusOK, gin.H{
		"token":         tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    int64(tokens.ExpiresAt.Sub(time.Now()).Seconds()),
	})
}
