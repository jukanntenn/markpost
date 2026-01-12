package handlers

import (
	"context"
	"net/http"

	apperrors "markpost/errors"
	"markpost/services"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type GitHubAuthURLGenerator interface {
	GenerateGitHubAuthURL(ctx context.Context) (string, error)
}

// GenerateGitHubOAuthURL godoc
// @Summary      Get GitHub OAuth URL
// @Description  Generate GitHub OAuth authorization URL
// @Tags         oauth
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /oauth/url [get]
func GenerateGitHubOAuthURL(authSvc GitHubAuthURLGenerator) gin.HandlerFunc {
	return func(c *gin.Context) {
		url, err := authSvc.GenerateGitHubAuthURL(c.Request.Context())
		if err != nil {
			apperrors.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": url})
	}
}

// LoginGitHub godoc
// @Summary      Login with GitHub
// @Description  Handle GitHub OAuth callback and authenticate user
// @Tags         oauth
// @Accept       json
// @Produce      json
// @Param        state  query    string  true  "OAuth state parameter"
// @Param        code   body     object  true  "GitHub OAuth code"
// @Success      200    {object}  AuthResponse
// @Failure      400    {object}  map[string]interface{}
// @Failure      401    {object}  map[string]interface{}
// @Failure      500    {object}  map[string]interface{}
// @Router       /oauth/login [post]
func LoginGitHub(authSvc services.AuthServiceInterface) gin.HandlerFunc {
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
		writeAuthResponse(c, user, tokens)
	}
}

type PasswordLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginWithPassword godoc
// @Summary      Login with username and password
// @Description  Authenticate user with credentials
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body  PasswordLoginRequest  true  "Login credentials"
// @Success      200  {object}  AuthResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/login [post]
func LoginWithPassword(authSvc services.AuthServiceInterface) gin.HandlerFunc {
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
		writeAuthResponse(c, user, tokens)
	}
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken godoc
// @Summary      Refresh JWT token
// @Description  Get new access token using refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body  RefreshTokenRequest  true  "Refresh token"
// @Success      200  {object}  AuthResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/refresh [post]
func RefreshToken(authSvc services.AuthServiceInterface) gin.HandlerFunc {
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
		writeAuthResponse(c, user, tokens)
	}
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword godoc
// @Summary      Change password
// @Description  Change authenticated user's password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  PasswordChangeRequest  true  "Password change request"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/change-password [post]
func ChangePassword(authSvc services.AuthServiceInterface) gin.HandlerFunc {
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

// QueryPostKey godoc
// @Summary      Get post key
// @Description  Query user's post key for creating posts
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  PostKeyResponse
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /post_key [get]
func QueryPostKey(authSvc services.AuthServiceInterface) gin.HandlerFunc {
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
