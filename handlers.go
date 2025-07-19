package main

import (
	"bytes"
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
)

type PostRequest struct {
	Title string `json:"title" binding:"required,titlesize"`
	Body  string `json:"body" binding:"required,bodysize"`
}

func CreatePostHandler(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户信息获取失败"})
		return
	}

	_ = user.(*User)

	var req PostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	post, err := CreatePost(req.Title, req.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建内容失败"})
		log.Printf("创建内容失败: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": post.ID,
	})
}

func RenderPostHandler(c *gin.Context) {
	id := c.Param("id")
	post, err := GetPostByID(id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			c.String(http.StatusNotFound, "Not Found")
		default:
			log.Printf("查询内容失败: %v", err)
			c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}

	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(post.Body), &buf); err != nil {
		log.Printf("Markdown 转换失败: %v", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	c.HTML(http.StatusOK, "post.html", gin.H{
		"Title": post.Title,
		"Body":  template.HTML(buf.String()),
	})
}

func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "markpost is running",
	})
}
