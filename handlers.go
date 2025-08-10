package main

import (
	"bytes"
	"context"
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
)

// TODO: rate limit
func GenerateGitHubOAuthURLHandler(c *gin.Context) {
	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "internal", "message": "Internal server error"})
		return
	}

	url := oauthConfig.AuthCodeURL(state)

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

func LoginGitHubHandler(c *gin.Context) {
	var header struct {
		XOAuthState string `header:"X-Oauth-State" binding:"required"`
	}
	if err := c.ShouldBindHeader(&header); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "required", "message": "Missing X-OAuth-State header"})
		return
	}

	var query struct {
		State string `form:"state" binding:"required"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "required", "message": "Missing state query parameter"})
		return
	}

	var body struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "required", "message": "Missing code field"})
		return
	}

	if header.XOAuthState != query.State {
		c.JSON(http.StatusBadRequest, gin.H{"code": "mismatch", "message": "State mismatch"})
		return
	}

	ctx := context.Background()
	token, err := oauthConfig.Exchange(ctx, body.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "internal", "message": "Internal server error"})
		return
	}

	githubUser, err := getGitHubUser(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "Unauthorized"})
		return
	}

	user, err := FindOrCreateUser(githubUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "internal", "message": "Internal server error"})
		return
	}

	tokenPair, err := generateJWTTokenPair(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "internal", "message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
	})
}

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

	userObj := user.(*User)

	var req PostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	post, err := CreatePost(req.Title, req.Body, userObj.ID)
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

func QueryPostKeyHandler(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户信息获取失败"})
		return
	}

	userObj := user.(*User)

	c.JSON(http.StatusOK, gin.H{
		"post_key":   userObj.PostKey,
		"created_at": userObj.CreatedAt,
	})
}

type PasswordLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func LoginWithPasswordHandler(c *gin.Context) {
	var req PasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "invalid_request",
			"message": "Invalid request format",
		})
		return
	}

	// 验证用户密码
	user, err := ValidateUserPassword(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    "invalid_credentials",
			"message": "Invalid username or password",
		})
		return
	}

	// 生成JWT token
	tokenPair, err := generateJWTTokenPair(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "internal",
			"message": "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
	})
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func ChangePasswordHandler(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户信息获取失败"})
		return
	}

	userObj := user.(*User)

	var req PasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "invalid_request",
			"message": "Invalid request format",
		})
		return
	}

	// 验证当前密码
	if err := CheckPassword(req.CurrentPassword, userObj.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    "invalid_current_password",
			"message": "当前密码错误",
		})
		return
	}

	// 检查新密码是否与当前密码相同
	if req.CurrentPassword == req.NewPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "same_password",
			"message": "新密码不能与当前密码相同",
		})
		return
	}

	// 哈希新密码
	hashedNewPassword, err := HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "internal",
			"message": "Internal server error",
		})
		return
	}

	// 更新用户密码
	if err := db.Model(&User{}).Where("id = ?", userObj.ID).Update("password", hashedNewPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "internal",
			"message": "Failed to update password",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密码修改成功",
	})
}

func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "markpost is running",
	})
}
