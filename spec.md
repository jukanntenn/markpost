# Go Web Service 完整规范说明

> **版本**: 1.0.0
> **日期**: 2026-03-11
> **状态**: 参考实现
> **基于**: AgentsMesh 生产项目

本文档定义了生产级 Go Web 服务的完整架构规范，涵盖技术栈、项目结构、设计模式、测试策略等核心内容。另一个项目可完全参照此 specification 实现架构一致的 Go Web 服务。

---

## 目录

1. [技术栈](#1-技术栈)
2. [项目结构](#2-项目结构)
3. [架构设计](#3-架构设计)
4. [数据库集成](#4-数据库集成)
5. [API 设计模式](#5-api-设计模式)
6. [认证与授权](#6-认证与授权)
7. [错误处理](#7-错误处理)
8. [配置管理](#8-配置管理)
9. [测试策略](#9-测试策略)
10. [前端集成](#10-前端集成)
11. [开发环境](#11-开发环境)
12. [质量保证](#12-质量保证)

---

## 1. 技术栈

### 1.1 核心框架

| 组件 | 技术选型 | 版本 | 说明 |
|------|---------|------|------|
| **HTTP 框架** | Gin | v1.9+ | 高性能路由 + 中间件 |
| **ORM** | GORM | v1.25+ | 数据库操作抽象 |
| **数据库** | PostgreSQL | 16+ | 主数据存储 |
| **缓存** | Redis | 7+ | 会话、OAuth State |
| **认证** | JWT | - | 无状态认证 |
| **配置** | Viper | - | 配置文件管理 |
| **日志** | slog (Go 1.21+) | - | 结构化日志 |

### 1.2 依赖管理

```bash
# go.mod 核心依赖
require (
    github.com/gin-gonic/gin v1.9.1
    gorm.io/gorm v1.25.12
    gorm.io/driver/postgres v1.5.9
    github.com/redis/go-redis/v9 v9.7.0
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/spf13/viper v1.18.2
)
```

### 1.3 Go 版本要求

```
go 1.24.0+
```

---

## 2. 项目结构

### 2.1 标准目录布局

```
project-root/
├── backend/                      # Go 后端
│   ├── cmd/
│   │   └── server/               # 应用入口
│   │       └── main.go
│   ├── internal/                 # 私有代码
│   │   ├── api/                  # API 层
│   │   │   ├── rest/             # REST API
│   │   │   │   └── v1/           # API 版本
│   │   │   │       ├── users.go
│   │   │   │       ├── auth.go
│   │   │   │       └── admin/    # 管理员 API
│   │   │   └── grpc/             # gRPC (可选)
│   │   ├── service/              # 业务逻辑层
│   │   │   ├── user/
│   │   │   ├── auth/
│   │   │   └── billing/
│   │   ├── domain/               # 领域层 (DDD)
│   │   │   ├── user/
│   │   │   │   ├── user.go       # 实体
│   │   │   │   └── repository.go  # 仓储接口
│   │   │   └── organization/
│   │   ├── infra/                # 基础设施层
│   │   │   ├── database/         # DB 实现
│   │   │   └── cache/            # Redis 实现
│   │   ├── middleware/           # 中间件
│   │   │   ├── auth.go
│   │   │   ├── tenant.go
│   │   │   └── cors.go
│   │   └── config/               # 配置
│   │       └── config.go
│   ├── pkg/                      # 公共库
│   │   ├── apierr/               # 错误处理
│   │   ├── auth/                 # JWT/OAuth
│   │   ├── crypto/               # 加密工具
│   │   └── i18n/                 # 国际化
│   ├── migrations/               # 数据库迁移
│   │   ├── 000001_init_schema.up.sql
│   │   └── 000001_init_schema.down.sql
│   ├── .golangci.yml             # Lint 配置
│   ├── go.mod
│   └── go.sum
├── web/                          # Next.js 前端
│   ├── src/
│   │   ├── app/                  # App Router
│   │   ├── components/           # React 组件
│   │   ├── lib/                  # 工具库
│   │   ├── hooks/                # 自定义 Hooks
│   │   ├── stores/               # 状态管理
│   │   └── messages/             # i18n
│   ├── eslint.config.mjs
│   ├── tsconfig.json
│   └── package.json
└── deploy/                       # 部署配置
    └── dev/                      # 开发环境
        ├── dev.sh                # 一键启动脚本
        ├── docker-compose.yml
        └── .env                  # 环境变量
```

### 2.2 命名约定

**目录**:
- 小写，连字符分隔: `user-service`, `billing-handler`
- 复数形式用于集合: `migrations/`, `handlers/`

**文件**:
- 小写下划线: `user_service.go`, `auth_middleware.go`
- 测试文件: `user_service_test.go`
- Mock 文件: `mock_auth_test.go`

---

## 3. 架构设计

### 3.1 DDD 分层架构

```
┌─────────────────────────────────────────────────────┐
│                    API Layer                         │
│         REST Handlers (internal/api/rest)            │
│   - 参数解析和验证                                    │
│   - 调用 Service 层                                   │
│   - 返回 HTTP 响应                                    │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│                  Service Layer                       │
│        业务逻辑编排 (internal/service)                │
│   - 实现 Domain 定义的接口                            │
│   - 编排多个 Domain 聚合                              │
│   - 事务管理                                          │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│                   Domain Layer                       │
│   领域模型 + 仓储接口 (internal/domain)               │
│   - 实体 (Entity)                                     │
│   - 值对象 (Value Object)                             │
│   - 仓储接口 (Repository Interface)                   │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│              Infrastructure Layer                    │
│    技术实现 (internal/infra)                          │
│   - Repository 实现 (PostgreSQL)                      │
│   - Cache 实现 (Redis)                                │
│   - 外部服务客户端                                    │
└─────────────────────────────────────────────────────┘
```

### 3.2 依赖倒置原则

```go
// Domain 层定义接口 (internal/domain/user/repository.go)
package user

type Repository interface {
    GetByID(ctx context.Context, id int64) (*User, error)
    Create(ctx context.Context, u *User) error
    // ...
}

// Service 层依赖接口 (internal/service/user/service.go)
package user

type Service struct {
    repo domain.Repository  // 依赖抽象
}

func NewService(repo domain.Repository) *Service {
    return &Service{repo: repo}
}

// Infra 层实现接口 (internal/infra/database/user_repository.go)
package database

type UserRepo struct {
    db *gorm.DB
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
    // PostgreSQL 实现
}
```

---

## 4. 数据库集成

### 4.1 连接配置

```go
// internal/config/database.go
package config

type DatabaseConfig struct {
    Host     string
    Port     int
    User     string
    Password string
    DBName   string
    SSLMode  string
}

func (c *DatabaseConfig) DSN() string {
    return fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
    )
}
```

### 4.2 GORM 初始化

```go
// internal/infra/database/db.go
package database

import (
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

func NewDB(dsn string) (*gorm.DB, error) {
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
        NowFunc: func() time.Time {
            return time.Now().UTC()
        },
    })
    if err != nil {
        return nil, err
    }

    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }

    // 连接池配置
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)

    return db, nil
}
```

### 4.3 数据库迁移

使用 `golang-migrate` 管理版本化迁移：

```bash
# 创建迁移
migrate create -ext sql -dir backend/migrations -seq add_feature

# 向上迁移
migrate -path backend/migrations -database "$DATABASE_URL" up

# 向下迁移
migrate -path backend/migrations -database "$DATABASE_URL" down 1
```

**迁移文件命名**:
```
000001_init_schema.up.sql
000001_init_schema.down.sql
000002_add_feature.up.sql
000002_add_feature.down.sql
```

### 4.4 Repository 实现模式

```go
// internal/infra/database/user_repository.go
package database

type userRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
    var u domain.User
    err := r.db.WithContext(ctx).First(&u, id).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, domain.ErrNotFound
        }
        return nil, err
    }
    return &u, nil
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
    return r.db.WithContext(ctx).Create(u).Error
}
```

---

## 5. API 设计模式

### 5.1 请求参数绑定

**查询参数 (GET)**:

```go
type ListPodsRequest struct {
    Status  string `form:"status"`
    Limit   int    `form:"limit"`
    Offset  int    `form:"offset"`
}

func (h *PodHandler) ListPods(c *gin.Context) {
    var req ListPodsRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        apierr.ValidationError(c, err.Error())
        return
    }

    // 设置默认值
    if req.Limit == 0 {
        req.Limit = 20
    }

    // 调用 Service
    pods, total, err := h.service.ListPods(c.Request.Context(), req)
    // ...
}
```

**JSON Body (POST/PUT)**:

```go
type CreatePodRequest struct {
    Name        string `json:"name" binding:"required,min=1,max=255"`
    AgentTypeID int64  `json:"agent_type_id" binding:"required,min=1"`
    RunnerID    *int64 `json:"runner_id" binding:"omitempty,min=1"`
}

func (h *PodHandler) CreatePod(c *gin.Context) {
    var req CreatePodRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        apierr.ValidationError(c, err.Error())
        return
    }

    pod, err := h.service.CreatePod(c.Request.Context(), req)
    // ...
}
```

### 5.2 常用验证标签

| 标签 | 说明 | 示例 |
|------|------|------|
| `required` | 必填 | `binding:"required"` |
| `min=N` | 最小值/长度 | `binding:"min=8"` |
| `max=N` | 最大值/长度 | `binding:"max=100"` |
| `email` | 邮箱格式 | `binding:"email"` |
| `oneof=X Y` | 枚举值 | `binding:"oneof=active inactive"` |
| `dive` | 验证数组元素 | `binding:"required,dive,required"` |

### 5.3 响应格式

**成功响应**:

```go
// 单个资源
c.JSON(http.StatusOK, gin.H{
    "pod": pod,
})

// 列表 + 分页
c.JSON(http.StatusOK, gin.H{
    "pods":   pods,
    "total":  total,
    "limit":  limit,
    "offset": offset,
})
```

**错误响应** (统一格式):

```json
{
  "error": "Validation failed: Email is required",
  "code": "VALIDATION_FAILED"
}
```

### 5.4 HTTP 状态码规范

| 状态码 | 使用场景 | 示例 |
|--------|---------|------|
| 200 | 成功 | GET /api/v1/pods |
| 201 | 创建成功 | POST /api/v1/pods |
| 400 | 参数错误 | 验证失败 |
| 401 | 未认证 | 缺少 Token |
| 403 | 无权限 | 非管理员操作 |
| 404 | 资源不存在 | GET /api/v1/pods/999 |
| 500 | 服务器错误 | 数据库连接失败 |

---

## 6. 认证与授权

### 6.1 JWT 认证流程

```
┌─────────┐                    ┌──────────┐
┌─────────┐                    │          │
│ Client  │────POST /auth/login──▶│ Backend  │
└─────────┘                    └──────────┘
                                        │
                                        ▼
                                 ┌─────────────┐
                                 │ Validate    │
                                 │ Credentials │
                                 └─────────────┘
                                        │
                                        ▼
                                 ┌─────────────┐
                                 │ Generate    │
                                 │ JWT Token   │
                                 └─────────────┘
                                        │
┌─────────┐                          │
│         │◀────— JWT Token ──────────┘
└─────────┘
                                        │
┌─────────┐                          │
│ Client  │────GET /api/v1/pods───────▶│ Backend  │
│         │   Authorization: Bearer   └──────────┘
└─────────┘
                                        │
                                        ▼
                                 ┌─────────────┐
                                 │ JWT Middleware│
                                 │ Verify Token │
                                 └─────────────┘
```

### 6.2 JWT Middleware 实现

```go
// internal/middleware/auth.go
package middleware

type AuthMiddleware struct {
    jwtSecret []byte
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Missing auth token")
            c.Abort()
            return
        }

        // 移除 "Bearer " 前缀
        token = strings.TrimPrefix(token, "Bearer ")

        claims, err := auth.ValidateJWT(token, m.jwtSecret)
        if err != nil {
            apierr.Unauthorized(c, apierr.INVALID_TOKEN, "Invalid token")
            c.Abort()
            return
        }

        // 设置用户上下文
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Next()
    }
}
```

### 6.3 租户隔离 Middleware

```go
// internal/middleware/tenant.go
package middleware

func TenantMiddleware(orgService OrganizationService) gin.HandlerFunc {
    return func(c *gin.Context) {
        slug := c.Param("slug")
        if slug == "" {
            apierr.BadRequest(c, apierr.INVALID_INPUT, "Organization slug required")
            c.Abort()
            return
        }

        userID := c.GetInt64("user_id")
        org, err := orgService.GetBySlugAndUserID(c.Request.Context(), slug, userID)
        if err != nil {
            apierr.ForbiddenAccess(c)
            c.Abort()
            return
        }

        // 设置租户上下文
        c.Set("tenant", &Tenant{
            OrganizationID: org.ID,
            UserID:         userID,
            UserRole:       org.Role,
        })

        c.Next()
    }
}
```

### 6.4 路由配置

```go
// cmd/server/main.go
func setupRoutes(r *gin.Engine) {
    authMiddleware := middleware.NewAuthMiddleware(config.JWTSecret)

    // 公开路由
    r.POST("/api/v1/auth/login", handlers.Login)
    r.POST("/api/v1/auth/register", handlers.Register)

    // 需要认证的路由
    auth := r.Group("/api/v1")
    auth.Use(authMiddleware.RequireAuth())
    {
        // 需要租户隔离的路由
        org := auth.Group("/organizations/:slug")
        org.Use(middleware.TenantMiddleware(orgService))
        {
            org.GET("/pods", handlers.ListPods)
            org.POST("/pods", handlers.CreatePod)
            org.GET("/tickets", handlers.ListTickets)
        }
    }
}
```

---

## 7. 错误处理

### 7.1 错误包设计

```go
// pkg/apierr/apierr.go
package apierr

import (
    "github.com/gin-gonic/gin"
    "net/http"
)

type ErrorResponse struct {
    Error string `json:"error"`
    Code  string `json:"code"`
}

// 响应函数
func ValidationError(c *gin.Context, msg string) {
    c.JSON(http.StatusBadRequest, ErrorResponse{
        Error: msg,
        Code:  "VALIDATION_FAILED",
    })
}

func Unauthorized(c *gin.Context, code, msg string) {
    c.JSON(http.StatusUnauthorized, ErrorResponse{
        Error: msg,
        Code:  code,
    })
}

func ForbiddenAccess(c *gin.Context) {
    c.JSON(http.StatusForbidden, ErrorResponse{
        Error: "Access denied",
        Code:  "ACCESS_DENIED",
    })
}

func ResourceNotFound(c *gin.Context, msg string) {
    c.JSON(http.StatusNotFound, ErrorResponse{
        Error: msg,
        Code:  "RESOURCE_NOT_FOUND",
    })
}

func InternalError(c *gin.Context, msg string) {
    c.JSON(http.StatusInternalServerError, ErrorResponse{
        Error: msg,
        Code:  "INTERNAL_ERROR",
    })
}
```

### 7.2 错误代码常量

```go
// pkg/apierr/codes.go
package apierr

const (
    // 验证错误
    VALIDATION_FAILED = "VALIDATION_FAILED"
    INVALID_INPUT     = "INVALID_INPUT"

    // 认证授权
    AUTH_REQUIRED    = "AUTH_REQUIRED"
    INVALID_TOKEN    = "INVALID_TOKEN"
    ACCESS_DENIED    = "ACCESS_DENIED"
    ADMIN_REQUIRED   = "ADMIN_REQUIRED"

    // 资源
    RESOURCE_NOT_FOUND = "RESOURCE_NOT_FOUND"
    ALREADY_EXISTS     = "ALREADY_EXISTS"

    // 通用
    INTERNAL_ERROR = "INTERNAL_ERROR"
)
```

### 7.3 错误处理最佳实践

```go
func (h *PodHandler) GetPod(c *gin.Context) {
    podKey := c.Param("key")

    // 1. 参数格式验证
    if podKey == "" {
        apierr.InvalidInput(c, "Pod key is required")
        return
    }

    // 2. 业务逻辑验证
    pod, err := h.service.GetPod(c.Request.Context(), podKey)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            apierr.ResourceNotFound(c, "Pod not found")
            return
        }
        apierr.InternalError(c, "Failed to get pod")
        return
    }

    // 3. 权限验证
    tenant := middleware.GetTenant(c)
    if pod.OrganizationID != tenant.OrganizationID {
        apierr.ForbiddenAccess(c)
        return
    }

    c.JSON(http.StatusOK, gin.H{"pod": pod})
}
```

---

## 8. 配置管理

### 8.1 配置结构

```go
// internal/config/config.go
package config

import "github.com/spf13/viper"

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    JWT      JWTConfig
}

type ServerConfig struct {
    Port int
    Host string
}

type DatabaseConfig struct {
    Host     string
    Port     int
    User     string
    Password string
    DBName   string
    SSLMode  string
}

type RedisConfig struct {
    Host     string
    Port     int
    Password string
    DB       int
}

type JWTConfig struct {
    Secret []byte
    Expiry time.Duration
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("/etc/app/")

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

### 8.2 环境变量支持

```yaml
# config.yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  host: ${DB_HOST}
  port: ${DB_PORT}
  user: ${DB_USER}
  password: ${DB_PASSWORD}
  db_name: ${DB_NAME}
```

### 8.3 配置加载

```go
// cmd/server/main.go
func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }

    db, err := database.NewDB(cfg.Database.DSN())
    if err != nil {
        log.Fatal("Failed to connect database:", err)
    }

    // ... 初始化服务
}
```

---

## 9. 测试策略

### 9.1 单元测试结构

```
internal/
├── service/
│   ├── user/
│   │   ├── service.go
│   │   ├── service_test.go          # 主测试文件
│   │   ├── service_auth_test.go     # 按功能拆分
│   │   └── mock_test.go             # Mock 实现
```

### 9.2 测试数据库设置

```go
// service_test.go
func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent),
    })
    if err != nil {
        t.Fatalf("failed to connect database: %v", err)
    }

    // 创建表结构
    db.AutoMigrate(&domain.User{}, &domain.Organization{})

    return db
}
```

### 9.3 使用 Testify

```go
func TestAuthenticate(t *testing.T) {
    db := setupTestDB(t)
    service := NewService(db)
    ctx := context.Background()

    // 创建测试数据
    user := &domain.User{
        Email:    "test@example.com",
        Password: bcryptHash("password123"),
    }
    db.Create(user)

    // 测试
    result, err := service.Authenticate(ctx, "test@example.com", "password123")

    // 断言
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "test@example.com", result.Email)
}
```

### 9.4 HTTP 测试

```go
func TestListPodsHandler(t *testing.T) {
    gin.SetMode(gin.TestMode)

    // 设置路由
    r := gin.New()
    r.GET("/api/v1/pods", handler.ListPods)

    // 创建请求
    req := httptest.NewRequest("GET", "/api/v1/pods?limit=20", nil)
    w := httptest.NewRecorder()

    // 执行请求
    r.ServeHTTP(w, req)

    // 验证响应
    assert.Equal(t, http.StatusOK, w.Code)
    var resp map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.NotEmpty(t, resp["pods"])
}
```

---

## 10. 前端集成

### 10.1 技术栈

| 组件 | 技术 | 版本 |
|------|------|------|
| **框架** | Next.js | 16.1.1+ |
| **语言** | TypeScript | 5.x |
| **样式** | Tailwind CSS | 3.x |
| **状态** | Zustand | 4.x |
| **HTTP** | fetch | 内置 |
| **i18n** | next-intl | 3.x |

### 10.2 前端目录结构

```
web/src/
├── app/                      # App Router
│   ├── (auth)/               # 认证页面组
│   │   ├── login/
│   │   └── register/
│   ├── (dashboard)/          # 仪表板页面组
│   │   └── [org]/            # 动态路由
│   │       ├── layout.tsx
│   │       └── page.tsx
│   └── api/                  # API Routes (可选)
├── components/               # React 组件
│   ├── ui/                   # 基础组件
│   └── layout/               # 布局组件
├── lib/                      # 工具库
│   ├── api.ts                # API 客户端
│   └── utils.ts
├── hooks/                    # 自定义 Hooks
│   ├── useAuth.ts
│   └── useTenant.ts
├── stores/                   # Zustand 状态
│   └── auth.ts
└── messages/                 # i18n 翻译
    └── en.json
```

### 10.3 API 客户端实现

```typescript
// web/src/lib/api.ts
const API_BASE = process.env.NEXT_PUBLIC_API_URL || '/api/v1';

interface RequestOptions {
  method?: string;
  headers?: Record<string, string>;
  body?: any;
}

async function request<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const { method = 'GET', headers = {}, body } = options;

  const config: RequestInit = {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...headers,
    },
  };

  if (body) {
    config.body = JSON.stringify(body);
  }

  const response = await fetch(`${API_BASE}${endpoint}`, config);

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Request failed');
  }

  return response.json();
}

// API 函数
export const api = {
  pods: {
    list: (params: ListPodsParams) =>
      request<PaginatedResponse<Pod>>('/pods', {
        method: 'GET',
      }),
    create: (data: CreatePodRequest) =>
      request<Pod>('/pods', {
        method: 'POST',
        body: data,
      }),
  },
};
```

### 10.4 认证状态管理

```typescript
// web/src/stores/auth.ts
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AuthState {
  token: string | null;
  user: User | null;
  setAuth: (token: string, user: User) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      setAuth: (token, user) => set({ token, user }),
      logout: () => set({ token: null, user: null }),
    }),
    { name: 'auth-storage' }
  )
);
```

---

## 11. 开发环境

### 11.1 Docker Compose 配置

```yaml
# deploy/dev/docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: agentsmesh
      POSTGRES_PASSWORD: agentsmesh_dev
      POSTGRES_DB: agentsmesh
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U agentsmesh"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  backend:
    build:
      context: ../../backend
      dockerfile: Dockerfile.dev
    volumes:
      - ../../backend:/app
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started
    environment:
      DATABASE_URL: postgres://agentsmesh:agentsmesh_dev@postgres:5432/agentsmesh?sslmode=disable
      REDIS_URL: redis://redis:6379

volumes:
  postgres_data:
  redis_data:
```

### 11.2 一键启动脚本

```bash
#!/bin/bash
# deploy/dev/dev.sh

set -e

# 生成 .env 文件
cat > .env <<EOF
WEB_PORT=8080
POSTGRES_PORT=5432
REDIS_PORT=6379
EOF

# 启动 Docker 服务
docker compose up -d postgres redis backend

# 运行数据库迁移
docker compose exec backend migrate -path /app/migrations -database "$DATABASE_URL" up

echo "✅ Development environment started!"
echo "🌐 API: http://localhost:${WEB_PORT}"
```

### 11.3 热重载配置

**Go (Air)**:

```toml
# backend/.air.toml
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/main ./cmd/server"
bin = "tmp/main"
include_ext = ["go"]
exclude_dir = ["tmp", "vendor"]
delay = 1000

[log]
time = true
```

**Next.js (Turbopack)**:

```bash
# 前端本地运行（非 Docker）
cd web
pnpm dev --turbopack
```

---

## 12. 质量保证

### 12.1 Lint 配置

```yaml
# .golangci.yml
linters:
  enable:
    - errcheck      # 未检查的错误
    - gosimple      # 简化建议
    - staticcheck   # 静态分析
    - unused        # 未使用的代码
    - ineffassign   # 无效赋值

linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: true

issues:
  exclude-rules:
    # 测试文件降低严格度
    - path: _test\.go
      linters:
        - errcheck
        - gosimple
```

### 12.2 TypeScript 配置

```json
{
  "compilerOptions": {
    "strict": true,
    "noEmit": true,
    "isolatedModules": true,
    "incremental": true,
    "target": "ES2020",
    "module": "ESNext",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "jsx": "preserve",
    "skipLibCheck": true
  }
}
```

### 12.3 ESLint 配置

```javascript
// eslint.config.mjs
import next from "eslint-config-next/core-web-vitals";

export default [
  ...next,
  {
    rules: {
      "@typescript-eslint/no-unused-vars": "error",
      "@typescript-eslint/no-explicit-any": "warn",
    },
  },
];
```

---

## 总结

本规范定义了一个完整的、生产级的 Go Web 服务架构，其核心特点：

### 架构原则
1. **DDD 分层**: API → Service → Domain → Infra
2. **依赖倒置**: Domain 定义接口，Infra 实现接口
3. **单一职责**: 每层有明确的职责边界

### 技术选型
1. **成熟稳定**: Gin + GORM + PostgreSQL
2. **高性能**: JWT 无状态认证 + Redis 缓存
3. **易维护**: 清晰的代码组织 + 统一的错误处理

### 开发体验
1. **一键启动**: Docker Compose + 热重载
2. **类型安全**: Go 强类型 + TypeScript 前端
3. **测试友好**: 内存数据库 + Mock 支持

### 可扩展性
1. **模块化**: 按域划分，易于添加新功能
2. **多租户**: 内置租户隔离机制
3. **前后端分离**: REST API + Next.js

遵循本规范，可以构建出结构清晰、易于维护、可扩展的生产级 Go Web 服务。
