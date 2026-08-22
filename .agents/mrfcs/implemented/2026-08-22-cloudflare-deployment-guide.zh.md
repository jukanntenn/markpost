# MRFC: Cloudflare 部署指南与规格对齐

Status: implemented

[English](2026-08-22-cloudflare-deployment-guide.md) | 中文

## Problem

SaaS 生产实例（`markpost.cc`，Cloudflare 之后的 VPS 源站）需要一份运维者可以端到端执行的操作指南 —— 控制台步骤、宿主机准备、部署、验证。[`docs/deployment.zh.md`](../../../docs/deployment.zh.md) 已长成一份覆盖完整多环境生命周期的参考（验收、dev、staging、production、Ansible 内部、运维任务、镜像标签语义），作为该 runbook 太长。更糟的是，持有 Cloudflare 设计事实的规格已经偏离 deploy 目录实际交付的内容：[`specs/backend/cloudflare.zh.md`](../../../specs/backend/cloudflare.zh.md) 把 `:7157` 的 Caddy 与 `443:7157` 映射描述为未实现的目标，把 Caddyfile 模板称为已跟踪的后续工作，并声称 `cloudflare_cidrs` 仍是占位符 —— 而 [`Caddyfile.production.j2`](../../../devops/ansible/templates/Caddyfile.production.j2) 交付的是 `:2053`，源站经一条把 443 改写为 2053 的 Origin Rule 到达，真实 CIDR 已位于 `group_vars/production/vars.yml`。按规格写 runbook 会与仓库矛盾；按仓库写会与规格矛盾。最深的漂移是客户端 IP 机制：规格描述一条在 TCP 层设防的纯 XFF 追加链，但模板的 `header_up X-Forwarded-For {http.request.header.CF-Connecting-IP}` 改写取代了那条链 —— 而且规格自述的设计按原文无法工作：gin（`SetTrustedProxies` 仅回环）从右到左遍历 XFF，会把 Cloudflare 边缘跳当作每个访客的客户端 IP 返回，以 IP 为键的限流器将坍缩。

## Decision

**一份聚焦生产的 runbook，旧指南原地冻结。** [`docs/deployment-cloudflare.zh.md`](../../../docs/deployment-cloudflare.zh.md)（与其英文镜像）只覆盖 Cloudflare 生命周期 —— 一次性控制台配置（已代理 A 记录、Full strict、Origin CA 证书、Origin Rule 443 → 2053）、一次性 VPS 准备（证书放置、宿主防火墙锁定到 Cloudflare CIDR 的 2053）、含获取步骤（GitHub OAuth 应用、Cloudflare API token）的必填 vault 密钥、固定版本的部署、经边缘的验证、CIDR 同步 —— 只陈述操作，不含设计理由，理由链接到规格。`docs/deployment.zh.md` 留在原路径，加冻结横幅把生产工作重定向到新指南（`PRINCIPLES.md` 的冻结先例），dev/staging/验收保留为参考快照；其三处入站链接（两种语言）保持有效。不附加词数预算：`scripts/doc_budgets.manifest.json` 只约束代理指令文件，"只写操作"本身就是编辑上限。

**规格向已部署现实对齐，经源码核实。** `specs/backend/cloudflare.zh.md` 与 `rate-limiting.zh.md`（两种语言）现在描述模板交付的内容：`:2053` 拓扑加 Origin Rule、已部署的 `Caddyfile.production.j2` 形状（`auto_https off`、Origin CA 证书路径、每块 `trusted_proxies` + `header_up`）、真实的 `cloudflare_cidrs` 归宿与其防火墙镜像，以及作为单值接力的客户端 IP 机制 —— Caddy 的用户头操作在其默认转发头处理之后执行（caddy `modules/caddyhttp/reverseproxy/reverseproxy.go`，头操作在代理循环中应用），因此到达 Go 的 XFF 恒为单个由 Cloudflare 断言的 `CF-Connecting-IP`；这一收敛是正确性所必需，因为 gin 的 `validateHeader`（gin `gin.go`）从右返回首个不受信 IP，仅有回环信任时将得出边缘跳。信任锚是 Cloudflare 边缘对 `CF-Connecting-IP` 的覆写加宿主防火墙的 CIDR 白名单；残余暴露 —— 防火墙被绕过者可伪造该头且改写会传播 —— 被如实记录，取代先前不成立的纵深防御主张。对 `gin.PlatformCloudflare` 的拒绝维持原状：它在应用层不做 CIDR 检查。

## Alternatives considered

**把旧指南改名为归档，把 `deployment.md` 名字让给新指南。** 它输了：两种语言共三处入站链接会为命名语义而无谓改动，原地冻结与 `PRINCIPLES.md` 先例一致，快照离其读者只有一跳。

**继续扩展单一的多环境指南。** 它输了：长度正是本决策的触发点；Cloudflare 路径尤其需要控制台级精确步骤，运维者无需涉读 dev/staging 内部即可执行。

**修模板以匹配规格** —— 去掉 `header_up`，恢复规格描述的纯 XFF 追加链。经源码核实后它输了：gin 仅信任回环时，追加链 `<real-client>, <cloudflare-hop>` 使 `ClientIP()` 对每个请求都返回 Cloudflare 边缘跳；已部署的改写正是让按 IP 作键的限流器保持可用的机制。

**采用 `gin.PlatformCloudflare`。** 如前所述它输了：应用层无 CIDR 检查，任何能直连端口者都可伪造该头；已部署设计把检查留在包层，防火墙本就在那里执行。

## Consequences

新文件对、横幅与重写的规格段落全部通过七个文档闸门；客户端 IP 主张的格式可引源（`~/Workspace/contexts/` 下的 caddy/gin 仓库），取代了描述一个无法工作的机制的散文。双指南状态带有重复成本 —— 冻结快照与活跃 runbook 在生产事实上重叠 —— 这被接受，因为横幅做了重定向且快照明确不再维护；当 dev/staging 流程下次变化时，其指南必须解冻或拆分，而不是原地编辑。规格重写还揭示并记录了旧散文掩盖的一个诚实残余风险：头的真实性落在防火墙上，因此 CIDR 同步职责（防火墙 + `cloudflare_cidrs`）是承重的，runbook 把它列为例行工作。决策时生产源站尚未服务流量（边缘应答、源站 502）—— 这份 runbook 就是通往绿色 `https://markpost.cc/api/v1/health` 的路径；冻结指南的散文仍携带源站 IP，这是一个公开性考量，runbook 用占位符规避了它。
