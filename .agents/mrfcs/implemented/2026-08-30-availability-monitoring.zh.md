# MRFC: 经由 uptime-kuma 的可用性监控

Status: implemented

[English](2026-08-30-availability-monitoring.md) | 中文

## Problem

markpost 没有外部故障发现。Docker healthcheck 只影响本地重启；CDN 在源站宕机时继续供应缓存页面；单一 Cloudflare 视点无法区分边缘故障与源站故障；而只做存活的 `/api/v1/health` 在 Postgres 宕机时依然返回绿色 —— 写路径先在用户侧失败，运维者却看不到任何变红的信号。故障只能靠人工发现。

## Decision

可用性监控横跨三个视点分层，由自托管的 uptime-kuma 实例探测 production 与 staging：

- `GET /api/v1/ready` 是就绪端点：驱动层数据库往返，返回 `200 {"status":"ready"}` 或 `503 {"status":"unavailable"}`，与 `/health` 一样注册在所有限流器之外。存活 `/health` 保持不变，仍是 Docker healthcheck 轮询的端点 —— 数据库死亡应当把服务标记为未就绪，而不是连坐杀死容器。
- 监控项清单、通知渠道（飞书为主、SMTP 兜底）、告警策略（60 s 间隔、3 次重试、约 4 分钟触发、恢复通知开、重复提醒关、证书 / 域名到期 7/14/21 天）与分诊表都在 [`docs/monitoring.zh.md`](../../../docs/monitoring.zh.md)。
- 生产 VPS 运行 supervisor 程序 `markpost-heartbeat`：循环探测 `http://127.0.0.1:8080/api/v1/ready` 并把判定推送到 kuma 的 push 端点，让 kuma 看到 Cloudflare 之外的源站自身视角；推送静默则覆盖主机死亡。push URL 是 vault 密钥（`kuma_heartbeat_url`）；仅当该变量存在时，部署才安装此程序。

## Alternatives considered

- **纯黑盒监控，不改应用。** 落选：`/health` 只表存活 —— Postgres 宕机时所有外部探针全绿而写路径已经失败。就绪端点才是根修；没有它，监控栈根本看不见数据库。
- **经 socket.io API 脚本化配置 kuma**（如第三方 `uptime-kuma-api` Python 库）。落选：为一次性配置六个监控项引入新依赖并承担 2.x 兼容风险；runbook 直接承载精确字段值，kuma 侧变更本身很少。
- **心跳用 systemd timer。** 落选，改用 supervisor：生产主机已为多个服务运行 supervisor，每台主机一套监管系统足矣，且 supervisor 的 `autorestart` 无需额外 timer 语义即可恢复崩溃的循环。
- **直连数据库与容器监控**（kuma 的 `postgres`/`docker` 类型）。落选：生产 Postgres 只经 Unix socket 卷可达，Docker 也不对外暴露，外部视点均够不着；就绪探针在不扩大攻击面的前提下覆盖了数据库健康。
- **覆盖发版的维护窗口。** 落选：约 4 分钟的重试阈值已吸收发版时的容器替换，而维护窗口可能把恰逢其时的真故障静默。

## Consequences

换来：源站、数据库、边缘路径与整机故障各自在约 4 分钟内浮出，分诊表凭红色监控项的组合即可区分。代价：VPS 上多一个 supervisor 程序、一个泄漏后必须轮换的 vault 密钥（持有者可伪造 up 心跳），以及 kuma 本身成为关键依赖 —— 它宕机意味着静默，而非误报。验证：就绪契约由 `ready_test.go` 覆盖（两种判定）；首个带 vault 变量的生产部署完成后，`sudo supervisorctl status markpost-heartbeat` 必须显示 RUNNING 且 kuma 收到心跳。kuma 侧配置（监控项、渠道、push URL 入库）是由 runbook 设定顺序承接的运维后续。
