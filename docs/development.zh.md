# 开发指南

[English](development.md) | 中文

如何在本地搭建并运行 markpost。文档规则见 [AGENTS.md](AGENTS.md)；部署见 [deployment.zh.md](deployment.zh.md)。

<a id="prerequisites"></a>

## 前置条件

| 工具                    | 版本        | 说明             | 安装                                                                    |
| ----------------------- | ----------- | ---------------- | ----------------------------------------------------------------------- |
| Go                      | 1.26+       | 后端语言         | [go.dev/dl](https://go.dev/dl/)                                         |
| Node.js                 | 24+         | 前端运行时       | [nodejs.org](https://nodejs.org/)                                       |
| pnpm                    | 11+         | 前端包管理器     | [pnpm.io/installation](https://pnpm.io/installation)                    |
| Docker & Docker Compose | Compose v2+ | 开发环境服务     | [docs.docker.com/get-docker](https://docs.docker.com/get-docker/)       |
| Python 3                | 3.12+       | 开发环境编排脚本 | [python.org](https://www.python.org/downloads/)                         |
| golangci-lint           | latest      | Go linter        | [golangci-lint.run/install](https://golangci-lint.run/welcome/install/) |
| air                     | latest      | 开发期 Go 热重载 | [github.com/cosmtrek/air](https://github.com/cosmtrek/air#installation) |

Swagger 生成使用经 `backend/go.mod` 中 `go tool` 钉住的 `swag` —— 无需单独安装。PostgreSQL 以容器运行；无需本地安装数据库。

<a id="quick-start"></a>

## 快速开始

<a id="option-1-dev-py-recommended"></a>

### 方式一 —— `dev.py`（推荐）

在 Docker Compose 中一并启动 PostgreSQL、后端与前端：

```bash
python3 devops/dev.py start   # start all services
python3 devops/dev.py stop    # stop all services
python3 devops/dev.py logs [backend|frontend|postgres]
```

- 前端：<http://localhost:3034>
- 后端：<http://localhost:7330>
- 数据库：`docker exec markpost-postgres psql -U markpost` —— 开发 compose 刻意不发布 Postgres 端口；用 `docker exec`。

<a id="option-2-vs-code-cursor-trae-compatible-ides"></a>

### 方式二 —— VS Code / Cursor / Trae 及兼容 IDE

项目附带 `.vscode/tasks.json`，含三个任务：

- **Start All** —— 并行运行后端与前端
- **Start Backend** —— 在 `backend/` 以开发 JWT 密钥启动 `air`
- **Start Frontend** —— 在 `frontend/` 启动 `pnpm dev`

打开命令面板（`Ctrl+Shift+P`）→ **Tasks: Run Task** → 选择任务。绑定快捷键（如 `Alt+R` → "Start All"）：打开键盘快捷方式 JSON（`Ctrl+Shift+P` → **Preferences: Open Keyboard Shortcuts (JSON)**）并添加：

```json
{
  "key": "alt+r",
  "command": "workbench.action.tasks.runTask",
  "args": "Start All"
}
```

注意：确保 `air` 与 `pnpm` 在 PATH 中。

<a id="option-3-manual"></a>

### 方式三 —— 手动

在宿主机上运行服务需要自备 PostgreSQL 17，可达于所配置的 DSN（开发 compose 不发布 5432）。

**后端**（air 热重载）：

```bash
cd backend
cp config.example.toml config.toml   # edit [db] dsn to point at your Postgres
air
```

开发服务器启动于 <http://localhost:7330>。在 `config.toml` 设置 `debug = true` 启用调试模式。

**前端：**

```bash
cd frontend
pnpm dev
```

开发服务器启动于 <http://localhost:3034>。

<a id="install-dependencies"></a>

## 安装依赖

`python3 devops/dev.py start` 首次运行会自动安装依赖。手动安装：

**后端：**

```bash
cd backend
go mod download
```

**前端：**

```bash
cd frontend
pnpm install
```

<a id="lint"></a>

## Lint

prek 拥有每一次 format/lint 调用（见根目录 `prek.toml`、`backend/prek.toml`、`frontend/prek.toml`）。`prek install` 之后，`git commit` 会运行检查；按需运行：

```bash
prek run --all-files          # everything CI's Lint job runs
prek run --group fmt --files <path>   # just the fixers, for specific files
```

各树的命令保持不变：`backend/` 里 `golangci-lint run`，`frontend/` 里 `pnpm lint`。

<a id="run-tests"></a>

## 运行测试

**后端**（需要运行中的 Docker 守护进程 —— 测试经 testcontainers-go 启动真实 PostgreSQL 容器）：

```bash
cd backend
go test ./...                        # all tests
go test ./internal/service/post/...  # specific package
```

Docker 不可用时设置 `TESTCONTAINERS_SKIP=1` 跳过容器测试。

**前端：**

```bash
cd frontend
pnpm test          # Vitest in watch mode
pnpm test:run      # single run (CI)
```

**E2E**（独立工作区；Playwright，仅 chromium）：

```bash
cd e2e
pnpm test                                                        # local run
dagger call -m e2e all --source ..                               # from repo root: full CI-fidelity run
dagger call -m e2e test --test-file=login.spec.ts --source ..    # single spec
```

每个 dagger spec 在隔离沙箱中运行（PostgreSQL + 后端 + 前端 + Playwright 容器）。

<a id="build"></a>

## 构建

**后端：**

```bash
cd backend
go build ./cmd/server
```

**前端**（静态导出到 `out/`）：

```bash
cd frontend
pnpm build
```

<a id="generate-swagger-docs"></a>

## 生成 Swagger 文档

```bash
cd backend
go generate ./...
```

这是唯一的再生成入口（`go tool swag` 钉在 go.mod，外加内嵌 CSS 构建）。后端以 `debug = true` 运行时，Swagger UI 位于 `/swagger/index.html`。

<a id="api-testing-with-yaak"></a>

## 用 yaak 测试 API

后端在 `backend/docs/swagger.json` 提供 Swagger 2.0 规范，[yaak](https://yaak.app/) 原生导入 —— 无需转换脚本。

<a id="import-into-yaak"></a>

### 导入 yaak

1. 打开 yaak
2. **File** → **Import Into Workspace**
3. 选择 `backend/docs/swagger.json`（yaak 自动识别 Swagger 2.0）
4. 将工作台 base URL 设为 `http://localhost:7330`
5. 对需要认证的端点，设置带 access token 的 `Authorization` 头

<a id="workflow"></a>

### 工作流

后端 API 变更时：

1. 更新 Go 代码中的 Swagger 注解
2. 运行 `go generate ./...` 重新生成 `backend/docs/swagger.json`
3. 重新导入 yaak（环境配置会被保留）

<a id="configuration"></a>

## 配置

后端从三个来源读取配置（优先级高者胜）：

1. **环境变量** —— 前缀 `MARKPOST_`，嵌套键用 `__`

   ```bash
   MARKPOST_DEBUG=true
   MARKPOST_SERVER__PORT=8080
   MARKPOST_DB__DSN="postgres://user:pass@localhost:5432/markpost?sslmode=disable"
   ```

2. **TOML 文件** —— 二进制旁的 `config.toml`，或经 `-c /path/to/config.toml`
3. **内建默认值** —— 完整参考见 `backend/config.example.toml`

环境变量是覆盖默认值的推荐方式。

前端只有一个环境变量：`BACKEND_URL`（默认 `http://127.0.0.1:7330`）。它供 `next.config.ts` 中的开发服务器重写使用，在开发期把 `/api/v1` 与 `/swagger` 代理到后端。生产没有前端服务器 —— Caddy 反代这些路径 —— 所以 `BACKEND_URL` 只影响本地开发。在 `frontend/.env.local`（gitignored）中覆盖。
