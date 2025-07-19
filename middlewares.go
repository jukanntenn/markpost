package main

import (
	"database/sql"
	"net/http"

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

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		postKey := c.Param("post_key")
		user, err := GetUserByPostKey(postKey)
		if err != nil {
			switch err {
			case sql.ErrNoRows:
				c.JSON(http.StatusForbidden, gin.H{"detail": "invalid post key"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"detail": "internal error"})
			}
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Next()
	}
}
