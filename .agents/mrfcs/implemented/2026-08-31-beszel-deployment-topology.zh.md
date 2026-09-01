# MRFC: Beszel deployment topology on ttyo

Status: implemented

[English](2026-08-31-beszel-deployment-topology.md) | 中文

## Problem

Beszel 的 agent 需要传输通道，仓库需要一条自动化边界。ttyo 入站默认拒绝（仅 SSH 与来自 Cloudflare 网段的 443），agent 的连接必须保持出站。hub 放哪、agent 怎么到达它、仓库自动化到哪一步——必须在不新开入站端口、不削弱现有姿态的前提下决定。

## Decision

[host-metrics MRFC](./2026-08-31-host-metrics-monitoring-beszel.zh.md) 选定了组件；本记录钉死它的拓扑与自动化边界。

- hub 置于独立的、运维管理的服务器上——绝不同驻被监控主机。它的部署、暴露、凭据、通知渠道与升级都是本仓库之外的运维手工事务；[`docs/monitoring.zh.md`](../../../docs/monitoring.zh.md) 承载的是运维清单，不是自动化。
- 仓库只自动化 ttyo 上的 agent：独立 compose 项目位于 `~/docker/beszel-agent`（`group_vars/all.yml` 的 `beszel_agent_path`），与应用的 compose 分离。[`deploy.yml`](../../../devops/ansible/deploy.yml) 从 [`beszel-agent-compose.yml.j2`](../../../devops/ansible/templates/beszel-agent-compose.yml.j2) 渲染：镜像钉版、host 网络、只读 `docker.sock` 挂载，并把 agent 指向运维提供的 `HUB_URL`——出站 WebSocket，防火墙零新增放行。任务仅在 `beszel_hub_url` 已定义时运行。
- agent 的 `KEY` 是 hub 的公钥——不是密钥——直接进模板。仓库不承载任何 hub 密钥：hub 管理凭据与通知 token 归运维管理的 hub 所有，在仓库 vault 之外。
- 自动化侧的移除像心跳的移除行一样写在运维手册里：部署模板永不卸载。hub 的移除是 hub 主机上普通的运维事务。

## Alternatives considered

- **hub 同驻 ttyo、藏在共享网关之后**（`beszel.markpost.cc` + 泛域名 Origin CA + Cloudflare，hub 只绑回环）。评审中落选：hub 与主机同死，资源层随它看护的机器一起熄火；而且还把仓库自动化绕在一个属于运维侧的面板上。
- **仓库自动化的 hub 部署**（ansible 模板化 hub compose、hub 凭据与通知 token 进仓库 vault）。评审中落选：hub 是运维基础设施——它的主机与生命周期归运维所有，仓库模板化会把运维主机未必共享的拓扑选择焊死。
- **在 agent 的 `docker.sock` 前加 socket-proxy 容器。** 多一个活动部件；直接只读 socket 是相称的。风险变陈年问题时再升级。
- **把 agent 并进 markpost 的 compose 文件。** 落选：应用 compose 保持应用形态（app + db + migrate）；监控须可独立部署、独立移除。
- **agent→hub 路径走 overlay 网络（Tailscale/WireGuard）。** 为一条出站 WebSocket 引入整套新网络子系统；`HUB_URL` 走普通 HTTPS 加密钥对认证已足够，overlay 留作运维侧的可选项。

## Consequences

买到：一条清晰的自动化边界——仓库只拥有并测试 agent 这一半，hub 的主机、暴露与密钥留在运维侧；模板里不漏 hub 拓扑，ttyo 的防火墙姿态分毫未动。代价：hub 主机与 ttyo→hub 路径进入依赖链（记录于选型层）；若路径走运维网络的隧道，隧道维护会静默暂停指标流——agent 离线告警是兜底；没有自动更新器时 Beszel 镜像钉版会悄然漂移，运维手册的升级行负责；激活等运维设好 `beszel_hub_url` 与 `beszel_agent_key`。验证：prek 门禁对每次自动化改动跑 `ansible-playbook --syntax-check` 与模板渲染解析；部署时检查由运维手册的配置顺序与前后 `ufw status` 覆盖。
