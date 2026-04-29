// Package main provides the entry point for the Markpost server.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	v1 "markpost/internal/api/rest/v1"
	"markpost/internal/config"
	"markpost/internal/domain/user"
	"markpost/internal/infra"
	"markpost/internal/middleware"
	"markpost/internal/service/admin"
	"markpost/internal/service/auth"
	deliverysvc "markpost/internal/service/delivery"
	postsvc "markpost/internal/service/post"

	"github.com/didip/tollbooth/v8"
	"github.com/gin-contrib/cors"
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/urfave/cli/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/text/language"
)

var authSvc *auth.Service
var postSvc *postsvc.Service
var jwtSvc *auth.JWTService

var userRepo user.Repository
var tokenRepo user.TokenRepository

func init() {
	_ = validator.New()
}

// UseCors configures CORS middleware for the gin router.
func UseCors(r *gin.Engine) {
	cfg := config.Get()
	c := cors.DefaultConfig()
	c.AllowOrigins = cfg.CORS.AllowOrigins
	c.AllowHeaders = cfg.CORS.AllowHeaders
	c.ExposeHeaders = cfg.CORS.ExposeHeaders
	r.Use(cors.New(c))
}

func main() {
	app := &cli.App{
		Name:  "markpost",
		Usage: "Markpost backend server and management commands",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to config file",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start HTTP server",
				Action: func(c *cli.Context) error {
					serve(c.String("config"))
					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			serve(c.String("config"))
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Failed to run markpost: %v", err)
	}
}

func serve(configPath string) {
	if err := config.Load(configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cfg := config.Get()

	dbInstance, err := infra.New(cfg.DB.DSN)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer func() {
		sqlDB, err := dbInstance.DB().DB()
		if err == nil && sqlDB != nil {
			_ = sqlDB.Close()
		}
	}()

	userRepo = infra.NewUserRepository(dbInstance.DB())
	tokenRepo = infra.NewTokenRepository(dbInstance.DB())

	RegisterValidators()

	jwtSvc = auth.NewJWTService(
		cfg.JWT.AccessSigningKey,
		cfg.JWT.RefreshSigningKey,
		cfg.JWT.AccessTokenExpire,
		cfg.JWT.RefreshTokenExpire,
	)

	authSvc = auth.NewService(
		userRepo,
		tokenRepo,
		&oauth2.Config{
			ClientID:     cfg.OAuth.GitHub.ClientID,
			ClientSecret: cfg.OAuth.GitHub.ClientSecret,
			RedirectURL:  cfg.OAuth.GitHub.RedirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		},
		jwtSvc,
		"markpost",
	)

	log.Printf("Initializing first admin user: %s", cfg.Admin.InitialUsername)
	if err := authSvc.InitializeFirstAdmin(context.Background(), cfg.Admin.InitialUsername); err != nil {
		log.Fatalf("Failed to initialize first admin: %v", err)
	}

	postRepo := infra.NewPostRepository(dbInstance.DB())
	postSvc = postsvc.NewService(postRepo, nil)

	deliveryRepo := infra.NewDeliveryChannelRepository(dbInstance.DB())
	deliverySvc := deliverysvc.NewService(deliveryRepo)

	adminSvc := admin.NewService(userRepo, postRepo, deliveryRepo)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	if err := r.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		log.Fatalf("Failed to set trusted proxies: %v", err)
	}

	r.Use(ginI18n.Localize(ginI18n.WithBundle(&ginI18n.BundleCfg{
		RootPath:         "./locales",
		AcceptLanguage:   []language.Tag{language.English, language.Chinese},
		DefaultLanguage:  language.English,
		UnmarshalFunc:    toml.Unmarshal,
		FormatBundleFile: "toml",
	})))

	r.Use(middleware.Fallback())

	UseCors(r)

	log.Printf("Initializing rate limiting...")
	SetupRoutes(r, deliverySvc, adminSvc)

	log.Println("Server starting...")
	listenAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	visitHost := cfg.Server.Host
	log.Println("Visit http://" + visitHost + ":" + strconv.FormatUint(uint64(cfg.Server.Port), 10))
	if err := r.Run(listenAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// SetupRoutes configures all API routes for the application.
func SetupRoutes(r *gin.Engine, deliverySvc *deliverysvc.Service, adminSvc *admin.Service) {
	cfg := config.Get()
	lmt := tollbooth.NewLimiter(float64(cfg.Ratelimit.PerSecond), nil)
	lmt.SetBurst(cfg.Ratelimit.Burst)
	lmt.SetMessageContentType("application/json; charset=utf-8")

	r.Use(middleware.RateLimitByIP(lmt))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	apiV1 := r.Group("/api/v1")

	oauthGroup := apiV1.Group("/oauth")
	{
		oauthGroup.GET("/url", v1.GenerateGitHubOAuthURL(authSvc))
		oauthGroup.POST("/login", v1.LoginGitHub(authSvc))
	}

	authGroup := apiV1.Group("/auth")
	{
		authGroup.POST("/login", v1.LoginWithUsername(authSvc))
		authGroup.POST("/refresh", v1.RefreshToken(authSvc))
	}

	jwtAuth := apiV1.Group("")
	jwtAuth.Use(middleware.AuthWithBlacklist(jwtSvc, userRepo, tokenRepo))
	{
		jwtAuth.GET("/post_key", v1.QueryPostKey(authSvc))
		jwtAuth.POST("/auth/logout", v1.Logout(authSvc))
		jwtAuth.POST("/auth/change-password", v1.ChangePassword(authSvc))
		jwtAuth.GET("/posts", v1.PostsList(postSvc))

		// Delivery channel routes
		deliveryGroup := jwtAuth.Group("/delivery/channels")
		{
			deliveryGroup.GET("", v1.ListDeliveryChannels(deliverySvc))
			deliveryGroup.POST("", v1.CreateDeliveryChannel(deliverySvc))
			deliveryGroup.PUT("/:id", v1.UpdateDeliveryChannel(deliverySvc))
			deliveryGroup.DELETE("/:id", v1.DeleteDeliveryChannel(deliverySvc))
		}

		// Admin routes
		adminGroup := jwtAuth.Group("/admin")
		adminGroup.Use(middleware.RequireAdmin())
		{
			adminGroup.GET("/users", v1.AdminListUsers(adminSvc))
			adminGroup.GET("/posts", v1.AdminListPosts(adminSvc))
			adminGroup.GET("/channels", v1.AdminListChannels(adminSvc))
		}
	}

	r.POST("/:post_key", middleware.PostKey(userRepo), v1.CreatePost(postSvc))

	var proxy gin.HandlerFunc
	if cfg.Server.FrontendURL != "" {
		proxy = middleware.NewFrontendProxy(cfg.Server.FrontendURL)
	}

	r.GET("/:id", v1.RenderPost(postSvc, proxy))
	r.GET("/health", v1.Health())

	r.Static("/ui/assets", "../dist/assets")
	r.StaticFile("/ui/markpost.svg", "../dist/markpost.svg")

	if proxy != nil {
		r.NoRoute(proxy)
	} else {
		r.NoRoute(func(c *gin.Context) {
			c.String(http.StatusNotFound, "Not Found")
		})
	}
}
