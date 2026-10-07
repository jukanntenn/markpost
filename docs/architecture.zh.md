# markpost 架构

[English](architecture.md) | 中文

改动源码树之前先读本文。它是代码库的有序地图——组件、边界、以及新行为的归属；决策理由在链接的 RFC（决策记录）里。

## 这个包是什么

markpost 是一个轻量的 Markdown 转 HTML 发布服务：作者通过 HTTP API 投递 Markdown，得到由渲染缓存支撑的稳定预渲染 HTML 页面。它以单个多架构 Docker 镜像交付（s6 监管 Caddy + Go），前置 Cloudflare CDN，由维护者自己的部署消费，另有一个驱动同一 API 的独立 CLI 与 MCP 服务器。

## 组件

| 组件        | 职责                                                                          | 公共接口                       |
| ----------- | ----------------------------------------------------------------------------- | ------------------------------ |
| `backend/`  | Go 服务：Gin HTTP handler、GORM 持久化、JWT 认证、异步投递扇出、OpenTelemetry | `/api/v1/*` REST               |
| `frontend/` | 管理/用户控制台：Next.js 16 静态导出、React 19                                | 由 Caddy 服务的静态产物        |
| `cli/`      | 面向脚本与 agent 的独立 `markpost` 客户端                                     | `markpost <子命令>`            |
| `mcp/`      | 向模型暴露 API 的 markpost-mcp MCP 服务器                                     | MCP 工具                       |
| `e2e/`      | 面向生产形态 compose 的 Playwright chromium 套件                              | `pnpm test`                    |
| `devops/`   | dev.py compose 环境、Dockerfile、ansible 部署                                 | `python3 devops/dev.py <命令>` |
| `docker/`   | 生产 s6 镜像构建                                                              | `docker/build.py`              |
| `specs/`    | 现状设计参考                                                                  | —                              |
| `docs/`     | 运维指南与文档标准                                                            | —                              |
| `.agents/`  | 驱动开发循环的 skills 与 RFC                                                  | —                              |

## 新行为去哪里

运行时行为从 `backend/internal/` 开始——新的 HTTP 路由在 `internal/web` 下加 handler，业务规则落在 `internal/service/<domain>`，持久化契约落在 `internal/repository` 加一次迁移。用户可见的控制台工作在 `frontend/` 内进行并遵守静态导出约束。横切契约（API 形态、认证、投递语义）在 `specs/` 规格化、在 `.agents/rfcs/` 的 RFC 里决策；运维流程住在 `docs/`。文档规则本身——[文档标准](AGENTS.md)、[双语文档契约](i18n/README.zh.md)与 [RFC 规则](../.agents/rfcs/README.zh.md)——拥有散文的归属。

贡献者入口见 [development.md](development.zh.md)。
