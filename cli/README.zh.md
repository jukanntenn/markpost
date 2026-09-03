# markpost CLI

[English](README.md) | 简体中文

[markpost](../README.zh.md) 服务器的官方命令行客户端——独立二进制(`cli/` 是独立 Go module,基于 urfave/cli/v2),为人类与 AI agent 双方设计。

```bash
cd cli && make build          # 或:go build -o markpost ./cmd/markpost
./markpost config set server https://mp.example.com
./markpost auth login          # 终端下交互提示;脚本用 flag
./markpost posts create --title "Hello" "# Hello World

Some **markdown**."
# → https://mp.example.com/p-AbCdEf...
```

## 命令

| 命令                                   | 用途                                                                                             |
| -------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `auth login / status / token / logout` | 会话生命周期;登录支持提示、flag、`--password-stdin` 或 `--token`                                 |
| `posts create / list / view / delete`  | 发布(打印文章 URL)、列表(表格或 `--json`)、查看(`--format raw/html/url`)、删除(脚本中用 `--yes`) |
| `post-key show / rotate`               | 发布 key                                                                                         |
| `api <endpoint>`                       | gh 式直通,覆盖类型化命令之外的一切;相对路径打到 `/api/v1`(`markpost api me/retention`)           |
| `status`                               | 服务器 health / readiness / version(`--json`)                                                    |
| `config get/set`                       | 本地设置(`server`)                                                                               |
| `version`                              | CLI 版本                                                                                         |

## 天然 agent / 脚本友好

- **绝不阻塞**:仅当 stdin 与 stdout 都是终端才出现提示;其余场合给出可执行的 flag 错误。`--file -` 与 `--password-stdin` 接受管道输入。
- **机器可读输出**:结果走 stdout,诊断走 stderr;读命令支持 `--json`(管道时紧凑,TTY 时缩进)。
- **稳定退出码**(gh 契约):`0` 成功、`1` 失败、`2` 取消、`4` 认证——认证错误附带登录提示。
- **agent 检测**:设置 `AI_AGENT`(或已知 agent 的环境变量)后,用法错误输出完整帮助,agent 一轮即可自我纠正;User-Agent 追加 `Agent/<name>`。
- **会话自动刷新**:401 触发一次 token 刷新(持久化到配置文件)并重试。

## 配置

会话与设置存放于 `config.toml`(0600):`$MARKPOST_CONFIG_DIR`,否则 `$XDG_CONFIG_HOME/markpost`,否则 `~/.config/markpost`。环境变量覆盖文件:

| 变量                | 作用                                  |
| ------------------- | ------------------------------------- |
| `MARKPOST_SERVER`   | 服务器基础 URL(等同 `--server` flag)  |
| `MARKPOST_TOKEN`    | 无头场景的访问令牌——无刷新;401 即终局 |
| `MARKPOST_POST_KEY` | 无需登录会话即可发布                  |

## 开发

```bash
make test         # 单测(httptest 假后端;无需 Docker)
make test-race
make lint
make acceptance   # e2e:需要 MARKPOST_E2E_BASE_URL/USERNAME/PASSWORD,否则跳过
```

设计:[specs/cli.zh.md](../specs/cli.zh.md) · 决策:[MRFC](../.agents/mrfcs/implemented/2026-09-03-standalone-agent-cli.zh.md) · 子树规范:[AGENTS.md](AGENTS.md)
