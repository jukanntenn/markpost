# Docker Build Specification

English | [中文](build-specification.zh.md)

## Base Images

| Image                | Stage                      | Version               | Size (compressed) |
| -------------------- | -------------------------- | --------------------- | ----------------- |
| `golang:1.26-alpine` | Backend builder            | Pinned to Alpine      | ~150MB            |
| `alpine:3.21`        | Backend runtime            | Pinned                | ~3MB              |
| `node:24-alpine3.21` | Frontend builder & runtime | Pinned to Alpine 3.21 | ~60MB             |

All base images are pinned to specific Alpine versions (`alpine:3.21`, `alpine3.21`) for build reproducibility. Unpinned `latest` tags are not used.

## Build Tool

**docker buildx** — Docker CLI plugin for multi-platform and cache-enabled builds.

Key features used:

- Multi-platform builds via QEMU emulation (`docker-container` driver)
- Local builder cache (registry-based cache is deliberately unused — see Build Cache below)
- Multi-stage Dockerfile builds

See the [buildx documentation](https://docs.docker.com/build/buildx/) for reference.

## Directory Structure

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

## Optimization Mechanisms

### Layer Cache Ordering

Dependencies are installed before source code is copied. This ensures that code changes don't invalidate the expensive dependency installation layer.

**Backend** (`go-build` stage of `docker/Dockerfile`):

1. `COPY go.mod go.sum` → `RUN go mod download` — cached unless dependencies change
2. `COPY . .` → `RUN go generate ./...` → `RUN CGO_ENABLED=0 go build` — only re-runs after source changes

**Frontend** (`node-build` stage):

1. `COPY package.json pnpm-lock.yaml pnpm-workspace.yaml` → `RUN pnpm install --frozen-lockfile` — cached unless dependencies change
2. `COPY . .` — invalidated by source changes
3. `RUN pnpm build` — only re-runs after source changes

### Static Build (Backend)

The backend builds with `CGO_ENABLED=0`: a pure-Go, statically linked binary with no C dependencies and no libc linkage concerns. The runtime image only needs `ca-certificates` and `tzdata` (plus Caddy and s6-overlay for the composite image).

### Static Export (Frontend)

Next.js is configured with `output: "export"`, which produces a plain static site under `out/`. The runtime image copies `out/` to `/app/frontend`, and Caddy serves it directly and reverse-proxies API paths to the Go backend — there is no Node process in the runtime image.

### Corepack (Frontend)

pnpm is activated via `corepack enable` instead of `npm install -g pnpm`. The exact pnpm version is pinned in `package.json`'s `packageManager` field, ensuring reproducible builds.

### Build Context Filtering

Every build surface — `docker/build.py` (production), the `devops/` dev compose files, and CI's `docker-publish.yml` — builds from the repo root, so the root `.dockerignore` is the only context filter; there are no subtree dockerignores.

No-slash patterns (`.env`, `*.log`) match the context root only, unlike gitignore where they match at any depth — nested local files therefore need explicit `**/` patterns: `**/.env*` and key material (`**/*.pem`, `**/*.key`, `**/id_rsa*`, …) keep local secrets out of the context even though git (and prek's detect-private-key) never sees them, and the `**/*.local*` families drop local variant configs (`docker/Caddyfile.local`, `docker/docker-compose.local.yml`). A future build that needs an in-tree file matching these patterns must add a `!` exception in the same change.

### Build Cache

Registry-based build cache (`--cache-to`/`--cache-from`) is deliberately NOT used: builds against the internal registry gain nothing from cross-machine cache layers (single builder machine) and the `mode=max` cache blobs would just consume registry disk. Only the local buildx builder cache applies; `--no-cache` disables it. CI release builds use GitHub Actions cache (`type=gha`) instead — see `.github/workflows/docker-publish.yml`.

## Build Script (docker/build.py)

### Behavior

The script performs two functions in order:

1. **Environment inspection** — verifies all requirements are met before building
2. **Image build** — invokes `docker buildx build` with the correct arguments

The script does **not** modify the environment. If requirements are not met, it exits with an error and instructions for manual resolution.

### Environment Checks

The following checks run before any build starts:

| Check                                     | Command                                | Failure |
| ----------------------------------------- | -------------------------------------- | ------- |
| Docker daemon running                     | `docker info`                          | Exit 2  |
| buildx plugin available                   | `docker buildx version`                | Exit 2  |
| Builder supports target platforms         | `docker buildx inspect`                | Exit 2  |
| QEMU registered for foreign architectures | `/proc/sys/fs/binfmt_misc/qemu-<arch>` | Exit 2  |

### CLI Flags

| Flag              | Description                                       | Default                           |
| ----------------- | ------------------------------------------------- | --------------------------------- |
| `--push`          | Push to registry (multi-platform)                 | Load locally (single platform)    |
| `--registry`      | Container registry address                        | `192.168.5.50:5000`               |
| `--tags`          | Additional image tags                             | `main` only (always incl., dedup) |
| `--platform`      | Target platform(s): `amd64`, `arm64`. Repeatable. | Host platform                     |
| `--all-platforms` | Build all target platforms (amd64 + arm64)        | Off                               |
| `--no-cache`      | Disable all build cache                           | Cache enabled                     |
| `--verbose`       | Full build output (no progress bar)               | Compact progress                  |

### Exit Codes

| Code | Meaning                                                                           |
| ---- | --------------------------------------------------------------------------------- |
| 0    | Success                                                                           |
| 1    | Build failure (buildx command failed)                                             |
| 2    | Environment check failure (missing tool, unregistered QEMU, unsupported platform) |
| 3    | Invalid arguments (conflicting flags, unknown platform)                           |

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

# Build arm64 explicitly
python3 docker/build.py --platform arm64

# Build with verbose output
python3 docker/build.py --verbose
```

1. Script checks environment (Docker daemon, buildx, builder)
2. Resolves target platforms (host platform by default; non-push collapses to a single platform)
3. Runs `docker buildx build --load`
4. Image available locally as `markpost:main` (plus any `--tags`)

### Normal Flow: Build and Push to Registry

```bash
# Push the host platform to the default registry, tagged main
python3 docker/build.py --push

# Push all platforms (cross-arch via QEMU) with an additional tag
python3 docker/build.py --push --all-platforms --tags 0.1.3
```

1. Script checks environment (Docker daemon, buildx, builder, QEMU for foreign architectures)
2. Resolves target platforms (all requested platforms for `--push`)
3. Runs `docker buildx build --push` (no registry cache)
4. Image pushed to the registry; multi-arch when more than one platform is requested

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
