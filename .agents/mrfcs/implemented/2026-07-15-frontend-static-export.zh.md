# MRFC: Frontend is a static export with dev-only API rewrites

Status: implemented

[English](2026-07-15-frontend-static-export.md) | 中文

## Problem

前端曾假设边缘处有一个 Node 服务器：把 `/api` 服务端代理到后端、请求时配置 i18n、服务端渲染路由。这个假设把 Node 运行时强塞进每次部署，并与单镜像部署模型作对 —— 在那个模型里，后端前面唯一的进程是 Caddy，而非 Node。

## Decision

`frontend/next.config.ts` 无条件钉住 `output: "export"`：构建产出 `frontend/out/` 下的纯静态文件，由 Caddy 服务。仅在开发期（`NODE_ENV !== "production"`），一个 `rewrites()` 块把 `/api/v1` 与 `/swagger` 代理到 `BACKEND_URL`（默认 `http://127.0.0.1:7330`），让 dev server 与本地后端对话；该块只在 dev 挂上，因为在静态导出下仅 `rewrites` 键的存在就会触发一条构建警告。前端永远不会获得 `middleware.ts` 或 `proxy.ts` —— 静态导出跑不了它们。i18n 是纯客户端的 next-intl；OAuth 以同页重定向完成；客户端路由守卫只是 UX，安全由后端强制（见 `specs/frontend/routes.md`）。

## Alternatives considered

**Standalone Next.js 服务器模式。** 保住服务器特性，但每次部署加一个 Node 进程和第二个 origin 或一次代理跳；产品里没有页面需要服务端渲染。

**`middleware.ts`/边缘代理转发 API 调用。** Next 原生机制，但它需要一个静态导出不产出的服务器运行时 —— dev-only `rewrites` 在有服务器的地方（dev server）交付同样的人体工学，生产由 Caddy 覆盖。

**Vercel 式受管托管。** 把运行时问题挪给平台；与自托管单镜像目标不兼容，并为部署增加一个外部依赖。

## Consequences

可部署的前端是一个静态文件目录 —— 没有 Node 运行时、没有任何服务端的东西，整棵 `out/` 树都可在 CDN 缓存。一切服务器交互都是客户端直连后端（生产经 Caddy 同源，本地经 dev rewrites）。任何需要服务器的东西（middleware、请求时 i18n、API 路由）在构造上就出界 —— 这正是要点。
