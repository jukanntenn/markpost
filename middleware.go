package main

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// LimiterMiddleware 创建限流中间件
func LimiterMiddleware(limit rate.Limit, burst int) gin.HandlerFunc {
	limiter := rate.NewLimiter(limit, burst)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(429, gin.H{"error": "Too many requests"})
			c.Abort()
			return
		}
		c.Next()
	}
} 