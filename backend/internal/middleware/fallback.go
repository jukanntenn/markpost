package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"markpost/internal/service"
)

// Fallback returns a middleware that recovers from panics.
func Fallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered", "error", r, "path", c.Request.URL.Path)
				abortWithError(c, service.New(service.ErrInternal, "internal server error"))
			}
		}()
		c.Next()
	}
}
