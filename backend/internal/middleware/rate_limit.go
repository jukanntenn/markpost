package middleware

import (
	"markpost/internal/service"

	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/gin-gonic/gin"
)

// RateLimitByIP returns a rate limiting middleware by IP address.
func RateLimitByIP(lmt *limiter.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}

		if httpErr := tollbooth.LimitByKeys(lmt, []string{ip}); httpErr != nil {
			abortWithError(c, service.NewServiceError(service.ErrRateLimited, "rate limit exceeded"))
			return
		}

		c.Next()
	}
}
