# AgentsMesh 普通用户认证授权规范文档

## 1. 概述

本文档详细描述 AgentsMesh 系统中普通用户（非多租户场景）的认证授权实现机制，包括登录流程、双 Token 自动刷新机制、API 端点定义、请求响应格式以及核心代码实现。

### 1.1 认证架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        Web 前端                                  │
│              (Next.js + Zustand + localStorage)                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                    HTTP/HTTPS + WebSocket
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Backend (Go + Gin)                          │
│  - 认证服务 (Auth Service)                                       │
│  - JWT 签发与验证                                                │
│  - Redis 存储 Refresh Token                                      │
│  - 认证中间件                                                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                    PostgreSQL + Redis
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      外部系统                                     │
│  - 邮件服务 (邮件验证、密码重置)                                  │
│  - OAuth 第三方登录 (GitHub, Google, GitLab, Gitee)             │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 认证流程时序图

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│   用户    │     │  前端    │     │  Backend  │     │  Redis   │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │                │
     │ 1.输入账号密码 │                │                │
     │──────────────>│                │                │
     │                │                │                │
     │                │ 2.POST /api/v1/auth/login        │
     │                │───────────────>│                │
     │                │                │                │
     │                │                │ 3.查询用户     │
     │                │                │──────────────>│
     │                │                │                │
     │                │                │ 4.bcrypt验证   │
     │                │                │<───────────────│
     │                │                │                │
     │                │                │ 5.生成Token对   │
     │                │                │────────────────>│
     │                │                │ (存储refresh)  │
     │                │                │                │
     │                │ 6.返回token    │                │
     │                │<───────────────│                │
     │                │                │                │
     │ 7.登录成功     │                │                │
     │<──────────────│                │                │
     │                │                │                │
```

## 2. 核心数据模型

### 2.1 用户实体

**文件位置**: `backend/internal/domain/user/user.go`

```go
type User struct {
    ID              int64      `gorm:"primaryKey" json:"id"`
    Email           string     `gorm:"uniqueIndex;not null" json:"email"`
    Username        string     `gorm:"uniqueIndex;not null" json:"username"`
    Name            string     `json:"name"`
    PasswordHash    *string    `gorm:"column:password_hash" json:"-"`
    AvatarURL       *string    `json:"avatar_url"`
    IsActive        bool       `gorm:"default:true" json:"is_active"`
    IsEmailVerified bool       `gorm:"default:false" json:"is_email_verified"`
    IsSystemAdmin   bool       `gorm:"default:false" json:"is_system_admin"`
    LastLoginAt     *time.Time `json:"last_login_at"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}
```

### 2.2 JWT Claims 结构

**文件位置**: `backend/pkg/auth/jwt.go` (第 16-23 行)

```go
type Claims struct {
    UserID         int64  `json:"user_id"`
    Email          string `json:"email"`
    Username       string `json:"username"`
    OrganizationID int64  `json:"organization_id,omitempty"`
    Role           string `json:"role,omitempty"`
    jwt.RegisteredClaims
}
```

### 2.3 Token 对结构

**文件位置**: `backend/internal/service/auth/auth_types.go` (第 60-66 行)

```go
type TokenPair struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
    TokenType    string    `json:"token_type"`
}
```

### 2.4 Refresh Token 数据结构

**文件位置**: `backend/internal/service/auth/auth_types.go` (第 68-75 行)

```go
type RefreshTokenData struct {
    UserID         int64     `json:"user_id"`
    OrganizationID int64     `json:"organization_id,omitempty"`
    Role           string    `json:"role,omitempty"`
    CreatedAt      time.Time `json:"created_at"`
    ExpiresAt      time.Time `json:"expires_at"`
}
```

### 2.5 登录结果结构

**文件位置**: `backend/internal/service/auth/auth_types.go` (第 95-101 行)

```go
type LoginResult struct {
    User         *user.User
    Token        string
    RefreshToken string
    ExpiresIn    int64
}
```

## 3. 登录流程实现

### 3.1 API 端点定义

**端点**: `POST /api/v1/auth/login`

**文件位置**: `backend/internal/api/rest/v1/auth_login.go` (第 18-52 行)

#### 3.1.1 请求格式

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**字段说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| email | string | 是 | 用户邮箱地址，格式需符合 email 规范 |
| password | string | 是 | 用户密码，最小长度无限制 |

#### 3.1.2 响应格式

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4...",
  "expires_in": 86400,
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "johndoe",
    "name": "John Doe",
    "avatar_url": null
  }
}
```

**字段说明**:

| 字段 | 类型 | 说明 |
|------|------|------|
| token | string | JWT Access Token |
| refresh_token | string | Refresh Token，用于刷新 Access Token |
| expires_in | int64 | Access Token 过期时间（秒），默认 86400 秒（24小时） |
| user | object | 用户基本信息 |
| user.id | int64 | 用户 ID |
| user.email | string | 用户邮箱 |
| user.username | string | 用户名 |
| user.name | string | 用户显示名称 |
| user.avatar_url | string/null | 用户头像 URL |

#### 3.1.3 登录处理器实现

```go
// 文件位置: backend/internal/api/rest/v1/auth_login.go

type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        apierr.ValidationError(c, err.Error())
        return
    }

    result, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
    if err != nil {
        if err == auth.ErrInvalidCredentials {
            apierr.Unauthorized(c, apierr.INVALID_CREDENTIALS, "Invalid email or password")
            return
        }
        if err == auth.ErrUserDisabled {
            apierr.Forbidden(c, apierr.USER_DISABLED, "Account is disabled")
            return
        }
        apierr.InternalError(c, "Login failed")
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "token":         result.Token,
        "refresh_token": result.RefreshToken,
        "expires_in":    result.ExpiresIn,
        "user": gin.H{
            "id":         result.User.ID,
            "email":      result.User.Email,
            "username":   result.User.Username,
            "name":       result.User.Name,
            "avatar_url": result.User.AvatarURL,
        },
    })
}
```

### 3.2 认证服务层

**文件位置**: `backend/internal/service/auth/auth_password.go` (第 10-33 行)

```go
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
    u, err := s.userService.Authenticate(ctx, email, password)
    if err != nil {
        if err == userService.ErrInvalidCredentials {
            return nil, ErrInvalidCredentials
        }
        if err == userService.ErrUserInactive {
            return nil, ErrUserDisabled
        }
        return nil, err
    }

    tokens, err := s.GenerateTokenPair(u, 0, "")
    if err != nil {
        return nil, err
    }

    return &LoginResult{
        User:         u,
        Token:        tokens.AccessToken,
        RefreshToken: tokens.RefreshToken,
        ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
    }, nil
}
```

### 3.3 用户密码验证

**文件位置**: `backend/internal/service/user/user_auth.go` (第 12-35 行)

```go
func (s *Service) Authenticate(ctx context.Context, email, password string) (*user.User, error) {
    u, err := s.GetByEmail(ctx, email)
    if err != nil {
        return nil, ErrInvalidCredentials
    }

    if !u.IsActive {
        return nil, ErrUserInactive
    }

    if u.PasswordHash == nil || *u.PasswordHash == "" {
        return nil, ErrInvalidCredentials
    }

    // 使用 bcrypt 进行密码验证
    if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
        return nil, ErrInvalidCredentials
    }

    // 更新最后登录时间
    now := time.Now()
    _ = s.repo.UpdateUserField(ctx, u.ID, "last_login_at", now)

    return u, nil
}
```

### 3.4 Token 配置

**文件位置**: `backend/internal/config/config_server.go` (第 51-55 行)

```go
type JWTConfig struct {
    Secret          string
    ExpirationHours int
}
```

**默认配置值** (`backend/internal/config/config.go` 第 84-87 行):

```go
JWT: JWTConfig{
    Secret:          getEnv("JWT_SECRET", "change-me-in-production"),
    ExpirationHours: getEnvInt("JWT_EXPIRATION_HOURS", 24),
},
```

**服务层配置初始化** (`backend/internal/service/auth/services_init.go` 第 94-96 行):

```go
JWTSecret:         cfg.JWT.Secret,
JWTExpiration:     time.Duration(cfg.JWT.ExpirationHours) * time.Hour,
RefreshExpiration: time.Duration(cfg.JWT.ExpirationHours*7) * time.Hour,
```

**Token 过期时间配置**:

| Token 类型 | 默认过期时间 | 公式 |
|------------|--------------|------|
| Access Token | 24 小时 | `JWTExpirationHours * 1` |
| Refresh Token | 168 小时（7天） | `JWTExpirationHours * 7` |

## 4. JWT Token 实现

### 4.1 Token 生成

**文件位置**: `backend/pkg/auth/jwt.go` (第 42-63 行)

```go
func (m *JWTManager) GenerateToken(userID int64, email, username string, orgID int64, role string) (string, error) {
    now := time.Now()
    expiresAt := now.Add(m.tokenDuration)

    claims := &Claims{
        UserID:         userID,
        Email:          email,
        Username:       username,
        OrganizationID: orgID,
        Role:           role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(expiresAt),
            IssuedAt:  jwt.NewNumericDate(now),
            NotBefore: jwt.NewNumericDate(now),
            Issuer:    m.issuer,
            Subject:   email,
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.secretKey)
}
```

### 4.2 Token 验证

**文件位置**: `backend/pkg/auth/jwt.go` (第 66-87 行)

```go
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, ErrInvalidToken
        }
        return m.secretKey, nil
    })

    if err != nil {
        if errors.Is(err, jwt.ErrTokenExpired) {
            return nil, ErrTokenExpired
        }
        return nil, ErrInvalidToken
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }

    return claims, nil
}
```

### 4.3 Token 对生成（高级）

**文件位置**: `backend/internal/service/auth/token_generate.go` (第 16-74 行)

```go
func (s *Service) GenerateTokenPair(u *user.User, orgID int64, role string) (*TokenPair, error) {
    now := time.Now()
    expiresAt := now.Add(s.config.JWTExpiration)
    refreshExpiresAt := now.Add(s.config.RefreshExpiration)

    // 生成 Access Token
    claims := &Claims{
        UserID:         u.ID,
        Email:          u.Email,
        Username:       u.Username,
        OrganizationID: orgID,
        Role:           role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(expiresAt),
            IssuedAt:  jwt.NewNumericDate(now),
            NotBefore: jwt.NewNumericDate(now),
            Issuer:    s.config.Issuer,
            Subject:   u.Email,
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    accessToken, err := token.SignedString([]byte(s.config.JWTSecret))
    if err != nil {
        return nil, err
    }

    // 生成 Refresh Token（随机 32 字节，Base64 编码）
    refreshBytes := make([]byte, 32)
    if _, err := rand.Read(refreshBytes); err != nil {
        return nil, err
    }
    refreshToken := base64.URLEncoding.EncodeToString(refreshBytes)

    // 将 Refresh Token 存储到 Redis
    if s.redis != nil {
        tokenData := &RefreshTokenData{
            UserID:         u.ID,
            OrganizationID: orgID,
            Role:           role,
            CreatedAt:      now,
            ExpiresAt:      refreshExpiresAt,
        }
        if err := s.storeRefreshToken(ctx, refreshToken, tokenData); err != nil {
            return nil, fmt.Errorf("failed to store refresh token: %w", err)
        }
    }

    return &TokenPair{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresAt:    expiresAt,
        TokenType:    "Bearer",
    }, nil
}
```

### 4.4 Refresh Token 存储

**文件位置**: `backend/internal/service/auth/token_generate.go` (第 76-88 行)

```go
func (s *Service) storeRefreshToken(ctx context.Context, refreshToken string, data *RefreshTokenData) error {
    // 使用 SHA-256 对 Token 进行哈希
    tokenHash := hashToken(refreshToken)
    key := refreshTokenPrefix + tokenHash  // 前缀: "auth:refresh:"

    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }

    // 计算 TTL 并存储到 Redis
    ttl := time.Until(data.ExpiresAt)
    return s.redis.Set(ctx, key, jsonData, ttl).Err()
}
```

**Token 哈希函数** (`backend/internal/service/auth/token_validate.go`):

```go
func hashToken(token string) string {
    hash := sha256.Sum256([]byte(token))
    return hex.EncodeToString(hash[:])
}
```

**Redis Key 格式**: `auth:refresh:{sha256(refresh_token)}`

## 5. Token 刷新机制

### 5.1 刷新 API 端点

**端点**: `POST /api/v1/auth/refresh`

**文件位置**: `backend/internal/api/rest/v1/auth_login.go` (第 129-154 行)

#### 5.1.1 请求格式

```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."
}
```

#### 5.1.2 响应格式

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "bmV3IHJlZnJlc2ggdG9rZW4...",
  "expires_in": 86400
}
```

#### 5.1.3 刷新处理器实现

```go
func (h *AuthHandler) RefreshToken(c *gin.Context) {
    var req struct {
        RefreshToken string `json:"refresh_token" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        apierr.ValidationError(c, err.Error())
        return
    }

    result, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
    if err != nil {
        if err == auth.ErrInvalidToken || err == auth.ErrInvalidRefreshToken {
            apierr.Unauthorized(c, apierr.INVALID_TOKEN, "Invalid refresh token")
            return
        }
        apierr.InternalError(c, "Failed to refresh token")
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "token":         result.Token,
        "refresh_token": result.RefreshToken,
        "expires_in":    result.ExpiresIn,
    })
}
```

### 5.2 刷新服务实现

**文件位置**: `backend/internal/service/auth/token_validate.go` (第 66-102 行)

```go
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*LoginResult, error) {
    if s.redis == nil {
        return nil, ErrInvalidRefreshToken
    }

    // 验证 Refresh Token
    tokenData, err := s.validateRefreshToken(ctx, refreshToken)
    if err != nil {
        return nil, err
    }

    // 获取用户信息
    u, err := s.userService.GetByID(ctx, tokenData.UserID)
    if err != nil {
        return nil, err
    }

    // 检查用户是否被禁用
    if !u.IsActive {
        return nil, ErrUserDisabled
    }

    // 撤销旧的 Refresh Token（Token 轮换，安全机制）
    if err := s.revokeRefreshToken(ctx, refreshToken); err != nil {
        // 记录日志但不中断流程
    }

    // 生成新的 Token 对
    tokens, err := s.GenerateTokenPairWithContext(ctx, u, tokenData.OrganizationID, tokenData.Role)
    if err != nil {
        return nil, err
    }

    return &LoginResult{
        User:         u,
        Token:        tokens.AccessToken,
        RefreshToken: tokens.RefreshToken,
        ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
    }, nil
}
```

### 5.3 Refresh Token 验证

**文件位置**: `backend/internal/service/auth/token_validate.go` (第 104-128 行)

```go
func (s *Service) validateRefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenData, error) {
    tokenHash := hashToken(refreshToken)
    key := refreshTokenPrefix + tokenHash

    // 从 Redis 获取存储的 Token 数据
    data, err := s.redis.Get(ctx, key).Result()
    if err != nil {
        if err == redis.Nil {
            return nil, ErrInvalidRefreshToken
        }
        return nil, fmt.Errorf("failed to validate refresh token: %w", err)
    }

    // 反序列化 Token 数据
    var tokenData RefreshTokenData
    if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
        return nil, fmt.Errorf("failed to parse refresh token data: %w", err)
    }

    // 检查是否过期
    if time.Now().After(tokenData.ExpiresAt) {
        s.redis.Del(ctx, key)
        return nil, ErrRefreshExpired
    }

    return &tokenData, nil
}
```

### 5.4 Token 轮换机制

刷新 Token 时，系统会：
1. 验证现有 Refresh Token 的有效性
2. 撤销（旧）Refresh Token（从 Redis 删除）
3. 生成并返回新的 Access Token + Refresh Token 对

这种机制（Token 轮换）提供了更好的安全性，即使旧的 Refresh Token 被窃取，也会在刷新后立即失效。

## 6. Logout 与 Token 撤销

### 6.1 登出 API 端点

**端点**: `POST /api/v1/auth/logout`

**认证**: 必选（需要有效的 Access Token）

#### 6.1.1 请求格式

```
Authorization: Bearer {access_token}
```

请求体为空。

#### 6.1.2 响应格式

```json
{
  "message": "Logged out successfully"
}
```

#### 6.1.3 登出处理器实现

**文件位置**: `backend/internal/api/rest/v1/auth_login.go` (第 156-167 行)

```go
// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
    // Get token from header
    token := c.GetHeader("Authorization")
    if token != "" && len(token) > 7 {
        token = token[7:] // Remove "Bearer " prefix
        // Optionally blacklist the token
        h.authService.RevokeToken(c.Request.Context(), token)
    }

    c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
```

### 6.2 Access Token 撤销

**文件位置**: `backend/internal/service/auth/token_revoke.go` (第 17-41 行)

```go
// RevokeToken revokes an access token by adding it to the blacklist
func (s *Service) RevokeToken(ctx context.Context, token string) error {
    if s.redis == nil {
        return nil
    }

    claims, err := s.ValidateToken(token)
    if err != nil && !errors.Is(err, ErrTokenExpired) {
        return nil
    }

    var ttl time.Duration
    if claims != nil && claims.ExpiresAt != nil {
        ttl = time.Until(claims.ExpiresAt.Time)
        if ttl <= 0 {
            return nil
        }
    } else {
        ttl = s.config.JWTExpiration
    }

    tokenHash := hashToken(token)
    key := tokenBlacklistKey + tokenHash
    return s.redis.Set(ctx, key, "1", ttl).Err()
}
```

**工作原理**:
1. 从 Access Token 中解析 Claims（忽略过期错误）
2. 计算 Token 的剩余有效时间（TTL）
3. 将 Token 哈希存入 Redis 黑名单，设置相同的 TTL
4. 在认证中间件中检查黑名单

**Redis Key 格式**: `auth:token:blacklist:{sha256(access_token)}`

### 6.3 Refresh Token 撤销

**文件位置**: `backend/internal/service/auth/token_revoke.go` (第 10-15 行)

```go
// revokeRefreshToken removes a refresh token from Redis
func (s *Service) revokeRefreshToken(ctx context.Context, refreshToken string) error {
    tokenHash := hashToken(refreshToken)
    key := refreshTokenPrefix + tokenHash
    return s.redis.Del(ctx, key).Err()
}
```

**工作原理**:
1. 对 Refresh Token 进行 SHA-256 哈希
2. 构建 Redis Key（格式: `auth:refresh:{hash}`）
3. 从 Redis 中删除该 Key

### 6.4 撤销用户所有 Token

**文件位置**: `backend/internal/service/auth/token_revoke.go` (第 43-66 行)

```go
// RevokeAllUserTokens revokes all tokens for a user
func (s *Service) RevokeAllUserTokens(ctx context.Context, userID int64) error {
    if s.redis == nil {
        return nil
    }

    pattern := refreshTokenPrefix + "*"
    iter := s.redis.Scan(ctx, 0, pattern, 0).Iterator()
    for iter.Next(ctx) {
        key := iter.Val()
        data, err := s.redis.Get(ctx, key).Result()
        if err != nil {
            continue
        }
        var tokenData RefreshTokenData
        if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
            continue
        }
        if tokenData.UserID == userID {
            s.redis.Del(ctx, key)
        }
    }
    return iter.Err()
}
```

**使用场景**:
- 用户主动注销账户
- 管理员禁用/删除用户
- 用户修改密码后强制下线其他设备

### 6.5 Token 黑名单检查

认证中间件在验证 Token 时，会检查 Token 是否在黑名单中：

```go
// 伪代码示例
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    // ... 解析 Token ...
    
    // 检查黑名单
    tokenHash := hashToken(tokenString)
    key := tokenBlacklistKey + tokenHash
    if exists, _ := m.redis.Exists(ctx, key).Result(); exists == 1 {
        return nil, ErrTokenRevoked
    }
    
    return claims, nil
}
```

### 6.6 登出流程时序图

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│   用户    │     │  前端    │     │  Backend  │     │  Redis   │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │                │
     │ 1.点击登出     │                │                │
     │──────────────>│                │                │
     │                │                │                │
     │                │ 2.POST /api/v1/auth/logout     │
     │                │ + Authorization: Bearer ...    │
     │                │───────────────>│                │
     │                │                │                │
     │                │                │ 3.解析 Token  │
     │                │                │   获取 Claims  │
     │                │                │                │
     │                │                │ 4.加入黑名单   │
     │                │                │───────────────>│
     │                │                │                │
     │                │ 5.返回成功    │                │
     │                │<───────────────│                │
     │                │                │                │
     │ 6.清除本地    │                │                │
     │   Token       │                │                │
     │<──────────────│                │                │
     │                │                │                │
```

## 7. 认证中间件

### 7.1 必选认证中间件

**文件位置**: `backend/internal/middleware/auth.go` (第 25-71 行)

```go
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        var tokenString string

        // 优先从 Authorization Header 获取
        authHeader := c.GetHeader("Authorization")
        if authHeader != "" {
            parts := strings.SplitN(authHeader, " ", 2)
            if len(parts) == 2 && parts[0] == "Bearer" {
                tokenString = parts[1]
            }
        }

        // 如果 Header 没有，尝试从 Query 参数获取（用于 WebSocket 连接）
        if tokenString == "" {
            tokenString = c.Query("token")
        }

        if tokenString == "" {
            apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "Authorization is required")
            return
        }

        // 解析和验证 JWT Token
        claims := &JWTClaims{}
        token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, errors.New("unexpected signing method")
            }
            return []byte(jwtSecret), nil
        })

        if err != nil || !token.Valid {
            apierr.AbortUnauthorized(c, apierr.INVALID_TOKEN, "Invalid or expired token")
            return
        }

        // 将用户信息设置到 Context
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("username", claims.Username)
        c.Set("claims", claims) // 用于 WebSocket 处理器的完整 Claims

        c.Next()
    }
}
```

### 7.2 可选认证中间件

**文件位置**: `backend/internal/middleware/auth.go` (第 91-122 行)

```go
func OptionalAuthMiddleware(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        var tokenString string

        authHeader := c.GetHeader("Authorization")
        if authHeader != "" {
            parts := strings.SplitN(authHeader, " ", 2)
            if len(parts) == 2 && parts[0] == "Bearer" {
                tokenString = parts[1]
            }
        }

        if tokenString == "" {
            // 没有 Token 时不终止请求，继续处理
            c.Next()
            return
        }

        claims := &JWTClaims{}
        token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, errors.New("unexpected signing method")
            }
            return []byte(jwtSecret), nil
        })

        if err != nil || !token.Valid {
            // Token 无效时不终止请求，继续处理
            c.Next()
            return
        }

        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("username", claims.Username)
        c.Set("claims", claims)

        c.Next()
    }
}
```

### 7.3 路由注册

**文件位置**: `backend/internal/api/rest/router.go`

#### 公开路由（无需认证）

```go
// 第 80-84 行
authHandler := v1.NewAuthHandler(svc.Auth, svc.User, emailSvc, cfg)
authGroup := apiV1.Group("/auth")
authGroup.Use(middleware.IPRateLimiter(redisClient, "auth", 20, time.Minute))
authHandler.RegisterRoutes(authGroup)
```

#### 受保护路由（需要认证）

```go
// 第 122-159 行
protected := apiV1.Group("")
protected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
{
    // 用户级别路由
    v1.RegisterUserRoutes(protected.Group("/users"), ...)
    
    // 组织路由
    v1.RegisterOrganizationRoutes(protected.Group("/orgs"), ...)
    
    // 组织范围的路由
    orgScoped := protected.Group("/orgs/:slug")
    orgScoped.Use(middleware.TenantMiddleware(svc.Org))
    // ...
}
```

## 8. API 端点总览

### 7.1 认证相关端点

| 端点 | 方法 | 认证 | 说明 |
|------|------|------|------|
| `/api/v1/auth/login` | POST | 否 | 用户登录 |
| `/api/v1/auth/register` | POST | 否 | 用户注册 |
| `/api/v1/auth/refresh` | POST | 否 | 刷新 Access Token |
| `/api/v1/auth/logout` | POST | 是 | 用户登出 |
| `/api/v1/auth/verify-email` | POST | 否 | 邮箱验证 |
| `/api/v1/auth/resend-verification` | POST | 否 | 重新发送验证邮件 |
| `/api/v1/auth/forgot-password` | POST | 否 | 忘记密码 |
| `/api/v1/auth/reset-password` | POST | 否 | 重置密码 |

### 7.2 OAuth 端点

| 端点 | 方法 | 认证 | 说明 |
|------|------|------|------|
| `/api/v1/auth/oauth/github` | GET | 否 | GitHub OAuth 跳转 |
| `/api/v1/auth/oauth/github/callback` | GET | 否 | GitHub OAuth 回调 |
| `/api/v1/auth/oauth/google` | GET | 否 | Google OAuth 跳转 |
| `/api/v1/auth/oauth/google/callback` | GET | 否 | Google OAuth 回调 |
| `/api/v1/auth/oauth/gitlab` | GET | 否 | GitLab OAuth 跳转 |
| `/api/v1/auth/oauth/gitlab/callback` | GET | 否 | GitLab OAuth 回调 |
| `/api/v1/auth/oauth/gitee` | GET | 否 | Gitee OAuth 跳转 |
| `/api/v1/auth/oauth/gitee/callback` | GET | 否 | Gitee OAuth 回调 |

### 7.3 受保护端点

所有通过 `protected.Use(middleware.AuthMiddleware(...))` 注册的路由都需要有效的 Access Token。

## 9. 前端实现

### 8.1 Auth Store

**文件位置**: `web/src/stores/auth.ts`

```typescript
interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  currentOrg: Organization | null;
  organizations: Organization[];
  _hasHydrated: boolean;

  setAuth: (token: string, user: User, refreshToken?: string) => void;
  setTokens: (token: string, refreshToken: string) => void;
  logout: () => void;
  isAuthenticated: () => boolean;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      refreshToken: null,
      user: null,
      
      setAuth: (token, user, refreshToken) => 
        set({ token, user, refreshToken: refreshToken || null }),
      
      setTokens: (token, refreshToken) => 
        set({ token, refreshToken }),
      
      logout: () => 
        set({ token: null, refreshToken: null, user: null, currentOrg: null, organizations: [] }),
      
      isAuthenticated: () => !!get().token,
    }),
    {
      name: 'agentsmesh-auth',  // localStorage 键名
      partialize: (state) => ({ 
        token: state.token, 
        refreshToken: state.refreshToken,
        user: state.user,
        currentOrg: state.currentOrg,
        organizations: state.organizations,
      }),
    }
  )
);
```

### 8.2 Auth API 客户端

**文件位置**: `web/src/lib/api/auth.ts`

```typescript
export const authApi = {
  login: (email: string, password: string) =>
    request<{
      token: string;
      refresh_token: string;
      expires_in: number;
      user: { id: number; email: string; username: string; name?: string };
    }>(
      '/api/v1/auth/login',
      {
        method: 'POST',
        body: { email, password },
        skipAuthRefresh: true,  // 登录接口不触发刷新
      }
    ),

  register: (data: { 
    email: string; 
    username: string; 
    password: string; 
    name?: string;
  }) =>
    request<{
      token: string;
      refresh_token: string;
      expires_in: number;
      user: { id: number; email: string; username: string; name?: string };
      message: string;
    }>(
      '/api/v1/auth/register',
      { method: 'POST', body: data, skipAuthRefresh: true }
    ),

  logout: () => 
    request('/api/v1/auth/logout', { method: 'POST' }),

  refreshToken: (refreshToken: string) =>
    request<{
      token: string;
      refresh_token: string;
      expires_in: number;
    }>(
      '/api/v1/auth/refresh',
      {
        method: 'POST',
        body: { refresh_token: refreshToken },
        skipAuthRefresh: true,  // 刷新接口不触发递归刷新
      }
    ),
};
```

### 8.3 自动 Token 刷新机制

**文件位置**: `web/src/lib/api/base.ts` (第 50-92 行)

```typescript
let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

async function refreshAccessToken(): Promise<boolean> {
  const { refreshToken, setTokens, logout } = useAuthStore.getState();

  if (!refreshToken) {
    return false;
  }

  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!response.ok) {
      logout();
      return false;
    }

    const data = await response.json();
    setTokens(data.token, data.refresh_token);
    return true;
  } catch {
    logout();
    return false;
  }
}

export async function handleTokenRefresh(): Promise<boolean> {
  // 防止多个请求同时触发刷新
  if (isRefreshing && refreshPromise) {
    return refreshPromise;
  }

  isRefreshing = true;
  refreshPromise = refreshAccessToken().finally(() => {
    isRefreshing = false;
    refreshPromise = null;
  });

  return refreshPromise;
}
```

### 8.4 请求拦截器

**文件位置**: `web/src/lib/api/base.ts`

```typescript
export async function request<T>(
  url: string,
  options: RequestOptions = {}
): Promise<T> {
  const { token, logout } = useAuthStore.getState();
  const { skipAuthRefresh = false, ...fetchOptions } = options;

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  };

  // 添加 Access Token
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE_URL}${url}`, {
    ...fetchOptions,
    headers,
  });

  // 401 处理：尝试刷新 Token
  if (response.status === 401 && !skipAuthRefresh) {
    const refreshed = await handleTokenRefresh();
    
    if (refreshed) {
      // 重新获取 Token 并重试请求
      const newToken = useAuthStore.getState().token;
      headers['Authorization'] = `Bearer ${newToken}`;
      
      const retryResponse = await fetch(`${API_BASE_URL}${url}`, {
        ...fetchOptions,
        headers,
      });
      
      if (!retryResponse.ok) {
        const error = await retryResponse.json();
        throw new Error(error.message || 'Request failed');
      }
      
      return retryResponse.json();
    } else {
      // 刷新失败，登出用户
      logout();
      throw new Error('Session expired');
    }
  }

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || 'Request failed');
  }

  return response.json();
}
```

### 8.5 前端刷新流程图

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  API 请求 │     │  拦截器  │     │  Backend │     │  Redis  │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │                │
     │ 1.发送请求     │                │                │
     │──────────────>│                │                │
     │                │                │                │
     │                │ 2.添加 Bearer  │                │
     │                │    Token       │                │
     │                │───────────────>│                │
     │                │                │                │
     │                │                │ 3.验证 Token   │
     │                │                │───────────────>│
     │                │                │                │
     │                │ 4.返回 401     │                │
     │                │<───────────────│                │
     │                │                │                │
     │                │ 5.检查是否     │                │
     │                │    正在刷新    │                │
     │                │                │                │
     │                │ 6.调用刷新 API │                │
     │                │───────────────>│                │
     │                │                │                │
     │                │                │ 7.验证 Refresh │
     │                │                │    Token       │
     │                │                │───────────────>│
     │                │                │                │
     │                │ 8.返回新 Token │                │
     │                │<───────────────│                │
     │                │                │                │
     │                │ 9.更新 Store   │                │
     │                │    并重试      │                │
     │                │───────────────>│                │
     │                │                │                │
```

## 10. 错误码定义

### 9.1 认证错误码

| 错误码 | HTTP 状态码 | 说明 |
|--------|-------------|------|
| `AUTH_REQUIRED` | 401 | 需要认证 |
| `INVALID_TOKEN` | 401 | Token 无效或已过期 |
| `INVALID_CREDENTIALS` | 401 | 用户名或密码错误 |
| `USER_DISABLED` | 403 | 用户账户已被禁用 |
| `REFRESH_EXPIRED` | 401 | Refresh Token 已过期 |
| `TOKEN_REVOKED` | 401 | Token 已被撤销 |

### 9.2 业务错误码

| 错误码 | HTTP 状态码 | 说明 |
|--------|-------------|------|
| `VALIDATION_ERROR` | 400 | 请求参数验证失败 |
| `USER_NOT_FOUND` | 404 | 用户不存在 |
| `EMAIL_EXISTS` | 409 | 邮箱已被注册 |
| `USERNAME_EXISTS` | 409 | 用户名已被使用 |

## 11. 安全最佳实践

### 10.1 Token 安全

1. **Token 存储**: Access Token 存储在内存中（Zustand Store），Refresh Token 存储在 localStorage
2. **Token 传输**: 始终使用 HTTPS 传输
3. **Token 轮换**: 每次刷新都会生成新的 Token 对，旧的 Refresh Token 立即失效
4. **Token 哈希**: Refresh Token 在 Redis 中以 SHA-256 哈希形式存储

### 10.2 密码安全

1. **密码加密**: 使用 bcrypt 进行密码哈希
2. **密码验证**: 不返回密码哈希值到前端
3. **密码重置**: 通过邮件发送临时重置链接

### 10.3 中间件安全

1. **必选认证**: 使用 `AuthMiddleware` 保护敏感 API
2. **可选认证**: 使用 `OptionalAuthMiddleware` 允许访客访问
3. **IP 限流**: 对登录接口实施 IP 级别限流

## 12. 相关文件索引

| 功能模块 | 文件路径 | 说明 |
|----------|----------|------|
| 登录处理器 | `backend/internal/api/rest/v1/auth_login.go` | HTTP 请求处理 |
| 认证服务 | `backend/internal/service/auth/auth_password.go` | 业务逻辑层 |
| 用户认证 | `backend/internal/service/user/user_auth.go` | 密码验证 |
| Token 生成 | `backend/internal/service/auth/token_generate.go` | Token 对生成 |
| Token 验证 | `backend/internal/service/auth/token_validate.go` | Token 刷新验证 |
| JWT 工具 | `backend/pkg/auth/jwt.go` | JWT 签发验证 |
| 认证中间件 | `backend/internal/middleware/auth.go` | 请求认证 |
| 路由注册 | `backend/internal/api/rest/router.go` | 路由与中间件配置 |
| 前端 Auth Store | `web/src/stores/auth.ts` | 状态管理 |
| 前端 API | `web/src/lib/api/auth.ts` | API 调用 |
| 前端请求拦截 | `web/src/lib/api/base.ts` | Token 自动刷新 |

## 13. 配置环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `JWT_SECRET` | `change-me-in-production` | JWT 签名密钥 |
| `JWT_EXPIRATION_HOURS` | `24` | Access Token 过期时间（小时） |

## 14. 总结

本文档详细描述了 AgentsMesh 系统中普通用户的认证授权实现机制：

1. **登录流程**: 用户通过邮箱密码登录，后端使用 bcrypt 验证密码，生成并返回 Access Token 和 Refresh Token
2. **双 Token 机制**: Access Token（24小时）用于 API 认证，Refresh Token（7天）用于自动刷新
3. **自动刷新**: 前端在 401 响应时自动调用刷新接口，实现无感知的 Token 续期
4. **Token 轮换**: 每次刷新都会生成新 Token 对，旧 Token 立即失效，提供安全保障
5. **中间件保护**: 使用 Gin 中间件对受保护 API 进行认证验证
6. **Redis 存储**: Refresh Token 存储在 Redis 中，支持分布式验证和自动过期清理
