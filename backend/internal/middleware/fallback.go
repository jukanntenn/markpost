package middleware

import (
	"log"
	"markpost/internal/service"

	"github.com/gin-gonic/gin"
)

// Fallback returns a middleware that recovers from panics.
func Fallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic recovered: %v", r)
				abortWithError(c, service.NewServiceError(service.ErrInternal, "internal server error"))
			}
		}()
		c.Next()
	}
}
