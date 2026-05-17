// Package middleware provides HTTP middleware functions for the application.
package middleware

import (
	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

func abortWithError(c *gin.Context, err error) {
	apierr.RespondError(c, err)
	c.Abort()
}

// ExtractUser extracts the authenticated user from the gin context.
func ExtractUser(c *gin.Context) (*user.User, bool) {
	u, ok := c.Get("user")
	if !ok {
		return nil, false
	}
	return u.(*user.User), true
}

// RequireAdmin returns a middleware that requires admin role.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := ExtractUser(c)
		if !ok {
			abortWithError(c, service.NewServiceError(service.ErrUnauthorized, "user not found in context"))
			return
		}

		if u.Role != user.RoleAdmin {
			abortWithError(c, service.NewServiceError(service.ErrForbidden, "admin access required"))
			return
		}

		c.Next()
	}
}
