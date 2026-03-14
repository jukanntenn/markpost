package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/internal/service/auth"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

// TokenBlacklistChecker defines the interface for checking blacklisted tokens.
type TokenBlacklistChecker interface {
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
}

// Auth returns an authentication middleware.
func Auth(jwtSvc *auth.JWTService, users user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrMissingAuthorizationHeader, "missing Authorization header", nil))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtSvc.ValidateAccess(tokenString)
		if err != nil {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrInvalidToken, "invalid token", err))
			c.Abort()
			return
		}

		u, err := users.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrInternal, "failed to get user information", err))
			c.Abort()
			return
		}

		if !u.IsActive {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrUserDisabled, "user account is disabled", nil))
			c.Abort()
			return
		}

		c.Set("user", u)
		c.Set("user_id", u.ID)
		c.Set("email", u.Email)
		c.Set("username", u.Username)
		c.Set("role", string(u.Role))
		c.Set("claims", claims)
		c.Next()
	}
}

// AuthWithBlacklist returns an authentication middleware with token blacklist checking.
func AuthWithBlacklist(jwtSvc *auth.JWTService, users user.Repository, tokenRepo user.TokenRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrMissingAuthorizationHeader, "missing Authorization header", nil))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		blacklisted, err := tokenRepo.IsTokenBlacklisted(c.Request.Context(), hashToken(tokenString))
		if err != nil {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrInternal, "failed to check token blacklist", err))
			c.Abort()
			return
		}
		if blacklisted {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrInvalidToken, "token has been revoked", nil))
			c.Abort()
			return
		}

		claims, err := jwtSvc.ValidateAccess(tokenString)
		if err != nil {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrInvalidToken, "invalid token", err))
			c.Abort()
			return
		}

		u, err := users.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrInternal, "failed to get user information", err))
			c.Abort()
			return
		}

		if !u.IsActive {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrUserDisabled, "user account is disabled", nil))
			c.Abort()
			return
		}

		c.Set("user", u)
		c.Set("user_id", u.ID)
		c.Set("email", u.Email)
		c.Set("username", u.Username)
		c.Set("role", string(u.Role))
		c.Set("claims", claims)
		c.Next()
	}
}

// OptionalAuth returns an optional authentication middleware.
func OptionalAuth(jwtSvc *auth.JWTService, users user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtSvc.ValidateAccess(tokenString)
		if err != nil {
			c.Next()
			return
		}

		u, err := users.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			c.Next()
			return
		}

		if !u.IsActive {
			c.Next()
			return
		}

		c.Set("user", u)
		c.Set("user_id", u.ID)
		c.Set("email", u.Email)
		c.Set("username", u.Username)
		c.Set("role", string(u.Role))
		c.Set("claims", claims)
		c.Next()
	}
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
