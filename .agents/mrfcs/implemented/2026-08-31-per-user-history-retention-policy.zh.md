# MRFC: Per-user history retention policy

Status: implemented

[English](2026-08-31-per-user-history-retention-policy.md) | 中文

## Problem

markpost 的保留策略是一刀切的：`[post] retention_days`（默认 7，0 = 永不过期）与 `[delivery] history_retention`（默认 168h）都是全局配置键，由两个 cron 调用的 CLI 命令（`prune-expired-posts`、`prune-delivery-history`）清扫。运营需要按用户的承诺——「这位用户的历史永久保存」「这批用户保留 30 天」——通常挂在 VIP 荣誉标记上，而 schema、API、admin UI 里都不存在任何按用户维度的东西。两笔相邻欠账被迫并入同一决策：仓库根本没有为 prune 命令安排任何调度（跳过手工步骤的部署就永远不删）；且缩短保留窗口在下一次清扫时是不可逆的删除——admin 的操作体验在这里是正确性问题，不是锦上添花。

## Decision

一个按用户的保留策略同时驱动 **`posts` 与 `delivery_history`**，保住「投递历史活得不超过文章自身寿命」的既有不变量（[delivery scheduler 规格](../../../specs/backend/delivery-scheduler.zh.md)）。

**数据模型 —— `users.retention_days INT NULL`**（迁移 `000010_user_retention`）。用一列，理由与当年把 `vip` 放上 `users` 相同（[VIP 标记 MRFC](./2026-08-23-user-vip-flag.zh.md)）：

| 取值   | 含义                                                                  |
| ------ | --------------------------------------------------------------------- |
| NULL   | 继承 —— 各表的全局配置生效（今天 = 7 天）                              |
| 0      | 永久保存（复用 `[post] retention_days` 既有的 0 = 永不编码）           |
| 1–3650 | 保留 N 天                                                             |

生效 cutoff 按行推导：posts 用 `created_at < now() − COALESCE(用户覆盖值, 全局)`；delivery_history 相同但走 `LEFT JOIN`——被 `ON DELETE SET NULL` 孤立的历史行回落全局窗口（匿名行带不了个人策略）。全局默认留在 config.toml：改它是部署级的重量，不是运营动作。

**prune 命令保形，携带按行谓词。** 两个 CLI 保留 `--dry-run`/`--batch-size`、批量子查询 LIMIT 循环（[delivery 队列 MRFC](./2026-07-10-persistent-best-effort-delivery-queue.zh.md)）与按 QID 的渲染缓存清理；单点 cutoff 变为按行的 `CASE`（`retention_days = 0` → 永不入选；NULL 比较把行排除）。每日 cron job（`markpost-retention-prune`，`devops/ansible/files/prune-retention.sh`）在 Ansible 管理的宿主机上调用两个命令，关掉调度欠账。dry-run 输出按生效策略合计待删数。

**VIP 类默认值，授予时物化。** 运行时 settings 键 `vip_retention_days`（值形状 `{"days": …}`，把 settings 表从[授予策略 MRFC](./2026-08-23-github-login-vip-grant-strategy.zh.md)的 `{"enabled": …}` 泛化）持有类默认。任何路径授予 VIP——手动 `PATCH /admin/users/:id/vip` 或 GitHub 登录自动授予——只要该用户仍是继承态，就在同一语句里把类默认写到用户行上（`UPDATE … retention_days = COALESCE(retention_days, $default)`）。撤销 VIP **保留**已落列的值：荣誉降级绝不把旧数据暴露回全局清扫。`scope:"vip"` 批量写入按需对齐存量 VIP。

**Admin API**，端到端镜像 `/vip` 端点（[VIP 徽章 MRFC](./2026-08-23-vip-badge-and-admin-management.zh.md)）：

- `PATCH /api/v1/admin/users/:id/retention` —— `{retention_days: null | 0 | 1–3650}`；`null` 清除回继承态；审计 `user.set_retention`（旧→新）。
- `POST /api/v1/admin/users/retention/bulk` —— `{user_ids: […]}`（≤ 200）或 `{scope: "vip"}`，加策略值；返回 `{updated}`；单条审计 `user.set_retention_bulk`（含范围与数量）。
- `POST /api/v1/admin/retention/impact` —— 同样的目标形状加候选值；返回 `{users_affected, posts_to_delete, history_to_delete}`——危险确认对话框的数据源。候选为 null 时按各表全局回退解析；0（永久）不命中任何行。
- `GET /api/v1/admin/retention/defaults` —— 全局回退窗口，供 UI 渲染继承策略的生效值。

**Admin UX**（四个接触面，一个共用对话框）：

- 用户列表带「保留策略」列，显示**生效值**——永久（徽章）/ N 天 / 默认 · 7 天（经 defaults 端点解析）——工具栏「批量选择」开关揭示复选框列、表头全选与悬浮操作条（已选 N · 设置保留策略 · 退出）。行 ⋮ 菜单加「保留策略…」作为单用户路径。
- 共用对话框：三段式选择器（继承全局默认 / 永久保存 / 保留 N 天），第三段展开预设芯片（7/30/90/365）与自由输入，校验 1–3650。
- 缩短窗口是危险流程：对话框取影响预览，删除量 > 0 时展示计数并要求确认（≥ 1000 篇文章时输入 `DELETE` 确认），延展 UserGovernance 模式。
- 用户页头带 VIP 策略栏，包住授予策略开关：类默认选择器（跟随全局 / 永久 / N 天）与「应用到全部 VIP 用户 (N)」按钮——以 vip-align 模式打开同一对话框。用户详情页 Profile 卡带当前值与「设置…」动作。

用户侧对自己保留策略的可见性（/posts 上的徽章）明确不在范围；运营提出需求时再跟进。

## Alternatives considered

**只覆盖 posts，或只覆盖 delivery_history。** 只保文章，VIP 用户会在永久内容旁留一份 7 天的通知记录；只保历史，保住的是主题 7 天后就被删掉的记录。两者都破坏同寿命不变量；翻倍的表面是自洽承诺的代价，且两个 prune 循环本就同形。

**实时解析链里的 VIP 类默认**（用户显式 > 类默认 > 全局）。它不碰授予路径就覆盖未来 VIP，但撤销 VIP 会瞬间把旧数据暴露回全局清扫——治理副作用——且每个读取方（prune SQL、admin 展示）都要吃一条推导链。授予时物化让存储值保持显式、降级无害；类承诺依然自维持，因为两条授予路径汇聚在同一钩子上。

**纯一次性批量**（不存类默认）。表面最简，但 VIP 会经 GitHub 策略在无人值守时自动出现；每个新 VIP 默默留在 7 天，承诺不可见地腐烂——恰是项目拒绝的静默退化。

**按用户策略表或 settings 复合键**（`retention:user:123`）。一个维度、一个写入方：VIP MRFC 已为单一维度否决过成员表，键值 settings 生来就是全局的。

**把全局默认搬进运行时 settings。** 可运行时改的全局保留是穿着运营外衣的部署级权力；config.toml 配得上这个动作的重量。settings 表仍会长出 `vip_retention_days` 键——那是运营按群体反复调整的类承诺，才是运行时重量。

## Consequences

换来的是：运营可以按用户、按批次、按 VIP 类承诺保留策略，授予钩子自维持，admin 列表显示生效值，每次缩短都有删除计数闸门；prune 任务真的被调度了，策略在新部署上是真实生效的。验证：testcontainers 覆盖显式永久、显式 N 天、缩短在下次清扫生效、继承、全局 0、被孤立的历史行回落全局窗口；授予时物化在两条授予路径上断言（继承者取得类默认、显式值在授予与撤销中幸存、撤销保留它）；bulk/impact/defaults 端点在 handler 层覆盖；admin 界面以 Playwright 截图验证——[`01` 保留策略列](./2026-08-31-per-user-history-retention-policy/01-users-retention-column.png)、[`02` 批量选择模式](./2026-08-31-per-user-history-retention-policy/02-bulk-select-mode.png)、[`03` 共用对话框](./2026-08-31-per-user-history-retention-policy/03-retention-dialog.png)、[`04` 天数输入](./2026-08-31-per-user-history-retention-policy/04-dialog-days-expanded.png)、[`05` 缩短窗口的影响确认](./2026-08-31-per-user-history-retention-policy/05-shorten-impact-confirm.png)、[`06` VIP 策略栏](./2026-08-31-per-user-history-retention-policy/06-vip-policy-bar.png)、[`07` VIP 对齐](./2026-08-31-per-user-history-retention-policy/07-vip-apply-all-dialog.png)、[`08` 详情行](./2026-08-31-per-user-history-retention-policy/08-user-detail-row.png)；i18n 键齐备于全部四个语言文件。

代价与遗留风险：缩短保留是下一次清扫时的不可逆删除——UI 用影响数与确认拦截，持 token 的 admin 绕过对话框（有审计；admin 信任边界）；策略设置前已被清扫的数据无法复活；settings 值形状从 `{"enabled"}` 泛化到含 `{"days"}`（错误形状双向 400 拒绝）；物化把保留策略耦合进授予路径——第三条授予路径必须汇入同一钩子（今天由两条路径共同调用的服务层接缝保证）；按行 `CASE` cutoff 让 prune SQL 比单时间戳比较更重——在约 0.12 次写/秒的量级下是噪声，且批量化本就限定了锁范围。
