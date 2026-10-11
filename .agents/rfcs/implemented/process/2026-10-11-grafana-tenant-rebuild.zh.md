# RFC: 依据平台契约重建 markpost 的 Grafana 租户资产

Status: implemented

[English](2026-10-11-grafana-tenant-rebuild.md) | 中文

## 问题

观测 NAS 上的 Grafana 平台——[OTLP 观测栈](2026-09-19-otlp-observability-stack.zh.md)的存储与查询半边——已移交外部 devops 运维仓库，由其从零重建：compose、`.env`、数据源与策略全部重造，数据卷清空，API key 重新签发。markpost 在旧实例上的一切——观测布线改造（[observability wiring overhaul](2026-10-04-observability-wiring-overhaul.zh.md)）记录为义务的交接材料、`~/docker/grafana/provisioning/` 下的 provisioning 文件——随旧卷一同消亡。租户按新的文件 provisioning 契约重新入驻，markpost 需要把租户侧收进仓库：按契约编写的看板与告警、可审计的推送、以及部署后平台确实加载成功的证明。重写还须吸收两个晚于旧文件的事实：[指标接线改造](../architecture/2026-10-11-metrics-wiring-runtime-source.zh.md)用 `go.*` 约定族替换了 `process.runtime.go.*`（照搬旧 runtime 面板只会一片空白），且 dev 现在也上报遥测，管线宕机探测器因此必须覆盖它。

## 决策

markpost 的租户资产位于本仓库 `devops/ansible/files/grafana/`，由部署 playbook 的租户同步块（[`devops/ansible/deploy.yml`](../../../../devops/ansible/deploy.yml)）将其任务委托给 hs1 推送；平台（compose、数据源、根策略、Grafana 数据卷）仍归 devops 仓库所有。runbook 的[告警归属一节](../../../../docs/monitoring.zh.md)承载当前状态。

### markpost 落地的租户入驻契约

`devops/ansible/files/grafana/provisioning/` 下的四文件集：

- **`dashboards/markpost.yml`**——看板 provider（`folder: markpost`、`allowUiUpdates: false`）；未设字段取官方默认，含让推送即时生效的 10 秒重扫。
- **`dashboards/markpost/*.json`**——四块看板，uid 为 `markpost-overview`、`markpost-http`、`markpost-runtime`、`markpost-delivery`。`markpost-*` 前缀是 markpost 的看板 uid 命名空间，删除后永不复用。契约中的 `T-*` uid 命名空间解读为平台侧租户对象（文件夹、数据源、平台自有策略），归 devops 仓库所有；markpost 自己的文件只携带 `markpost-*` uid——在此明说，供评审检验。
- **`alerting/markpost-rules.yaml`**——告警规则（清单见下）。
- **`alerting/markpost-contact-points.yaml`**——一个接触点（`markpost-email`，email 类型，经 mailrise SMTP 网关投递到 markpost 的飞书频道），配一条把 `grafana_folder = markpost` 告警路由到它的通知策略条目。路由与接触点同文件发布，让这对组合一起过评审。已对照 Grafana 源码核实的警告：provisioning 一条 `policies` 条目会整体替换 org 的策略树根，共享实例上只有一份策略文件能持有根——平台自己的策略文件必须内嵌这条路由，它才能在重新 provisioning 后幸存。

数据源引用限定契约 uid：全部指标面板与规则用 `victoriametrics`，HTTP 看板上唯一的 ERROR 趋势日志面板用 `victorialogs`（暂停中的 ERROR-log-rate 规则的实时替身）；没有面板引用 `jaeger`。查询使用 markpost 实际输出的指标与属性名（`deployment.environment.name` 由 compose 模板钉死），这假设平台的 collector 以原生 OTLP 转发到 VictoriaMetrics——若中间有 Prometheus 翻译跳板，点号名称会变成下划线。

每条告警规则都链接其归属看板：规则的 `dashboardUid`/`panelId` 字段加带看板 URL 的 `dashboard` 注解，与 `summary`、`runbook` 注解并列。Grafana 看板最佳实践把"大多数看板应当被告警链接"列为定向浏览的目标，通知模板会渲染该链接（并可附面板截图），加快分诊。

### 推送及其断言

[`deploy.yml`](../../../../devops/ansible/deploy.yml) 的租户同步块将任务委托给 hs1，逐文件复制白名单内的租户文件——每个文件一个任务，绝不对 `provisioning/` 做目录级复制，其他租户的文件因此不可触碰。该块在每次部署每个环境时运行：租户资产与环境无关，重复部署幂等——未变更的文件为零操作拷贝，断言均为读取。删除以显式 `state: absent` 任务承载（注释示例给出退役形态）。看板推送经 10 秒 provisioning 重扫无需重启即生效；告警 provisioning 由该块调用 admin reload 端点应用。推送后断言从 hs1 访问 Grafana HTTP API，任何一项缺失即令部署失败：文件夹 `markpost` 存在且恰好容纳四块看板、每个看板 uid 经看板 API 可解析、ruler API 显示文件夹内全部十二条规则、每条通知策略的 receiver 都解析到现存接触点。API token 是 hs1 上的 vault 变量（`host_vars/hs1.yml` 中的 `grafana_api_token`，身份 `markpost-observability`）；推送免 token 运行，断言以变量存在为门——与 otel-token 同步相同的 setup 顺序契约。

### 规则清单：十一条在评估，加一条暂停

[runbook 的规则表](../../../../docs/monitoring.zh.md)共六行，展开为三个组（`http`、`delivery`、`telemetry`，均 60s）里的十一条在评估规则：

1. `markpost-http-5xx-warn`——5xx 占请求速率比例超 5%（5m 窗口），持续 5m。
2. `markpost-http-5xx-crit`——5xx 占比超 20%（2m 窗口），持续 2m。
3. `markpost-http-p95`——p95 请求时长超 1s，持续 10m。
4. `markpost-http-rate-collapse`——10m 请求速率低于一小时前基线的 20%，持续 10m。
5. `markpost-delivery-backlog`——`markpost.delivery.pending` 超 100，持续 10m。
6. `markpost-delivery-backlog-growth`——积压 15m 内增长 50+，持续 0m。
7. `markpost-delivery-failures`——投递失败速率超 0，按 `error_category` 分维，持续 15m。
8. `markpost-cdn-purge-failures`——CDN 清除失败速率超 0，持续 15m。
9. `markpost-telemetry-gap-dev`——`markpost-dev` 的 `go.goroutine.count` 10m 无数据，持续 0m。
10. `markpost-telemetry-gap-staging`——对 `markpost-staging` 的同款规则。
11. `markpost-telemetry-gap-production`——对 production 的同款规则。

第十二条 `markpost-error-log-rate` 以文档记录的原因暂停 provisioning（日志数据源输出的整数帧被 Grafana 13.2.1 的告警表达式拒绝）。重建的计数与布线改造时代"十一条中十条在评估"有两处不同，均为有意：dev 获得自己的 telemetry-gap 规则，因为 dev 现在上报遥测（其 vars 已设 `otel_otlp_endpoint`）；表中单行"Delivery / CDN purge failures"被 provision 成两条规则，让投递与 CDN 失败信号以独立阈值告警。所有规则的 `noDataState` 均为 OK——序列缺失意味着"无流量"或"管线宕机"，而管线宕机恰是 `absent_over_time` telemetry-gap 规则负责触发的情况；可用性本身仍是 kuma 的职责。路由标签按 `alertname` 与 `env` 分组：按环境的 gap 规则携带 `env` 标签，因为 Alertmanager 格式的标签名无法承载带点号的 `deployment.environment.name`。

### 顺序

runtime 看板与 telemetry-gap 规则直接按[指标接线改造](../architecture/2026-10-11-metrics-wiring-runtime-source.zh.md)发布的 `go.*` 族编写；早于该后端到达某环境部署本栈，会让该环境的 runtime 看板空白、gap 规则误触发——应在 [#135](https://github.com/jukanntenn/markpost/issues/135) 的后端之后部署，而非之前。租户同步随部署 playbook 运行，其委托任务与桥接变量需要 `hs1` inventory 条目——此处应用与 [#133](https://github.com/jukanntenn/markpost/issues/133) 修订相同的顶层主机（用户 alice），让本树在该合并前后运行如一。

## 考虑过的替代方案

**照搬旧 NAS 的 JSON 文件。** 省力之最——但契约上这是零兼容重建：旧卷被清空，旧实例上没有任何需要匹配的东西，且旧资产带着重写正是要丢弃的 v1 时代假设（仪器已不存在的 `process.runtime.go.*` runtime 面板、无 dev 遥测缺口、缺失的告警链接）。照搬等于引入死重并延续漂移。

**API 驱动的 provisioning（脚本经 HTTP API 推看板与规则）。** 免去 provisioning 路径假设，也允许窄权限的按租户 token——但平台选定文件 provisioning 作为租户契约，正因为它是声明式、git 可评审、无需凭据即热加载的；API 脚本要重新发明 provisioner 已拥有的状态管理（创建、更新、删除的次序），推送也不再是可审计的纯文件复制。

**告警链接不进注解。** 规则文件更小——但官方指引要求从告警直达看板，链接只花一行注解；缺了它，每次告警都要在分诊表里人工找对看板。

## 后果

markpost 的看板与告警成为本仓库可评审的工件，由部署 playbook 推送，并以 API 断言证明加载成功——Grafana 侧的可观测性故事不再依赖任何仓库都不追踪的服务器上的交接材料。代价：重建从空数据卷起步（无导入历史——"阈值是起点"的调优基线自重新入驻之日重计）；Grafana API key 必须重签并以新身份（`markpost-observability`）入库于 hs1 的 host_vars，旧 NAS 的 provisioning 文件随旧卷消失而非归档；平台实际的 provisioning 路径、端口与 mailrise apprise 地址在 hs1 的 host_vars 与资产文件中是标注过的假设，待评审或首跑修正；policies 条目的根替换语义使 markpost 的路由只有被平台自己的策略文件承载时才能幸存——配对断言会在下次推送时抓到它的丢失。
