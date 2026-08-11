package v1

import (
	"context"
	"net/http"
	"time"

	"markpost/internal/apierr"
	"markpost/internal/domain/user"
	"markpost/internal/middleware"
	"markpost/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// GitHubAuthURLGenerator defines the interface for generating GitHub OAuth authorization URLs.
type GitHubAuthURLGenerator interface {
	GenerateGitHubAuthURL(ctx context.Context) (url, state string, err error)
}

// GenerateGitHubOAuthURL godoc
// @Summary Get GitHub OAuth authorization URL
// @Tags oauth
// @Produce json
// @Success 200 {object} OAuthURLResponse
// @Failure 500 {object} apierr.ErrorResponse
// @Router /api/v1/oauth/url [get]
func GenerateGitHubOAuthURL(authSvc GitHubAuthURLGenerator) gin.HandlerFunc {
	return func(c *gin.Context) {
		url, state, err := authSvc.GenerateGitHubAuthURL(c.Request.Context())
		if err != nil {
			apierr.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, OAuthURLResponse{URL: url, State: state})
	}
}

// AuthService defines the interface for authentication operations.
type AuthService interface {
	LoginWithGitHub(ctx context.Context, code, state string) (*user.User, *auth.JWTTokenPair, error)
	LoginWithEmail(ctx context.Context, username, password string) (*user.User, *auth.JWTTokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*user.User, *auth.JWTTokenPair, error)
	Logout(ctx context.Context, accessToken string) error
	ChangePassword(ctx context.Context, userID int, current, newPassword string) (*auth.JWTTokenPair, error)
	QueryPostKey(ctx context.Context, userID int) (string, time.Time, error)
	RotatePostKey(ctx context.Context, userID int) (string, error)
	ListSessions(ctx context.Context, userID int) ([]user.RefreshToken, error)
	RevokeSession(ctx context.Context, userID, tokenID int) error
	RevokeAllSessions(ctx context.Context, userID int) error
}

func writeAuthResult(c *gin.Context, u *user.User, tokens *auth.JWTTokenPair, err error) {
	if err != nil {
		apierr.RespondError(c, err)
		return
	}
	writeAuthResponse(c, u, tokens)
}

// LoginGitHub godoc
// @Summary Login with GitHub OAuth code
// @Tags oauth
// @Accept json
// @Produce json
// @Param body body GitHubLoginRequest true "GitHub OAuth code and state"
// @Success 200 {object} AuthResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/oauth/login [post]
func LoginGitHub(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GitHubLoginRequest
		if !bindJSON(c, &req) {
			return
		}
		u, tokens, err := authSvc.LoginWithGitHub(c.Request.Context(), req.Code, req.State)
		writeAuthResult(c, u, tokens, err)
	}
}

// LoginWithUsername godoc
// @Summary Login with username and password
// @Tags auth
// @Accept json
// @Produce json
// @Param body body UsernameLoginRequest true "Login credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/auth/login [post]
func LoginWithUsername(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UsernameLoginRequest
		if !bindJSON(c, &req) {
			return
		}
		u, tokens, err := authSvc.LoginWithEmail(c.Request.Context(), req.Username, req.Password)
		writeAuthResult(c, u, tokens, err)
	}
}

// RefreshToken godoc
// @Summary Refresh authentication token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} RefreshTokenResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func RefreshToken(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RefreshTokenRequest
		if !bindJSON(c, &req) {
			return
		}

		_, tokens, err := authSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
		if err != nil {
			apierr.RespondError(c, err)
			return
		}
		writeRefreshResponse(c, tokens)
	}
}

// Logout godoc
// @Summary Logout the current user
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} MessageResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/auth/logout [post]
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

// ChangePassword godoc
// @Summary Change the current user's password
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body PasswordChangeRequest true "Current and new password"
// @Success 200 {object} v1.ChangePasswordResponse
// @Failure 400 {object} apierr.ErrorResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/auth/change-password [post]
func ChangePassword(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
			var req PasswordChangeRequest
			if !bindJSON(c, &req) {
				return
			}

			tokens, err := authSvc.ChangePassword(c.Request.Context(), u.ID, req.CurrentPassword, req.NewPassword)
			if err != nil {
				apierr.RespondError(c, err)
				return
			}

			c.JSON(http.StatusOK, ChangePasswordResponse{
				TokenFields: tokenFieldsFromPair(tokens),
			})
		})
	}
}

// RotatePostKey godoc
// @Summary Rotate the current user's post key (C2.5)
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} v1.RotatePostKeyResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/post-key/rotate [post]
func RotatePostKey(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
			postKey, err := authSvc.RotatePostKey(c.Request.Context(), u.ID)
			if err != nil {
				apierr.RespondError(c, err)
				return
			}
			c.JSON(http.StatusOK, RotatePostKeyResponse{PostKey: postKey})
		})
	}
}

// ListSessions godoc
// @Summary List the current user's sessions (I.12)
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} v1.SessionsResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/auth/sessions [get]
func ListSessions(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
			tokens, err := authSvc.ListSessions(c.Request.Context(), u.ID)
			if err != nil {
				apierr.RespondError(c, err)
				return
			}
			c.JSON(http.StatusOK, SessionsResponse{Sessions: tokens})
		})
	}
}

// RevokeSession godoc
// @Summary Revoke one of the current user's sessions (I.12)
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Param token_id path int true "Session (refresh token) ID"
// @Success 200 {object} map[string]bool
// @Failure 401 {object} apierr.ErrorResponse
// @Failure 404 {object} apierr.ErrorResponse
// @Router /api/v1/auth/sessions/{token_id} [delete]
func RevokeSession(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
			tokenID, err := parseIDParam(c, "token_id")
			if err != nil {
				return
			}
			if err := authSvc.RevokeSession(c.Request.Context(), u.ID, tokenID); err != nil {
				apierr.RespondError(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"revoked": true})
		})
	}
}

// RevokeAllSessions godoc
// @Summary Revoke all of the current user's sessions except the present one (I.12)
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]bool
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/auth/sessions [delete]
func RevokeAllSessions(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
			if err := authSvc.RevokeAllSessions(c.Request.Context(), u.ID); err != nil {
				apierr.RespondError(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"revoked": true})
		})
	}
}

// QueryPostKey godoc
// @Summary Get the current user's post key
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} PostKeyResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/post_key [get]
func QueryPostKey(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
			postKey, createdAt, err := authSvc.QueryPostKey(c.Request.Context(), u.ID)
			if err != nil {
				apierr.RespondError(c, err)
				return
			}

			c.JSON(http.StatusOK, PostKeyResponse{PostKey: postKey, CreatedAt: createdAt})
		})
	}
}

func tokenFieldsFromPair(tokens *auth.JWTTokenPair) TokenFields {
	return TokenFields{
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresInSeconds(),
	}
}

func writeAuthResponse(c *gin.Context, u *user.User, tokens *auth.JWTTokenPair) {
	c.JSON(http.StatusOK, AuthResponse{
		User:        newUserResponse(*u),
		TokenFields: tokenFieldsFromPair(tokens),
	})
}

func writeRefreshResponse(c *gin.Context, tokens *auth.JWTTokenPair) {
	c.JSON(http.StatusOK, RefreshTokenResponse{
		TokenFields: tokenFieldsFromPair(tokens),
	})
}
