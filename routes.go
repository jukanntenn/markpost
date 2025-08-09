package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 配置所有路由
func SetupRoutes(r *gin.Engine) {
	// 初始化限流中间件
	rateLimitConfig := RateLimitConfig{
		IPPerMinute:      config.RateLimit.IP.PerMinute,
		IPPerDay:         config.RateLimit.IP.PerDay,
		PostKeyPerMinute: config.RateLimit.PostKey.PerMinute,
		PostKeyPerDay:    config.RateLimit.PostKey.PerDay,
	}

	rateLimitMiddlewares, err := initRateLimiters(rateLimitConfig)
	if err != nil {
		log.Printf("初始化限流器失败: %v", err)
		// 使用空的中间件数组，允许请求通过（fail-open 策略）
		rateLimitMiddlewares = []gin.HandlerFunc{}
	}

	// 认证相关路由
	auth := r.Group("/auth")
	{
		auth.GET("/github/url", GenerateGitHubAuthURL)     // 生成授权 URL
		auth.GET("/github/callback", HandleGitHubCallback) // 处理回调
	}

	// 为 create post 路由添加限流中间件
	middlewares := append(rateLimitMiddlewares, AuthMiddleware())
	handlers := append(middlewares, CreatePostHandler)
	r.POST("/:post_key", handlers...)

	// 其他路由不受限流影响
	r.GET("/:id", RenderPostHandler)
	r.GET("/health", HealthHandler)
}
