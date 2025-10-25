package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	ginlimiter "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func PostKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		postKey := c.Param("post_key")
		user, err := database.GetUserRepository().GetUserByPostKey(postKey)
		if err != nil {
			switch err {
			case sql.ErrNoRows:
				c.JSON(http.StatusForbidden, gin.H{"code": "invalid_post_key", "message": "Invalid post key"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": "Internal server error"})
			}
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "missing_authorization_header", "message": "Missing Authorization header"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := validateJWTToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "invalid_token", "message": "Invalid token"})
			c.Abort()
			return
		}

		user, err := database.GetUserRepository().GetUserByID(claims.UserID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": ErrNotFound, "message": "User not found"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

type RateLimitConfig struct {
	IPPerMinute      int
	IPPerDay         int
	PostKeyPerMinute int
	PostKeyPerDay    int
}

func initRateLimiters(config RateLimitConfig) ([]gin.HandlerFunc, error) {
	store := memory.NewStore()

	ipMinuteRate := limiter.Rate{Period: time.Minute, Limit: int64(config.IPPerMinute)}
	ipDayRate := limiter.Rate{Period: 24 * time.Hour, Limit: int64(config.IPPerDay)}
	postKeyMinuteRate := limiter.Rate{Period: time.Minute, Limit: int64(config.PostKeyPerMinute)}
	postKeyDayRate := limiter.Rate{Period: 24 * time.Hour, Limit: int64(config.PostKeyPerDay)}

	ipMinuteLimiter := limiter.New(store, ipMinuteRate)
	ipDayLimiter := limiter.New(store, ipDayRate)
	postKeyMinuteLimiter := limiter.New(store, postKeyMinuteRate)
	postKeyDayLimiter := limiter.New(store, postKeyDayRate)

	middlewares := []gin.HandlerFunc{
		ginlimiter.NewMiddleware(ipMinuteLimiter,
			ginlimiter.WithKeyGetter(getIPMinuteKey),
			withCustomErrorHandler(),
			withCustomLimitReachedHandler()),
		ginlimiter.NewMiddleware(ipDayLimiter,
			ginlimiter.WithKeyGetter(getIPDayKey),
			withCustomErrorHandler(),
			withCustomLimitReachedHandler()),
		ginlimiter.NewMiddleware(postKeyMinuteLimiter,
			ginlimiter.WithKeyGetter(getPostKeyMinuteKey),
			withCustomErrorHandler(),
			withCustomLimitReachedHandler()),
		ginlimiter.NewMiddleware(postKeyDayLimiter,
			ginlimiter.WithKeyGetter(getPostKeyDayKey),
			withCustomErrorHandler(),
			withCustomLimitReachedHandler()),
	}

	log.Printf("Rate limiters initialized: ip=%d/min %d/day, post_key=%d/min %d/day",
		config.IPPerMinute, config.IPPerDay, config.PostKeyPerMinute, config.PostKeyPerDay)

	return middlewares, nil
}

func getIPMinuteKey(c *gin.Context) string {
	return "ip:minute:" + c.ClientIP()
}

func getIPDayKey(c *gin.Context) string {
	return "ip:day:" + c.ClientIP()
}

func getPostKeyMinuteKey(c *gin.Context) string {
	return "postkey:minute:" + c.Param("post_key")
}

func getPostKeyDayKey(c *gin.Context) string {
	return "postkey:day:" + c.Param("post_key")
}

func withCustomErrorHandler() ginlimiter.Option {
	return ginlimiter.WithErrorHandler(func(c *gin.Context, err error) {
		log.Printf("rate-limit error ip=%s path=%s err=%v", c.ClientIP(), c.Request.URL.Path, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ErrInternal,
			"message": "Rate limit service unavailable",
		})
	})
}

func withCustomLimitReachedHandler() ginlimiter.Option {
	return ginlimiter.WithLimitReachedHandler(func(c *gin.Context) {
		logRateLimitEvent("limit_reached", c.ClientIP(), c.Param("post_key"), "too_many_requests")
		c.JSON(http.StatusTooManyRequests, gin.H{
			"code":    "too_many_requests",
			"message": "Too many requests",
		})
	})
}

func logRateLimitEvent(eventType, ip, postKey, reason string) {
	timestamp := time.Now().Format(time.RFC3339)
	log.Printf("rate-limit event=%s ip=%s post_key=%s reason=%s ts=%s",
		eventType, ip, postKey, reason, timestamp)
}

func FallbackMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if path == "/" {
			c.File("./dist/index.html")
			c.Abort()
			return
		}

		if strings.HasPrefix(path, "/ui") {
			if strings.HasPrefix(path, "/ui/assets") || path == "/ui/markpost.svg" {
				c.Next()
				return
			}
			c.File("./dist/index.html")
			c.Abort()
			return
		}

		c.Next()
	}
}