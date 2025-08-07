# GitHub 登录 API 技术方案设计

## 架构概述

本方案基于现有的 Gin 框架和 GORM 数据库，集成 GitHub OAuth2 认证和 JWT 令牌生成，实现用户通过 GitHub 账户登录的功能。

## 技术栈

- **Web 框架**: Gin
- **数据库 ORM**: GORM (SQLite)
- **OAuth2 库**: `golang.org/x/oauth2`
- **JWT 库**: `github.com/appleboy/gin-jwt/v2`
- **配置管理**: Viper

## 系统架构

### 1. 配置层

扩展现有的配置结构，添加 GitHub OAuth2 和 JWT 相关配置：

```go
type Config struct {
    // 现有配置...

    // GitHub OAuth2 配置
    GitHub struct {
        ClientID     string `mapstructure:"client_id"`
        ClientSecret string `mapstructure:"client_secret"`
        RedirectURL  string `mapstructure:"redirect_url"`
    } `mapstructure:"github"`

    // JWT 配置
    JWT struct {
        SecretKey     string `mapstructure:"secret_key"`
        AccessTokenExpire  time.Duration `mapstructure:"access_token_expire"`
        RefreshTokenExpire time.Duration `mapstructure:"refresh_token_expire"`
    } `mapstructure:"jwt"`
}
```

### 2. 数据模型层

扩展用户模型，添加 GitHub ID 字段：

```go
type User struct {
    ID        int    `json:"id" gorm:"primaryKey;autoIncrement"`
    Username  string `json:"username" gorm:"unique;not null"`
    PostKey   string `json:"post_key" gorm:"not null"`
    GitHubID  *int64 `json:"github_id" gorm:"uniqueIndex"` // 使用指针类型，允许 NULL
}
```

**数据库迁移解决方案：**

由于数据库中已有数据，GitHubID 字段设置为可空（使用指针类型），避免迁移问题：

1. 新用户通过 GitHub 登录时，GitHubID 会被设置
2. 现有用户可以通过其他方式登录，GitHubID 保持为 NULL
3. 当现有用户首次通过 GitHub 登录时，系统会更新其 GitHubID 字段

### 3. 处理层

保持与现有目录结构一致，在 `handlers.go` 中添加 GitHub 认证相关处理函数：

```go
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

// 生成 GitHub 授权 URL
func GenerateGitHubAuthURL(c *gin.Context)

// 处理 GitHub 回调
func HandleGitHubCallback(c *gin.Context)

// 辅助函数
func initGitHubOAuth()
func exchangeCodeForToken(code string) (*oauth2.Token, error)
func getUserInfo(token *oauth2.Token) (*GitHubUser, error)
func findOrCreateUser(githubUser *GitHubUser) (*User, error)
func generateTokenPair(user *User) (*TokenPair, error)
```

## 数据流设计

### GitHub 登录流程

1. **前端请求授权 URL**: 前端调用 `/auth/github/url` 获取授权 URL
2. **后端生成授权 URL**: 后端生成包含 state 参数的 GitHub 授权 URL
3. **前端重定向**: 前端重定向用户到 GitHub 授权页面
4. **GitHub 授权**: 用户在 GitHub 完成授权
5. **回调处理**: GitHub 重定向到 `/auth/github/callback`
6. **令牌交换**: 后端使用授权码换取访问令牌
7. **用户信息获取**: 使用访问令牌获取用户 GitHub ID
8. **用户关联**: 查找或创建本地用户记录
9. **JWT 生成**: 生成访问令牌和刷新令牌
10. **响应返回**: 返回用户信息和 JWT 令牌给前端

## 安全设计

### 1. OAuth2 安全

- 使用 HTTPS 进行所有通信
- 验证 state 参数防止 CSRF 攻击
- 安全存储 Client Secret

### 2. JWT 安全

- 使用强密钥签名 JWT
- 设置合理的令牌过期时间（访问令牌 24 小时，刷新令牌 1 个月）

### 3. 数据库安全

- GitHub ID 使用唯一索引
- 防止重复用户创建
- 数据验证和清理

## 错误处理策略

### 1. OAuth2 错误

- 授权码无效: 返回 400 Bad Request
- GitHub API 错误: 返回 503 Service Unavailable
- 配置错误: 返回 500 Internal Server Error

### 2. JWT 错误

- 令牌生成失败: 返回 500 Internal Server Error

### 3. 数据库错误

- 连接失败: 返回 500 Internal Server Error
- 唯一约束冲突: 返回 409 Conflict

## 配置示例

```toml
[github]
client_id = "your_github_client_id"
client_secret = "your_github_client_secret"
redirect_url = "http://localhost:8080/auth/github/callback"

[jwt]
secret_key = "your_jwt_secret_key"
access_token_expire = "24h"
refresh_token_expire = "720h"  # 30 days
```

## 路由设计

```go
// 认证相关路由
auth := r.Group("/auth")
{
    auth.GET("/github/url", GenerateGitHubAuthURL)      // 生成授权 URL
    auth.GET("/github/callback", HandleGitHubCallback)  // 处理回调
}
```

## API 接口设计

### 1. 生成 GitHub 授权 URL

**请求**: `GET /auth/github/url`

**响应**:

```json
{
  "auth_url": "https://github.com/login/oauth/authorize?client_id=...&state=...",
  "state": "random_state_string"
}
```

### 2. GitHub 回调处理

**请求**: `GET /auth/github/callback?code=...&state=...`

**响应**:

```json
{
  "success": true,
  "user": {
    "id": 1,
    "username": "user123",
    "post_key": "key123",
    "github_id": 123456
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 86400
  },
  "message": "登录成功"
}
```

## 测试策略

### 1. 单元测试

- GitHub OAuth2 配置测试
- JWT 令牌生成测试
- 用户查找和创建逻辑测试
- 错误处理测试

### 2. 集成测试

- 完整的登录流程测试
- 数据库迁移测试
- 错误处理测试

### 3. 端到端测试

- 与 GitHub API 的集成测试
- 前端与后端的集成测试
