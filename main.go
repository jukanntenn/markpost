package main

import (
	"bytes"
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
	"golang.org/x/time/rate"
)

// PostRequest 上传请求结构
type PostRequest struct {
	Title string `json:"title" binding:"required"`
	Body  string `json:"body" binding:"required"`
}

func main() {
	// 加载配置
	LoadConfig()

	// 初始化数据库
	InitDB()
	defer CloseDB()

	// 创建 Gin 路由器
	r := gin.Default()

	// 加载 HTML 模板
	r.LoadHTMLGlob("templates/*")

	// 添加限流中间件（如果配置了限流）
	if config.APIRateLimit > 0 {
		// 将每分钟的限制转换为每秒限制
		limitPerSecond := float64(config.APIRateLimit) / 60.0
		r.Use(LimiterMiddleware(rate.Limit(limitPerSecond), config.APIRateLimit))
		log.Printf("已启用限流: 每分钟 %d 次请求", config.APIRateLimit)
	}

	// POST /:post_key - 上传 markdown 内容
	r.POST("/:post_key", func(c *gin.Context) {
		postKey := c.Param("post_key")

		// 验证 post_key 是否有效
		_, err := GetUserByPostKey(postKey)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 post_key"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败"})
			return
		}

		var req PostRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
			return
		}

		// 检查 title 长度
		if config.TitleMaxSize > 0 && len([]byte(req.Title)) > config.TitleMaxSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "标题长度超过限制: " + strconv.Itoa(config.TitleMaxSize) + " 字节",
			})
			return
		}

		// 检查 body 长度
		if config.BodyMaxSize > 0 && len([]byte(req.Body)) > config.BodyMaxSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "内容长度超过限制: " + strconv.Itoa(config.BodyMaxSize) + " 字节",
			})
			return
		}

		// 创建新的内容记录
		post, err := CreatePost(req.Title, req.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建内容失败"})
			log.Printf("创建内容失败: %v", err)
			return
		}

		// 返回 JSON 响应
		c.JSON(http.StatusOK, gin.H{
			"id":      post.ID,
			"title":   post.Title,
			"message": "内容创建成功",
		})
	})

	// GET /:id - 获取内容并渲染 HTML 页面
	r.GET("/:id", func(c *gin.Context) {
		id := c.Param("id")

		post, err := GetPostByID(id)
		if err != nil {
			if err == sql.ErrNoRows {
				c.HTML(http.StatusNotFound, "error.html", gin.H{
					"Error": "内容不存在",
					"Code":  "404",
				})
				return
			}
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": "数据库查询失败",
				"Code":  "500",
			})
			log.Printf("查询内容失败: %v", err)
			return
		}

		// 将 markdown 转换为 HTML
		var buf bytes.Buffer
		if err := goldmark.Convert([]byte(post.Body), &buf); err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": "Markdown 转换失败",
				"Code":  "500",
			})
			log.Printf("Markdown 转换失败: %v", err)
			return
		}

		// 渲染文章页面模板
		c.HTML(http.StatusOK, "article.html", gin.H{
			"ID":        post.ID,
			"Title":     post.Title,
			"Body":      template.HTML(buf.String()),
			"CreatedAt": post.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	})

	// 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "markpost is running",
		})
	})

	// 启动服务器
	log.Println("服务器启动中...")
	log.Println("访问 http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
