# markpost

一个简单的 Go Web 项目，提供 API 来上传 markdown 文本内容并查询转换后的 HTML 内容。

## 功能特性

- 支持通过 API 上传 markdown 文本内容
- 支持查询已上传的内容
- 支持配置文件管理
- 内置限流中间件
- 使用 SQLite3 数据库，开启 WAL 模式提高并发性能

## 技术栈

- **gin**: Web 框架
- **viper**: 配置文件管理
- **goldmark**: Markdown 转 HTML
- **go-nanoid**: 生成唯一 ID
- **modernc.org/sqlite**: 纯 Go SQLite 数据库驱动（无 CGO 依赖）

## 快速开始

### 方法一：直接运行（需要 Go 环境）

#### 1. 安装依赖

```bash
go mod download
```

#### 2. 运行项目

```bash
go run .
```

### 方法二：Docker 部署（推荐）

#### 1. 使用部署脚本（推荐）

```bash
./deploy.sh
```

#### 2. 手动 Docker 部署

```bash
# 创建数据目录
mkdir -p ./data

# 构建并启动
docker-compose up --build -d

# 查看日志
docker-compose logs -f markpost

# 停止服务
docker-compose down
```

程序启动后会：

- 自动创建 `data/db.sqlite3` 数据库文件
- 自动创建必要的数据表
- 如果不存在 admin 用户，会自动创建并生成 post_key
- 启动 HTTP 服务器，监听 8080 端口

### 3. 配置文件

项目会读取 `markpost.toml` 配置文件，可以配置以下参数：

```toml
# title 的最大字节数
TITLE_MAX_SIZE = 200

# body 的最大字节数
BODY_MAX_SIZE = 1048576

# 接口限流频率，每分钟允许请求次数
API_RATE_LIMIT = 60
```

## API 文档

### 上传内容

**POST** `/:post_key`

使用有效的 post_key 上传 markdown 内容。

请求体：

```json
{
  "title": "文章标题",
  "body": "markdown 内容"
}
```

响应：

```json
{
  "id": "生成的nanoid"
}
```

### 获取内容

**GET** `/:id`

根据 ID 获取已上传的内容，返回转换后的 HTML 格式。

响应：

```json
{
  "id": "nanoid",
  "title": "文章标题",
  "body": "<h1>转换后的 HTML 内容</h1><p>这是一些 <strong>HTML</strong> 内容。</p>",
  "created_at": "2023-01-01T00:00:00Z"
}
```

## 示例用法

首先启动服务器，查看日志获取生成的 post_key：

```bash
go run .
```

然后使用 curl 测试 API：

```bash
# 上传内容（替换 YOUR_POST_KEY 为实际的 post_key）
curl -X POST http://localhost:8080/YOUR_POST_KEY \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试标题",
    "body": "# 这是一个测试\n\n这是一些 **markdown** 内容。"
  }'

# 获取内容（替换 RETURNED_ID 为上一步返回的 ID）
# 返回的 body 字段将是转换后的 HTML 内容
curl http://localhost:8080/RETURNED_ID
```

## 项目结构

```
markpost/
├── config.go          # 配置管理
├── db.go              # 数据库操作
├── middleware.go      # 中间件定义
├── utils.go           # 辅助函数
├── main.go            # 主程序入口
├── markpost.toml      # 配置文件
├── go.mod             # Go 模块文件
├── Dockerfile         # Docker 镜像构建文件
├── docker-compose.yml # Docker Compose 配置
├── deploy.sh          # 部署脚本
├── .dockerignore      # Docker 忽略文件
├── Makefile          # Make 工具配置
├── DOCKER.md         # Docker 部署指南
├── MIGRATION.md      # SQLite 驱动迁移说明
├── data/             # 数据目录（挂载点）
└── README.md         # 项目说明
```

## Docker 部署说明

### 特性

- **多阶段构建**：优化镜像大小，最终镜像基于 Alpine Linux
- **纯 Go 构建**：使用无 CGO 依赖的 SQLite 驱动，构建更简单、部署更可靠
- **数据持久化**：数据库文件保存在 `./data` 目录中
- **配置文件挂载**：支持外部配置文件修改
- **健康检查**：内置健康检查端点 `/health`
- **非 root 用户**：容器内使用非 root 用户运行，提高安全性
- **自动重启**：服务异常时自动重启

### 环境要求

- Docker
- Docker Compose

### 端口说明

- **8080**：HTTP API 服务端口

### 数据持久化

数据库文件保存在宿主机的 `./data` 目录中，即使容器被删除，数据也不会丢失。
