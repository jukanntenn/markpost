# MRFC: 回源落位宿主 443 端口，由宿主机 Caddy 共享网关终结 TLS

Status: implemented

[English](2026-08-23-origin-port-443-shared-gateway.md) | 中文

## Problem

SaaS 源站此前运行在宿主端口 2053，靠 Cloudflare Origin Rule 把 443 改写到 2053（[Cloudflare 部署指南 MRFC](2026-08-22-cloudflare-deployment-guide.zh.md) 记录的拓扑）。经真实 zone 的边缘缓存从未生效：所有响应 —— 包括带 `Cache-Control: public, max-age=31536000, immutable` 的默认缓存扩展名（woff2、CSS）—— 一律返回 `CF-Cache-Status: DYNAMIC`，站点容量被源站 3 Mbps 链路封顶，CDN 作为承重设计的前提落空（[缓存规格](../../../specs/backend/caching.zh.md)）。根因在 Cloudflare 文档快照中（`fundamentals/reference/network-ports.mdx`）：除 443 外的每个代理 HTTPS 端口都列在 "Ports supported by Cloudflare, but with caching disabled"（2052、2053、2082、2083、2086、2087、2095、2096、8880、8443），且经 `additional_cacheable_ports` 重新启用仅限 Enterprise（`cache/how-to/cache-rules/settings.mdx`）。在 SSL 模式为 Full (strict) 的前提下，443 是 Free 套餐下唯一能承载边缘缓存的回源端口。原端口决策只核对了规则可用性（"Origin Rules are available on every plan, Free included"），从未与缓存端口矩阵交叉验证；容量压测的 MISS → HIT 验证是直连源站做的，缺口直到 zone 实测才暴露。随之暴露的第二个约束：这台 VPS 不是单服务机器（第二个容器已占用 8443；还有一批服务 `*.bytehome.fun` 的子域代理），而 443 只能被一个进程绑定。

## Decision

**回源落位宿主端口 443，由宿主机 systemd Caddy 作为共享网关终结 TLS；markpost 以一个站点块接入。** 网关（生产机上已在运行，配置为 `import /etc/caddy/conf.d/*.caddy`）出示现有的 15 年 `*.markpost.cc` Origin CA 证书 —— playbook 把它复制到 `/etc/caddy/certs/markpost/`（root:caddy，0640）—— 并把 `markpost.cc` 反向代理到 `127.0.0.1:8080`。markpost 容器的 Caddy 彻底去掉 TLS：[`Caddyfile.production.j2`](../../../devops/ansible/templates/Caddyfile.production.j2) 在内部端口服务明文 HTTP，仅发布到回环（`127.0.0.1:8080:2053`），唯一可能的对端就是宿主网关。客户端 IP 接力机制不变：`CF-Connecting-IP` 经网关原样透传（非 hop-by-hop 头），容器 Caddy 仍从它改写 `X-Forwarded-For`；`trusted_proxies` 从 Cloudflare CIDR 列表收缩为 `private_ranges`（对端是网关经网桥 NAT 后的地址）。控制台删除 Origin Rule；宿主防火墙把"仅放行来自 Cloudflare 网段的 2053"换成"443"；`group_vars/production/vars.yml` 的 `cloudflare_cidrs` 仍是该列表在文档中的归宿（供防火墙参考），但不再喂给任何模板。后续服务各自丢一个 `conf.d` 站点块即可接入 —— 端口绑定是独占的，主机名不是；不需要边缘缓存的服务可以留在缓存禁用端口上（8443 上现有的服务正是如此）。

## Alternatives considered

**留在 2053，用规则让该端口可缓存。** 落选：`additional_cacheable_ports` 仅限 Enterprise，Free 套餐没有任何 Cache Rule 能越过端口闸门 —— Cache Rule active 且表达式匹配（`Eligible for cache`、`starts_with(http.request.uri.path, "/p-")`）时全站仍为 `DYNAMIC`，已实测确认。

**markpost 容器直接发布到宿主 443，不设网关。** 落选：一个端口只能绑定一个进程，这会冻结多服务计划（第二个服务已经存在），并让未来每个服务与 markpost 的 compose 生命周期耦合；宿主 Caddy 网关当时已在运行且闲置。

**Cloudflare Tunnel（完全不开入站端口）。** 暂时落选：每台机器要跑一个 cloudflared 守护进程，且规格刚稳定下来的防火墙 / trusted-proxies / CIDR 设计需要重写；本地文档快照没有覆盖经 Tunnel 的缓存行为，需先实测。若入站端口管理将来成为负担，值得重提。

## Consequences

边缘缓存从"完全不可达"变为可达；验收标准是 `verify-cf.sh` 的 MISS → HIT 序列加上 woff2 金丝雀（默认缓存、与规则无关）。切换需要一个短暂维护窗口且顺序固定 —— 先在控制台删除 Origin Rule，再跑 playbook（网关站点块先落位、容器后重建）；deploy.yml 末尾的边缘验证只有在两半都就位时才通过。容器内少了一跳 TLS（健康检查在所有环境都是明文 HTTP —— `tls_profile: http`，与 dev/staging 一致），宿主上多了一跳；Origin CA 证书有了两处归宿（`~/docker/markpost/certs/` 是 playbook 复制的源，`/etc/caddy/certs/markpost/` 是网关读取处）。针对生产源站的直连压测必须改为隧道到 `127.0.0.1:8080`，不再打 `:2053`；staging 保持 2053 形态，压测默认值不受影响。客户端 IP 接力的残余暴露面收窄：伪造 `CF-Connecting-IP` 现在要求要么是 Cloudflare（防火墙在 443 上强制），要么已在本机 —— 因为容器端口仅回环发布。[Cloudflare 部署指南 MRFC](2026-08-22-cloudflare-deployment-guide.zh.md) 的端口拓扑被本决策取代并交叉链接；其客户端 IP 分析仍然成立。
