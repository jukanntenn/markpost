package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	rateLimitMiddlewares, err := initRateLimiters(RateLimitConfig{
		IPPerMinute:      config.RateLimit.IP.PerMinute,
		IPPerDay:         config.RateLimit.IP.PerDay,
		PostKeyPerMinute: config.RateLimit.PostKey.PerMinute,
		PostKeyPerDay:    config.RateLimit.PostKey.PerDay,
	})
	if err != nil {
		log.Fatalf("Failed to initialize rate limiters: %v", err)
	}

	api := r.Group("/api")

	oauth := api.Group("/oauth")
	{
		oauth.GET("/url", GenerateGitHubOAuthURLHandler)
		oauth.POST("/login", LoginGitHubHandler)
	}

  auth := api.Group("/auth")
  {
    auth.POST("/login", LoginWithPasswordHandler)
    auth.POST("/refresh", RefreshTokenHandler)
  }

	jwtAuth := api.Group("")
	jwtAuth.Use(AuthMiddleware())
	{
		jwtAuth.GET("/post_key", QueryPostKeyHandler)
		jwtAuth.POST("/auth/change-password", ChangePasswordHandler)
		jwtAuth.GET("/posts", PostsListHandler)
	}

	middlewares := append(rateLimitMiddlewares, PostKeyMiddleware())
	handlers := append(middlewares, CreatePostHandler)
	r.POST("/:post_key", handlers...)

	r.GET("/:id", RenderPostHandler)
	r.GET("/health", HealthHandler)

	r.Static("/ui/assets", "./dist/assets")
	r.StaticFile("/ui/markpost.svg", "./dist/markpost.svg")

	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})
}
