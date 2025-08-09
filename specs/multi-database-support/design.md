# 设计文档

## 概述

本设计文档描述了为 markpost 应用程序添加多数据库支持功能的技术实现。该功能将使应用程序能够同时支持 PostgreSQL 和 SQLite 数据库，通过配置文件中的数据库 URL 进行连接配置，同时保持现有代码的兼容性。

应用程序当前使用 GORM 作为 ORM，这为多数据库支持提供了良好的基础。我们将通过简单地修改 `InitDB()` 函数来实现数据库类型选择，保持架构简洁。

## 架构

### 整体架构图

```mermaid
graph TB
    A[应用程序启动] --> B[配置加载器]
    B --> C{数据库类型}
    C -->|PostgreSQL| D[PostgreSQL 驱动器]
    C -->|SQLite| E[SQLite 驱动器]
    D --> F[GORM 实例]
    E --> F
    F --> G[业务逻辑层]
    G --> H[API 处理器]

    I[配置文件] --> B
```

### 架构说明

应用程序通过修改后的 `InitDB()` 函数根据配置中的数据库类型和 URL 初始化相应的数据库连接。整个过程保持简单直接，无需额外的抽象层。

## 组件和接口

### 配置管理

#### DatabaseConfig 结构体

```go
type DatabaseConfig struct {
    Type string `toml:"type"` // "postgresql" 或 "sqlite"
    URL  string `toml:"url"`  // 数据库连接 URL
}
```

#### 配置文件格式

```toml
[database]
type = "sqlite"
url = "./data/db.sqlite3"

# PostgreSQL 示例
# type = "postgresql"
# url = "postgres://username:password@localhost:5432/markpost?sslmode=disable"
```

#### URL 格式说明

**SQLite URL 格式：**

- 相对路径：`./data/db.sqlite3`
- 绝对路径：`/path/to/database.sqlite3`
- 内存数据库：`:memory:`

**PostgreSQL URL 格式：**

- 标准格式：`postgres://username:password@host:port/database?sslmode=disable`
- 示例：`postgres://postgres:password@localhost:5432/markpost?sslmode=disable`

## 数据模型

### 模型兼容性

现有的 `User` 和 `Post` 模型保持不变，GORM 会自动处理不同数据库间的类型映射：

```go
type User struct {
    ID       int    `json:"id" gorm:"primaryKey;autoIncrement"`
    Username string `json:"username" gorm:"unique;not null"`
    PostKey  string `json:"post_key" gorm:"not null"`
    GitHubID *int64 `json:"github_id" gorm:"column:github_id;uniqueIndex"`
}

type Post struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    Title     string    `json:"title" gorm:"not null"`
    Body      string    `json:"body" gorm:"not null"`
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
    UserID    *int      `json:"user_id" gorm:"index;foreignKey:ID;references:users"`
}
```

### 数据类型映射

| GORM 类型 | SQLite   | PostgreSQL |
| --------- | -------- | ---------- |
| int       | INTEGER  | INTEGER    |
| string    | TEXT     | VARCHAR    |
| time.Time | DATETIME | TIMESTAMP  |
| bool      | BOOLEAN  | BOOLEAN    |

## 错误处理

### 错误处理策略

1. **连接错误** - 应用程序启动时失败，提供清晰的错误信息
2. **配置错误** - 应用程序启动时失败，指出配置问题
3. **不支持的数据库类型** - 启动时报错并退出
4. **URL 格式错误** - 启动时验证并报错

## 实现计划

### 核心修改

1. **扩展配置文件**

   - 在现有 config.toml 中添加 `[database]` 配置段
   - 添加数据库类型和 URL 配置项

2. **修改 InitDB 函数**

   - 根据配置的数据库类型选择相应的 GORM 驱动
   - 支持 SQLite 和 PostgreSQL 连接
   - 保持现有的 AutoMigrate 和初始化逻辑

3. **添加 PostgreSQL 依赖**
   - 在 go.mod 中添加 PostgreSQL 驱动依赖

## 向后兼容性

1. **现有 API 保持不变** - 所有现有的 HTTP 端点和响应格式保持兼容
2. **数据库函数签名不变** - `GetUserByPostKey`、`CreatePost` 等函数接口保持不变
3. **配置可选性** - 如果未指定数据库配置，默认使用 SQLite
4. **渐进式迁移** - 可以在不影响现有功能的情况下逐步启用新功能

## 安全考虑

1. **配置安全** - 数据库密码等敏感信息建议通过环境变量配置
2. **连接安全** - PostgreSQL 连接支持 SSL/TLS 加密
3. **访问控制** - 维持现有的用户认证和授权机制
4. **SQL 注入防护** - 继续使用 GORM 的参数化查询防护

## 运维考虑

1. **数据库选择** - 开发环境可使用 SQLite，生产环境推荐使用 PostgreSQL
2. **配置管理** - 敏感的数据库连接信息建议通过环境变量管理
3. **备份策略** - 不同数据库需要相应的备份和恢复策略
