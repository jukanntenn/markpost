# markpost

一个简单的 Go Web 项目，提供 API 来上传 markdown 文本内容并查询转换后的 HTML 内容。

**申明**
目前所有文件和代码均为 AI 生成，基本实现了想要的功能，但个人对 go 开发并不熟悉，代码没有 review。目前来看 AI 生成的代码还有很多问题，后续的主要任务就是使用 AI 开发+人工 review 的方式来重构和优化。快速体验请直接跳到 [方法二：Docker 部署（推荐）](#方法二docker-部署推荐)

## 功能特性

- 支持通过 API 上传 markdown 文本内容
- 支持查询已上传的内容并渲染为美观的 HTML 页面
- 支持配置文件管理（标题长度、内容长度、限流等）
- 内置限流中间件
- 使用 SQLite3 数据库，开启 WAL 模式提高并发性能
- 自动生成 post_key 用于内容上传
- 内置健康检查端点
- 响应式 HTML 模板，支持移动端访问

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

```yml
services:
  markpost:
    image: jukanntenn/markpost:latest
    container_name: markpost
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    restart: unless-stopped
```

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

使用有效的 post_key 上传 markdown 内容。系统启动时会自动生成一个 admin 用户的 post_key，请查看启动日志获取。

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
  "id": "生成的nanoid",
  "title": "文章标题",
  "message": "内容创建成功"
}
```

错误响应：

```json
{
  "error": "错误信息"
}
```

### 获取内容

**GET** `/:id`

根据 ID 获取已上传的内容，返回渲染后的 HTML 页面（不是 JSON 格式）。

- 成功：返回完整的 HTML 页面，包含转换后的 markdown 内容
- 失败：返回错误页面

### 健康检查

**GET** `/health`

检查服务状态。

响应：

```json
{
  "status": "ok",
  "message": "markpost is running"
}
```

## 示例用法

首先启动服务器，查看日志获取生成的 post_key：

```bash
go run .
```

启动日志会显示类似以下内容：

```
已创建 admin 用户，post_key: abc12345
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
# 返回完整的 HTML 页面，可在浏览器中打开
curl http://localhost:8080/RETURNED_ID

# 或者直接在浏览器中访问
# http://localhost:8080/RETURNED_ID

# 健康检查
curl http://localhost:8080/health
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
├── templates/         # HTML 模板目录
│   ├── article.html   # 文章页面模板
│   ├── error.html     # 错误页面模板
│   └── success.html   # 成功页面模板
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
