# 设计文档

## 概览

本设计文档描述了为 create post 接口实现多层次限流功能的技术方案。该系统将使用 `ulule/limiter` 库实现基于 IP 地址和 post_key 的双重限流机制，限流状态保存在内存中，通过配置文件进行参数管理。

## 架构

### 整体架构

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   HTTP Request  │───▶│  Rate Limiting   │───▶│ Authentication  │
│                 │    │   Middleware     │    │   Middleware    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ ulule/limiter    │    │ CreatePost      │
                       │ (Memory Store)   │    │ Handler         │
                       └──────────────────┘    └─────────────────┘
```

### 限流层次结构

1. **第一层：IP 限流**

   - 防止单个 IP 地址的恶意请求
   - 按分钟和天进行限制

2. **第二层：Post Key 限流**
   - 防止单个用户的垃圾内容发布
   - 按分钟和天进行限制

### 技术栈集成

- **Web 框架**: Gin (现有)
- **限流库**: ulule/limiter v3
- **存储后端**: Memory Store (内存)
- **配置管理**: Viper (现有)

## 组件和接口

### 1. 配置扩展

扩展现有的 Config 结构，添加限流相关配置：

```go
type Config struct {
    // 现有字段...
    TitleMaxSize int `mapstructure:"TITLE_MAX_SIZE"`
    BodyMaxSize  int `mapstructure:"BODY_MAX_SIZE"`
    APIRateLimit int `mapstructure:"API_RATE_LIMIT"`

    // 新增：限流配置
    RateLimit struct {
        IP struct {
            PerMinute int `mapstructure:"per_minute"`
            PerDay    int `mapstructure:"per_day"`
        } `mapstructure:"ip"`
        PostKey struct {
            PerMinute int `mapstructure:"per_minute"`
            PerDay    int `mapstructure:"per_day"`
        } `mapstructure:"post_key"`
    } `mapstructure:"rate_limit"`

    // 现有配置...
    Database struct { ... }
    GitHub   struct { ... }
    JWT      struct { ... }
}
```

### 2. 限流器初始化

使用 ulule/limiter 框架提供的组件：

```go
import (
    "time"
    "github.com/ulule/limiter/v3"
    ginlimiter "github.com/ulule/limiter/v3/drivers/middleware/gin"
    "github.com/ulule/limiter/v3/drivers/store/memory"
)

// 初始化内存存储和限流器
func initRateLimiters(config RateLimitConfig) ([]gin.HandlerFunc, error) {
    store := memory.NewStore()

    // 定义限流速率
    ipMinuteRate := limiter.Rate{Period: time.Minute, Limit: int64(config.IPPerMinute)}
    ipDayRate := limiter.Rate{Period: 24 * time.Hour, Limit: int64(config.IPPerDay)}
    postKeyMinuteRate := limiter.Rate{Period: time.Minute, Limit: int64(config.PostKeyPerMinute)}
    postKeyDayRate := limiter.Rate{Period: 24 * time.Hour, Limit: int64(config.PostKeyPerDay)}

    // 创建限流器实例
    ipMinuteLimiter := limiter.New(store, ipMinuteRate)
    ipDayLimiter := limiter.New(store, ipDayRate)
    postKeyMinuteLimiter := limiter.New(store, postKeyMinuteRate)
    postKeyDayLimiter := limiter.New(store, postKeyDayRate)

    // 创建 Gin 中间件
    middlewares := []gin.HandlerFunc{
        ginlimiter.NewMiddleware(ipMinuteLimiter, ginlimiter.WithKeyGetter(getIPKey)),
        ginlimiter.NewMiddleware(ipDayLimiter, ginlimiter.WithKeyGetter(getIPKey)),
        ginlimiter.NewMiddleware(postKeyMinuteLimiter, ginlimiter.WithKeyGetter(getPostKeyKey)),
        ginlimiter.NewMiddleware(postKeyDayLimiter, ginlimiter.WithKeyGetter(getPostKeyKey)),
    }

    return middlewares, nil
}

// IP 键提取器
func getIPKey(c *gin.Context) string {
    return "ip:" + c.ClientIP()
}

// Post Key 键提取器
func getPostKeyKey(c *gin.Context) string {
    return "postkey:" + c.Param("post_key")
}
```

### 3. 中间件集成

在路由配置中应用限流中间件：

```go
// routes.go 中的路由配置
func SetupRoutes(r *gin.Engine) {
    // 初始化限流中间件
    rateLimitMiddlewares, err := initRateLimiters(config.RateLimit)
    if err != nil {
        log.Printf("初始化限流器失败: %v", err)
        // 使用空的中间件数组，允许请求通过
        rateLimitMiddlewares = []gin.HandlerFunc{}
    }

    // 认证相关路由
    auth := r.Group("/auth")
    {
        auth.GET("/github/url", GenerateGitHubAuthURL)
        auth.GET("/github/callback", HandleGitHubCallback)
    }

    // 为 create post 路由添加限流中间件
    middlewares := append(rateLimitMiddlewares, AuthMiddleware())
    r.POST("/:post_key", middlewares..., CreatePostHandler)

    // 其他路由不受限流影响
    r.GET("/:id", RenderPostHandler)
    r.GET("/health", HealthHandler)
}
```

### 4. 自动响应头管理

ulule/limiter 的 Gin 中间件自动处理响应头：

- **X-RateLimit-Limit**: 限流器的最大请求数
- **X-RateLimit-Remaining**: 当前时间窗口内剩余的请求数
- **X-RateLimit-Reset**: 限流重置的时间戳
- **Retry-After**: 当达到限制时，建议的重试时间

## 数据模型

### 限流键格式

使用标准化的键格式来区分不同类型的限流：

```
IP 限流键：
- "ip:minute:{ip_address}"
- "ip:day:{ip_address}"

Post Key 限流键：
- "postkey:minute:{post_key}"
- "postkey:day:{post_key}"
```

### 内存存储结构

ulule/limiter 的内存存储使用以下结构：

```go
// 内部使用的限流计数器结构
type Counter struct {
    Count     int64
    ExpiresAt time.Time
}

// 存储映射: key -> Counter
store: map[string]*Counter
```

## 错误处理

### 1. 限流错误响应

当请求被限流时，返回标准的 HTTP 429 响应：

```json
{
    "error": "Rate limit exceeded",
    "message": "IP rate limit exceeded" | "Post key rate limit exceeded",
    "retry_after": 60,
    "details": {
        "limit": 100,
        "remaining": 0,
        "reset": 1703648400
    }
}
```

### 2. 系统错误处理

- **配置错误**: 使用默认值并记录警告日志
- **限流器初始化失败**: 记录错误并继续服务（允许请求通过）
- **内存不足**: 记录错误并清理过期的限流记录

### 3. 降级策略

当限流系统出现故障时：

1. 记录错误日志
2. 允许请求通过（fail-open 策略）
3. 尝试重新初始化限流器

## 实现细节

### 1. 配置文件扩展

在 `config.toml` 中添加限流配置：

```toml
# 限流配置
[rate_limit]
  [rate_limit.ip]
  per_minute = 100
  per_day = 1000

  [rate_limit.post_key]
  per_minute = 10
  per_day = 100
```

### 2. 主程序集成

在 main.go 中初始化和使用限流器：

```go
// main.go
func main() {
    if err := LoadConfig(); err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 初始化 GitHub OAuth2 配置
    initGitHubOAuth()

    InitDB()
    defer CloseDB()

    // 验证器注册
    if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
        v.RegisterValidation("titlesize", validateTitleSize)
        v.RegisterValidation("bodysize", validateBodySize)
    }

    r := gin.Default()
    r.LoadHTMLGlob("templates/*")

    // 保留现有的全局限流器（如果配置了）
    if config.APIRateLimit > 0 {
        limitPerSecond := float64(config.APIRateLimit) / 60.0
        r.Use(LimiterMiddleware(rate.Limit(limitPerSecond), config.APIRateLimit))
        log.Printf("已启用全局限流: 每分钟 %d 次请求", config.APIRateLimit)
    }

    // 设置路由（包含 create post 的专用限流）
    SetupRoutes(r)

    log.Println("服务器启动中...")
    log.Println("访问 http://localhost:8080")
    if err := r.Run(":8080"); err != nil {
        log.Fatalf("启动服务器失败: %v", err)
    }
}
```

### 3. 自定义错误处理

可选：自定义 ulule/limiter 的错误响应格式：

```go
// 自定义错误响应格式的中间件选项
func withCustomErrorHandler() ginlimiter.Option {
    return ginlimiter.WithErrorHandler(func(c *gin.Context, err error) {
        log.Printf("[RATE_LIMIT] Error for IP %s: %v", c.ClientIP(), err)
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Rate limit service temporarily unavailable",
        })
    })
}

func withCustomLimitReachedHandler() ginlimiter.Option {
    return ginlimiter.WithLimitReachedHandler(func(c *gin.Context) {
        log.Printf("[RATE_LIMIT] Limit reached for IP %s, path %s",
            c.ClientIP(), c.Request.URL.Path)
        c.JSON(http.StatusTooManyRequests, gin.H{
            "error": "Rate limit exceeded",
            "message": "Too many requests. Please try again later.",
        })
    })
}

// 在创建中间件时使用自定义选项
ginlimiter.NewMiddleware(
    ipMinuteLimiter,
    ginlimiter.WithKeyGetter(getIPKey),
    withCustomErrorHandler(),
    withCustomLimitReachedHandler(),
)
```

### 4. 监控指标

记录关键的限流指标：

- 限流触发次数（按类型分类）
- 限流器的内存使用情况
- 限流检查的平均响应时间
- 当前活跃的限流键数量

## 性能考虑

### 1. 内存管理

- **自动清理**: ulule/limiter 自动清理过期的限流记录
- **内存限制**: 监控内存使用，必要时手动清理
- **垃圾回收**: 定期清理长时间未使用的键值对

### 2. 并发安全

- **线程安全**: ulule/limiter 内置线程安全机制
- **锁竞争**: 使用分片存储减少锁竞争
- **原子操作**: 确保计数器更新的原子性

### 3. 性能优化

- **预分配**: 预分配常用键的存储空间
- **批量操作**: 支持批量检查多个限流条件
- **缓存友好**: 优化数据结构的缓存局部性
