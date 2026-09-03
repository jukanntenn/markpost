# Docker 构建规范

[English](build-specification.md) | 中文

<a id="base-images"></a>

## 基础镜像

| 镜像                 | 阶段             | 版本               | 体积（压缩后） |
| -------------------- | ---------------- | ------------------ | -------------- |
| `golang:1.26-alpine` | 后端构建器       | 锁定到 Alpine      | ~150MB         |
| `alpine:3.21`        | 后端运行时       | 锁定               | ~3MB           |
| `node:24-alpine3.21` | 前端构建与运行时 | 锁定到 Alpine 3.21 | ~60MB          |

所有基础镜像都锁定到具体的 Alpine 版本（`alpine:3.21`、`alpine3.21`），保证构建可复现。不使用未锁定的 `latest` 标签。

<a id="build-tool"></a>

## 构建工具

**docker buildx** —— 用于多平台与缓存构建的 Docker CLI 插件。

用到的关键特性：

- 通过 QEMU 仿真做多平台构建（`docker-container` 驱动）
- 本地构建器缓存（刻意不使用基于 registry 的缓存 —— 见下文"构建缓存"）
- 多阶段 Dockerfile 构建

参考 [buildx 文档](https://docs.docker.com/build/buildx/)。

<a id="directory-structure"></a>

## 目录结构

```
markpost/
├── .dockerignore                    # The only dockerignore — every build uses the repo root as context
├── docker/                          # Production image building
│   ├── build.py                     # Build script (environment check + buildx invocation)
│   ├── Dockerfile                   # Single multi-stage production image (backend + frontend + s6 runtime)
│   ├── Caddyfile / Caddyfile.local  # Caddy configs baked into / referenced by the image
│   ├── docker-compose.yml           # Acceptance compose (production topology, local source)
│   ├── docker-compose.local.yml     # Local run compose
│   └── s6/                          # s6-overlay service definitions (Caddy + Go backend)
├── devops/                          # Development environment
│   ├── dev.py                       # Dev environment manager (all services in Docker)
│   ├── docker-compose.yml           # Dev services (backend, frontend, postgres)
│   ├── backend.Dockerfile           # Backend dev image (with hot-reload)
│   ├── frontend.Dockerfile          # Frontend dev image
│   └── ansible/                     # Deployment playbooks and templates
├── backend/                         # No subtree dockerignore — root **/ patterns cover nested paths
└── frontend/
    └── package.json                 # Contains "packageManager" field for corepack
```

<a id="optimization-mechanisms"></a>

## 优化机制

<a id="layer-cache-ordering"></a>

### 层缓存排序

依赖先于源码复制安装。这保证代码变更不会让昂贵的依赖安装层失效。

**后端**（`docker/Dockerfile` 的 `go-build` 阶段）：

1. `COPY go.mod go.sum` → `RUN go mod download` —— 除非依赖变化，否则命中缓存
2. `COPY . .` → `RUN go generate ./...` → `RUN CGO_ENABLED=0 go build` —— 仅在源码变化后重新执行

**前端**（`node-build` 阶段）：

1. `COPY package.json pnpm-lock.yaml pnpm-workspace.yaml` → `RUN pnpm install --frozen-lockfile` —— 除非依赖变化，否则命中缓存
2. `COPY . .` —— 源码变化使其失效
3. `RUN pnpm build` —— 仅在源码变化后重新执行

<a id="static-build-backend"></a>

### 静态构建（后端）

后端以 `CGO_ENABLED=0` 构建：纯 Go、静态链接的二进制，无 C 依赖、无 libc 链接问题。运行时镜像只需要 `ca-certificates` 和 `tzdata`（组合镜像再加上 Caddy 与 s6-overlay）。

<a id="static-export-frontend"></a>

### 静态导出（前端）

Next.js 配置了 `output: "export"`，在 `out/` 下产出纯静态站点。运行时镜像把 `out/` 复制到 `/app/frontend`，Caddy 直接伺服它并把 API 路径反向代理到 Go 后端 —— 运行时镜像中没有 Node 进程。

<a id="corepack-frontend"></a>

### Corepack（前端）

pnpm 通过 `corepack enable` 激活，而非 `npm install -g pnpm`。确切的 pnpm 版本锁定在 `package.json` 的 `packageManager` 字段，保证构建可复现。

<a id="build-context-filtering"></a>

### 构建上下文过滤

所有构建面 —— `docker/build.py`（生产）、`devops/` 的 dev compose、CI 的 `docker-publish.yml` —— 都以仓库根为上下文，因此根 `.dockerignore` 是唯一生效的过滤器；不存在子树 dockerignore。

无斜杠模式（`.env`、`*.log`）只匹配上下文根，这一点与 gitignore（匹配任意深度）不同 —— 嵌套的本地文件因此需要显式 `**/` 模式：`**/.env*` 与密钥类（`**/*.pem`、`**/*.key`、`**/id_rsa*` 等）让本地密钥进不了上下文，即使 git（以及 prek 的 detect-private-key）根本看不到它们；`**/*.local*` 家族排除本地变体配置（`docker/Caddyfile.local`、`docker/docker-compose.local.yml`）。未来构建若需要命中这些模式的入树文件，必须在同一变更中加 `!` 例外。

<a id="build-cache"></a>

### 构建缓存

刻意不使用基于 registry 的构建缓存（`--cache-to`/`--cache-from`）：对内部 registry 的构建无法从跨机器缓存层获益（单一构建机），`mode=max` 缓存 blob 只会消耗 registry 磁盘。只有本地 buildx 构建器缓存生效；`--no-cache` 将其禁用。CI 发布构建改用 GitHub Actions 缓存（`type=gha`）—— 见 `.github/workflows/docker-publish.yml`。

<a id="build-script-dockerbuildpy"></a>

## 构建脚本（docker/build.py）

<a id="behavior"></a>

### 行为

脚本按顺序做两件事：

1. **环境检查** —— 构建前确认所有要求满足
2. **镜像构建** —— 以正确参数调用 `docker buildx build`

脚本**不**修改环境。要求不满足时，它带错误信息与手工解决指引退出。

<a id="environment-checks"></a>

### 环境检查

任何构建开始前运行以下检查：

| 检查                 | 命令                                   | 失败时 |
| -------------------- | -------------------------------------- | ------ |
| Docker daemon 运行中 | `docker info`                          | Exit 2 |
| buildx 插件可用      | `docker buildx version`                | Exit 2 |
| 构建器支持目标平台   | `docker buildx inspect`                | Exit 2 |
| 异构架构已注册 QEMU  | `/proc/sys/fs/binfmt_misc/qemu-<arch>` | Exit 2 |
| 版本串可解析         | `scripts/build_version.py`             | Exit 2 |

<a id="version-string"></a>

### 版本串

`VERSION` build-arg —— 经 `-X main.version` 烘焙进 Go 二进制、由 `/api/v1/version` 上报 —— 由 `scripts/build_version.py` 计算；它是与部署 playbook 的 dev 版本检查（`devops/ansible/deploy.yml`）共享的唯一实现：

- **干净树：** `git describe --tags --always`。发布镜像由 CI 从干净的 tag checkout 构建，因此发布版本串就是 tag 本身。
- **脏树：** `<describe>-dirty.<8 hex>` —— 基底提交加工作区增量的确定性摘要（tracked diff 与未跟踪且未 ignore 文件的内容）。同提交、不同内容的两场脏构建互不相等；重建完全相同的树会复现同一字符串。部署检查因此对重建镜像放行、对构建后编辑过的 checkout 失败（MRFC 2026-09-03-dirty-tree-image-version-string）。

版本解析失败以退出码 2 结束（环境检查失败）。无提交的仓库烘焙 `dev`。

<a id="cli-flags"></a>

### CLI 标志

| 标志              | 说明                                 | 默认值                  |
| ----------------- | ------------------------------------ | ----------------------- |
| `--push`          | 推送到 registry（多平台）            | 本地加载（单平台）      |
| `--registry`      | 容器 registry 地址                   | `192.168.5.50:5000`     |
| `--tags`          | 额外镜像标签                         | 仅 `main`（总含，去重） |
| `--platform`      | 目标平台：`amd64`、`arm64`。可重复。 | 宿主平台                |
| `--all-platforms` | 构建全部目标平台（amd64 + arm64）    | 关                      |
| `--no-cache`      | 禁用全部构建缓存                     | 缓存启用                |
| `--verbose`       | 完整构建输出（无进度条）             | 紧凑进度                |

<a id="exit-codes"></a>

### 退出码

| 码  | 含义                                              |
| --- | ------------------------------------------------- |
| 0   | 成功                                              |
| 1   | 构建失败（buildx 命令失败）                       |
| 2   | 环境检查失败（缺工具、QEMU 未注册、平台不受支持） |
| 3   | 无效参数（标志冲突、未知平台）                    |

<a id="error-output-format"></a>

### 错误输出格式

所有环境错误遵循此格式：

```
ERROR: <description of the problem>
HINT: <command or action to resolve>
AGENT: Stop all subsequent actions. Report this error to the user. Do not attempt to resolve automatically.
```

<a id="build-workflows"></a>

## 构建工作流

<a id="normal-flow-build-and-load-locally"></a>

### 常规流程：本地构建并加载

```bash
# Build both images for the host platform
python3 docker/build.py

# Build arm64 explicitly
python3 docker/build.py --platform arm64

# Build with verbose output
python3 docker/build.py --verbose
```

1. 脚本检查环境（Docker daemon、buildx、构建器）
2. 解析目标平台（默认宿主平台；非推送时坍缩为单平台）
3. 运行 `docker buildx build --load`
4. 镜像以 `markpost:main`（加任意 `--tags`）在本地可用

<a id="normal-flow-build-and-push-to-registry"></a>

### 常规流程：构建并推送到 registry

```bash
# Push the host platform to the default registry, tagged main
python3 docker/build.py --push

# Push all platforms (cross-arch via QEMU) with an additional tag
python3 docker/build.py --push --all-platforms --tags 0.1.3
```

1. 脚本检查环境（Docker daemon、buildx、构建器、异构架构的 QEMU）
2. 解析目标平台（`--push` 时为全部所请求的平台）
3. 运行 `docker buildx build --push`（无 registry 缓存）
4. 镜像推送到 registry；请求多于一个平台时为多架构

<a id="abnormal-flow-environment-failure"></a>

### 异常流程：环境故障

```bash
$ python3 docker/build.py --push --platform arm64
ERROR: QEMU binfmt for arm64 is not registered — required for cross-platform build (linux/arm64).
HINT: Run: docker run --rm --privileged tonistiigi/binfmt --install arm64
AGENT: Stop all subsequent actions. Report this error to the user. Do not attempt to resolve automatically.
```

脚本以退出码 2 退出。不尝试构建。用户（或 AI agent）必须先解决环境问题再重试。

<a id="abnormal-flow-build-failure"></a>

### 异常流程：构建失败

```bash
$ python3 docker/build.py
ERROR: Build failed for markpost (exit code 1)!
```

脚本以退出码 1 退出。buildx 的错误输出在 stderr 可见。用户应检查输出中的编译或依赖错误。
