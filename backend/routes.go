package main

import (
	"net/http"

	"markpost/handlers"
	"markpost/middlewares"

	"github.com/didip/tollbooth/v8"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	lmt := tollbooth.NewLimiter(2, nil)
	lmt.SetBurst(10)

	r.Use(middlewares.RateLimitByIP(lmt))

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
