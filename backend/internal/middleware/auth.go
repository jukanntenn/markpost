package middleware

import (
	"strings"

	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/internal/service/auth"
	"markpost/pkg/utils"

	"github.com/gin-gonic/gin"
)

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
		abortWithError(c, service.New(service.ErrUnauthorized, "missing Authorization header"))
		return "", false
	}
	return token, true
}

// ExtractAccessToken retrieves the access token from the gin context.
func ExtractAccessToken(c *gin.Context) (string, bool) {
	t, ok := c.Get("access_token")
	if !ok {
		return "", false
	}
	s, ok := t.(string)
	if !ok {
		return "", false
	}
	return s, true
}

func setUserFields(c *gin.Context, u *user.User) {
	c.Set("user", u)
	c.Set("user_id", u.ID)
	c.Set("email", u.Email)
	c.Set("username", u.Username)
	c.Set("role", string(u.Role))
}

func validateBearerToken(c *gin.Context, tokenString string, jwtSvc *auth.JWTService, users user.Repository) (*user.User, *auth.AccessClaims, error) {
	claims, err := jwtSvc.ValidateAccess(tokenString)
	if err != nil {
		return nil, nil, service.Wrap(auth.ErrInvalidToken, "invalid token", err)
	}

	u, err := users.GetByID(c.Request.Context(), claims.UserID)
	if err != nil {
		return nil, nil, service.WrapNotFoundOrInternal(err, "user not found", "failed to get user information")
	}

	if !u.IsActive {
		return nil, nil, service.New(auth.ErrUserDisabled, "user account is disabled")
	}

	return u, claims, nil
}

func tryAuthenticate(c *gin.Context, tokenString string, jwtSvc *auth.JWTService, users user.Repository) (*user.User, *auth.AccessClaims, error) {
	u, claims, err := validateBearerToken(c, tokenString, jwtSvc, users)
	if err != nil {
		return nil, nil, err
	}

	setUserFields(c, u)
	c.Set("claims", claims)
	c.Set("access_token", tokenString)
	return u, claims, nil
}

func requireAuth(jwtSvc *auth.JWTService, users user.Repository, tokenRepo user.TokenRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, ok := extractBearerToken(c)
		if !ok {
			return
		}

		if tokenRepo != nil {
			blacklisted, err := tokenRepo.IsTokenBlacklisted(c.Request.Context(), utils.HashToken(tokenString))
			if err != nil {
				abortWithError(c, service.Wrap(service.ErrInternal, "failed to check token blacklist", err))
				return
			}
			if blacklisted {
				abortWithError(c, service.New(auth.ErrInvalidToken, "token has been revoked"))
				return
			}
		}

		if _, _, err := tryAuthenticate(c, tokenString, jwtSvc, users); err != nil {
			abortWithError(c, err)
			return
		}
		c.Next()
	}
}

// Auth returns a middleware that requires a valid JWT access token.
func Auth(jwtSvc *auth.JWTService, users user.Repository) gin.HandlerFunc {
	return requireAuth(jwtSvc, users, nil)
}

// AuthWithBlacklist returns a middleware that requires a valid JWT access token and checks against the token blacklist.
func AuthWithBlacklist(jwtSvc *auth.JWTService, users user.Repository, tokenRepo user.TokenRepository) gin.HandlerFunc {
	return requireAuth(jwtSvc, users, tokenRepo)
}

// OptionalAuth returns an optional authentication middleware.
func OptionalAuth(jwtSvc *auth.JWTService, users user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, ok := getBearerToken(c)
		if !ok {
			c.Next()
			return
		}

		_, _, _ = tryAuthenticate(c, tokenString, jwtSvc, users)
		c.Next()
	}
}
