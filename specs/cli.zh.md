# CLI 规格

[English](cli.md) | 中文

`markpost` CLI(`cli/`,独立 Go module `markpost/cli`)的当前状态设计。决策理由见[独立 agent CLI MRFC](../.agents/mrfcs/implemented/2026-09-03-standalone-agent-cli.zh.md)。

## 范围与技术栈

面向单个 markpost 服务器的客户端:会话处理、发布、通用 API 直通——为人类与 AI agent 双方设计。框架:`urfave/cli/v2`(与 backend 相同;不支持 cobra 式穿插解析——flag 在位置参数之前)。架构沿用 gh(cli/cli):惰性依赖工厂 `Factory`(`internal/cliapp`)、`IOStreams`(`internal/iostreams`)、单一请求核心之上的类型化 REST 客户端(`internal/api`)。

## 命令

| 命令                       | 作用                                                                        | 鉴权                                                              |
| -------------------------- | --------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| `auth login`               | 用户名/密码(flag、提示或 `--password-stdin`)或 `--token`;保存会话           | —                                                                 |
| `auth status`              | 服务器、会话、过期时间;经 `/api/v1/me/retention` 验证                       | 可选                                                              |
| `auth token`               | 打印访问令牌                                                                | 会话                                                              |
| `auth logout`              | 尽力在服务端吊销,清除本地凭据,保留服务器配置                                | 会话                                                              |
| `posts create`             | 发布 markdown(`--file <路径                                                 | ->`、位置参数或管道 stdin)并打印文章 URL;`--json`给出`{qid, url}` | post key(flag/env)或会话查询 |
| `posts list`               | 分页表格或 `--json` 信封;`--search/--page/--limit`                          | 会话                                                              |
| `posts view <qid>`         | `--format raw`(默认)、`html` 或 `url`                                       | —(公开路由)                                                       |
| `posts delete <qid>`       | 非交互需 `--yes`;TTY 下提示确认                                             | 会话                                                              |
| `post-key show` / `rotate` | 打印 key(rotate 立即作废旧 key)                                             | 会话                                                              |
| `api <endpoint>`           | 直通;相对路径解析到 `/api/v1` 下,绝对路径原样;`-X <method>`、`--input <文件 | ->`;响应体原样输出                                                | 有会话即携带                 |
| `status`                   | health + readiness + version;`--json`                                       | —                                                                 |
| `config get/set`           | 本地配置键:`server`                                                         | —                                                                 |
| `version`                  | CLI 版本                                                                    | —                                                                 |

## 会话与配置

- 配置文件:`config.toml`,路径 `$MARKPOST_CONFIG_DIR` > `$XDG_CONFIG_HOME/markpost` > `~/.config/markpost`,原子写入 0600、目录 0700。字段:服务器、用户身份、访问令牌、刷新令牌、过期时间。
- 环境变量:`MARKPOST_SERVER`(也是全局 `--server` flag 的 env)、`MARKPOST_TOKEN`(无刷新的会话——401 即终局)、`MARKPOST_POST_KEY`。
- 优先级:`--server` flag > `MARKPOST_SERVER` > 已存服务器;`MARKPOST_TOKEN` > 已存令牌。已存会话只提供给签发它的服务器。
- 持有刷新令牌时遇 401,客户端刷新一次、持久化新令牌对并重试原请求;不可恢复的情况以认证错误呈现。
- `auth login --token` 先验证令牌再保存。

## 输出与退出码

- stdout 承载结果(可解析);诊断走 stderr。读命令支持 `--json`——管道时紧凑,TTY 时缩进。
- 退出码(对齐 gh):`0` 成功、`1` 失败、`2` 取消、`4` 认证——认证错误附带登录提示。
- agent 检测(`internal/agentenv`,gh 的环境变量约定,`AI_AGENT` 优先)让用法错误输出完整帮助,并在 User-Agent 追加 `Agent/<name>`。提示绝不阻塞非交互调用者。

## 测试

- 单测(默认 `go test ./...`):各包测试用精确输出断言,对着 `internal/testserver` 的共享 httptest 后端;命令测试以缓冲流在进程内运行整个 app。
- 验收(`go test -tags acceptance ./acceptance`):对真实服务器执行编译出的二进制;需要 `MARKPOST_E2E_BASE_URL/USERNAME/PASSWORD`,否则跳过。内含针对 [#84](https://github.com/jukanntenn/markpost/issues/84)(同一秒登出重登会拉黑重签令牌)的、带注释的 sleep 规避。
