# MRFC: OTLP 遥测经 frp 链路的传输与暴露架构

[English](2026-09-19-otlp-transport-frp-exposure.md) | 中文

Status: implemented

## Problem

遥测的生产方与观测栈分居 NAT 两侧。markpost 跑在公网 VPS（vps1）上；观测栈跑在无公网 IP、无入站端口的家用/办公 NAS 上；唯一的回传路径是 frp 链路——frps 在另一台 VPS（vps2，3Mbps，前面是负责 TLS 终结的 caddy），frpc 在 NAS 上主动外连。由于生产方（vps1）与 frp 入口（vps2）是不同的机器，OTLP 摄取端点必然要经 vps2 的 caddy 对公网可达：不存在 localhost 捷径。因此传输安全、认证与带宽保护是本设计的一等需求，而非事后加固。

## Decision

整条外部链路由外部运维方按交付材料运维；本仓库拥有且仅拥有一个触点：markpost 的 OTLP 环境变量（staging 与生产 `OTEL_EXPORTER_OTLP_ENDPOINT=https://otlp.bytehome.fun`，`markpost-dev` 内网直连，各环境 bearer token 以 `otel_otlp_token` 入库 vault）。

```
markpost@vps1 ──HTTPS(OTLP+gzip+Bearer)──► caddy@vps2:443 (otlp vhost, IP-allowlisted)
                                              │ 127.0.0.1 (loopback only)
                                           frps@vps2 ──encrypted tunnel──► frpc@NAS
                                                                           │ docker network
                                                                   otelcol-contrib ─► stores
```

- **公网暴露面**（全部在 vps2，全部由外部配置）：caddy `:80`/`:443` 加 frps 控制端口。frps 代理端口保持公网绑定（其他租户需要直连，如 SMTP），提案期的"仅回环"设想因此让位于补偿控制：每条路径上的 collector token 认证（失效即关闭），以及 otlp 站点上持有生产服务器 IP 的 `remote_ip` 白名单。
- **两个 caddy 站点**（规格已交付）：`grafana.bytehome.fun` 面向运营人员（Grafana 自有账号、关闭匿名），`otlp.bytehome.fun` 面向生产方，均反代 vps2 回环上的 frps 代理端口。
- **Collector 认证**：`bearertokenauth` 以每环境一个 token（`markpost-dev` / `markpost-staging` / `markpost`）守卫 OTLP receiver；collector 对每条入口都要求有效 token（无 token → 401、错 token → 401、有效 token → 接受——rollout 期间经内网与公网两条路径分别验证）。
- **带宽保护**：SDK 侧 gzip 与批量由仓库负责；frp 代理级 `transport.bandwidthLimit` 限制 otlp 代理向外部运维方提请，遥测永远饿不死 3Mbps 中继。
- **降级**：vps2 或隧道不可用时，OTel SDK 重试后丢弃；生产机本地双写的 `app-*.jsonl` 保留事件现场。

## Alternatives considered

**同机回环绑定**（生产方 → `127.0.0.1:<frps 代理端口>`）：最廉价的暴露方式——摄取完全不触公网。败于真实拓扑：只有生产方与 frps 同机时才成立，而 markpost（vps1）不同机。保留为"未来与 frps 同机的生产方"的必备模式。

**frps 代理端口直接公网绑定（绕过 caddy）**：少一跳。败因：公网段明文、无 TLS 终结、无 IP 白名单层，还要多防一个公网端口——而 caddy 本就部署在 vps2 上、为一切终结 TLS。回环绑定加 caddy 的代价是一段反代配置，同时关掉三个缺口。

**WireGuard 站点互联替代发布 OTLP 端点**：最强的网络层隔离——完全没有公网应用端口。当前否决：它引入内核网络依赖和横跨三台机器（公网 VPS 与 NAS 属不同运维关切）的第二信任根，而 frps+caddy 已部署且在工作。当多个生产方让逐机隧道管理成为常态时再议。

**frp stcp（秘密隧道）visitor 模式**：无任何公网端口、凭 token 的私有暴露。败因：每个生产方都得跑 frpc visitor——对管理员笔记本合理，作为每个业务系统的接入要求是错的。

## Consequences

公网链路已端到端验证：`https://grafana.bytehome.fun` 经完整链路服务 Grafana；otlp 站点对白名单外答 403、无 token 或错 token 答 401、接受来自生产服务器的认证推送。staging 刻意走同一公网入口——先于生产验证生产路径。

rollout 期间浮出的运维事实，现为义务：

- **`docker compose restart` 不重读 `.env`**——改 env 必须 `up -d`（重建）。一次 env 失配让 collector 静默拒绝一切推送数日，正因 SDK 侧失败无声；摄取断流告警与[观测栈 MRFC](./2026-09-19-otlp-observability-stack.zh.md)的后续工作一并跟踪。
- **接口漂移是外部运维链路的长期风险**：站点、代理端口、限速与网络接入都经外部运维方。交付文档写明必需属性，黑盒检查（403/401/接受）验证外部可观测项——无论配置出自谁手。
- **token 轮换是两侧动作**：collector 的 token 列表与各生产方的环境必须同步移动，否则摄取静默中断；轮换顺序为先 collector（追加新 token）、后生产方。
