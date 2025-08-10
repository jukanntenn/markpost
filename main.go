package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"golang.org/x/time/rate"
)

func main() {
	// 检查是否是清理命令
	if len(os.Args) > 1 && os.Args[1] == "cleanup" {
		// 移除 "cleanup" 参数，让 flag 包正确解析剩余参数
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		CleanupCommand()
		return
	}

	if err := LoadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化 GitHub OAuth2 配置
	initOAuthConfig()

	InitDB()
	defer CloseDB()

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("titlesize", validateTitleSize)
		v.RegisterValidation("bodysize", validateBodySize)
	}

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "x-oauth-state", "Authorization")
	r.Use(cors.New(corsConfig))

	if config.APIRateLimit > 0 {
		limitPerSecond := float64(config.APIRateLimit) / 60.0
		r.Use(LimiterMiddleware(rate.Limit(limitPerSecond), config.APIRateLimit))
		log.Printf("已启用全局限流: 每分钟 %d 次请求", config.APIRateLimit)
	}

	log.Printf("正在初始化 create post 专用限流功能...")
	SetupRoutes(r)

	// 启动限流监控
	startRateLimitMonitoring()

	log.Printf("限流功能初始化完成")

	log.Println("服务器启动中...")
	log.Println("访问 http://localhost:7330")
	if err := r.Run(":7330"); err != nil {
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
