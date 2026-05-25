# Docker Build Specification

## Base Images

| Image | Stage | Version | Size (compressed) |
|-------|-------|---------|-------------------|
| `golang:1.26-alpine` | Backend builder | Pinned to Alpine | ~150MB |
| `alpine:3.21` | Backend runtime | Pinned | ~3MB |
| `node:24-alpine3.21` | Frontend builder & runtime | Pinned to Alpine 3.21 | ~60MB |

All base images are pinned to specific Alpine versions (`alpine:3.21`, `alpine3.21`) for build reproducibility. Unpinned `latest` tags are not used.

## Build Tool

**docker buildx** — Docker CLI plugin for multi-platform and cache-enabled builds.

Key features used:
- Multi-platform builds via QEMU emulation (`docker-container` driver)
- Registry-based build cache (`--cache-to`/`--cache-from`)
- Multi-stage Dockerfile builds

See [buildx reference](../../wiki/buildx-reference.md) for detailed buildx knowledge.

## Directory Structure

```
markpost/
├── docker/                          # Production image building
│   ├── build.py                     # Build script (environment check + buildx invocation)
│   ├── backend.Dockerfile           # Backend multi-stage production image
│   └── frontend.Dockerfile          # Frontend multi-stage production image
├── devops/                          # Development environment
│   ├── dev.py                       # Dev environment manager
│   ├── docker-compose.yml           # Dev services (backend, postgres)
│   ├── backend.Dockerfile           # Backend dev image (with hot-reload)
│   └── ansible/                     # Provisioning playbooks
├── backend/
│   └── .dockerignore                # Excludes tests, docs, tools from build context
├── frontend/
│   ├── .dockerignore                # Excludes .env.local from build context
│   └── package.json                 # Contains "packageManager" field for corepack
└── wiki/
    └── buildx-reference.md          # Buildx knowledge base
```

## Optimization Mechanisms

### Layer Cache Ordering

Dependencies are installed before source code is copied. This ensures that code changes don't invalidate the expensive dependency installation layer.

**Backend** (`backend.Dockerfile`):
1. `COPY go.mod go.sum` → `RUN go mod download` — cached unless dependencies change
2. `COPY . .` — invalidated by source changes
3. `RUN CGO_ENABLED=1 CGO_LDFLAGS="-static" go build` — only re-runs after source changes

**Frontend** (`frontend.Dockerfile`):
1. `COPY package.json pnpm-lock.yaml pnpm-workspace.yaml` → `RUN pnpm install --frozen-lockfile` — cached unless dependencies change
2. `COPY . .` — invalidated by source changes
3. `RUN pnpm build` — only re-runs after source changes

### Static Linking (Backend)

The backend binary is statically linked with `CGO_LDFLAGS="-static"`. This embeds sqlite3 and musl libc into the binary, eliminating runtime shared library dependencies. The runtime image only needs `ca-certificates` and `tzdata`.

### Standalone Output (Frontend)

Next.js is configured with `output: "standalone"` which produces a minimal server bundle. The runtime image copies only:
- `.next/standalone/` — the server and its dependencies
- `.next/static/` — static assets
- `public/` — public assets

No `node_modules` in the runtime stage.

### Corepack (Frontend)

pnpm is activated via `corepack enable` instead of `npm install -g pnpm`. The exact pnpm version is pinned in `package.json`'s `packageManager` field, ensuring reproducible builds.

### Build Context Filtering

Each build context has a `.dockerignore` that excludes non-essential files:
- **Backend**: test files, generated docs, dev tools, config files, IDE files
- **Frontend**: `.env.local`

### Registry-Based Build Cache

When pushing images (`--push`), build cache is stored in the same registry using `--cache-to`/`--cache-from` with `mode=max`. Cache is scoped per-platform to avoid cross-contamination between architecture-specific build outputs.

Cache reference pattern: `<registry>/<image>:cache` (e.g., `192.168.5.50:5000/markpost:cache`).

## Build Script (`docker/build.py`)

### Behavior

The script performs two functions in order:

1. **Environment inspection** — verifies all requirements are met before building
2. **Image build** — invokes `docker buildx build` with the correct arguments

The script does **not** modify the environment. If requirements are not met, it exits with an error and instructions for manual resolution.

### Environment Checks

The following checks run before any build starts:

| Check | Command | Failure |
|-------|---------|---------|
| Docker daemon running | `docker info` | Exit 2 |
| buildx plugin available | `docker buildx version` | Exit 2 |
| Builder supports target platforms | `docker buildx inspect` | Exit 2 |
| QEMU registered for foreign architectures | `/proc/sys/fs/binfmt_misc/qemu-<arch>` | Exit 2 |

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--push` | Push to registry (multi-platform) | Load locally (single platform) |
| `--registry` | Container registry address | `192.168.5.50:5000` |
| `--tags` | Additional image tags | `dev` only |
| `--backend-only` | Build only the backend image | Build both |
| `--frontend-only` | Build only the frontend image | Build both |
| `--platform` | Target platform(s): `amd64`, `arm64`. Repeatable. | Both platforms |
| `--no-cache` | Disable all build cache | Cache enabled |
| `--verbose` | Full build output (no progress bar) | Compact progress |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Build failure (buildx command failed) |
| 2 | Environment check failure (missing tool, unregistered QEMU, unsupported platform) |
| 3 | Invalid arguments (conflicting flags, unknown platform) |

### Error Output Format

All environment errors follow this format:

```
ERROR: <description of the problem>
HINT: <command or action to resolve>
AGENT: Stop all subsequent actions. Report this error to the user. Do not attempt to resolve automatically.
```

## Build Workflows

### Normal Flow: Build and Load Locally

```bash
# Build both images for the host platform
python3 docker/build.py

# Build only arm64
python3 docker/build.py --platform arm64

# Build only backend with verbose output
python3 docker/build.py --backend-only --verbose
```

1. Script checks environment (Docker daemon, buildx, builder)
2. Resolves target platforms (single host platform for `--load`)
3. Runs `docker buildx build --load` for each selected image
4. Images available locally as `markpost:dev` and/or `markpost-web:dev`

### Normal Flow: Build and Push to Registry

```bash
# Push both platforms to default registry
python3 docker/build.py --push

# Push arm64 only with additional tag
python3 docker/build.py --push --platform arm64 --tags v1.2.0
```

1. Script checks environment (Docker daemon, buildx, builder, QEMU)
2. Resolves target platforms (all specified platforms for `--push`)
3. Runs `docker buildx build --push` with `--cache-from`/`--cache-to` for each image
4. Images pushed to registry with multi-architecture manifest

### Abnormal Flow: Environment Failure

```bash
$ python3 docker/build.py --push --platform arm64
ERROR: QEMU binfmt for arm64 is not registered — required for cross-platform build (linux/arm64).
HINT: Run: docker run --rm --privileged tonistiigi/binfmt --install arm64
AGENT: Stop all subsequent actions. Report this error to the user. Do not attempt to resolve automatically.
```

The script exits with code 2. No build is attempted. The user (or AI agent) must resolve the environment issue before retrying.

### Abnormal Flow: Build Failure

```bash
$ python3 docker/build.py
ERROR: Build failed for markpost (exit code 1)!
```

The script exits with code 1. The buildx error output is visible in stderr. The user should inspect the output for compilation or dependency errors.
