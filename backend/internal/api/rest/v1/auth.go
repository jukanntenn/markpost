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
	LoginWithEmail(ctx context.Context, username, password string) (*user.User, *auth.JWTTokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*user.User, *auth.JWTTokenPair, error)
	Logout(ctx context.Context, accessToken string) error
	ChangePassword(ctx context.Context, userID int, current, newPassword string) error
	QueryPostKey(ctx context.Context, userID int) (string, time.Time, error)
}

func writeAuthResult(c *gin.Context, u *user.User, tokens *auth.JWTTokenPair, err error) {
	if err != nil {
		apierr.RespondError(c, err)
		return
	}
	writeAuthResponse(c, u, tokens)
}

func LoginGitHub(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GitHubLoginRequest
		if !bindJSON(c, &req) {
			return
		}
		u, tokens, err := authSvc.LoginWithGitHub(c.Request.Context(), req.Code)
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

func ChangePassword(authSvc AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		withUser(c, func(u *user.User) {
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
		})
	}
}

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
