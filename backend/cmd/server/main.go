// Package main provides the entry point for the Markpost server.
//
// @title Markpost API
// @version 1.0
// @description Markpost backend API for post management and delivery.
// @host localhost:7330
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description "Bearer <access_token>"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"markpost/cmd"
	_ "markpost/docs"
	v1 "markpost/internal/api/rest/v1"
	"markpost/internal/config"
	"markpost/internal/domain/user"
	"markpost/internal/infra"
	"markpost/internal/middleware"
	"markpost/internal/service/admin"
	"markpost/internal/service/auth"
	deliverysvc "markpost/internal/service/delivery"
	postsvc "markpost/internal/service/post"

	"github.com/gin-contrib/cors"
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
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
			{
				Name:  "reset-password",
				Usage: "Reset a user's password and revoke all sessions",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "username",
						Aliases:  []string{"u"},
						Usage:    "Username of the user to reset",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "password",
						Aliases: []string{"p"},
						Usage:   "New password (generates a random one if omitted)",
					},
				},
				Action: func(c *cli.Context) error {
					return cmd.RunResetPassword(c.String("config"), c.String("username"), c.String("password"))
				},
			},
			{
				Name:  "import-fake-posts",
				Usage: "Import fake posts from a JSON file (for load-test seeding)",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    "Path to the JSON file produced by tools/generate_fake_data",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return cmd.RunImportFakePosts(c.String("config"), c.String("file"))
				},
			},
			{
				Name:  "seed-users",
				Usage: "Create test users with post_keys (and optional delivery channels) for load testing",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "count",
						Usage: "Number of users to create",
						Value: 100,
					},
					&cli.StringFlag{
						Name:  "prefix",
						Usage: "Username prefix for seeded users",
						Value: "loadtest",
					},
					&cli.StringFlag{
						Name:  "password",
						Usage: "Password for seeded users",
						Value: "loadtestpass",
					},
					&cli.IntFlag{
						Name:  "channels",
						Usage: "Number of Feishu delivery channels to attach per user",
						Value: 0,
					},
					&cli.StringFlag{
						Name:  "channel-keywords",
						Usage: "Keyword filter expression for each delivery channel",
						Value: "",
					},
				},
				Action: func(c *cli.Context) error {
					return cmd.RunSeedUsers(
						c.String("config"),
						c.Int("count"),
						c.String("prefix"),
						c.String("password"),
						c.Int("channels"),
						c.String("channel-keywords"),
					)
				},
			},
			{
				Name:  "prune-expired-posts",
				Usage: "Delete posts older than the retention window (cron-invoked)",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Count expired posts without deleting",
					},
					&cli.IntFlag{
						Name:  "batch-size",
						Usage: "Rows deleted per statement (0 = server default)",
						Value: 0,
					},
				},
				Action: func(c *cli.Context) error {
					return cmd.RunPruneExpiredPosts(c.String("config"), c.Bool("dry-run"), c.Int("batch-size"))
				},
			},
			{
				Name:  "prune-delivery-history",
				Usage: "Delete delivery_history rows older than the retention window (cron-invoked)",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Count old rows without deleting",
					},
					&cli.IntFlag{
						Name:  "batch-size",
						Usage: "Rows deleted per statement (0 = server default)",
						Value: 0,
					},
				},
				Action: func(c *cli.Context) error {
					return cmd.RunPruneDeliveryHistory(c.String("config"), c.Bool("dry-run"), c.Int("batch-size"))
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
		if err := dbInstance.Close(); err != nil {
			log.Printf("Failed to close database: %v", err)
		}
	}()

	userRepo = infra.NewUserRepository(dbInstance.DB(), cfg.PostKeyLength)
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
	if stateStore, err := auth.NewRistrettoOAuthStateStore(); err == nil {
		authSvc = authSvc.WithOAuthStateStore(stateStore)
	} else {
		log.Printf("OAuth state store disabled (ristretto init failed: %v)", err)
	}

	log.Printf("Initializing first admin user: %s", cfg.Admin.InitialUsername)
	if err := authSvc.InitializeFirstAdmin(context.Background(), cfg.Admin.InitialUsername); err != nil {
		log.Fatalf("Failed to initialize first admin: %v", err)
	}

	postRepo := infra.NewPostRepository(dbInstance.DB())

	deliveryRepo := infra.NewDeliveryChannelRepository(dbInstance.DB())
	attemptRepo := infra.NewAttemptRepository(dbInstance.DB())
	deliverySvc := deliverysvc.NewService(deliveryRepo, attemptRepo)

	postDeliverySvc := deliverysvc.NewPostDeliveryService()
	deliveryDispatcher := deliverysvc.NewDispatcher(attemptRepo, deliveryRepo, postRepo, postDeliverySvc)
	dispatcherCtx, dispatcherCancel := context.WithCancel(context.Background())
	defer dispatcherCancel()
	deliveryDispatcher.Start(dispatcherCtx)
	defer deliveryDispatcher.Stop()

	postSvc = postsvc.NewService(postRepo, deliveryDispatcher)

	adminSvc := admin.NewService(userRepo, postSvc, deliverySvc, attemptRepo)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	if err := r.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		log.Fatalf("Failed to set trusted proxies: %v", err)
	}

	r.Use(ginI18n.Localize(ginI18n.WithBundle(&ginI18n.BundleCfg{
		RootPath: "./locales",
		AcceptLanguage: []language.Tag{
			language.English,
			language.SimplifiedChinese,
			language.TraditionalChinese,
			language.Japanese,
		},
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
		dispatcherCancel()
		deliveryDispatcher.Stop()
		log.Fatalf("Failed to start server: %v", err)
	}
}

// SetupRoutes configures all API routes for the application.
func SetupRoutes(r *gin.Engine, deliverySvc *deliverysvc.Service, adminSvc *admin.Service) {
	cfg := config.Get()

	// Three independent rate limiters, each scoped to a route class and keyed on
	// the dimension that identifies the actor. They replace the single global
	// limiter so read throttling is no longer coupled to write throttling.
	l1Read := middleware.NewLimiter(cfg.Ratelimit.Read.PerSecond, cfg.Ratelimit.Read.Burst)
	l2Write := middleware.NewLimiter(cfg.Ratelimit.L2.PerSecond, cfg.Ratelimit.L2.Burst)
	l2Daily := middleware.NewLimiter(cfg.Ratelimit.L2.DailyPerSec, cfg.Ratelimit.L2.DailyBurst)
	l3Write := middleware.NewLimiter(cfg.Ratelimit.L3.PerSecond, cfg.Ratelimit.L3.Burst)

	if cfg.Debug {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))
	}

	apiV1 := r.Group("/api/v1")
	apiV1.GET("/health", v1.Health())

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
		jwtAuth.GET("/posts", v1.PostsList(postSvc))

		// L3: authenticated state changes keyed on user_id (from JWT). Reads
		// (GET) stay outside the limiter so listing does not consume the write
		// budget.
		jwtWrite := jwtAuth.Group("")
		jwtWrite.Use(middleware.RateLimitByUserID(l3Write))
		{
			jwtWrite.POST("/auth/logout", v1.Logout(authSvc))
			jwtWrite.POST("/auth/change-password", v1.ChangePassword(authSvc))
			jwtWrite.DELETE("/posts/:id", v1.DeleteOwnPost(postSvc))
		}

		deliveryGroup := jwtAuth.Group("/delivery/channels")
		{
			deliveryGroup.GET("", v1.ListDeliveryChannels(deliverySvc))
			deliveryGroup.POST("", middleware.RateLimitByUserID(l3Write), v1.CreateDeliveryChannel(deliverySvc))
			deliveryGroup.PATCH("/:id", middleware.RateLimitByUserID(l3Write), v1.UpdateDeliveryChannel(deliverySvc))
			deliveryGroup.DELETE("/:id", middleware.RateLimitByUserID(l3Write), v1.DeleteDeliveryChannel(deliverySvc))
		}
		jwtAuth.GET("/delivery/history", v1.ListDeliveryHistory(deliverySvc))

		adminGroup := jwtAuth.Group("/admin")
		adminGroup.Use(middleware.RequireAdmin())
		{
			adminGroup.GET("/users", v1.AdminListUsers(adminSvc))
			adminGroup.GET("/posts", v1.AdminListPosts(adminSvc))
			adminGroup.GET("/delivery/channels", v1.AdminListChannels(adminSvc))
			adminGroup.GET("/delivery/history", v1.AdminListDeliveryHistory(adminSvc))
			adminGroup.DELETE("/posts/:id", middleware.RateLimitByUserID(l3Write), v1.DeleteAnyPost(postSvc))
		}
	}

	// L2: public writes keyed on user_id, resolved by PostKey. The 10/min and
	// 1000/day limiters chain so both must pass.
	r.POST("/:post_key", middleware.PostKey(userRepo), middleware.RateLimitByUserID(l2Write, l2Daily), v1.CreatePost(postSvc))
	r.GET("/static/:filename", v1.StaticCSS())
	// L1: public reads keyed on client IP.
	r.GET("/:id", middleware.RateLimitByIP(l1Read), v1.RenderPost(postSvc))

	r.NoRoute(v1.NotFound())
}
