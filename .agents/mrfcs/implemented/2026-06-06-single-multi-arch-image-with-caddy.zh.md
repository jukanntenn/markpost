# MRFC: One multi-arch Docker image, Caddy as the edge

Status: implemented

[English](2026-06-06-single-multi-arch-image-with-caddy.md) | 中文

## Problem

把后端与前端作为各自独立的制品发布，意味着逐环境组装：前端要一个 Node 进程或静态主机，每次部署都要挑选并配置一个反向代理，而且每次部署两个半区之间都存在版本偏斜。每个环境（staging、生产、homelab）都在用不同的答案重解同一套管道问题。

## Decision

markpost 以单一带版本的多架构 Docker 镜像发布，由 `docker/build.py` 从 `docker/Dockerfile` 构建。容器内部，s6-overlay 监管 Go 后端与 Caddy；前端以其静态导出的形式烘焙进镜像。Caddy 服务导出的前端，并把 `/api/v1` 与 `/swagger` 反向代理到后端，因此公共面是单一 origin，无需任何跨域配置。环境差异活在 `devops/ansible/` 的 group_vars 与 Caddyfile 模板里，不在镜像里。

## Alternatives considered

**前端与后端各自成像。** 每个半区可独立定版，但每次部署都必须锁定一对兼容版本，且每个环境仍需要一套代理与 TLS 方案；版本偏斜变成运维者的问题。

**前端用 Node 服务器（SSR/standalone）。** 保住服务端渲染的可用性，但为一个产品用不上的运行时付费 —— 应用完全由客户端渲染 —— 还为监管增加第二个进程，渲染收益为零。

**用 nginx 替代 Caddy。** 作为反向代理能力相当，但 Caddy 的自动 TLS 签发和显著更小的配置更契合自托管目标。

## Consequences

一个制品、一个版本号、一个部署单元：e2e 测试的东西（`dagger call -m e2e all`）正是生产里运行的东西。容器构建是多架构的，因此更慢。Caddy 如今是承重组件 —— 它的逐环境模板是部署面的一部分，并在 prek 中接受语法检查。
