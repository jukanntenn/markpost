# markpost-mcp：AI agent 集成

[English](mcp.md) | 中文

`markpost-mcp` 是独立的 MCP 服务器，让 AI agent（Claude Desktop、Cursor、ZCode、VS Code 等）操作 markpost 实例：发布与读取帖子、管理投递渠道，admin 凭据下还可运行完整管理面。设计参考见 [specs/mcp/mcp-server.zh.md](../specs/mcp/mcp-server.zh.md)。

## 1. 安装

三选一：

- **Release 二进制**——从 [releases 页面](https://github.com/jukanntenn/markpost/releases) 下载 `markpost-mcp-<os>-<arch>.tar.gz`，把二进制放入 `PATH`。
- **Go 安装**——`go install github.com/jukanntenn/markpost/mcp/cmd/markpost-mcp@latest`。
- **Docker**——`docker run -e MARKPOST_MCP_URL=... -e MARKPOST_MCP_USERNAME=... -e MARKPOST_MCP_PASSWORD=... -p 127.0.0.1:8973:8973 jukanntenn/markpost-mcp`（镜像提供 HTTP 传输）。

## 2. 配置 MCP host

服务器以环境变量中的一个 markpost 用户身份认证（刻意不设 flag，避免凭据进入 shell 历史）。`.mcp.json` / host 配置中的 stdio 条目：

```json
{
  "mcpServers": {
    "markpost": {
      "type": "stdio",
      "command": "markpost-mcp",
      "args": ["stdio", "--url", "https://markpost.example.com"],
      "env": {
        "MARKPOST_MCP_USERNAME": "alice",
        "MARKPOST_MCP_PASSWORD": "…"
      }
    }
  }
}
```

认证全自动：服务器启动即登录（凭据错误即刻失败）并自动续期——markpost 轮换 refresh token，服务器跟踪轮换，401 时刷新，刷新被拒时重新登录。经 GitHub OAuth 创建且无本地密码的用户暂不支持。

## 3. 工具集

默认启用的工具集覆盖发布工作流：`posts`（创建/列表/读取/删除）、`delivery`（渠道 CRUD + 测试、历史、统计）、`account`（保留期、会话、post key、密码）。`admin` 工具集（28 个，镜像 `/api/v1/admin`）按需开启：

```json
"args": ["stdio", "--url", "https://markpost.example.com", "--toolsets", "all"]
```

`--read-only` 在服务端移除全部写工具——交给只该读的 agent 最安全。工具 schema 由快照测试锁定，工具面只会经意变更。

## 4. 远程服务（HTTP 传输）

`markpost-mcp http` 默认在 `127.0.0.1:8973/mcp` 提供 streamable-http 传输：

```bash
markpost-mcp http --url https://markpost.example.com --toolsets all \
  --addr 0.0.0.0:8973 --http-token "$(openssl rand -hex 32)"
```

客户端以 `{"type": "http", "url": "https://mcp.example.com/mcp", "headers": {"Authorization": "Bearer …"}}` 连接。监听离开环回地址时务必设置 `--http-token`；缺省时端点无认证。MCP 原生 OAuth 已延后；静态 bearer 是 v1 的保护手段。

## 5. 验证

```bash
MARKPOST_MCP_USERNAME=… MARKPOST_MCP_PASSWORD=… \
  markpost-mcp stdio --url https://markpost.example.com --toolsets all
```

然后让 agent 列出工具，或在另一终端向运行中服务器的 stdin 发送 `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`。每个工具原样返回 backend 的 REST JSON；失败携带 backend 的错误码、消息与 HTTP 状态。
