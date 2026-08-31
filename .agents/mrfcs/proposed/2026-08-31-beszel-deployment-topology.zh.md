# MRFC: Beszel deployment topology on ttyo

Status: proposed

[English](2026-08-31-beszel-deployment-topology.md) | 中文

## Problem

Beszel 的 hub 需要运维入口，agent 需要传输通道。ttyo 入站默认拒绝（仅 SSH 与来自 Cloudflare 网段的 443），身前是 `import /etc/caddy/conf.d/*.caddy` 的宿主 Caddy 共享网关，持有 `markpost.cc, *.markpost.cc` 的 Origin CA 泛域名证书，且部署 playbook 已在模板化网关站点块。hub 放哪、怎么到达、密钥怎么流转——必须在不新开入站端口、不削弱现有姿态的前提下决定。

## Proposal

[host-metrics MRFC](./2026-08-31-host-metrics-monitoring-beszel.zh.md) 选定了组件；本层钉死它的拓扑。

- hub 与 agent 同驻 ttyo，跑在独立的 compose project 里、与应用的 compose 分离——监控部署与应用部署互不牵连。hub 只绑回环；部署 playbook 从模板渲染 compose 文件，版本由 `group_vars` 钉住。
- 运维入口复用共享网关机制（[origin-port-443 MRFC](../implemented/2026-08-23-origin-port-443-shared-gateway.zh.md)）：一条 proxied 的 `beszel.markpost.cc` DNS 记录 + 一个网关站点块反代到 hub 的回环端口——现有 Origin CA 泛域名证书直接覆盖、防火墙零变更、Cloudflare 在前。hub 自身的登录是应用层闸门；Cloudflare Access 是记录在案的升级路径。
- agent 用 host 网络加只读 `docker.sock` 挂载。其 `KEY` 是 hub 的公钥——不是密钥——直接进模板；hub 管理凭据与通知 token 是 `group_vars/production/vault.yml` 里的逐变量 vault 条目（avpm），与 `kuma_heartbeat_url` 同款。
- 移除像心跳的移除行一样写进运维手册：部署模板永不卸载。

## Alternatives considered

- **hub 放家里（oect/fn），ttyo 只跑 agent。** 能挺过 ttyo 死亡、保住历史，但把生产可观测性耦到家内机器与其隧道 上。主机死亡本就是 kuma 心跳的信号，边际收益买来的是新依赖。若跨宕机的历史保留开始要紧再重议。
- **仅 SSH 隧道访问、不开公共域名。** 面最小，但单人运维日常摩擦大；网关 + Cloudflare 路径复用既有机制、零新端口、hub 仍在登录之后。
- **在 `docker.sock` 前加 socket-proxy 容器。** 现阶段多一个活动部件；origin 门禁之后的只读 socket 是相称的。风险变陈年问题时再升级。
- **把 Beszel 并进 markpost 的 compose 文件。** 落选：应用 compose 保持应用形态（app + db + migrate）；监控必须可独立部署、独立移除。
- **用 overlay 网络（Tailscale/WireGuard）做访问。** 为一个面板引入整套新网络子系统；对单人运维，Cloudflare 前置 + 登录已足够。

## Acceptance criteria

- `ansible-playbook devops/ansible/deploy.yml -e target=production` 幂等地渲染监控 compose 与网关站点块；站点块变更时重载网关。
- `https://beszel.markpost.cc` 经 Cloudflare 到达 hub 登录页；前后 `ufw status` 完全一致。
- 模板与 git 中无密钥值：hub 凭据与通知 token 均为 vault 变量；模板中的 agent `KEY` 是公钥。
- `docs/monitoring.md` 及中文对承载本层的安装顺序与移除行。

## Risks

- 公开的 hub URL 会招来扫描流量；Cloudflare 加 hub 登录足以吸收，Cloudflare Access 是点名过的升级路径。
- 没有自动更新器时 Beszel 镜像钉版会悄然过时；运维手册的升级行负责。
- 共享网关约定一旦变化，此站点块须随之迁移——与心跳程序已接受的耦合相同。
