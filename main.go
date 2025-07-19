package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"golang.org/x/time/rate"
)

func main() {
	if err := LoadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	InitDB()
	defer CloseDB()

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("titlesize", validateTitleSize)
		v.RegisterValidation("bodysize", validateBodySize)
	}

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	if config.APIRateLimit > 0 {
		limitPerSecond := float64(config.APIRateLimit) / 60.0
		r.Use(LimiterMiddleware(rate.Limit(limitPerSecond), config.APIRateLimit))
		log.Printf("已启用限流: 每分钟 %d 次请求", config.APIRateLimit)
	}

	SetupRoutes(r)

	log.Println("服务器启动中...")
	log.Println("访问 http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
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
