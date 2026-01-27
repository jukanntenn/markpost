package main

// @title           Markpost API
// @version         1.0
// @description     A simple pastebin-like markdown blog service with OAuth and JWT authentication
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    https://github.com/yourusername/markpost
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:7330
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

import (
	"fmt"
	"log"
	"os"
	"strconv"

	appcmd "markpost/cmd"
	"markpost/conf"
	"markpost/middlewares"
	"markpost/models"
	"markpost/repositories"
	"markpost/services"
	"markpost/utils"

	"github.com/gin-contrib/cors"
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/text/language"
)

var authSvc *services.AuthService
var postSvc *services.PostService
var jwtSvc *services.JWTService

var userRepo repositories.UserRepoInterface
var postRepo repositories.PostRepoInterface

var database *models.Database

var validate *validator.Validate

func init() {
	validate = validator.New()
}

func UseCors(r *gin.Engine) {
	cfg := conf.Conf()
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
				Name:  "prune-expired-posts",
				Usage: "Prune expired posts from database",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "dry-run",
						Aliases: []string{"d"},
						Usage:   "Only print number of posts to be deleted",
					},
					&cli.IntFlag{
						Name:    "batch-size",
						Aliases: []string{"b"},
						Value:   100,
						Usage:   "Number of records to delete per batch",
					},
				},
				Action: func(c *cli.Context) error {
					return appcmd.RunPruneExpiredPosts(
						c.String("config"),
						c.Bool("dry-run"),
						c.Int("batch-size"),
					)
				},
			},
			{
				Name:  "import-fake-posts",
				Usage: "Import fake posts from JSON file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Usage:   "Path to fake.json file",
						Value:   "fake.json",
					},
				},
				Action: func(c *cli.Context) error {
					return appcmd.RunImportFakePosts(
						c.String("config"),
						c.String("file"),
					)
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
	if err := conf.LoadConfig(configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cfg := conf.Conf()

	dbInstance, err := models.NewDatabase(cfg.DB.DSN)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer func() {
		sqlDB, err := dbInstance.DB().DB()
		if err == nil && sqlDB != nil {
			sqlDB.Close()
		}
	}()
	database = dbInstance

	if err := database.DB().AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	userRepo = repositories.NewUserRepo(database)
	postRepo = repositories.NewPostRepo(database)

	RegisterValidators()

	jwtSvc = services.NewJWTService(
		cfg.JWT.AccessSigningKey,
		cfg.JWT.RefreshSigningKey,
		cfg.JWT.AccessTokenExpire,
		cfg.JWT.RefreshTokenExpire,
	)

	authSvc = services.NewAuthService(userRepo, &oauth2.Config{
		ClientID:     cfg.OAuth.GitHub.ClientID,
		ClientSecret: cfg.OAuth.GitHub.ClientSecret,
		RedirectURL:  cfg.OAuth.GitHub.RedirectURL,
		Scopes:       []string{},
		Endpoint:     github.Endpoint,
	}, jwtSvc)

	log.Printf("Initializing first admin user: %s", cfg.Admin.InitialUsername)
	if err := authSvc.InitializeFirstAdmin(cfg.Admin.InitialUsername); err != nil {
		log.Fatalf("Failed to initialize first admin: %v", err)
	}

	postSvc = services.NewPostService(postRepo)

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
		Loader:           utils.ActiveLocaleLoader{},
	})))

	r.Use(middlewares.Fallback())

	UseCors(r)

	log.Printf("Initializing rate limiting for create post...")
	SetupRoutes(r)

	log.Println("Server starting...")
	listenAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	visitHost := cfg.Server.Host
	log.Println("Visit http://" + visitHost + ":" + strconv.FormatUint(uint64(cfg.Server.Port), 10))
	if err := r.Run(listenAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
