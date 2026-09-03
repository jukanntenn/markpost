# MRFC: 面向 AI agent 集成的 MCP server

Status: implemented

[English](2026-09-03-mcp-server.md) | 中文

## Problem

markpost 的产品契约是 HTTP API，服务的客户端是人类（仪表盘）与脚本（curl）。AI agent 如今是一等操作者：能*发布*、_回读_、_管理投递_、*实施管理*的 agent 把实例变成 agent 可达的发布端点——正是产品想要的集成面。但 agent 不会原生说 REST，它们说 MCP（带 schema 的工具，经 stdio 或 streamable-http 列出与调用）。仓库里没有任何东西服务于此：唯一的 MCP 提及是 `.mcp.json` 里给编码 agent 的第三方工具。MCP server 必须新建，而其中三个形态决策是承重的：与 backend 耦合还是包装其公开契约；在 markpost 没有 personal access token、access token 24 小时过期的前提下如何认证；以及 47 个工具的面（admin 镜像天然是破坏性的）如何做安全门控。业界收敛的参考——github/github-mcp-server——对 api.github.com 回答了同样三个问题，是架构与测试形态的对齐目标。

## Decision

`mcp/` 是独立 Go module（`github.com/jukanntenn/markpost/mcp`），以薄的类型化 HTTP 客户端包装 REST API，不 import backend 的任何内容——即 github-mcp-server 与 api.github.com 的关系，支撑独立构建与发布。MCP 层为官方 `modelcontextprotocol/go-sdk` v1.7.0（黄金参考的锁定版本）；工具是类型化 handler，JSON Schema 由参数结构体推导，输出为 backend REST JSON 原样。认证使用 `MARKPOST_MCP_*` 环境变量中的用户名/密码（不设 flag），会话维护全自动：启动即登录、客户端跟踪 refresh token 轮换、客户端互斥锁下的单飞 401 恢复（先刷新、被拒再重登录）——并发刷新会触发后端的令牌盗用检测；`change_my_password` 采纳 backend 返回的全新令牌对。工具面为四个工具集——`posts`、`delivery`、`account` 默认开启；`admin`（28 个工具镜像 `/api/v1/admin`）按需开启——另有 `--read-only` 在注册时移除全部写工具（服务端保证）。传输：面向本地 host 的 `stdio` 与面向远程的 `http`（无状态 streamable-http、可选常数时间 bearer 守卫）；CLI 与 backend 同为 urfave/cli v2。测试对齐黄金参考：逐工具的 httptest 契约测试、进程内会话的工具测试、锁定 47 工具面的 schema 快照、以及 `--tags e2e` 套件（起 postgres testcontainer、构建真实 backend、经 stdio 驱动真实二进制）。分发搭产品的 release tag：GitHub release 附交叉编译二进制、发布 `jukanntenn/markpost-mcp` 多架构镜像、支持 `go install`。完整设计见 [specs/mcp/mcp-server.zh.md](../../../specs/mcp/mcp-server.zh.md)；操作指南见 [docs/mcp.zh.md](../../../docs/mcp.zh.md)。

## Alternatives considered

**嵌入 backend、共享 service 层。** 无重复 DTO、无 HTTP 跳数——但 server 无法独立于 backend 分发与版本化，无法指向远端实例，backend 每次重构都会波及 agent 面。否决：独立分发是需求本身，REST API 已是稳定契约。

**用 mark3labs/mcp-go 而非官方 SDK。** 社区 SDK 早于官方出现；github-mcp-server 自己已从它迁移到 `modelcontextprotocol/go-sdk`。绿地 v1 否决：黄金参考与协议官方示例都基于官方 SDK，对齐它们是本项目声明的标准。

**环境变量放静态 access token。** 字面上最贴近黄金参考的 PAT-in-env 形态——但默认每 24 小时失效，工具日常不可用。凭据登录加轮换跟踪，是对"没有 PAT 的令牌体系"最诚实的 PAT 等价物。给 backend 增加真正的 PAT 概念作为 v1 的范围蔓延被否决（未来 MRFC 的候选）。

**把 markpost-mcp 作为 s6 服务内置进主镜像。** 自托管者零额外部署，但把 agent 面的发布与产品镜像耦合，还迫使镜像内出现认证故事。延后至未来 MRFC；v1 只发独立镜像与二进制。

## Consequences

这一取舍换来的：任何支持 MCP 的 agent 经 47 个带 schema 的工具操作 markpost 实例，输出是 backend 自己的 JSON、错误携带 backend 自己的错误码；该面构造即安全（admin 按需开启、只读是服务端保证、schema 被快照锁定），会话从不要求人工去轮换什么。提案的全部验收条件在落地时逐项核实：e2e 套件对真实 backend 经 stdio 走通 创建→读取→列表→删除 及 account、delivery、admin 三个面（CI 对 `mcp/**` 与 `backend/**` 路径重跑）；401 恢复与刷新被拒→重登录两条序列钉死在客户端契约测试里；只读与 admin 关闭的 ListTools 断言在服务端证明门控有效；`testdata/tools.json` 与交付面一致；release 工作流在 tag 时附二进制并发布 `jukanntenn/markpost-mcp` 镜像。代价：手工维护的 REST 客户端必须跟踪 backend DTO 变化（e2e 路径过滤器是绊网——backend 契约变更会对真实服务器重跑套件）；密码存在于 agent host 的环境里，信任模型与黄金参考赋予 PAT 的相同；OAuth-only 用户在 MCP 原生 OAuth 或 PAT 概念落地前无法使用；admin 工具集虽是按需开启，仍给了 agent 破坏性触达，部署者须有意识地启用。随同落地：prek workspace 现在并发运行多个 golangci-lint 项目（backend、mcp——另有 standalone-agent-cli MRFC 的 cli），暴露了 golangci-lint 的 `/tmp` 文件锁；各树声明 `run.allow-parallel-runners` 并使用独立缓存，入队的 `build_version_test.py` 则剥离 `GIT_*` 环境才能在 prek hook 环境下存活。刻意留在范围外的后续项：HTTP 传输的 MCP 原生 OAuth、backend 的 PAT 概念、镜像内 s6 打包。
