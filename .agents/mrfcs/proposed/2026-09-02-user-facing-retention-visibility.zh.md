# MRFC: 用户侧保留策略可见性

Status: proposed

[English](2026-09-02-user-facing-retention-visibility.md) | 中文

## Problem

markpost 的保留策略是一份关于用户数据的承诺，却从未展示给数据的主人。窗口放在全局配置里（`[post] retention_days`、`[delivery] history_retention`），而按用户策略（[per-user retention MRFC](2026-08-31-per-user-history-retention-policy.zh.md)）落地后，一条覆盖值可以直接承诺「此人的数据永久保留」。数据主人对此一无所见：`/posts` 与 `/delivery/history` 只是在清扫时刻默默丢失行。默认持久的用户撞上无声删除，默认即逝的用户把内容无限期留存在库里。该 MRFC 有意把主人侧可见性留作后继（「a badge on /posts… a follow-up if operations asks for it」）；现在运营提出了这个要求。可展示的值是*生效*策略——覆盖值 ?? 全局——只有服务器能按调用者解析；静态文案恰恰会对策略所要保护的那批被覆盖用户撒谎。

## Proposal

一个认证端点，加一个共用的页面级提示。

**`GET /api/v1/me/retention`**（JWT，纯读——不加专用限流器）返回调用者的生效策略 `{posts_days, history_days}`，每个值 `0`（永久，沿用既有零值编码）或整天数。解析与 prune 谓词逐字镜像（`post_repo`/`delivery_attempt_repo` 的按行 `CASE`）：显式覆盖对两类同时生效；继承的用户读各自表的全局值，两个数字只会在全局值彼此漂移时不同。`history_retention` 是 Go duration；按整天展示，非零余数向上取整——展示绝不能在读作「0 天」时实际含义是「非永久」。该端点开启 `/me` 命名空间：自作用域的读取今后归拢于此。

**共用的 `RetentionHint`** —— `/posts` 与 `/delivery/history` 上 `PageHeading` 与列表之间的一行弱化文本，由单个 query 供给（`staleTime` 5 分钟；该值只在管理员操作或部署改配置时变动）。文案只说结果、不说机制：永久 →「数据将永久保留」；N 天 →「数据保留 N 天，到期后自动清除」。`/posts` 渲染 `posts_days`；`/delivery/history` 渲染 `history_days`。继承/VIP 机制留在 admin 侧（[per-user retention MRFC](2026-08-31-per-user-history-retention-policy.zh.md) 拥有那批界面）；数据主人只读结果。

schema 前置已落地——`users.retention_days` 与解析语义随按用户栈的 prune 层合并；该栈其余 admin 层与本设计正交，不构成顺序约束。

## Alternatives considered

**每行到期时间戳。** 与维护者当面否决：一个用户的行共享同一策略，页面级一行已完整回答「我的数据保留多久」；每行日期把同一个数复述成视觉噪音。

**全局配置烘死的静态文案，不做 API。** 最便宜——且恰恰对策略为之存在的用户出错：永久覆盖的 VIP 会读到全局的「7 天」。只有服务器能按调用者解析覆盖值 ?? 全局。

**把提示折进列表端点的响应信封。** 让提示耦合到帖子列表首页 3 秒轮询，并把解析逻辑复制进两个载荷；一个专用轻量读取镜像 admin 层的 `GET /admin/retention/defaults`，且只有一处归属。

**连 admin 列表一起覆盖。** 与维护者当面否决：按用户栈的 admin UX 已给用户列表提供生效保留策略列；admin 的帖子/历史列表复述的是治理数据，不是本功能为之存在的主人承诺。

## Acceptance criteria

- testcontainers：显式 0 → `{0, 0}`；显式 N → `{N, N}`；继承且默认全局 → `{7, 7}`；继承且 `[post] retention_days = 0` → posts 永久；未认证 401。duration→天数舍入单测（168h → 7、36h → 2）。
- 两个页面在永久与 N 天两种形态下渲染提示；四语言键集一致性测试通过。
- 对 dev 栈 Playwright：两页 × 两形态的截图证据。
- [api-schema](../../../specs/backend/api-schema.zh.md) 双语对补端点行；不新增 spec 页，不改 `specs/index.md`。

## Risks

- 文案承诺清除（「到期后自动清除」）；清扫本身是 prune 层部署的每日 cron。提示陈述的是策略而非删除计时——若运营停用 cron，文字会比机制活得久。记录在案，避免有人把提示当作计时器解读。
- `/me` 开启一个命名空间；今后的自作用域端点应归拢于此，而不是另起新前缀。
- 提示最多滞后策略变更 5 分钟 `staleTime`；对一行安心文案可接受，写明以理解数值新鲜度。
- 历史按整天展示而底层 cutoff 是时间戳；清扫边界与展示的日边界可能相差数小时。
