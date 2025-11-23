#!/usr/bin/env bash
set -euo pipefail

# 用法：
#   REGISTRY=jukanntenn/markpost VERSION=0.0.7 ./scripts/publish-multiarch.sh
#   REGISTRY=jukanntenn/markpost ./scripts/publish-multiarch.sh   # 仅 latest

REGISTRY=${REGISTRY:-jukanntenn/markpost}
VERSION=${VERSION:-}

echo "[1/5] 登录镜像仓库"
docker login

echo "[2/5] 初始化 Buildx"
docker buildx create --use --name markpost-builder >/dev/null 2>&1 || true
docker buildx inspect --bootstrap >/dev/null

echo "[3/5] 检查 binfmt/QEMU 支持 (如首次运行可能需要 sudo)"
if ! docker run --privileged --rm tonistiigi/binfmt --version >/dev/null 2>&1; then
  echo "安装 binfmt 支持多架构仿真..."
  docker run --privileged --rm tonistiigi/binfmt --install all
fi

echo "[4/5] 构建并推送 latest 多架构镜像"
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ${REGISTRY}:latest \
  -f Dockerfile . --push

if [[ -n "${VERSION}" ]]; then
  echo "[5/5] 构建并推送 ${VERSION} 多架构镜像"
  docker buildx build \
    --platform linux/amd64,linux/arm64 \
    -t ${REGISTRY}:${VERSION} \
    -f Dockerfile . --push
else
  echo "[5/5] 跳过版本标签推送 (未设置 VERSION)"
fi

echo "完成：已推送 ${REGISTRY}:latest${VERSION:+ 和 ${REGISTRY}:${VERSION}}"
