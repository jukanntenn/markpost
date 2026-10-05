# 可用性监控

[English](monitoring.md) | 中文

markpost 的可用性由自托管的 [uptime-kuma](https://github.com/louislam/uptime-kuma) 实例从外部探测 production 与 staging，并由各应用主机反向推送心跳；源站上的 Beszel agent 另将主机与容器指标上报给异地 hub；观测服务器上的 Grafana 承载应用、业务与遥测管线告警。告警发往飞书（主）与经 mailrise 网关的邮件（兜底）。本 runbook 承载监控项清单、命名标准、通知渠道配置、告警归属边界与心跳的部署 / 卸载流程。

<a id="probe-model"></a>

## 探测模型

单一 URL 盯不住 CDN 前置的源站：edge 缓存的页面在源站宕机时依然保持绿色，而只盯源站的探针又看不到用户实际经过的边缘路径。监控项因此覆盖三个视点：

| 视点                          | 看到什么                                                                              | 承载者                                   |
| ----------------------------- | ------------------------------------------------------------------------------------- | ---------------------------------------- |
| 边缘路径（kuma → 公网 URL）   | 完整用户路径：DNS、Cloudflare、网关、容器、静态导出                                   | 首页监控项                               |
| 源站探针（kuma → 未缓存端点） | Go 进程，并经 `/api/v1/ready` 触达数据库 —— `/api/v1/*` 携带 `no-store`，请求必达源站 | 就绪监控项                               |
| 反向心跳（应用主机 → kuma）   | 主机自身的本地判定，完全绕过 Cloudflare；静默即主机死亡                               | 每环境一组 Push 监控项 + supervisor 循环 |

端点语义（`/health` 存活 vs `/ready` 就绪）见 [`api-schema.zh.md`](../specs/backend/api-schema.zh.md)；两者均豁免限流（[`rate-limiting.zh.md`](../specs/backend/rate-limiting.zh.md)）。支撑分层的 CDN 行为规范在 [`caching.zh.md`](../specs/backend/caching.zh.md) 与 [`cloudflare.zh.md`](../specs/backend/cloudflare.zh.md)。

<a id="monitors"></a>

## 监控项

**命名标准**（MRFC 2026-10-04-observability-wiring-overhaul）：监控项挂在两个 kuma 分组监控之下——`markpost · production` 与 `markpost · staging`——两环境形态完全一致，晋升前先在 staging 彩排。叶子监控命名遵循 `markpost · <env> · <对象> (<layer>)`，并打三个标签：`env:production` / `env:staging`、`service:markpost`、`layer:edge|origin|push`。侧栏的 Tags 过滤与分组布局都依赖这套约定。

所有监控项的公共设置：Heartbeat Interval `60`、Retries `3`、Retry Interval `60`，挂接两个通知渠道，不配维护窗口（重试阈值已吸收发版重启；维护窗口反而会把真故障静默）。Push 监控显式设置 retries：心跳 `2`、每日备份检查 `1`（单次漏推不该立刻告警）。

| 分组                  | 监控项                                  | 类型                 | 目标 / 关键字段                                                                           |
| --------------------- | --------------------------------------- | -------------------- | ----------------------------------------------------------------------------------------- |
| markpost · production | markpost · prod · homepage (edge)       | HTTP(s)              | URL `https://markpost.cc/`，接受状态 200；开启 `markpost.cc` 的证书与域名到期通知         |
| markpost · production | markpost · prod · origin readiness      | HTTP(s) - Json Query | URL `https://markpost.cc/api/v1/ready`；Json Query `status`、运算符 `==`、期望值 `ready`  |
| markpost · production | markpost · prod · host heartbeat (push) | Push                 | Interval `120`、Retries `2`；push URL 是 vault 里的秘密（见[心跳](#heartbeat)）           |
| markpost · production | markpost · prod · db backup (push)      | Push                 | Interval `86400`、Retries `1`；由每日备份检查喂入（见 [docs/backup.zh.md](backup.zh.md)） |
| markpost · staging    | markpost · stg · homepage (edge)        | HTTP(s)              | URL `https://markpost.bytehome.fun/`，接受状态 200                                        |
| markpost · staging    | markpost · stg · origin readiness       | HTTP(s) - Json Query | URL `http://192.168.5.50:8089/api/v1/ready`；Json Query `status` == `ready`               |
| markpost · staging    | markpost · stg · host heartbeat (push)  | Push                 | Interval `120`、Retries `2`——与生产对称                                                   |
| markpost · staging    | markpost · stg · db backup (push)       | Push                 | Interval `86400`、Retries `1`                                                             |

**Push URL 契约**：vault 中的 push URL 是**裸**端点、不带查询后缀，且按各生产者的可达性选择主机形态：生产（vps1 上）走公网路由（`https://uptime-kuma.bytehome.fun/api/push/<token>`），staging 主机走内网（`http://192.168.5.50:3001/api/push/<token>`）——它到 vps2 的出站会挂起（家宽 IP 在那边不被接受），公网形态会静默超时。脚本侧契约处处一致：`heartbeat.py` 以 `?status=…` 追加，`pgbackrest-check.py` 同样以 `?status=…` 追加。若带上第二段 `?status=…`（kuma 界面复制出来的形态）会产生重复参数，kuma 把 status 读成"非 up"——每次真实推送都会把监控项打成 down。

证书与域名到期通知按 kuma 全局阈值触发（默认剩余 7/14/21 天）。stg · origin readiness 直探内网地址，因此入口故障与实例故障可区分。

<a id="notification-channels"></a>

## 通知渠道

| 渠道         | kuma 配置                                                                                                                                         |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| 飞书（主）   | Notification Type `Feishu`；Webhook URL = 飞书群机器人 webhook                                                                                    |
| 邮件（兜底） | Notification Type `Email (SMTP)`；主机 `mailrise.bytehome.fun`、端口 `465`、TLS 开、用户按 vault/mailrise 配置、收件人 = mailrise 的 apprise 地址 |

两个渠道都建好后，设为默认通知（Settings → Notifications → apply as default），让每个监控项都走双通道。mailrise 网关把邮件转换成 apprise 通知——收件人地址编码了投递目标；调整 mailrise 映射后要回来核对。

<a id="alert-ownership"></a>

## 告警归属

一个故障只有一个 owner——三个系统不得重复叫:

| 层                 | 归属    | 信号                                       |
| ------------------ | ------- | ------------------------------------------ |
| 边缘 / 源站可用性  | kuma    | 首页、源站就绪、心跳、证书/域名到期        |
| 主机资源           | beszel  | 磁盘、内存、CPU、agent 掉线                |
| 应用 / 业务 / 管线 | Grafana | RED 信号、投递积压、错误日志速率、遥测断流 |

Grafana-managed 告警（接触点、通知策略、规则）在观测服务器上以代码形式 provision（`~/docker/grafana/provisioning/alerting/`），按 `alertname` + `deployment.environment.name` 路由，覆盖 staging 与 production。规则清单（阈值为起点，跑两周基线后再调）:

| 规则                           | 条件                                   | For    | 级别                |
| ------------------------------ | -------------------------------------- | ------ | ------------------- |
| HTTP 5xx 比例超 5% / 20%       | 5xx 占请求速率比例（5m）               | 5m/2m  | warning/critical    |
| HTTP p95 延迟超 1s             | `histogram_quantile(0.95, …)`          | 10m    | warning             |
| 请求速率塌陷                   | 10m 速率 < 1 小时前基线的 20%          | 10m    | critical            |
| 投递积压超 100 / 15m 增长超 50 | `markpost.delivery.pending`            | 10m/0m | warning（领先指标） |
| 投递 / CDN 清除失败            | 失败计数器速率 > 0                     | 15m    | warning             |
| 遥测断流（按环境）             | runtime gauge 10m 无数据——管线自身挂了 | 0m     | critical            |

ERROR 日志速率规则当前为 **paused**:日志数据源输出整数帧,Grafana 13.2.1 的告警表达式拒绝;待插件输出数值帧后启用(Runtime 看板 meantime 持续展示 ERROR 速率)。

<a id="alert-policy"></a>

## 告警策略（kuma）

- 监控项只在连续失败约 4 分钟后告警（间隔 60 s × 重试 3）；边缘抖动与发版时的容器切换保持静默。
- 恢复通知开启:down→up 转换一定通知。
- 不做重复提醒（单人运维服务;恢复通知闭环）。
- 证书与域名到期在剩余 7/14/21 天通知。

<a id="heartbeat"></a>

## 心跳（staging + production）

每台应用主机上，supervisor 程序 `markpost-heartbeat` 运行安装于 `<app_path>/heartbeat.py` 的静态脚本 [`heartbeat.py`](../devops/ansible/files/heartbeat.py)：每 60 s 探测 `http://127.0.0.1:<host_port>/api/v1/ready` 并把判定推送到 kuma 的 push 端点。探测 URL 与间隔走命令行；秘密 push URL 经 supervisor 程序的 `environment=` 注入（来自 vault 变量——因此 conf 为 0600）。当 `down` 判定到达（应用级故障，含数据库故障）或推送停止（主机死亡）时，kuma 把监控项打成 down。日志在 `<app_path>/data/heartbeat.log`。接线按设计对称——staging 彩排的正是生产运行的形态（MRFC 2026-10-04-observability-wiring-overhaul）。

[`deploy.yml`](../devops/ansible/deploy.yml) 中的部署任务只在对应环境的 vault 变量 `kuma_heartbeat_url` 已定义时安装脚本与程序，setup 顺序为:

1. 在 kuma 中把该环境的心跳监控项（Push、interval 120、retries 2）建在对应分组下，复制其**裸** push URL——持有者可伪造心跳，务必当秘密对待。
2. 入库:`python3 scripts/vault.py set <env> kuma_heartbeat_url`（在隐藏提示处粘贴 push URL）
3. 部署:`ansible-playbook devops/ansible/deploy.yml -e target=<env> --ask-become-pass`——handler 执行 `supervisorctl reread && update` 并启动程序。
4. 验证:`sudo supervisorctl status markpost-heartbeat` 显示 RUNNING，且 kuma 收到心跳。

卸载是手动的（部署从不卸载）：删除 `/etc/supervisor/conf.d/markpost-heartbeat.conf`，然后 `sudo supervisorctl reread && sudo supervisorctl update`，并删除 vault 变量。

<a id="host-metrics"></a>

## 主机指标（Beszel）

上面的监控项回答"是否活着"，Beszel agent 回答"为什么"——主机与逐容器资源历史及阈值告警，上报给观测服务器上的自托管 hub。设计记录:[主机指标 MRFC](../.agents/rfcs/implemented/2026-08-31-host-metrics-monitoring-beszel.zh.md)、[拓扑 MRFC](../.agents/rfcs/implemented/2026-08-31-beszel-deployment-topology.zh.md)、WebSocket 接线见[agent-token MRFC](../.agents/rfcs/implemented/2026-10-02-beszel-agent-websocket-token.zh.md)。

**Hub（独立生命周期，观测服务器上）。** 部署于 192.168.5.57 的 `~/docker/beszel`（边缘走 `beszel.bytehome.fun`）：固定 `henrygd/beszel`、bind-mount `beszel_data`（上游内嵌 PocketBase、不支持外接数据库）、`DISABLE_SSH=true`——纯 WebSocket 拓扑,agent 外连 hub,hub 从不回连。整个 `beszel_data` 目录一起备份（内含 agent `KEY` 验证用的 hub 密钥对）。

**Agent（仓库自动化，仅生产）。** 单服务 compose 项目 `~/docker/beszel-agent`,由 [`beszel-agent-compose.yml.j2`](../devops/ansible/templates/beszel-agent-compose.yml.j2) 渲染：固定 `henrygd/beszel-agent`、host 网络、只读 `docker.sock`。采集主机 CPU/内存/磁盘/负载/网络及 `markpost`、`markpost-postgres` 的容器统计，经出站 WebSocket 连接 hub（`HUB_URL` + vault 的 `TOKEN`;`KEY` 验证 hub）——防火墙不开任何入站。

**告警。** 阈值配在 hub 上；起步值如下，跑一周曲线后再调（beszel 每个指标一个阈值，只用 critical 档）：

| 指标                   | 阈值（持续）     |
| ---------------------- | ---------------- |
| 磁盘使用               | 90% 持续 2 分钟  |
| 内存                   | 92% 持续 2 分钟  |
| CPU                    | 95% 持续 2 分钟  |
| 系统状态（agent 掉线） | down 持续 1 分钟 |

通知发往与 kuma 相同的飞书群（hub 上配置 shoutrrr `lark://` URL）。

**Setup 顺序。**

1. Hub：为该主机添加 system，取 Add-System 对话框展示的公钥与每系统 token。
2. 在 `devops/ansible/group_vars/production/vars.yml` 写入 `beszel_hub_url` 与 `beszel_agent_key`（公钥——非秘密），token 入库为该环境 vault 的 `beszel_agent_token`。
3. `ansible-playbook devops/ansible/deploy.yml -e target=production`——两个变量齐备后部署才会安装 agent。
4. 验证:`docker compose -f ~/docker/beszel-agent/docker-compose.yml ps` 显示 agent 运行，hub 的 system 页面出现实时数据。

卸载是手动的（部署从不卸载）：`docker compose -f ~/docker/beszel-agent/docker-compose.yml down`，删除 `~/docker/beszel-agent`，删除变量。

<a id="alert-triage"></a>

## 告警分诊

| 红掉的监控项                      | 含义                                  | 首个动作                                                                    |
| --------------------------------- | ------------------------------------- | --------------------------------------------------------------------------- |
| user path + readiness + heartbeat | 源站整体宕机                          | SSH 到应用主机；`docker compose ps`、容器日志                               |
| user path + readiness 红，心跳绿  | Cloudflare / 边缘路径故障；源站还活着 | Cloudflare 控制台；源站无恙                                                 |
| 仅 user path 红                   | 静态导出或 CDN 缓存问题               | curl 对比 `/` 与 `/api/v1/ready`；检查发布                                  |
| readiness（503）+ 心跳 `down`     | 数据库故障                            | `docker compose logs postgres`；磁盘空间                                    |
| 仅心跳红                          | 心跳循环、supervisor 或 kuma 可达性   | `sudo supervisorctl status markpost-heartbeat`；心跳日志                    |
| 遥测断流（Grafana,按环境）        | 应用活着但遥测路径断了                | 该环境到 collector 的可达性；观测服务器上 `docker logs otelcol`             |
| Beszel agent 离线（经飞书）       | ttyo 活着但 agent/路由/hub 故障       | `docker compose -f ~/docker/beszel-agent/docker-compose.yml ps`；hub 可达性 |
