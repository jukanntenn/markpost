package handlers

import (
	"net/http"

	apperrors "markpost/errors"
	"markpost/services"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func GenerateGitHubOAuthURL(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		url, err := authSvc.GenerateGitHubAuthURL(c.Request.Context())
		if err != nil {
			apperrors.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": url})
	}
}

func LoginGitHub(authSvc *services.AuthService) gin.HandlerFunc {
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

		user, tokens, err := authSvc.LoginWithGitHub(c.Request.Context(), body.Code)
		if err != nil {
			apperrors.RespondError(c, err)
			return
		}
		sendAuthResponse(c, user, tokens)
	}
}

type PasswordLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func LoginWithPassword(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req PasswordLoginRequest
		if !bindJSON(c, &req) {
			return
		}

		user, tokens, err := authSvc.LoginWithPassword(req.Username, req.Password)
		if err != nil {
			apperrors.RespondError(c, err)
			return
		}
		sendAuthResponse(c, user, tokens)
	}
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func RefreshToken(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RefreshTokenRequest
		if !bindJSON(c, &req) {
			return
		}

		user, tokens, err := authSvc.RefreshToken(req.RefreshToken)
		if err != nil {
			apperrors.RespondError(c, err)
			return
		}
		sendAuthResponse(c, user, tokens)
	}
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func ChangePassword(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := ExtractUser(c)
		if !ok {
			err := services.NewServiceErrorWrap(services.ErrFailedGetUser, "failed to get user from context", nil)
			apperrors.RespondError(c, err)
			return
		}

		var req PasswordChangeRequest
		if !bindJSON(c, &req) {
			return
		}

		if err := authSvc.ChangePassword(user.ID, req.CurrentPassword, req.NewPassword); err != nil {
			apperrors.RespondError(c, err)
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

func QueryPostKey(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := ExtractUser(c)
		if !ok {
			err := services.NewServiceErrorWrap(services.ErrFailedGetUser, "failed to get user from context", nil)
			apperrors.RespondError(c, err)
			return
		}

		postKey, createdAt, err := authSvc.QueryPostKey(user.ID)
		if err != nil {
			apperrors.RespondError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"post_key": postKey, "created_at": createdAt})
	}
}
