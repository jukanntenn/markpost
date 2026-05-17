package middleware

import (
	"context"
	"strings"

	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/internal/service/auth"
	"markpost/pkg/utils"

	"github.com/gin-gonic/gin"
)

// TokenBlacklistChecker defines the interface for checking blacklisted tokens.
type TokenBlacklistChecker interface {
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
}

func getBearerToken(c *gin.Context) (string, bool) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}
	return authHeader[7:], true
}

func extractBearerToken(c *gin.Context) (string, bool) {
	token, ok := getBearerToken(c)
	if !ok {
		abortWithError(c, service.NewServiceError(service.ErrMissingAuthorizationHeader, "missing Authorization header"))
		return "", false
	}
	return token, true
}

func ExtractAccessToken(c *gin.Context) (string, bool) {
	t, ok := c.Get("access_token")
	if !ok {
		return "", false
	}
	return t.(string), true
}

func setUserFields(c *gin.Context, u *user.User) {
	c.Set("user", u)
	c.Set("user_id", u.ID)
	c.Set("email", u.Email)
	c.Set("username", u.Username)
	c.Set("role", string(u.Role))
}

func setUserContext(c *gin.Context, u *user.User, claims *auth.AccessClaims) {
	setUserFields(c, u)
	c.Set("claims", claims)
}

func validateBearerToken(c *gin.Context, tokenString string, jwtSvc *auth.JWTService, users user.Repository) (*user.User, *auth.AccessClaims, error) {
	claims, err := jwtSvc.ValidateAccess(tokenString)
	if err != nil {
		return nil, nil, service.NewServiceErrorWrap(service.ErrInvalidToken, "invalid token", err)
	}

	u, err := users.GetByID(c.Request.Context(), claims.UserID)
	if err != nil {
		return nil, nil, service.NewServiceErrorWrap(service.ErrInternal, "failed to get user information", err)
	}

	if !u.IsActive {
		return nil, nil, service.NewServiceError(service.ErrUserDisabled, "user account is disabled")
	}

	return u, claims, nil
}

func authenticateUser(c *gin.Context, tokenString string, jwtSvc *auth.JWTService, users user.Repository) bool {
	u, claims, err := validateBearerToken(c, tokenString, jwtSvc, users)
	if err != nil {
		abortWithError(c, err)
		return false
	}

	setUserContext(c, u, claims)
	c.Set("access_token", tokenString)
	return true
}

// Auth returns an authentication middleware.
func Auth(jwtSvc *auth.JWTService, users user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, ok := extractBearerToken(c)
		if !ok {
			return
		}

		if !authenticateUser(c, tokenString, jwtSvc, users) {
			return
		}
		c.Next()
	}
}

// AuthWithBlacklist returns an authentication middleware with token blacklist checking.
func AuthWithBlacklist(jwtSvc *auth.JWTService, users user.Repository, tokenRepo user.TokenRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, ok := extractBearerToken(c)
		if !ok {
			return
		}

		blacklisted, err := tokenRepo.IsTokenBlacklisted(c.Request.Context(), utils.HashToken(tokenString))
		if err != nil {
			abortWithError(c, service.NewServiceErrorWrap(service.ErrInternal, "failed to check token blacklist", err))
			return
		}
		if blacklisted {
			abortWithError(c, service.NewServiceError(service.ErrInvalidToken, "token has been revoked"))
			return
		}

		if !authenticateUser(c, tokenString, jwtSvc, users) {
			return
		}
		c.Next()
	}
}

// OptionalAuth returns an optional authentication middleware.
func OptionalAuth(jwtSvc *auth.JWTService, users user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, ok := getBearerToken(c)
		if !ok {
			c.Next()
			return
		}

		u, claims, err := validateBearerToken(c, tokenString, jwtSvc, users)
		if err != nil {
			c.Next()
			return
		}

		setUserContext(c, u, claims)
		c.Next()
	}
}
