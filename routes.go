package main

import "github.com/gin-gonic/gin"

// SetupRoutes 配置所有路由
func SetupRoutes(r *gin.Engine) {
	r.POST("/:post_key", AuthMiddleware(), CreatePostHandler)
	r.GET("/:id", RenderPostHandler)
	r.GET("/health", HealthHandler)
}
