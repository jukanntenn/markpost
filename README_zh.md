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
- 🏠 **自托管** — 单个 Docker 容器，支持 SQLite 或 PostgreSQL，随处运行

## 快速开始

详细部署说明请参阅[部署指南](docs/deployment.md)。

部署完成后，打开 `http://<你的服务器IP或域名>:7157`，使用以下凭据登录：

- **用户名：** `markpost`
- **密码：** `<your-secret-admin-password>`，默认为 `markpost`

> ⚠️ 首次登录后请立即修改默认密码。

登录后，**Post 密钥** 将显示在控制台首页。通过 API 创建文章时需要用到它（见下方）。

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

## 开发

请参阅[开发指南](docs/development.md)。

## 许可证

[MIT](LICENSE)
