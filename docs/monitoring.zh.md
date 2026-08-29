# 可用性监控

[English](monitoring.md) | 中文

markpost 的可用性由自托管的 [uptime-kuma](https://github.com/louislam/uptime-kuma) 实例从外部探测 production 与 staging，并由生产主机反向推送心跳。告警发往飞书（主）与邮件（兜底）。本 runbook 承载监控项清单、通知渠道配置、告警策略与心跳的部署 / 卸载流程。

<a id="probe-model"></a>

## 探测模型

单一 URL 盯不住 CDN 前置的源站：edge 缓存的页面在源站宕机时依然保持绿色，而只盯源站的探针又看不到用户实际经过的边缘路径。监控项因此覆盖三个视点：

| 视点                          | 看到什么                                                                              | 承载者                                  |
| ----------------------------- | ------------------------------------------------------------------------------------- | --------------------------------------- |
| 边缘路径（kuma → 公网 URL）   | 完整用户路径：DNS、Cloudflare、网关、容器、静态导出                                   | 首页监控项                              |
| 源站探针（kuma → 未缓存端点） | Go 进程，并经 `/api/v1/ready` 触达数据库 —— `/api/v1/*` 携带 `no-store`，请求必达源站 | 就绪监控项                              |
| 反向心跳（VPS → kuma）        | 源站自身的本地判定，完全绕过 Cloudflare；静默即主机死亡                               | Push 监控项 + ttyo 上的 supervisor 循环 |

端点语义（`/health` 存活 vs `/ready` 就绪）见 [`api-schema.zh.md`](../specs/backend/api-schema.zh.md)；两者均豁免限流（[`rate-limiting.zh.md`](../specs/backend/rate-limiting.zh.md)）。支撑分层的 CDN 行为规范在 [`caching.zh.md`](../specs/backend/caching.zh.md) 与 [`cloudflare.zh.md`](../specs/backend/cloudflare.zh.md)。

<a id="monitors"></a>

## 监控项

所有监控项的公共设置：Heartbeat Interval `60`、Retries `3`、Retry Interval `60`，挂接两个通知渠道，不配维护窗口（重试阈值已吸收发版重启；维护窗口反而会把真故障静默）。

| 监控项                  | 类型                 | 目标 / 关键字段                                                                              |
| ----------------------- | -------------------- | -------------------------------------------------------------------------------------------- |
| prod · user path        | HTTP(s)              | URL `https://markpost.cc/`，接受状态 200；开启证书到期通知                                   |
| prod · origin readiness | HTTP(s) - Json Query | URL `https://markpost.cc/api/v1/ready`；Json Query `status`、运算符 `==`、期望值 `ready`     |
| prod · heartbeat        | Push                 | Interval `120`、Retries `2`；push URL 是保存在 ansible vault 的密钥（见 [心跳](#heartbeat)） |
| prod · domain           | 开关                 | 在 prod · user path 监控项上开启 `markpost.cc` 的域名到期通知                                |
| stg · user path         | HTTP(s)              | URL `https://markpost.bytehome.fun/`，接受状态 200                                           |
| stg · origin readiness  | HTTP(s) - Json Query | URL `http://192.168.5.50:8089/api/v1/ready`；Json Query `status` == `ready`                  |

证书与域名到期通知按 kuma 全局阈值（默认剩余 7/14/21 天）触发。stg · origin readiness 直探局域网地址，入口故障与实例故障因此可区分。

<a id="notification-channels"></a>

## 通知渠道

| 渠道         | kuma 配置                                                                 |
| ------------ | ------------------------------------------------------------------------- |
| 飞书（主）   | Notification Type `Feishu`；Webhook URL = 飞书群机器人 webhook            |
| 邮件（兜底） | Notification Type `Email (SMTP)`；SMTP 主机 / 端口 / 凭据、发件人、收件人 |

两个渠道都添加后，设为默认通知（Settings → Notifications → apply as default），所有监控项即同时经双渠道告警。

<a id="alert-policy"></a>

## 告警策略

- 监控项仅在连续失败约 4 分钟后（间隔 60 s × 重试 3 次）触发告警；瞬时边缘抖动与发版时的容器替换保持静默。
- 恢复通知开启：down→up 转换必通知。
- 重复提醒关闭（单人服务；恢复通知闭环）。
- 证书与域名到期在剩余 7/14/21 天时通知。

<a id="heartbeat"></a>

## 心跳（production）

生产 VPS 上的 supervisor 程序 `markpost-heartbeat` 运行 [`heartbeat.sh.j2`](../devops/ansible/templates/heartbeat.sh.j2)（渲染到 `~/docker/markpost/heartbeat.sh`）：每 60 秒探测 `http://127.0.0.1:8080/api/v1/ready`，并把判定推送给 kuma 的 push 端点。收到 `down` 判定（应用级故障，含数据库问题）或推送停止（主机死亡）时，kuma 将监控项置为 down。日志位于 `~/docker/markpost/data/heartbeat.log`。

[`deploy.yml`](../devops/ansible/deploy.yml) 中的部署任务仅在 vault 变量 `kuma_heartbeat_url` 已定义时安装脚本与程序，因此配置顺序是：

1. 在 kuma 添加 prod · heartbeat 监控项（Push，interval 120，retries 2）并复制其 push URL —— 持有者可伪造 up 心跳掩盖故障，按密钥对待。
2. 入库：`ansible-vault encrypt_string '<push-url>' --name kuma_heartbeat_url >> devops/ansible/group_vars/production/vault.yml`
3. 部署：`ansible-playbook devops/ansible/deploy.yml -e target=production` —— handler 执行 `supervisorctl reread && update` 并启动程序。
4. 验证：`sudo supervisorctl status markpost-heartbeat` 显示 RUNNING，且 kuma 收到心跳。

卸载是手动的（部署不会卸载）：删除 `/etc/supervisor/conf.d/markpost-heartbeat.conf`，执行 `sudo supervisorctl reread && sudo supervisorctl update`，并删除 vault 变量。

<a id="alert-triage"></a>

## 告警分诊

| 红色监控项                          | 含义                                | 第一步                                                   |
| ----------------------------------- | ----------------------------------- | -------------------------------------------------------- |
| user path + readiness + heartbeat   | 源站整体宕机                        | SSH 到 ttyo；`docker compose ps`、容器日志               |
| user path + readiness，heartbeat 绿 | Cloudflare / 边缘路径故障；源站存活 | Cloudflare 控制台；源站无恙                              |
| 仅 user path                        | 静态导出或 CDN 缓存问题             | curl 对比 `/` 与 `/api/v1/ready`；检查发布               |
| readiness（503）+ heartbeat `down`  | 数据库故障                          | `docker compose logs postgres`；磁盘空间                 |
| 仅 heartbeat                        | 心跳循环、supervisor 或 kuma 可达性 | `sudo supervisorctl status markpost-heartbeat`；心跳日志 |
