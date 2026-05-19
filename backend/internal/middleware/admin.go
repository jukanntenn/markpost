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
			abortWithError(c, service.NewServiceError(service.ErrUnauthorized, "user not found in context"))
			return
		}

		if !u.IsAdmin() {
			abortWithError(c, service.NewServiceError(service.ErrForbidden, "admin access required"))
			return
		}

		c.Next()
	}
}
