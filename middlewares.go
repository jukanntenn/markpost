package main

import (
	"database/sql"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	ginlimiter "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
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

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "missing authorization header"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := validateJWTToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "invalid token"})
			c.Abort()
			return
		}

		user, err := GetUserByID(claims.UserID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"detail": "user not found"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

// RateLimitConfig 限流配置结构
type RateLimitConfig struct {
	IPPerMinute      int
	IPPerDay         int
	PostKeyPerMinute int
	PostKeyPerDay    int
}

// initRateLimiters 初始化限流器并返回 Gin 中间件数组
func initRateLimiters(config RateLimitConfig) ([]gin.HandlerFunc, error) {
	store := memory.NewStore()

	// 定义限流速率
	ipMinuteRate := limiter.Rate{Period: time.Minute, Limit: int64(config.IPPerMinute)}
	ipDayRate := limiter.Rate{Period: 24 * time.Hour, Limit: int64(config.IPPerDay)}
	postKeyMinuteRate := limiter.Rate{Period: time.Minute, Limit: int64(config.PostKeyPerMinute)}
	postKeyDayRate := limiter.Rate{Period: 24 * time.Hour, Limit: int64(config.PostKeyPerDay)}

	// 创建限流器实例
	ipMinuteLimiter := limiter.New(store, ipMinuteRate)
	ipDayLimiter := limiter.New(store, ipDayRate)
	postKeyMinuteLimiter := limiter.New(store, postKeyMinuteRate)
	postKeyDayLimiter := limiter.New(store, postKeyDayRate)

	// 创建 Gin 中间件，使用自定义错误处理器和不同的键格式
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

	log.Printf("限流器初始化成功: IP限制 %d/分钟 %d/天, PostKey限制 %d/分钟 %d/天",
		config.IPPerMinute, config.IPPerDay, config.PostKeyPerMinute, config.PostKeyPerDay)

	return middlewares, nil
}

// getIPMinuteKey IP 分钟限流键提取器
func getIPMinuteKey(c *gin.Context) string {
	return "ip:minute:" + c.ClientIP()
}

// getIPDayKey IP 天限流键提取器
func getIPDayKey(c *gin.Context) string {
	return "ip:day:" + c.ClientIP()
}

// getPostKeyMinuteKey PostKey 分钟限流键提取器
func getPostKeyMinuteKey(c *gin.Context) string {
	return "postkey:minute:" + c.Param("post_key")
}

// getPostKeyDayKey PostKey 天限流键提取器
func getPostKeyDayKey(c *gin.Context) string {
	return "postkey:day:" + c.Param("post_key")
}

// withCustomErrorHandler 自定义错误处理器
func withCustomErrorHandler() ginlimiter.Option {
	return ginlimiter.WithErrorHandler(func(c *gin.Context, err error) {
		log.Printf("[RATE_LIMIT] 限流系统错误 - IP: %s, 路径: %s, 错误: %v",
			c.ClientIP(), c.Request.URL.Path, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "限流服务暂时不可用",
			"message": "Rate limit service temporarily unavailable",
		})
	})
}

// withCustomLimitReachedHandler 自定义限流达到处理器
func withCustomLimitReachedHandler() ginlimiter.Option {
	return ginlimiter.WithLimitReachedHandler(func(c *gin.Context) {
		// 记录详细的限流事件
		logRateLimitEvent("LIMIT_REACHED", c.ClientIP(), c.Param("post_key"), "请求频率超出限制")

		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":   "请求频率过高",
			"message": "Too many requests. Please try again later.",
			"details": "您的请求频率超出了限制，请稍后再试",
		})
	})
}

// startRateLimitMonitoring 启动限流监控（定期记录内存使用情况）
func startRateLimitMonitoring() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute) // 每10分钟记录一次
		defer ticker.Stop()

		for range ticker.C {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			log.Printf("[RATE_LIMIT] 内存监控 - 当前分配: %d KB, 总分配: %d KB, 堆对象数: %d",
				memStats.Alloc/1024,
				memStats.TotalAlloc/1024,
				memStats.HeapObjects)
		}
	}()

	log.Printf("[RATE_LIMIT] 已启动限流状态监控，每10分钟记录一次内存使用情况")
}

// logRateLimitEvent 记录限流事件的详细信息
func logRateLimitEvent(eventType, ip, postKey, reason string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	log.Printf("[RATE_LIMIT] %s | 时间: %s | IP: %s | PostKey: %s | 原因: %s",
		eventType, timestamp, ip, postKey, reason)
}
