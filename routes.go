package main

import (
	"log"
	"net/http"

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

	// API 路由组
	api := r.Group("/api")

	// OAuth 认证相关路由
	oauth := api.Group("/oauth")
	{
		oauth.GET("/url", GenerateGitHubOAuthURLHandler) // 生成授权 URL
		oauth.POST("/login", LoginGitHubHandler)         // 处理回调
	}

	// 密码认证相关路由
	auth := api.Group("/auth")
	{
		auth.POST("/login", LoginWithPasswordHandler) // 密码登录
	}

	// JWT 认证的路由组
	jwtAuth := api.Group("")
	jwtAuth.Use(JWTMiddleware())
	{
		jwtAuth.GET("/post_key", QueryPostKeyHandler)                // 查询用户的 post_key
		jwtAuth.POST("/auth/change-password", ChangePasswordHandler) // 修改密码
	}

	// 为 create post 路由添加限流中间件
	middlewares := append(rateLimitMiddlewares, AuthMiddleware())
	handlers := append(middlewares, CreatePostHandler)
	r.POST("/:post_key", handlers...)

	// 其他路由不受限流影响
	r.GET("/:id", RenderPostHandler)
	r.GET("/health", HealthHandler)

	// 根路径路由 - 返回 index.html
	r.GET("/", func(c *gin.Context) {
		c.File("./dist/index.html")
	})

	// 静态文件服务 - 服务前端构建的资源文件
	r.Static("/ui/assets", "./dist/assets")
	r.StaticFile("/ui/markpost.svg", "./dist/markpost.svg")

	// 前端路由处理 - 精确匹配常见前端路由路径
	uiRoutes := []string{"/ui", "/ui/", "/ui/login", "/ui/dashboard", "/ui/settings"}
	for _, route := range uiRoutes {
		r.GET(route, func(c *gin.Context) {
			c.File("./dist/index.html")
		})
	}

	// 其他未匹配的路由处理 - 统一处理前端路由和404
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// 检查是否为前端路由路径
		if path == "/" || (len(path) >= 4 && path[:4] == "/ui/") {
			// 排除静态文件路径（这些应该由静态文件服务处理）
			if len(path) > 11 && path[:11] == "/ui/assets" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Asset not found"})
				return
			}

			// 返回前端的 index.html
			c.File("./dist/index.html")
		} else {
			// 其他路径返回 404
			c.JSON(http.StatusNotFound, gin.H{"error": "Route not found"})
		}
	})
}
