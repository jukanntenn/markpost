package middleware

import (
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
			c.AbortWithStatusJSON(httpErr.StatusCode, gin.H{
				"error":  httpErr.Message,
				"code":   "rate_limit_exceeded",
				"detail": "Too many requests",
			})
			return
		}

		c.Next()
	}
}
