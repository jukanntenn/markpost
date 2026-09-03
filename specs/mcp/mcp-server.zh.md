# MCP Server 设计规格

English | [中文](mcp-server.md)

本文档定义 `markpost-mcp`：把运行中的 markpost 实例暴露给 AI agent 的独立 MCP（Model Context Protocol）服务器。它是终态设计参考；决策依据见 [MCP server MRFC](../../.agents/mrfcs/proposed/2026-09-03-mcp-server.zh.md)。

<a id="architecture"></a>

## 1. 架构

`markpost-mcp` 是位于 `mcp/` 的独立 Go module（`github.com/jukanntenn/markpost/mcp`，自带 `go.mod`），只通过 markpost 的公开契约——`/api/v1` REST API 与两个公开帖子端点——与实例通信。它不 import backend 模块的任何内容，也因此支持独立构建、独立发布，一个二进制可指向任意本地或远端实例。架构对齐黄金参考 [github/github-mcp-server](https://github.com/github/github-mcp-server)：薄的类型化 API 客户端（对应 go-github）加上 MCP 工具 handler，后者把 backend 的 JSON 原样作为 text 内容返回。

MCP 层构建在官方 [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) v1.7.0 之上（协议版本覆盖到 2025-06-18/2026-07-28），与黄金参考锁定同一 SDK 同一版本。工具以类型化 handler 签名声明；SDK 从 handler 的参数结构体推导每个工具的 JSON Schema，`jsonschema` 结构体标签承载参数描述。

## 2. 工具集

工具分为四个工具集（toolset），各自镜像一块 REST 面。注册前先校验全部请求的名字，拼写错误在启动时即失败，不会留下半注册的服务器。`--toolsets`（或 `MARKPOST_MCP_TOOLSETS`）选择启用的集合；`all` 展开为全部。

| 工具集     | 默认 | 工具                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| ---------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `posts`    | 开   | `create_post`、`list_posts`、`get_post`、`delete_post`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `delivery` | 开   | `list_channels`、`create_channel`、`update_channel`、`delete_channel`、`test_channel`、`list_delivery_history`、`list_latest_deliveries`、`get_delivery_stats`、`list_pending_deliveries`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `account`  | 开   | `get_my_retention`、`list_my_sessions`、`revoke_my_session`、`revoke_my_other_sessions`、`rotate_post_key`、`change_my_password`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `admin`    | 关   | `admin_list_users`、`admin_get_user`、`admin_create_user`、`admin_delete_user`、`admin_set_user_role`、`admin_reset_user_password`、`admin_set_user_active`、`admin_set_user_vip`、`admin_set_user_retention`、`admin_bulk_set_retention`、`admin_retention_impact`、`admin_get_retention_defaults`、`admin_list_user_sessions`、`admin_revoke_user_sessions`、`admin_revoke_session`、`admin_list_posts`、`admin_delete_post`、`admin_list_channels`、`admin_create_channel`、`admin_set_channel_enabled`、`admin_delete_channel`、`admin_list_delivery_history`、`admin_get_delivery_stats`、`admin_list_locked_channels`、`admin_list_audit_logs`、`admin_get_stats`、`admin_get_settings`、`admin_set_setting` |

共四十七个工具。命名遵循黄金参考的 `动词_名词` 惯例；admin 工具带 `admin_` 前缀，与 REST 的 `/admin` 命名空间一致。admin 工具集默认关闭：多数凭据不是 admin，且其工具破坏面最大；三个默认工具集已覆盖 agent 所需的发布工作流。

`--read-only`（或 `MARKPOST_MCP_READ_ONLY`）在注册时移除全部写工具——这是服务端保证，而非客户端承诺。读工具携带 `ReadOnlyHint: true`；破坏性工具携带 `DestructiveHint: true`。

`create_post` 先经认证端点 `GET /api/v1/post-key` 解析调用者的 post key，再向公开端点 `POST /{post_key}` 提交 `{title, body}`（markpost 唯一的创建路径），返回 `{id, url}`，渲染 URL 由实例 base URL 拼出。`get_post` 请求 `GET /{qid}?format=raw`，返回 markdown 源文（`# 标题` + 正文）而非渲染后的 HTML。

## 3. 认证

markpost 没有 personal access token，access token 默认 24 小时过期，因此 MCP server 使用环境变量中的用户名/密码凭据（`MARKPOST_MCP_USERNAME` / `MARKPOST_MCP_PASSWORD`——仅环境变量、不设 flag，避免凭据进入 shell 历史）认证，并自动维持会话：

- 启动时客户端立即登录，URL 或凭据错误即刻失败，而非等到第一次工具调用。
- markpost 轮换 refresh token；客户端持久化 `/auth/refresh` 返回的每个新令牌对。
- 任何认证调用收到 401 时，在客户端互斥锁下执行一次恢复：用当前 refresh token 刷新；若刷新本身被拒，则完整重新登录。基于失败 access token 的单飞比较防止并发刷新——同一 refresh token 的重复使用会触发后端的令牌盗用检测。
- `change_my_password` 采纳 backend 返回的全新令牌对，MCP 会话在自己的密码变更后继续存活。

已知限制：经 GitHub OAuth 创建且无本地密码的用户无法认证；HTTP 传输的 MCP 原生 OAuth 留待后续 MRFC。

## 4. 传输与配置

CLI 采用 `urfave/cli/v2`（与 backend 相同的框架），两个子命令。公共 flag 在两个子命令上重复声明（urfave v2 不向下传播 App 级 flag），并绑定 `MARKPOST_MCP_*` 环境变量：

| Flag                  | 环境变量                  | 默认值                   | 含义                               |
| --------------------- | ------------------------- | ------------------------ | ---------------------------------- |
| `--url`               | `MARKPOST_MCP_URL`        | —（必填）                | 实例 base URL                      |
| `--toolsets`          | `MARKPOST_MCP_TOOLSETS`   | `posts,delivery,account` | 启用的工具集（逗号分隔，或 `all`） |
| `--read-only`         | `MARKPOST_MCP_READ_ONLY`  | false                    | 移除全部写工具                     |
| `--addr` (http)       | `MARKPOST_MCP_HTTP_ADDR`  | `127.0.0.1:8973`         | HTTP 监听地址                      |
| `--path` (http)       | `MARKPOST_MCP_HTTP_PATH`  | `/mcp`                   | HTTP 端点路径                      |
| `--http-token` (http) | `MARKPOST_MCP_HTTP_TOKEN` | —                        | MCP 客户端须出示的 bearer token    |

`stdio` 在 stdin/stdout 上运行服务器（本地 MCP host 的传输）；日志只写 stderr。`http` 以无状态模式提供 streamable-http 传输（`mcp.NewStreamableHTTPHandler`，黄金参考远程服务器的模式），可选常数时间 bearer 校验保护；未配置 token 时默认只监听环回地址，是否暴露由部署者决定。

工具输出为 backend REST JSON 原样（缩进后）的 text 内容；工具错误携带 backend 自己的错误码与消息及 HTTP 状态码，agent 看到的是 markpost 的语义，而非客户端转述。

## 5. 测试

三层，与黄金参考对齐：

- **单元**——REST 客户端对着记录请求的 `httptest` 假后端做契约测试（登录/刷新/重试序列、查询构造、错误映射）；每个工具经进程内 MCP 会话（`mcp.NewInMemoryTransports`）覆盖成功与错误路径。
- **工具快照**——`internal/tools/testdata/tools.json` 锁定完整工具面（名称、描述、注解、输入 schema）；任何变更都是有意的、经评审的 diff（用 `-update` 再生成）。
- **E2E**——`mcp/e2e` 位于 `--tags e2e` 构建标签之后（对齐黄金参考的门控）：启动 postgres testcontainer、从 `backend/` 构建并运行真实后端（迁移、种子 admin）、以 stdio 启动真实 `markpost-mcp` 二进制，经 SDK 客户端驱动每个工具集。

## 6. 分发

一个产品版本、一个发布节奏：release tag 为 GitHub release 附上交叉编译的二进制（linux/darwin × amd64/arm64、windows amd64；禁用 CGO），并发布独立镜像 `jukanntenn/markpost-mcp`（多架构，入口为 `http` 传输）。`go install github.com/jukanntenn/markpost/mcp/cmd/markpost-mcp@latest` 从源码安装。操作指南见 [docs/mcp.zh.md](../../docs/mcp.zh.md)。
