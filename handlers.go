package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/yuin/goldmark"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GitHub OAuth2 配置
var githubOAuthConfig *oauth2.Config

// GitHub 用户信息结构
type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

// JWT 令牌对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// JWT Claims
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// 初始化 GitHub OAuth2 配置
func initGitHubOAuth() {
	githubOAuthConfig = &oauth2.Config{
		ClientID:     config.GitHub.ClientID,
		ClientSecret: config.GitHub.ClientSecret,
		RedirectURL:  config.GitHub.RedirectURL,
		Scopes:       []string{"read:user"},
		Endpoint:     github.Endpoint,
	}
}

// 生成随机 state 参数
func generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// 使用授权码交换访问令牌
func exchangeCodeForToken(code string) (*oauth2.Token, error) {
	ctx := context.Background()
	token, err := githubOAuthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// 获取 GitHub 用户信息
func getUserInfo(token *oauth2.Token) (*GitHubUser, error) {
	ctx := context.Background()
	client := githubOAuthConfig.Client(ctx, token)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var githubUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, err
	}

	return &githubUser, nil
}

// 生成 JWT 令牌对
func generateTokenPair(user *User) (*TokenPair, error) {
	now := time.Now()

	// 生成访问令牌
	accessClaims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(config.JWT.AccessTokenExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(config.JWT.SecretKey))
	if err != nil {
		return nil, err
	}

	// 生成刷新令牌
	refreshClaims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(config.JWT.RefreshTokenExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(config.JWT.SecretKey))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(config.JWT.AccessTokenExpire.Seconds()),
	}, nil
}

// 生成 GitHub 授权 URL
func GenerateGitHubAuthURL(c *gin.Context) {
	// 检查配置是否完整
	if config.GitHub.ClientID == "" || config.GitHub.ClientSecret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "GitHub OAuth2 配置不完整",
		})
		return
	}

	// 获取自定义的 redirect_uri 参数
	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		redirectURI = config.GitHub.RedirectURL
	}

	// 创建临时的 OAuth2 配置
	tempOAuthConfig := &oauth2.Config{
		ClientID:     config.GitHub.ClientID,
		ClientSecret: config.GitHub.ClientSecret,
		RedirectURL:  redirectURI,
		Scopes:       []string{"read:user"},
		Endpoint:     github.Endpoint,
	}

	// 生成随机 state 参数
	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "生成 state 参数失败",
		})
		return
	}

	// 生成授权 URL
	authURL := tempOAuthConfig.AuthCodeURL(state)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// 处理 GitHub 回调
func HandleGitHubCallback(c *gin.Context) {
	// 获取授权码和错误参数
	code := c.Query("code")
	error := c.Query("error")

	// 检查是否有错误
	if error != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "GitHub 授权失败: " + error,
		})
		return
	}

	// 验证必要参数
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少授权码",
		})
		return
	}

	// 使用授权码交换访问令牌
	token, err := exchangeCodeForToken(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "交换访问令牌失败",
		})
		log.Printf("交换访问令牌失败: %v", err)
		return
	}

	// 获取 GitHub 用户信息
	githubUser, err := getUserInfo(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取用户信息失败",
		})
		log.Printf("获取用户信息失败: %v", err)
		return
	}

	// 查找或创建用户
	user, err := FindOrCreateUser(githubUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "用户处理失败",
		})
		log.Printf("用户处理失败: %v", err)
		return
	}

	// 生成 JWT 令牌
	tokenPair, err := generateTokenPair(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "生成令牌失败",
		})
		log.Printf("生成令牌失败: %v", err)
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"post_key":  user.PostKey,
			"github_id": user.GitHubID,
		},
		"tokens": gin.H{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"expires_in":    tokenPair.ExpiresIn,
		},
		"message": "登录成功",
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

func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "markpost is running",
	})
}
