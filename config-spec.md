# Go 应用配置管理规范

## 1. 概述

本文档定义 Go 应用程序配置管理的标准化实现机制，涵盖配置结构设计、加载策略、验证机制、安全处理等方面，可作为各类 Go 项目的参考规范。

### 1.1 设计目标

- **多源融合**: 配置文件、环境变量、命令行参数、默认值多层级融合
- **类型安全**: 强类型配置结构体，编译期类型检查
- **验证完备**: 启动时配置验证，快速失败
- **安全合规**: 敏感信息安全存储与处理
- **开发友好**: 合理默认值，本地开发零配置启动

---

## 2. 技术栈与依赖

### 2.1 核心依赖

| 库 | 版本 | 用途 |
|---|------|------|
| `github.com/spf13/viper` | v1.21.0+ | 配置管理核心库 |
| `github.com/spf13/pflag` | v1.0.10+ | 命令行参数解析 |
| `github.com/spf13/afero` | v1.15.0+ | 虚拟文件系统（配置文件读取） |
| `github.com/go-viper/mapstructure/v2` | v2.4.0+ | 配置映射到结构体 |
| `github.com/pelletier/go-toml/v2` | v2.2.4+ | TOML 格式支持 |

### 2.2 go.mod 配置

```go
module github.com/org/myapp

go 1.23

require (
    github.com/spf13/viper v1.21.0
    github.com/go-viper/mapstructure/v2 v2.4.0
    github.com/pelletier/go-toml/v2 v2.2.4
)
```

### 2.3 Viper 特性

- 支持 JSON、TOML、YAML、HCL、INI、ENV 等多种格式
- 支持环境变量自动绑定
- 支持嵌套配置结构
- 支持配置热加载
- 支持多配置文件合并

---

## 3. 配置文件格式

### 3.1 格式选择：TOML

**推荐使用 TOML 格式**，原因如下：

| 特性 | TOML | YAML | JSON |
|------|------|------|------|
| 可读性 | 优秀 | 优秀 | 一般 |
| 注释支持 | 是 | 是 | 否 |
| 类型明确 | 是 | 否（隐式类型） | 是 |
| 解析稳定性 | 高 | 低（空白敏感） | 高 |
| 错误提示 | 清晰 | 模糊 | 清晰 |
| 工具支持 | 良好 | 良好 | 优秀 |

### 3.2 配置文件示例 (TOML)

```toml
# ============================================================
# 应用配置文件
# ============================================================

# 服务连接配置
server_url = "https://api.example.com"
grpc_endpoint = "grpc.example.com:9443"

# 节点标识
node_id = "runner-001"
description = "Production runner node 1"

# 组织信息
org_slug = "my-org"

# 容量配置
max_concurrent_tasks = 5

# 工作空间配置
workspace_root = "/var/lib/myapp/workspace"
log_level = "info"

# MCP 配置
[mcp]
config_path = "/etc/myapp/mcp.json"
port = 19000

# 健康检查
[health_check]
port = 9090
enabled = true

# 自动更新配置
[auto_update]
enabled = true
check_interval = "24h"
channel = "stable"          # stable | beta
max_wait_time = "30m"
auto_apply = true

# gRPC/mTLS 证书配置
[grpc]
cert_file = "/etc/myapp/certs/client.crt"
key_file = "/etc/myapp/certs/client.key"
ca_file = "/etc/myapp/certs/ca.crt"

# 日志配置
[logging]
level = "info"              # debug | info | warn | error
file = "/var/log/myapp/app.log"
format = "json"             # json | text

# PTY 调试日志
[logging.pty]
enabled = false
directory = "/tmp/myapp/pty-logs"
```

---

## 4. 配置结构体设计

### 4.1 结构体定义规范

```go
package config

import (
    "errors"
    "os"
    "path/filepath"
    "runtime"
    "time"

    "github.com/spf13/viper"
)

// Config 应用主配置结构体
type Config struct {
    // ========== 服务连接 ==========
    ServerURL    string `mapstructure:"server_url"`
    GRPCEndpoint string `mapstructure:"grpc_endpoint"`

    // ========== 节点标识 ==========
    NodeID      string `mapstructure:"node_id"`
    Description string `mapstructure:"description"`

    // ========== 组织信息 ==========
    OrgSlug string `mapstructure:"org_slug"`

    // ========== 容量配置 ==========
    MaxConcurrentTasks int `mapstructure:"max_concurrent_tasks"`

    // ========== 工作空间 ==========
    WorkspaceRoot string `mapstructure:"workspace_root"`
    LogLevel      string `mapstructure:"log_level"`

    // ========== 子配置组 ==========
    MCP         MCPConfig         `mapstructure:"mcp"`
    HealthCheck HealthCheckConfig `mapstructure:"health_check"`
    AutoUpdate  AutoUpdateConfig  `mapstructure:"auto_update"`
    GRPC        GRPCConfig        `mapstructure:"grpc"`
    Logging     LoggingConfig     `mapstructure:"logging"`

    // ========== 程序化字段（不来自配置文件）==========
    // 使用 mapstructure:"-" 标签标记为忽略
    Version        string `mapstructure:"-"` // 构建时注入
    ConfigFilePath string `mapstructure:"-"` // 配置文件路径追踪
}

// MCPConfig MCP 服务配置
type MCPConfig struct {
    ConfigPath string `mapstructure:"config_path"`
    Port       int    `mapstructure:"port"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
    Port    int  `mapstructure:"port"`
    Enabled bool `mapstructure:"enabled"`
}

// AutoUpdateConfig 自动更新配置
type AutoUpdateConfig struct {
    Enabled       bool          `mapstructure:"enabled"`
    CheckInterval time.Duration `mapstructure:"check_interval"`
    Channel       string        `mapstructure:"channel"`        // stable | beta
    MaxWaitTime   time.Duration `mapstructure:"max_wait_time"`
    AutoApply     bool          `mapstructure:"auto_apply"`
}

// GRPCConfig gRPC 连接配置
type GRPCConfig struct {
    CertFile string `mapstructure:"cert_file"`
    KeyFile  string `mapstructure:"key_file"`
    CAFile   string `mapstructure:"ca_file"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
    Level  string        `mapstructure:"level"`
    File   string        `mapstructure:"file"`
    Format string        `mapstructure:"format"`
    PTY    PTYLogConfig  `mapstructure:"pty"`
}

// PTYLogConfig PTY 日志配置
type PTYLogConfig struct {
    Enabled   bool   `mapstructure:"enabled"`
    Directory string `mapstructure:"directory"`
}
```

### 4.2 结构体标签规范

| 标签 | 用途 | 示例 |
|------|------|------|
| `mapstructure:"name"` | Viper 字段映射 | `mapstructure:"server_url"` |
| `mapstructure:"-"` | 忽略该字段 | `mapstructure:"-"` |
| `json:"name"` | JSON 序列化（可选） | `json:"serverUrl"` |
| `yaml:"name"` | YAML 序列化（可选） | `yaml:"serverUrl"` |

### 4.3 字段命名规范

- **配置文件字段**: 小写 + 下划线 (`server_url`, `max_concurrent_tasks`)
- **结构体字段**: 驼峰命名 (`ServerURL`, `MaxConcurrentTasks`)
- **环境变量**: 全大写 + 下划线 (`SERVER_URL`, `MAX_CONCURRENT_TASKS`)

---

## 5. 配置加载机制

### 5.1 加载优先级

```
命令行参数 > 环境变量 > 配置文件 > 默认值
```

### 5.2 配置文件搜索路径

```go
// 按优先级搜索配置文件
// 1. 命令行指定路径
// 2. 当前目录
// 3. 用户目录
// 4. 系统配置目录 (Unix)
```

| 优先级 | 位置 | 路径示例 |
|--------|------|----------|
| 1 | 命令行指定 | `--config /path/to/config.toml` |
| 2 | 当前目录 | `./myapp.toml` |
| 3 | 用户目录 | `~/.myapp/config.toml` |
| 4 | 系统目录 (Unix) | `/etc/myapp/config.toml` |

### 5.3 加载实现

```go
// Load 加载配置
func Load(configFile string) (*Config, error) {
    v := viper.New()

    // ========== 1. 设置默认值 ==========
    setDefaults(v)

    // ========== 2. 环境变量绑定 ==========
    v.SetEnvPrefix("MYAPP")           // 环境变量前缀: MYAPP_*
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    v.AutomaticEnv()                   // 自动读取环境变量

    // ========== 3. 配置文件加载 ==========
    if configFile != "" {
        // 命令行指定配置文件
        v.SetConfigFile(configFile)
    } else {
        // 搜索默认位置
        v.SetConfigName("config")      // 配置文件名（不含扩展名）
        v.SetConfigType("toml")        // 显式指定格式
        v.AddConfigPath(".")           // 当前目录
        if home, err := os.UserHomeDir(); err == nil {
            v.AddConfigPath(filepath.Join(home, ".myapp"))
        }
        if runtime.GOOS != "windows" {
            v.AddConfigPath("/etc/myapp")
        }
    }

    // ========== 4. 读取配置文件 ==========
    if err := v.ReadInConfig(); err != nil {
        // 配置文件不存在是允许的（可完全依赖环境变量）
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("failed to read config: %w", err)
        }
    }

    // ========== 5. 解析到结构体 ==========
    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    // ========== 6. 后处理 ==========
    if err := postProcess(&cfg); err != nil {
        return nil, err
    }

    // 记录配置文件路径
    cfg.ConfigFilePath = v.ConfigFileUsed()

    return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper) {
    // 基础配置
    v.SetDefault("server_url", "https://api.example.com")
    v.SetDefault("max_concurrent_tasks", 5)
    v.SetDefault("workspace_root", defaultWorkspaceRoot())
    v.SetDefault("log_level", "info")

    // 子配置组
    v.SetDefault("mcp.port", 19000)
    v.SetDefault("health_check.port", 9090)
    v.SetDefault("health_check.enabled", true)

    // 自动更新
    v.SetDefault("auto_update.enabled", true)
    v.SetDefault("auto_update.check_interval", 24*time.Hour)
    v.SetDefault("auto_update.channel", "stable")
    v.SetDefault("auto_update.max_wait_time", 30*time.Minute)
    v.SetDefault("auto_update.auto_apply", true)

    // 日志
    v.SetDefault("logging.level", "info")
    v.SetDefault("logging.format", "json")
}

// postProcess 配置后处理
func postProcess(cfg *Config) error {
    // 生成默认节点 ID
    if cfg.NodeID == "" {
        hostname, _ := os.Hostname()
        if hostname == "" {
            hostname = "node"
        }
        cfg.NodeID = hostname
    }

    // 展开环境变量
    if cfg.WorkspaceRoot != "" {
        cfg.WorkspaceRoot = os.ExpandEnv(cfg.WorkspaceRoot)
    }

    return nil
}

// defaultWorkspaceRoot 平台适配的默认工作空间
func defaultWorkspaceRoot() string {
    if runtime.GOOS == "windows" {
        if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
            return filepath.Join(localAppData, "myapp", "workspace")
        }
        if home, err := os.UserHomeDir(); err == nil {
            return filepath.Join(home, ".myapp", "workspace")
        }
    }
    return "/var/lib/myapp/workspace"
}
```

---

## 6. 配置验证机制

### 6.1 验证层次

```
┌─────────────────────────────────────────────────────────┐
│                    配置验证层次                          │
├─────────────────────────────────────────────────────────┤
│  Level 1: 类型验证      - Viper 自动处理               │
│  Level 2: 必填验证      - Validate() 方法              │
│  Level 3: 格式验证      - Validate() 方法              │
│  Level 4: 语义验证      - Validate() 方法              │
│  Level 5: 依赖验证      - Validate() 方法              │
│  Level 6: 文件存在验证  - Validate() 方法              │
└─────────────────────────────────────────────────────────┘
```

### 6.2 验证实现

```go
// Validate 验证配置有效性
func (c *Config) Validate() error {
    // ========== 必填验证 ==========
    if c.ServerURL == "" {
        return errors.New("server_url is required")
    }

    // ========== 数值范围验证 ==========
    if c.MaxConcurrentTasks < 1 {
        return errors.New("max_concurrent_tasks must be at least 1")
    }
    if c.MaxConcurrentTasks > 100 {
        return errors.New("max_concurrent_tasks must not exceed 100")
    }

    // ========== 枚举值验证 ==========
    validChannels := map[string]bool{"stable": true, "beta": true}
    if !validChannels[c.AutoUpdate.Channel] {
        return errors.New("auto_update.channel must be 'stable' or 'beta'")
    }

    validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
    if !validLogLevels[c.LogLevel] {
        return errors.New("log_level must be one of: debug, info, warn, error")
    }

    // ========== 条件验证 ==========
    if c.GRPC.CertFile != "" || c.GRPC.KeyFile != "" || c.GRPC.CAFile != "" {
        // 如果配置了任意证书，则必须全部配置
        if err := c.validateGRPCConfig(); err != nil {
            return err
        }
    }

    // ========== 目录创建验证 ==========
    if c.WorkspaceRoot != "" {
        if err := os.MkdirAll(c.WorkspaceRoot, 0755); err != nil {
            return fmt.Errorf("failed to create workspace_root: %w", err)
        }
    }

    return nil
}

// validateGRPCConfig 验证 gRPC 配置
func (c *Config) validateGRPCConfig() error {
    if c.GRPCEndpoint == "" {
        return errors.New("grpc_endpoint is required when gRPC is enabled")
    }
    if c.GRPC.CertFile == "" {
        return errors.New("grpc.cert_file is required for gRPC mode")
    }
    if c.GRPC.KeyFile == "" {
        return errors.New("grpc.key_file is required for gRPC mode")
    }
    if c.GRPC.CAFile == "" {
        return errors.New("grpc.ca_file is required for gRPC mode")
    }

    // 验证证书文件存在
    files := []struct {
        path string
        name string
    }{
        {c.GRPC.CertFile, "certificate"},
        {c.GRPC.KeyFile, "private key"},
        {c.GRPC.CAFile, "CA certificate"},
    }

    for _, f := range files {
        if _, err := os.Stat(f.path); os.IsNotExist(err) {
            return fmt.Errorf("%s file not found: %s", f.name, f.path)
        }
    }

    return nil
}

// WarnInsecureDefaults 警告不安全的默认值
func (c *Config) WarnInsecureDefaults() {
    if c.ServerURL == "https://api.example.com" {
        slog.Warn("SECURITY: server_url is using default value, " +
            "this should be changed in production")
    }
}
```

### 6.3 验证时机

```go
func main() {
    // 1. 加载配置
    cfg, err := config.Load(configFile)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 2. 验证配置
    if err := cfg.Validate(); err != nil {
        log.Fatalf("Invalid config: %v", err)
    }

    // 3. 安全警告
    cfg.WarnInsecureDefaults()

    // 4. 启动应用
    // ...
}
```

---

## 7. 环境变量支持

### 7.1 环境变量命名规则

| 配置字段 | 环境变量 | 说明 |
|----------|----------|------|
| `server_url` | `MYAPP_SERVER_URL` | 前缀 + 大写 + 下划线 |
| `max_concurrent_tasks` | `MYAPP_MAX_CONCURRENT_TASKS` | 同上 |
| `mcp.port` | `MYAPP_MCP_PORT` | 嵌套字段用下划线连接 |
| `auto_update.enabled` | `MYAPP_AUTO_UPDATE_ENABLED` | 同上 |

### 7.2 环境变量优先级示例

```bash
# 配置文件中的值
server_url = "https://api.example.com"

# 环境变量覆盖
export MYAPP_SERVER_URL="https://prod.example.com"

# 最终值: https://prod.example.com
```

### 7.3 敏感信息环境变量

```bash
# 推荐使用环境变量传递敏感信息
export MYAPP_GRPC_KEY_FILE="/etc/myapp/secrets/client.key"
export MYAPP_DATABASE_PASSWORD="secret123"

# 或在 .env 文件中（不提交到版本控制）
# .env
MYAPP_DATABASE_PASSWORD=secret123
MYAPP_API_KEY=sk-xxxxx
```

---

## 8. 安全信息处理

### 8.1 敏感信息分类

| 类别 | 示例 | 存储方式 |
|------|------|----------|
| API 密钥 | `api_key`, `secret_key` | 环境变量 |
| 数据库密码 | `db_password` | 环境变量 |
| TLS 私钥 | `key_file` | 文件 (权限 0600) |
| TLS 证书 | `cert_file` | 文件 (权限 0644) |

### 8.2 证书文件安全存储

```go
// SaveCertificates 安全保存证书文件
func (c *Config) SaveCertificates(certPEM, keyPEM, caPEM []byte) error {
    // 证书目录
    certsDir := filepath.Join(os.UserHomeDir(), ".myapp", "certs")
    
    // 创建目录，权限 0700（仅所有者可访问）
    if err := os.MkdirAll(certsDir, 0700); err != nil {
        return fmt.Errorf("failed to create certs dir: %w", err)
    }

    // 保存私钥，权限 0600（仅所有者可读写）
    keyPath := filepath.Join(certsDir, "client.key")
    if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
        return fmt.Errorf("failed to save private key: %w", err)
    }

    // 保存证书，权限 0644（所有者读写，其他只读）
    certPath := filepath.Join(certsDir, "client.crt")
    if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
        return fmt.Errorf("failed to save certificate: %w", err)
    }

    // 保存 CA 证书
    caPath := filepath.Join(certsDir, "ca.crt")
    if err := os.WriteFile(caPath, caPEM, 0644); err != nil {
        return fmt.Errorf("failed to save CA certificate: %w", err)
    }

    // 更新配置路径
    c.GRPC.CertFile = certPath
    c.GRPC.KeyFile = keyPath
    c.GRPC.CAFile = caPath

    return nil
}
```

### 8.3 配置文件权限

```bash
# 配置文件权限建议
chmod 600 ~/.myapp/config.toml     # 仅所有者读写
chmod 700 ~/.myapp/                # 目录仅所有者访问

# 证书文件权限
chmod 600 ~/.myapp/certs/client.key  # 私钥
chmod 644 ~/.myapp/certs/client.crt  # 证书
chmod 644 ~/.myapp/certs/ca.crt      # CA 证书
```

---

## 9. 配置更新与持久化

### 9.1 运行时更新配置文件

```go
// UpdateConfigFile 更新配置文件中的特定字段
func UpdateConfigFile(configFile, key, value string) error {
    data, err := os.ReadFile(configFile)
    if err != nil {
        return fmt.Errorf("failed to read config: %w", err)
    }

    content := string(data)
    lines := strings.Split(content, "\n")
    found := false

    // 查找并更新现有字段
    for i, line := range lines {
        if strings.HasPrefix(strings.TrimSpace(line), key+" =") {
            lines[i] = fmt.Sprintf("%s = \"%s\"", key, value)
            found = true
            break
        }
    }

    // 字段不存在则追加
    if !found {
        if len(lines) > 0 && lines[len(lines)-1] != "" {
            lines = append(lines, "")
        }
        lines = append(lines, fmt.Sprintf("%s = \"%s\"", key, value))
    }

    // 写回文件
    newContent := strings.Join(lines, "\n")
    if err := os.WriteFile(configFile, []byte(newContent), 0600); err != nil {
        return fmt.Errorf("failed to write config: %w", err)
    }

    return nil
}
```

### 9.2 配置热加载（可选）

```go
import "github.com/fsnotify/fsnotify"

// WatchConfig 监听配置文件变化
func WatchConfig(v *viper.Viper, onReload func(*Config)) {
    v.WatchConfig()
    v.OnConfigChange(func(e fsnotify.Event) {
        log.Printf("Config file changed: %s", e.Name)
        
        var cfg Config
        if err := v.Unmarshal(&cfg); err != nil {
            log.Printf("Failed to reload config: %v", err)
            return
        }
        
        if err := cfg.Validate(); err != nil {
            log.Printf("Invalid config after reload: %v", err)
            return
        }
        
        onReload(&cfg)
    })
}
```

---

## 10. 命令行集成

### 10.1 命令行参数定义

```go
import "github.com/spf13/pflag"

// ParseFlags 解析命令行参数
func ParseFlags() (configFile string, err error) {
    pflag.StringVarP(&configFile, "config", "c", "", 
        "Path to config file (default: ./config.toml)")
    
    pflag.Parse()
    
    return configFile, nil
}
```

### 10.2 完整启动流程

```go
func main() {
    // 1. 解析命令行
    configFile, err := config.ParseFlags()
    if err != nil {
        log.Fatalf("Failed to parse flags: %v", err)
    }

    // 2. 加载配置
    cfg, err := config.Load(configFile)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 3. 验证配置
    if err := cfg.Validate(); err != nil {
        log.Fatalf("Invalid config: %v", err)
    }

    // 4. 安全警告
    cfg.WarnInsecureDefaults()

    // 5. 启动应用
    app.Run(cfg)
}
```

---

## 11. 测试配置

### 11.1 单元测试配置加载

```go
package config_test

import (
    "os"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
    // 创建临时配置文件
    tmpFile, err := os.CreateTemp("", "config-*.toml")
    require.NoError(t, err)
    defer os.Remove(tmpFile.Name())

    configContent := `
server_url = "https://test.example.com"
max_concurrent_tasks = 10
log_level = "debug"

[auto_update]
enabled = false
`
    _, err = tmpFile.WriteString(configContent)
    require.NoError(t, err)
    tmpFile.Close()

    // 加载配置
    cfg, err := config.Load(tmpFile.Name())
    require.NoError(t, err)

    // 验证
    assert.Equal(t, "https://test.example.com", cfg.ServerURL)
    assert.Equal(t, 10, cfg.MaxConcurrentTasks)
    assert.Equal(t, "debug", cfg.LogLevel)
    assert.False(t, cfg.AutoUpdate.Enabled)
}

func TestLoadWithEnvOverride(t *testing.T) {
    // 设置环境变量
    os.Setenv("MYAPP_SERVER_URL", "https://env.example.com")
    defer os.Unsetenv("MYAPP_SERVER_URL")

    cfg, err := config.Load("")
    require.NoError(t, err)

    assert.Equal(t, "https://env.example.com", cfg.ServerURL)
}

func TestValidate(t *testing.T) {
    tests := []struct {
        name    string
        config  config.Config
        wantErr bool
    }{
        {
            name: "valid config",
            config: config.Config{
                ServerURL:         "https://example.com",
                MaxConcurrentTasks: 5,
            },
            wantErr: false,
        },
        {
            name: "missing server_url",
            config: config.Config{
                MaxConcurrentTasks: 5,
            },
            wantErr: true,
        },
        {
            name: "invalid max_concurrent_tasks",
            config: config.Config{
                ServerURL:         "https://example.com",
                MaxConcurrentTasks: 0,
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

---

## 12. 最佳实践总结

### 12.1 配置设计原则

1. **显式优于隐式**: 关键配置无默认值，强制用户明确设置
2. **安全默认值**: 开发环境友好，生产环境有警告
3. **最小权限**: 敏感文件使用最严格权限
4. **快速失败**: 启动时验证配置，避免运行时错误

### 12.2 配置文档维护

- 维护 `config.example.toml` 示例文件
- 每个配置项添加注释说明
- 标注必填项和可选项
- 说明默认值和推荐值

### 12.3 变更流程

1. 更新配置结构体
2. 更新默认值设置
3. 更新验证逻辑
4. 更新示例配置文件
5. 更新测试用例
6. 更新文档

---

## 13. 参考资源

### 13.1 库文档

- [Viper 官方文档](https://github.com/spf13/viper)
- [TOML 规范](https://toml.io/cn/)
- [mapstructure 文档](https://github.com/mitchellh/mapstructure)

### 13.2 相关规范

- [Twelve-Factor App - 配置](https://12factor.net/zh_cn/config)
- [Go 项目布局标准](https://github.com/golang-standards/project-layout)

---

## 14. 版本历史

| 版本 | 日期 | 变更说明 |
|------|------|----------|
| 1.0 | 2026-03-13 | 初始版本 |
