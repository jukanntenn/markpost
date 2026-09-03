# MRFC: 独立的 agent 优先 markpost CLI

[English](2026-09-03-standalone-agent-cli.md) | 中文

Status: implemented

## Problem

markpost 此前只有 Web 仪表盘和裸 curl 两种客户端。脚本——以及越来越多的 AI agent——需要一个稳定、可编程的客户端:会话管理(JWT 登录、刷新、登出)、经 post key 发布、列表/查看/删除,以及可预期的机器可读行为。每个 curl 使用者都要重新实现 token 刷新、`{items, total, page, limit, total_pages}` 分页信封与 `{code, message, errors}` 错误解码;一个挂起的交互提示或不可用的退出码对自动化是致命的。markpost 明确希望 AI agent 成为一等 API 消费者,因此客户端必须天然 agent 友好,而非碰巧可用。

## Decision

`cli/` 是独立 Go module(`markpost/cli`),产出 `markpost` 二进制,独立于服务端镜像分发。框架采用 backend 已依赖的 `urfave/cli/v2` v2.27.4;gh(cli/cli,黄金参考,克隆于 `.local/contexts/cli`)提供的是架构模式而非框架:

- **Factory + 惰性闭包**(`internal/cliapp`):`Config`、`SaveConfig`、`Client` 是记忆化的函数字段,`markpost version` 既不碰磁盘也不碰网络,测试可只替换一个闭包。
- **IOStreams**(`internal/iostreams`):缓冲与 TTY 事实(stdout/stdin)收进一个可注入对象;提示仅在两个流都是终端时出现,非交互调用者得到 flag 错误而非挂起。v1 输出全部纯文本;TTY 决定 `--json` 紧凑或缩进。
- **类型化 REST 客户端**(`internal/api`):单一 `do`/`send` 核心注入 bearer token,401 时经 `/api/v1/auth/refresh` 恰好重试一次,触发持久化新 token 对的 `TokensChanged` 回调,并解码服务端错误信封。wire 类型在此重新声明(镜像服务端 DTO),不导入服务端 module。
- **配置**(`internal/config`):单一 `config.toml`,路径 `$MARKPOST_CONFIG_DIR` > `$XDG_CONFIG_HOME/markpost` > `~/.config/markpost`,原子写入 0600,目录 0700。环境变量覆盖:`MARKPOST_SERVER`、`MARKPOST_TOKEN`(无刷新;401 即终局)、`MARKPOST_POST_KEY`。已存会话只提供给签发它的服务器。
- **命令集**:`auth login|status|token|logout`、`posts create|list|view|delete`、`post-key show|rotate`、`api`(gh 式直通;相对路径解析到 `/api/v1` 下)、`status`、`config get|set`、`version`。读命令支持 `--json`。
- **agent 友好**:`internal/agentenv` 按 gh 的环境变量约定检测驱动 agent(`AI_AGENT` 优先)→ 用法错误时输出完整帮助而非一行 usage、User-Agent 携带 `Agent/<name>`、绝不阻塞等待输入。退出码对齐 gh:0 成功、1 错误、2 取消、4 认证(附登录提示)。
- **urfave 语义照单全收**:flag 必须在位置参数之前(v2 不支持穿插解析);`App.OnUsageError` 显式下发给每个子命令,因为 urfave 只把它复制到根命令。
- **测试对齐参考实现的分层**:包级单测用精确输出断言(testify `require`/`assert`),对着共享的 httptest 假后端(`internal/testserver`);命令级测试在进程内运行整个 app;验收测试(build tag `acceptance`)对真实服务器执行编译出的二进制,由 `MARKPOST_E2E_*` 环境变量门控,未设置时干净跳过。

## Alternatives considered

- **cobra(gh 的框架)**:落选——backend 已标准化于 urfave/cli/v2,且任务固定了框架选择。移植的是 gh 的模式(工厂、iostreams、错误分类、agent 检测),不是它的框架。
- **`cli/` 并入 backend Go module**:落选——独立分发需要隔离的依赖图;共享 module 会把 testcontainers 与 postgres 拖进 CLI 构建,并让分发客户端耦合服务端内部。重新声明 wire 类型与 gh 对 GitHub API 的取舍相同。
- **API 测试用 httpmock 式 transport 注册表**:落选——httptest.Server 以零新依赖执行真实 HTTP(URL 构造、头、body 编码),与本仓库真实 postgres 的测试哲学一致。
- **`--jq`/`--template` 输出(gh 的 json_flags)**:暂缓——jq 引擎是新增的重量级依赖,v1 价值有限;`--json` 配合标准工具链已足够覆盖 agent。
- **YAML 配置(gh 的选择)**:落选——本项目配置方言是 TOML(backend viper 配置、devops compose);markpost 内部一致性胜过模仿参考实现。
- **为穿插解析改造 urfave**:落选——与框架解析循环对抗只为参数书写自由,维护代价真实存在;flags-first 约定已写入每个 UsageText。

## Consequences

CLI 需手工跟进服务端 REST v1 契约(`internal/api/types.go`);API 变更需要配套 CLI 更新,`api` 直通命令在新端点获得类型化命令前可作缓冲。refresh token 以明文存放于 `config.toml`(与 gh 的 hosts.yml 回退相同),权限 0600/0700——v1 不接 OS keyring。会话有意不跨服务器泄漏。验收工作暴露了服务端 bug——同一秒内登出后重新登录会签发与已拉黑 JWT 逐字节相同的新令牌([#84](https://github.com/jukanntenn/markpost/issues/84))——在服务端修复前,验收套件以注释过的 sleep 规避。验证:`go test ./...` 与 `-race` 全绿、golangci-lint 零问题、验收测试对 dev 栈全绿。
