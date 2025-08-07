package main

import "github.com/gin-gonic/gin"

// SetupRoutes 配置所有路由
func SetupRoutes(r *gin.Engine) {
	// 认证相关路由
	auth := r.Group("/auth")
	{
		auth.GET("/github/url", GenerateGitHubAuthURL)     // 生成授权 URL
		auth.GET("/github/callback", HandleGitHubCallback) // 处理回调
	}

	// 现有路由
	r.POST("/:post_key", AuthMiddleware(), CreatePostHandler)
	r.GET("/:id", RenderPostHandler)
	r.GET("/health", HealthHandler)
}
