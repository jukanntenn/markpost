[English](README.md) | 简体中文

<div align="center">

# Markpost

**轻量级 Markdown 转 HTML 发布服务。** 通过 API 上传 Markdown，即可获得渲染后的 HTML 页面。简单、自托管、快速。

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://www.docker.com/)

</div>

---

## 功能特性

- ✍️ **Markdown 发布** — 通过一个 `POST` 请求上传 Markdown，即可获得带唯一 URL 的渲染 HTML 页面
- 🌐 **Web 控制台** — 管理文章、查看统计、配置推送通道
- 📬 **推送通道** — 将文章转发至 Webhook（飞书、Slack、自定义），支持关键词过滤
- 🏠 **自托管** — 单个 Docker Compose 栈，支持 PostgreSQL，随处运行

## 快速开始

### 前置要求

- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/install/)（v2+）

### 1. 下载文件

```bash
mkdir markpost && cd markpost
BASE=https://raw.githubusercontent.com/jukanntenn/markpost/main/docker
curl -o docker-compose.yml $BASE/docker-compose.yml
curl -o Caddyfile $BASE/Caddyfile
curl -o .env.example $BASE/.env.example
```

### 2. 配置

```bash
cp .env.example .env
```

编辑 `.env` — 至少修改密码和 JWT 签名密钥：

```env
MARKPOST_ADMIN__INITIAL_PASSWORD=<你的密码>
MARKPOST_JWT__ACCESS_SIGNING_KEY=<随机字符串>
MARKPOST_JWT__REFRESH_SIGNING_KEY=<随机字符串>
```

### 3. 启动

```bash
docker compose up -d
```

### 4. 访问

打开 `http://<你的服务器IP>:2053`，使用你在 `.env` 中设置的凭据登录（默认：`markpost` / `markpost`）。

> ⚠️ 首次登录后请立即修改默认密码。

**Post 密钥** 显示在控制台首页，通过 API 创建文章时需要用到（见下方）。

## API 参考

### 创建文章

POST /:post-key

```json
{ "title": "My Post", "body": "# Hello World\nThis is **Markdown**." }
```

**响应** `201 Created`

```json
{ "id": "p-abc123" }
```

你的 Post 密钥（以 `mpk-` 开头）在首次登录时自动生成，可在控制台首页查看。

### 查看文章

**渲染后的 HTML：**

GET /:qid

**原始 Markdown：**

GET /:qid?format=raw

## 配置

所有配置通过环境变量设置，可在 `.env` 文件中指定或直接传递给容器。

| 变量                                | 说明                           | 默认值      |
| ----------------------------------- | ------------------------------ | ----------- |
| `MARKPOST_ADMIN__INITIAL_USERNAME`  | 管理员用户名（仅首次启动生效） | `markpost`  |
| `MARKPOST_ADMIN__INITIAL_PASSWORD`  | 管理员密码（仅首次启动生效）   | `markpost`  |
| `MARKPOST_JWT__ACCESS_SIGNING_KEY`  | JWT 访问令牌签名密钥           | `change-me` |
| `MARKPOST_JWT__REFRESH_SIGNING_KEY` | JWT 刷新令牌签名密钥           | `change-me` |
| `MARKPOST_SERVER__PUBLIC_URL`       | 服务公网地址                   | _（空）_    |
| `MARKPOST_TIMEZONE`                 | 容器时区                       | `UTC`       |
| `MARKPOST_POST__RETENTION_DAYS`     | 文章保留天数                   | `7`         |

完整变量列表请参阅 [`.env.example`](docker/.env.example)。

## 许可证

[MIT](LICENSE)
