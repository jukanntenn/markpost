package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"markpost/internal/service"
)

// Fallback returns a middleware that recovers from panics. The recovered panic
// is logged via the context-aware slog default logger so the record carries the
// request's trace_id/span_id (observability.md §trace↔log 关联).
func Fallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(c.Request.Context(), "panic recovered",
					"error", r, "method", c.Request.Method, "path", c.Request.URL.Path)
				abortWithError(c, service.New(service.ErrInternal, "internal server error"))
			}
		}()
		c.Next()
	}
}
