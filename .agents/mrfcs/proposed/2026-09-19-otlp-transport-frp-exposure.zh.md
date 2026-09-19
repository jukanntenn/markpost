# MRFC: OTLP 遥测经 frp 链路的传输与暴露架构

[English](2026-09-19-otlp-transport-frp-exposure.md) | 中文

Status: proposed

## Problem

遥测的生产方与观测栈分居 NAT 两侧。markpost 跑在公网 VPS（vps1）上；观测栈跑在无公网 IP、无入站端口的家用/办公 NAS 上；唯一的回传路径是已部署的 frp 链路——frps 在另一台 VPS（vps2，3Mbps，前面是负责 TLS 终结的 caddy），frpc 在 NAS 上主动外连。由于生产方（vps1）与 frp 入口（vps2）是不同的机器，OTLP 摄取端点必然要经 vps2 的 caddy 对公网可达：不存在 localhost 捷径。因此传输安全、认证与带宽保护是本设计的一等需求，而非事后加固。

## Proposal

端到端链路：

```
markpost@vps1 ──HTTPS(OTLP+gzip+Bearer)──► caddy@vps2:443 (otlp vhost, IP-allowlisted)
                                              │ 127.0.0.1 (loopback only)
                                           frps@vps2 ──encrypted tunnel──► frpc@NAS
                                                                           │ docker network
                                                                   otelcol-contrib ─► stores
```

- **公网暴露面**（全部在 vps2）：caddy `:80`/`:443` + frps 控制端口 `:7000`（token 认证、`transport.tls.force = true`）。frps 代理端口绑定回环（`proxyBindAddr = "127.0.0.1"`），任何 frp 代理都不能被公网独立触达。NAS 零入站连接——frpc 只向外拨。
- **两个 caddy 站点**：`grafana.<域名>` 面向运营人员（Grafana 自有账号体系，关闭匿名访问）；`otlp.<域名>` 面向生产方，用 `remote_ip` 白名单持有生产方 IP（当前仅 vps1；未来每个业务系统加一行）。
- **Collector 认证**：otelcol-contrib 的 `bearertokenauth` 扩展守卫 OTLP receiver；每个业务系统持有独立静态 token，collector 按 token 身份强制打 `service.name` 标签，生产方无法伪造其他业务的遥测。认证从第一天就强制——端点公网可达，这就是真正的门，不是纵深冗余。
- **带宽保护**：SDK 侧 gzip 与批量；frp 代理级 `transport.bandwidthLimit` 限制 otlp 代理（初始 512KB/s）。markpost 单服务遥测压缩后每天几 MB，远低于 3Mbps 中继——但封顶保证遥测永远饿不死 Grafana 及 vps2 上的其他流量。
- **降级行为**：vps2 或隧道不可用时，OTel SDK 批量重试后丢弃。vps1 本地 `app-*.jsonl` 双写（由[观测栈 MRFC](./2026-09-19-otlp-observability-stack.zh.md) 承担）保留事件现场；失明的是可视化，不是服务。
- 三个触点的配置样稿（markpost 环境变量、caddy 站点、frpc 代理块）随实施栈落地。

## Alternatives considered

**同机回环绑定**（生产方 → `127.0.0.1:<frps 代理端口>`）：最廉价的暴露方式——摄取完全不触公网。败于真实拓扑：只有生产方与 frps 同机时才成立，而 markpost（vps1）不同机。保留为"未来与 frps 同机的生产方"的必备模式。

**frps 代理端口直接公网绑定（绕过 caddy）**：少一跳。败因：公网段明文、无 TLS 终结、无 IP 白名单层，还要多防一个公网端口——而 caddy 本就部署在 vps2 上、为一切终结 TLS。回环绑定加 caddy 的代价是一段反代配置，同时关掉三个缺口。

**WireGuard 站点互联替代发布 OTLP 端点**：最强的网络层隔离——完全没有公网应用端口。当前否决：它引入内核网络依赖和横跨三台机器（公网 VPS 与 NAS 属不同运维关切）的第二信任根，而 frps+caddy 已部署且在工作。当多个生产方让逐机隧道管理成为常态时再议。

**frp stcp（秘密隧道）visitor 模式**：无任何公网端口、凭 token 的私有暴露。败因：每个生产方都得跑 frpc visitor——对管理员笔记本合理，作为每个业务系统的接入要求是错的。

## Acceptance criteria

- 公网监听精确为：vps2 `:80`/`:443`（caddy）与 `:7000`（frps 控制，token + `transport.tls.force`）；无其他；frps 代理端口仅回环；NAS 零入站。
- `otlp` 站点拒绝非白名单 IP；collector 拒绝无认证 OTLP；token 身份映射到存储遥测的 `service.name`。
- 公网段无明文：浏览器→caddy 与 markpost→caddy 走 TLS；frps→frpc 走 `useEncryption`。
- otlp 代理限速生效；遥测满流时 Grafana 仍可用。
- 隧道中断优雅降级：SDK 重试后丢弃；事件现场保留在 vps1 本地 `app-*.jsonl`。

## Risks

- **Token 生命周期**：静态 bearer token 需要轮换规程（随实施写入运维 runbook）；单个 token 泄露的影响面是一个生产方的遥测流。
- **域名与 DNS**：两个站点需要 DNS 名字与证书（caddy ACME）；换域名会触及所有生产方的 endpoint 配置。
- **caddy 配置错误是新的单点暴露面**：设计按"失效即关闭"——白名单失守时 collector 认证依然生效——且验收条件对两层分别验证。
- **vps2 是共享单点**：它宕机只致可视化失明，不影响服务；该窗口内由本地崩溃通道日志覆盖事件取证。
