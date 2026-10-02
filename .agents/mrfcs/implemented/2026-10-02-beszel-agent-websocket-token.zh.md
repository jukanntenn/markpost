# MRFC: Beszel agent authenticates to the hub with a vaulted WebSocket token

Status: implemented

[English](2026-10-02-beszel-agent-websocket-token.md) | 中文

## Problem

hub 现已自托管在可观测服务器（192.168.5.57，边缘为 `beszel.bytehome.fun`），不再是对外运维的机器，并采用纯 WebSocket 拓扑（`DISABLE_SSH=true`）：agent 主动外拨 hub，生产防火墙保持关闭。而仓库的 agent 模板只渲染 `KEY` + `HUB_URL` —— 这是 beszel 0.18 时代的半成品状态，其 SSH 时代的假设早于基于令牌的 WebSocket 认证：缺少 `TOKEN` 时 agent 无法向 hub 认证，任何一次渲染该模板的部署都会静默折断主机指标线。agent 镜像 pin 也已过时（0.18.8 对 0.20.0），且 0.19 起 agent 会校验 hub 的 TLS 证书（与真实证书的边缘兼容）。另一独立事实：hub 内嵌 PocketBase，不支持外部数据库——"用 PostgreSQL/MySQL/MariaDB"的预期在上游无法成立（`go.mod` 无驱动、无 env），故 hub 以 SQLite 跑在 bind mount 上，用 PocketBase 备份作为持久性方案。

## Decision

**agent 以 vault 的每系统令牌（`beszel_agent_token`）认证。** `beszel-agent-compose.yml.j2` 在 `HUB_URL`/`KEY` 之外渲染 `TOKEN: "{{ beszel_agent_token }}"`；deploy.yml 的四个 agent 任务同时要求 `beszel_hub_url` 与 `beszel_agent_token`，agent 永远不会在凭据对残缺时被渲染（setup-order 契约相应点名两者）。`group_vars/production/vars.yml` 设置 `beszel_hub_url: https://beszel.bytehome.fun`、hub 公钥（`beszel_agent_key`，非密钥），并把 pin 升至 `0.20.0`。hub 本身留在仓库之外（可观测服务器上的独立 compose）：SQLite 于 bind mount 的 `beszel_data`，飞书告警走 shoutrrr `lark://`，阈值 CPU 95 / 内存 92 / 磁盘 90（2 分钟）与 agent 离线 1 分钟——每指标仅一阈值，取 critical 档。

## Alternatives considered

**维持 SSH 模式（hub 反连 agent）。** 落选：hub 在内网，够不到 VPS；对 45876 开入站违背拓扑 MRFC 确立的"防火墙零放行"，且 hub 的 `DISABLE_SSH` 已将其排除。

**给 hub 接外部数据库。** 落选：上游不存在该能力（内嵌 PocketBase，仅 SQLite）——持久性即 bind mount 加 hub 侧备份；要改只能换产品，不是改配置。

**只填 group_vars、不动模板。** 落选：渲染 `HUB_URL` 而无 `TOKEN` 会得到一个外拨失败认证、回落到死 SSH 监听的 agent——正是本变更要消灭的静默断线形态；联合守卫使两个变量原子生效。

## Consequences

下次生产部署将（重新）接管 ttyo 上的 agent compose——此前的手写 compose 视作被取代，因其 env 集合与模板完全一致故为安全替换。hub URL、公钥与 pin 为明文变量；本次新增的唯一密钥是令牌。beszel 0.20 的 agent 会校验 hub 证书，边缘必须继续为 `beszel.bytehome.fun` 提供真实证书。令牌泄漏时的轮换在 hub 侧完成（Add-System 对话框重新生成）加一次 vault 修改。
