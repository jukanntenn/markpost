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
	LoginWithPassword(ctx context.Context, username, password string) (*user.User, *auth.JWTTokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*user.User, *auth.JWTTokenPair, error)
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

type PasswordLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func LoginWithPassword(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req PasswordLoginRequest
		if !bindJSON(c, &req) {
			return
		}

		u, tokens, err := authSvc.LoginWithPassword(c.Request.Context(), req.Username, req.Password)
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
		writeAuthResponse(c, u, tokens)
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
