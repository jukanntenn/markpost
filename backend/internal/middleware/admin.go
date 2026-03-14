// Package middleware provides HTTP middleware functions for the application.
package middleware

import (
	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

// RequireAdmin returns a middleware that requires admin role.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := c.Get("user")
		if !ok {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrUnauthorized, "user not found in context", nil))
			c.Abort()
			return
		}

		currentUser, ok := u.(*user.User)
		if !ok {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrUnauthorized, "invalid user type in context", nil))
			c.Abort()
			return
		}

		if currentUser.Role != user.RoleAdmin {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrUnauthorized, "admin access required", nil))
			c.Abort()
			return
		}

		c.Next()
	}
}
