package main

import (
	"net/http"

	"markpost/conf"
	"markpost/handlers"
	"markpost/middlewares"

	"github.com/didip/tollbooth/v8"
	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "markpost/docs"
)

func SetupRoutes(r *gin.Engine) {
	cfg := conf.Conf()
	lmt := tollbooth.NewLimiter(float64(cfg.Ratelimit.PerSecond), nil)
	lmt.SetBurst(cfg.Ratelimit.Burst)
	lmt.SetMessageContentType("application/json; charset=utf-8")

	r.Use(middlewares.RateLimitByIP(lmt))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api")

	oauth := api.Group("/oauth")
	{
		oauth.GET("/url", handlers.GenerateGitHubOAuthURL(authSvc))
		oauth.POST("/login", handlers.LoginGitHub(authSvc))
	}

	auth := api.Group("/auth")
	{
		auth.POST("/login", handlers.LoginWithPassword(authSvc))
		auth.POST("/refresh", handlers.RefreshToken(authSvc))
	}

	jwtAuth := api.Group("")
	jwtAuth.Use(middlewares.Auth(jwtSvc, userRepo))
	{
		jwtAuth.GET("/post_key", handlers.QueryPostKey(authSvc))
		jwtAuth.POST("/auth/change-password", handlers.ChangePassword(authSvc))
		jwtAuth.GET("/posts", handlers.PostsList(postSvc))
		jwtAuth.GET("/delivery/channels", handlers.ListDeliveryChannels(deliveryChannelSvc))
		jwtAuth.POST("/delivery/channels", handlers.CreateDeliveryChannel(deliveryChannelSvc))
		jwtAuth.PUT("/delivery/channels/:id", handlers.UpdateDeliveryChannel(deliveryChannelSvc))
		jwtAuth.DELETE("/delivery/channels/:id", handlers.DeleteDeliveryChannel(deliveryChannelSvc))
	}

	admin := api.Group("/admin")
	admin.Use(middlewares.Auth(jwtSvc, userRepo))
	admin.Use(middlewares.RequireAdmin())
	{
		userHandler := handlers.NewUserHandler(authSvc)
		adminHandler := handlers.NewAdminHandler(adminSvc)
		admin.GET("/users", userHandler.ListAllUsers)
		admin.POST("/users", adminHandler.CreateUser)
		admin.PUT("/users/:id/role", adminHandler.UpdateUserRole)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)
		admin.POST("/users/:id/reset-password", adminHandler.ResetUserPassword)
		admin.GET("/posts", adminHandler.ListAllPosts)
		admin.PUT("/posts/:id", adminHandler.UpdatePost)
		admin.DELETE("/posts/:id", adminHandler.DeletePost)
		admin.GET("/channels", adminHandler.ListAllDeliveryChannels)
		admin.PUT("/channels/:id", adminHandler.UpdateDeliveryChannel)
		admin.DELETE("/channels/:id", adminHandler.DeleteDeliveryChannel)
	}

	r.POST("/:post_key", middlewares.PostKey(userRepo), handlers.CreatePost(postSvc))

	r.GET("/:id", handlers.RenderPost(postSvc))
	r.GET("/health", handlers.Health())

	r.Static("/ui/assets", "../dist/assets")
	r.StaticFile("/ui/markpost.svg", "../dist/markpost.svg")

	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})
}
