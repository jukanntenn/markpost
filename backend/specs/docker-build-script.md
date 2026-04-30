# Docker Build Script Specification

This specification defines the mechanism, constraints, and rules for a Python-based Docker build script that produces multi-architecture container images using Docker buildx.

## Requirements

- Python 3.8+, no third-party dependencies
- `argparse` for command-line argument parsing
- `logging` module with `StreamHandler` for console output
- `subprocess` for invoking Docker CLI commands

## Command-Line Interface

### Positional Arguments

None. All configuration is via optional flags.

### Optional Arguments

| Flag | Type | Default | Description |
|---|---|---|---|
| `--push` | flag (boolean) | off (load mode) | Push images to registry instead of loading locally |
| `--registry` | string | Project-specific default | Container registry hostname/address |
| `--tags` | string (repeatable) | empty | Additional image tags beyond the mandatory `dev` tag |

`--tags` uses `action="extend"` with `nargs="+"`, accepting one or more values per occurrence:

```bash
# Both forms are valid and combinable:
--tags v1.0.0 latest
--tags v1.0.0 --tags latest
```

### Derived Values (not configurable)

| Property | Value | Rationale |
|---|---|---|
| Image name | Project-specific, fixed | Hardcoded, no flag to override |
| Platforms | `linux/amd64`, `linux/arm64` | Hardcoded multi-architecture support |
| Mandatory tag | `dev` | Always applied, represents a snapshot of the latest codebase |

## Behavior

### Tag Resolution

The `dev` tag is always applied. If `--tags` is specified, those tags are appended.

| Scenario | Tags Applied |
|---|---|
| (no `--tags`) | `dev` |
| `--tags v1.0.0` | `dev`, `v1.0.0` |
| `--tags v1.0.0 latest` | `dev`, `v1.0.0`, `latest` |

### Load Mode (default)

- Triggered when `--push` is **not** specified.
- Builds for the **host architecture only** (single platform).
- Image tags have **no registry prefix**.
- Uses `--load` flag in the buildx command.

Tag format: `<image-name>:<tag>`

Example: `myapp:dev`, `myapp:v1.0.0`

### Push Mode

- Triggered when `--push` is specified.
- Builds for **all supported platforms** (`linux/amd64`, `linux/arm64`).
- Image tags include the **registry prefix**.
- Uses `--push` flag in the buildx command.

Tag format: `<registry>/<image-name>:<tag>`

Example: `registry.example.com/myapp:dev`, `registry.example.com/myapp:v1.0.0`

## Buildx Handling

### What the Script Does

1. Check that `docker buildx version` exits successfully.
2. Inspect the currently active builder to determine supported platforms.
3. If required platforms are missing, register QEMU binfmt for foreign architectures (see QEMU Registration below).
4. After QEMU registration, bootstrap the active builder (`docker buildx inspect --bootstrap`) so it re-detects newly available platforms.
5. Verify the builder now supports all required platform(s).

### What the Script Does NOT Do

- Create builder instances.
- Switch the active builder (`docker buildx use`).
- Modify any buildx state beyond bootstrapping the already active builder.

If verification fails after all remediation, the script must output a human- and AI-agent-friendly error message describing what was expected vs. what was found, then exit with a non-zero code.

## QEMU Registration

Cross-platform builds (e.g. arm64 on an amd64 host) require QEMU user-mode emulation registered via the kernel's `binfmt_misc` interface. This registration is volatile — lost on reboot and after `docker system prune`.

The script handles this automatically:

1. Before building, determine which target platforms are "foreign" (not the host architecture).
2. Check if QEMU is already registered by testing `/proc/sys/fs/binfmt_misc/qemu-<arch>`.
3. If not registered, run `docker run --rm --privileged tonistiigi/binfmt --install <archs>` to register.
4. Verify registration succeeded by re-checking the binfmt entries.
5. If registration fails, output an error with the manual command and exit.

### Architecture Mapping

| Docker Platform | QEMU binfmt name |
|---|---|
| `linux/arm64` | `qemu-aarch64` |

This mapping is extensible — additional foreign architectures can be added to the script's `QEMU_ARCH_MAP` as needed.

## Logging

- Use `logging` module with `StreamHandler`.
- `INFO` level messages go to `stdout`.
- `WARNING` and `ERROR` level messages go to `stderr`.
- No `--verbose` or `-v` flag. Key observability information is always output at `INFO` level.

### What to Log at INFO

- Build configuration summary (image, tags, mode, platforms, registry, builder).
- The full `docker buildx build` command being executed.
- `docker buildx build` stdout/stderr passed through to the terminal unchanged.
- Final result on success.

### What to Log at ERROR

- Buildx unavailability.
- Platform support mismatch.
- Build failure context (see Error Output below).

## Error Output

On build failure, output a structured context block to `stderr` containing all information relevant for debugging:

```
Fields:
  - Image:       full image name with all tags
  - Mode:        load or push
  - Platforms:   target platform(s)
  - Builder:     active builder name and driver type
  - Context:     build context directory (absolute path)
  - Dockerfile:  Dockerfile path (absolute path)
  - Command:     the exact docker buildx build command that failed
  - Error:       the error message from the failed command
```

Do **not** include speculative "possible causes" or troubleshooting suggestions. The context block and the error message together provide sufficient information for a human or AI agent to diagnose the issue.

## Host Architecture Detection

For load mode, detect the host architecture using `platform.machine()`:

| `platform.machine()` | Docker Platform |
|---|---|
| `x86_64` | `linux/amd64` |
| `aarch64` | `linux/arm64` |
| Any other | `linux/amd64` (fallback) |

## Infrastructure Prerequisites

The script assumes the following infrastructure is correctly configured before it runs. It does **not** manage any of these — they are the operator's responsibility.

### Buildx Builder

The script uses the **currently active** buildx builder. It must be a `docker-container` driver instance (not the default `docker` driver) for multi-platform and push support.

The builder must already exist and be active before the script runs. Create it once:

```bash
docker buildx create --name my-builder --driver docker-container --use
```

### Self-Hosted HTTP Registry

When pushing to a self-hosted registry that serves plain HTTP (no TLS), two separate configurations are required — Docker CLI and the buildkit builder are independent processes with independent configs.

**1. Docker daemon** — add the registry as an insecure registry:

```json
// /etc/docker/daemon.json
{
  "insecure-registries": ["my-registry:5000"]
}
```

Then `sudo systemctl restart docker`.

**2. Buildkit builder** — the `docker-container` driver does NOT inherit `/etc/docker/daemon.json`. The registry must be configured in the builder's BuildKit config:

```toml
# buildkitd.toml
[registry."my-registry:5000"]
  http = true
```

Create the builder with this config:

```bash
docker buildx create --name my-builder \
  --driver docker-container \
  --config buildkitd.toml \
  --use
```

If the builder already exists without this config, it must be deleted and recreated.

### Summary

| Component | Config scope | What to configure |
|---|---|---|
| Docker daemon | `/etc/docker/daemon.json` | `insecure-registries` for the registry host |
| Buildkit builder | `buildkitd.toml` via `--config` | `[registry."<host>"] http = true` |
| QEMU binfmt | kernel `binfmt_misc` | Auto-managed by the script |

## Script Conventions

- Shebang: `#!/usr/bin/env python3`
- File location: `docker/build.py` (relative to project root)
- Working directory: script auto-detects project root as the parent of the `docker/` directory containing the script.
- Dockerfile path: `docker/Dockerfile` relative to project root.
- Exit code: 0 on success, 1 on failure.

## Example Usage

```bash
# Local build for host architecture, dev tag only
./docker/build.py

# Local build with additional tags
./docker/build.py --tags v1.0.0

# Push multi-architecture image to default registry
./docker/build.py --push

# Push to a custom registry with multiple tags
./docker/build.py --push --registry ghcr.io/myorg --tags v2.0.0 latest
```
