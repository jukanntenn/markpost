package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var authSvc *AuthService
var postSvc *PostService

func main() {
	if len(os.Args) > 1 && os.Args[1] == "cleanup" {
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		CleanupCommand()
		return
	}

	if err := LoadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	InitI18n()

	initOAuthConfig()

	dbInstance, err := NewDatabase(config.Database.URL)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer func() {
		sqlDB, err := dbInstance.GetDB().DB()
		if err == nil && sqlDB != nil {
			sqlDB.Close()
		}
	}()
	database = dbInstance

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("titlesize", validateTitleSize)
		v.RegisterValidation("bodysize", validateBodySize)
	}

	authSvc = NewAuthService(database.GetUserRepository(), oauthConfig)
	postSvc = NewPostService(database.GetPostRepository())

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.Use(FallbackMiddleware())
	r.Use(I18nMiddleware())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "x-oauth-state", "Authorization")
	r.Use(cors.New(corsConfig))

	log.Printf("Initializing rate limiting for create post...")
	SetupRoutes(r)

	log.Println("Server starting...")
	log.Println("Visit http://localhost:7330")
	if err := r.Run(":7330"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func validateTitleSize(fl validator.FieldLevel) bool {
	if config.TitleMaxSize <= 0 {
		return true
	}
	return len([]byte(fl.Field().String())) <= config.TitleMaxSize
}

func validateBodySize(fl validator.FieldLevel) bool {
	if config.BodyMaxSize <= 0 {
		return true
	}
	return len([]byte(fl.Field().String())) <= config.BodyMaxSize
}