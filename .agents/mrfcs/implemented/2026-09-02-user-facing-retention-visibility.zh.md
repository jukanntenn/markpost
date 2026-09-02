# MRFC: 用户侧保留策略可见性

Status: implemented

[English](2026-09-02-user-facing-retention-visibility.md) | 中文

## Problem

markpost 的保留策略是一份关于用户数据的承诺，却从未展示给数据的主人。窗口放在全局配置里（`[post] retention_days`、`[delivery] history_retention`），而按用户策略（[per-user retention MRFC](2026-08-31-per-user-history-retention-policy.zh.md)）落地后，一条覆盖值可以直接承诺「此人的数据永久保留」。数据主人对此一无所见：`/posts` 与 `/delivery/history` 只是在清扫时刻默默丢失行。默认持久的用户撞上无声删除，默认即逝的用户把内容无限期留存在库里。该 MRFC 有意把主人侧可见性留作后继（「a badge on /posts… a follow-up if operations asks for it」）；随后运营提出了这个要求。可展示的值是*生效*策略——覆盖值 ?? 全局——只有服务器能按调用者解析；静态文案恰恰会对策略所要保护的那批被覆盖用户撒谎。

## Decision

`GET /api/v1/me/retention`（JWT，纯读——不加专用限流器）返回调用者的生效策略 `{posts_days, history_days}`，每个值 `0`（永久，沿用既有零值编码）或整天数。解析与 prune 谓词逐字镜像（`post_repo`/`delivery_attempt_repo` 的按行 `CASE`）：显式覆盖对两类同时生效；继承的用户读各自表的全局值，两个数字只会在全局值彼此漂移时不同。认证中间件每个请求重载用户行，端点直接从上下文用户解析、无仓储往返；`history_retention` 按整天展示，非零余数向上取整——展示绝不会在读作「0 天」时实际含义是「非永久」。该端点开启了 `/me` 命名空间：自作用域的读取归拢于此。

共用的 `RetentionHint` 在 `/posts` 与 `/delivery/history` 的 `PageHeading` 与列表之间渲染一行弱化文本，由单个 query 供给（`staleTime` 5 分钟；该值只在管理员操作或部署改配置时变动）。文案只说结果、不说机制：永久 →「数据将永久保留」；N 天 →「数据保留 N 天，到期后自动清除」——四语言齐备。`/posts` 渲染 `posts_days`；`/delivery/history` 渲染 `history_days`；继承/VIP 机制留在 admin 侧。

## Alternatives considered

**每行到期时间戳。** 与维护者当面否决：一个用户的行共享同一策略，页面级一行已完整回答「我的数据保留多久」；每行日期把同一个数复述成视觉噪音。

**全局配置烘死的静态文案，不做 API。** 最便宜——且恰恰对策略为之存在的用户出错：永久覆盖的 VIP 会读到全局的「7 天」。只有服务器能按调用者解析覆盖值 ?? 全局。

**把提示折进列表端点的响应信封。** 让提示耦合到帖子列表首页 3 秒轮询，并把解析逻辑复制进两个载荷；一个专用轻量读取镜像 admin 层的 `GET /admin/retention/defaults`，且只有一处归属。

**连 admin 列表一起覆盖。** 与维护者当面否决：按用户栈的 admin UX 已给用户列表提供生效保留策略列；admin 的帖子/历史列表复述的是治理数据，不是本功能为之存在的主人承诺。

## Consequences

这笔取舍买到的是：数据主人读到系统对自己数据的承诺，按调用者解析——永久覆盖的用户不再读到对自己不适用的全局「7 天」，默认即逝的用户知道删除已被排期。读取是一次廉价 JSON 响应、无仓储往返。验证：解析矩阵（显式永久、显式 N、默认全局下继承、全局 posts 0 下继承）与时长取整（168h → 7、36h → 2、不足一天 → 1）由 service 单测钉住；handler 在 gin 引擎里跑同一矩阵外加无上下文用户的 fail-closed 路径，未认证 401 归 `AuthWithBlacklist` 中间件测试套件所有——落地后的解析是纯函数，矩阵因此跑在快速测试而非 testcontainers；`RetentionHint` 的形态由 MSW 组件测试覆盖；[api-schema](../../../specs/backend/api-schema.zh.md) 双语对记录了端点；两个页面经 Playwright 截图验证——两页 × 永久与 N 天形态，验收证据以交付 PR（#81）的评论承载而非二进制入库存放。代价是：文案承诺清除，而清扫本身是 prune 层部署的每日 cron——提示陈述策略而非删除计时；提示最多滞后策略变更 5 分钟 `staleTime`；历史按整天展示而底层 cutoff 是时间戳，清扫边界与展示的日边界可能相差数小时；`/me` 从此是一个命名空间，今后的自作用域端点加入它而不是另起新前缀。
