// Package middleware provides HTTP middleware for gin routers.
package middleware

import (
	"markpost/internal/service"

	"github.com/gin-gonic/gin"
)

// RequireAdmin returns a middleware that requires admin role.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := ExtractUser(c)
		if !ok {
			abortWithError(c, service.New(service.ErrUnauthorized, "user not found in context"))
			return
		}

		if !u.IsAdmin() {
			abortWithError(c, service.New(service.ErrForbidden, "admin access required"))
			return
		}

		c.Next()
	}
}
