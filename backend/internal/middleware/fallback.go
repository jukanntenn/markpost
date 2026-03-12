package middleware

import (
	"markpost/internal/service"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

func Fallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrInternal, "internal server error", nil))
				c.Abort()
			}
		}()
		c.Next()
	}
}
