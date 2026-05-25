# Markpost

**轻量级 Markdown 转 HTML 发布服务。** 通过 API 上传 Markdown，即可获得渲染后的 HTML 页面。简单、自托管、快速。

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

[English](README.md) | 简体中文

---

## 功能特性

- **API 优先** — 只需一个 `POST` 请求即可上传 Markdown，获得唯一 URL
- **Web 控制台** — 管理文章、查看统计、配置推送通道
- **自托管** — 可在 Docker、裸机或云平台上运行
- **多数据库** — SQLite 简单易用，PostgreSQL 支持规模化扩展
- **推送通道** — 将文章转发至 Webhook（飞书、Slack、自定义），支持关键词过滤
- **OAuth 支持** — 支持 GitHub 登录或用户名/密码登录

## 快速开始

### Docker Compose（推荐）

创建 `docker-compose.yml`：

```yaml
services:
  frontend:
    image: jukanntenn/markpost-web:latest
    container_name: markpost-frontend
    ports:
      - "7330:3000"
    environment:
      - API_PROXY_TARGET=http://backend:7330
    depends_on:
      backend:
        condition: service_healthy
    restart: unless-stopped

  backend:
    image: jukanntenn/markpost:latest
    container_name: markpost-backend
    volumes:
      - ./data:/app/data
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://127.0.0.1:7330/api/v1/health"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s
    restart: unless-stopped
```

```bash
docker compose up -d
```

打开 `http://localhost:7330`，使用以下凭据登录：
- **用户名：** `markpost`（默认）
- **密码：** `markpost`（生产环境请务必修改！）

登录后，**Post 密钥** 将显示在控制台首页。

### 仅后端（无头 / 纯 API 模式）

如果不需要 Web 控制台，可以单独运行后端：

```bash
docker run -d \
  --name markpost \
  -p 7330:7330 \
  -v ./data:/app/data \
  --restart unless-stopped \
  jukanntenn/markpost:latest
```

此模式提供 API 端点和文章渲染（`GET /:id`），但不包含 Web 界面。

## API 参考

### 创建文章

```bash
curl -X POST http://localhost:7330/YOUR_POST_KEY \
  -H "Content-Type: application/json" \
  -d '{"title": "My Post", "body": "# Hello World\nThis is **Markdown**."}'
```

响应：
```json
{ "id": "p-abc123" }
```

### 查看文章

访问 `http://localhost:7330/p-abc123` — 将渲染为带样式的 HTML 页面。

## 配置

Markpost 使用 TOML 配置文件或环境变量。

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MARKPOST_SERVER__HOST` | 绑定地址 | `127.0.0.1` |
| `MARKPOST_SERVER__PORT` | HTTP 端口 | `7330` |
| `MARKPOST_DB__DRIVER` | 数据库驱动 | `sqlite` |
| `MARKPOST_DB__DSN` | 连接字符串 | `file:./data/markpost.db` |
| `MARKPOST_JWT__ACCESS_SIGNING_KEY` | **必填。** JWT 签名密钥 | — |

完整配置参考请查看 [config.example.toml](backend/config.example.toml)。

## 开发

环境要求：Go 1.26+、Node.js 24+、pnpm、Docker

```bash
# 启动开发环境（PostgreSQL、后端热重载）
python3 devops/dev.py start

# 前端开发服务器
cd frontend && pnpm install && pnpm dev

# 运行后端测试
cd backend && go test ./...

# 运行前端测试
cd frontend && pnpm test
```

详细说明请参阅[开发指南](docs/development.md)。

## 部署

请参阅[部署指南](docs/deployment.md)，了解：
- 生产环境 Docker 配置
- Ansible 自动化
- 反向代理配置
- PostgreSQL 配置

## 许可证

[MIT](LICENSE)
