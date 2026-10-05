# RFC: Observability wiring overhaul (three-env telemetry, kuma rework, Grafana alerting)

Status: implemented

[English](2026-10-04-observability-wiring-overhaul.md) | 中文

## Problem

三类发现破坏了观测栈设计时的故事:

1. **staging 的遥测从未到达。** staging 原本接公网 OTLP 入口(`otlp.bytehome.fun`),但从 staging 主机该域名解析到内网反向代理——上面没有 otlp vhost(TLS 直接失败),且 vps2 caddy 的 `remote_ip` 白名单里也只有生产机的 IP。`markpost-staging` 的 metrics、logs、traces 自栈建成起就缺席三个存储——"staging 作为晋升门"从未验证过遥测路径。
2. **kuma 推送链给出错误判决。** vault 中的 push URL 带着 kuma 复制出来的查询后缀(`?status=up&msg=OK&ping=`),而 `pgbackrest-check.py` 又用 `&` 追加了第二组 `status`/`msg`。kuma 把重复参数解析成数组,读到"非 up"的 status,把每一次真实的备份检查推送都记成 down 心跳、消息损坏为 `[object Object]`——备份监控显示 0% 可用率,而备份链本身是健康的。另一处:生产心跳的 supervisor conf 装好了却从未执行 `supervisorctl reread && update`,程序从未启动,心跳监控一条心跳都没收到。
3. **可用性之外没有任何告警,也没有环境身份。** Grafana 零告警规则、零通知接触点;日志管线把生产的 DEBUG 记录送进了 collector(otelslog 桥没有级别过滤,文件 handler 却有);遥测不带部署属性——环境区分全靠 `service.name` 后缀,而它没法干净地驱动看板与告警路由。

## Decision

1. **标准部署属性。** 每个接线环境导出 `OTEL_RESOURCE_ATTRIBUTES=deployment.environment.name=<env>`(compose 模板)。看板与告警规则按该标签切分,不再解析 `service.name`。
2. **staging 改 LAN 直连。** staging 的 `otel_otlp_endpoint` 改为内网直达 collector(`http://192.168.5.57:4318`)——与 dev 同路径;生产保留公网入口。此条取代[frp 传输 MRFC](2026-09-19-otlp-transport-frp-exposure.zh.md)中 staging 走公网的决定。
3. **裸 push URL 契约。** vault 中的 kuma URL 是不带查询后缀的裸 push 端点,并按各生产者的可达性选择主机形态——生产走公网路由,staging 走内网(它到 vps2 的出站会挂起,公网形态会静默超时)。`heartbeat.py` 以 `?query` 追加(不变);`pgbackrest-check.py` 现在也用 `?query`(原为 `&`——那只对带后缀的 URL 成立,正是重复参数 bug 的来源)。
4. **心跳对称。** 部署在 staging 与 production 都安装 supervisor 程序,按环境的 vault 变量 `kuma_heartbeat_url` 守卫;staging 有自己的 push 监控与 vault 变量。两个环境彩排完全相同的形态。
5. **kuma 清单标准。** 两个分组监控(`markpost · production`、`markpost · staging`)作为叶子监控的父级;每个监控带自描述标签(`env:production` / `env:staging`、`service:markpost`、`layer:edge|origin|push`);命名遵循 `markpost · <env> · <对象> (<layer>)`;push 监控显式设置 retries(心跳 2,每日备份检查 1)。
6. **双通知渠道。** 飞书(主)+ 经 mailrise SMTP 网关的邮件(兜底),都设为默认并应用到全部监控,兑现 runbook 的承诺。
7. **告警归属边界。** kuma 负责边缘/源站可用性及证书与域名到期;beszel 负责主机资源;Grafana 负责应用、业务与遥测管线信号——包括监视管线自身的"遥测断流"告警。Grafana-managed 告警(接触点、通知策略、规则)在观测服务器上以代码形式 provision;规则清单记录在 runbook。Grafana 13.2.1 OSS 不带 Feishu 接收器集成,故 Grafana 经 mailrise SMTP 网关投递(email 接触点 → apprise → 同一个飞书群);11 条规则中 10 条正常评估,ERROR 日志速率规则暂时置为 paused,等日志数据源输出 SSE 兼容的数值帧后再启用。

## Alternatives considered

- **修复 staging 的公网路径**(staging 主机加 hosts/DNS override,并把家宽出口 IP 加进 vps2 caddy 白名单)。落选:两个由外部运维持有的活动部件——内网 split-horizon DNS 和动态家宽 IP——必须永远同时正确,而这条路径的生产者代码与 LAN 直连逐字节相同;公网传输由生产环境全天候彩排。
- **在 collector 注入部署属性**而非生产者。落选:collector 是外部运维的 handoff material,而部署环境是生产者自身的身份——仓库恰拥有这个接触点。
- **重建 kuma 监控来改名**。落选:新 push token 会废掉 vault 里的 URL,强制重存三份秘密(外加新的 staging 心跳共四份)。重命名加重新挂父级可保持 token 稳定。
- **每环境一套看板副本**。落选:三份副本静默漂移;由 `deployment.environment.name` 驱动的单套看板加多维告警规则,原生产出按环境区分的序列与告警实例。
- **用 VictoriaMetrics 内的 vmalert 做告警**。落选:多一个要 provision 的外部运维组件;Grafana-managed 告警本就能查询三个数据源、支持 provisioning 文件并有统一的通知路由。

## Consequences

换来的:staging 遥测以与生产相同的生产者代码落地,三个存储全部可见;看板与告警按标准 semconv 属性路由;两条 kuma 推送链给出真实判决,清单靠标签与分组可扩展;生产的 DEBUG 记录不再泄漏进 collector,而本地 JSONL 文件仍是崩溃通道;应用/业务层终于有了告警,且管线监视自身。

付出的:四份 vault push URL 泄漏即需轮换(裸形态、同等保密、更简单的契约);staging 不再彩排公网传输——这是有意的取舍,该传输属外部运维基础设施且生产全天候覆盖;staging 多一个 supervisor 程序;观测服务器上的 Grafana provisioning 文件是 handoff material,须与 runbook 的规则清单保持同步;Grafana 告警经 mailrise 才能到达飞书(多一跳,且 SMTP 密码填入前保持静默),ERROR 日志速率规则因上游类型兼容问题暂缓。

验证:`markpost-staging` 带着 `deployment.environment.name=staging` 出现在 VictoriaMetrics、VictoriaLogs、Jaeger;kuma 两个分组下的子监控变绿(部署后数分钟内心跳到达,备份判决在每日检查时到达);把任一环境的遥测掐断十分钟会触发 telemetry-gap 告警到飞书;VictoriaLogs 中生产日志不再出现 DEBUG 记录。
