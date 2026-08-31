# MRFC: Per-user history retention policy

Status: proposed

[English](2026-08-31-per-user-history-retention-policy.md) | 中文

## Problem

markpost 的保留策略是一刀切的：`[post] retention_days`（默认 7，0 = 永不过期）与 `[delivery] history_retention`（默认 168h）都是全局配置键，由两个 cron 调用的 CLI 命令（`prune-expired-posts`、`prune-delivery-history`）清扫。运营现在需要按用户的承诺——「这位用户的历史永久保存」「这批用户保留 30 天」——通常挂在 VIP 荣誉标记上，而 schema、API、admin UI 里都不存在任何按用户维度的东西。两笔相邻欠账被迫并入同一决策：仓库根本没有为 prune 命令安排任何调度（跳过手工步骤的部署就永远不删）；且缩短保留窗口在下一次清扫时是不可逆的删除——admin 的操作体验在这里是正确性问题，不是锦上添花。

## Proposal

加一个按用户的保留策略，同时驱动 **`posts` 与 `delivery_history`** 两张表，保住「投递历史活得不超过文章自身寿命」的既有不变量（[delivery scheduler 规格](../../../specs/backend/delivery-scheduler.zh.md)）。

**数据模型 —— `users.retention_days INT NULL`**（迁移 `000010_user_retention`）。用一列，理由与当年把 `vip` 放上 `users` 相同（[VIP 标记 MRFC](../implemented/2026-08-23-user-vip-flag.zh.md)）：

| 取值   | 含义                                                                  |
| ------ | --------------------------------------------------------------------- |
| NULL   | 继承 —— 各表的全局配置生效（今天 = 7 天）                              |
| 0      | 永久保存（复用 `[post] retention_days` 既有的 0 = 永不编码）           |
| 1–3650 | 保留 N 天                                                             |

生效 cutoff 按行推导：posts 用 `created_at < now() − COALESCE(用户覆盖值, 全局)`；delivery_history 相同但走 `LEFT JOIN`——被 `ON DELETE SET NULL` 孤立的历史行回落全局窗口（匿名行带不了个人策略）。全局默认留在 config.toml：改它是部署级的重量，不是运营动作。

**prune 命令保形，换谓词。** 两个 CLI 保留 `--dry-run`/`--batch-size`、批量子查询 LIMIT 循环（[delivery 队列 MRFC](../implemented/2026-07-10-persistent-best-effort-delivery-queue.zh.md)）与按 QID 的渲染缓存清理；单点 cutoff 变为按行的 `CASE`（`retention_days = 0` → 永不入选；NULL 比较把行排除）。Ansible 部署一个每日 systemd timer 调用两个命令，顺手关掉调度欠账——没有它，策略在新部署上是空转的表演。

**VIP 类默认值，授予时物化。** 新的运行时 settings 键 `vip_retention_days`（值形状 `{"days": …}`，把 settings 表从[授予策略 MRFC](../implemented/2026-08-23-github-login-vip-grant-strategy.zh.md)的 `{"enabled": …}` 泛化）持有类默认。任何路径授予 VIP——手动 `PATCH /admin/users/:id/vip` 或 GitHub 登录自动授予——只要该用户仍是继承态，就在同一动作里把类默认写到用户行上。撤销 VIP **保留**已落列的值：荣誉降级绝不能把两年的数据暴露给 7 天清扫。`scope:"vip"` 批量写入保留，用于对存量 VIP 做一次性对齐。

**Admin API**，端到端镜像 `/vip` 端点（[VIP 徽章 MRFC](../implemented/2026-08-23-vip-badge-and-admin-management.zh.md)）：

- `PATCH /api/v1/admin/users/:id/retention` —— `{retention_days: null | 0 | 1–3650}`；`null` 清除回继承态。
- `POST /api/v1/admin/users/retention/bulk` —— `{user_ids: […]}`（≤ 200）或 `{scope: "vip"}`，加策略值；返回 `{updated}`；写一条审计（`user.set_retention_bulk`，含范围与数量），而非 N 条。
- `POST /api/v1/admin/retention/impact` —— 同样的目标形状加候选值；返回 `{users_affected, posts_to_delete, history_to_delete}`。影响预览是一等端点，因为它是危险确认 UI 的数据源。
- PATCH 单用户写审计 `user.set_retention`（含旧→新叙事）。

**Admin UX**（四个接触面，一个共用对话框）：

- 用户列表加「保留策略」列，显示**生效值**——永久（徽章）/ N 天 / 默认 · 7 天（解析后的值，全局配置一改显示跟着走）；工具栏「批量选择」开关揭示复选框列、表头全选与悬浮操作条（已选 N · 设置保留策略 · 退出）。行 ⋮ 菜单加「保留策略…」作为单用户路径。
- 共用对话框：三段式选择器（继承全局默认 / 永久保存 / 保留 N 天），第三段展开预设芯片（7/30/90/365）与自由输入，校验 1–3650。
- 缩短窗口是危险流程：对话框取影响数，删除量 > 0 时展示计数并要求确认（≥ 1000 篇文章时输入 `DELETE` 确认），延展 UserGovernance 模式。
- 用户页头长出 VIP 策略栏，包住既有授予策略开关：类默认选择器（跟随全局 / 永久 / N 天）与「应用到全部 VIP 用户 (N)」按钮——以 vip-align 模式打开同一对话框并聚合影响数。用户详情页 Profile 卡加一行当前值与「设置…」动作。

用户侧对自己保留策略的可见性（/posts 上的徽章）明确不在本次范围；运营提出需求时再跟进。

## Alternatives considered

**只覆盖 posts，或只覆盖 delivery_history。** 只保文章，VIP 用户会在永久内容旁留一份 7 天的通知记录；只保历史，保住的是主题 7 天后就被删掉的记录。两者都破坏同寿命不变量；翻倍的表面是自洽承诺的代价，且两个 prune 循环本就同形。

**实时解析链里的 VIP 类默认**（用户显式 > 类默认 > 全局）。它不碰授予路径就覆盖未来 VIP，但撤销 VIP 会瞬间把旧数据暴露回全局清扫——治理副作用——且每个读取方（prune SQL、admin 展示）都要吃一条推导链。授予时物化让存储值保持显式、降级无害；类承诺依然自维持，因为两条授予路径汇聚在同一钩子上。

**纯一次性批量**（不存类默认）。表面最简，但 VIP 会经 GitHub 策略在无人值守时自动出现；每个新 VIP 默默留在 7 天，承诺不可见地腐烂——恰是项目拒绝的静默退化。

**按用户策略表或 settings 复合键**（`retention:user:123`）。一个维度、一个写入方：VIP MRFC 已为单一维度否决过成员表，键值 settings 生来就是全局的。

**把全局默认搬进运行时 settings。** 可运行时改的全局保留是穿着运营外衣的部署级权力；config.toml 配得上这个动作的重量。settings 表仍会长出 `vip_retention_days` 键——那是运营按群体反复调整的类承诺，才是运行时重量。

## Acceptance criteria

- `000010_user_retention` up/down 干净应用；所有用户为 NULL 时，两个 prune 命令删除的集合与今天完全一致。
- testcontainers 覆盖：显式永久、显式 N 天、继承、全局 `retention_days = 0`、用户删除（SET NULL）孤立的历史行回落全局窗口。
- 授予时物化在两条授予路径上都触发、用户已有值时为无操作；撤销不碰已落列的值；第三条授予路径不经过钩子就无法落地（测试级强制）。
- 批量端点：ids 与 `scope:"vip"` 都逐用户写值、遵守 200 上限、只产出一条审计；impact 端点对单用户、多选、vip 范围返回正确计数。
- Ansible playbook 在临时主机上部署每日 timer，两次 prune 在日志中可观测；失败可见，不静默。
- admin 流程（单用户、批量选择、VIP 对齐、带影响数的缩短）用 Playwright 对 dev 栈验证；删除影响 > 0 时缩短确认拦截；i18n 键齐备于全部四个语言文件。

## Risks

- 缩短保留是下一次清扫时的不可逆删除；UI 用影响数与确认拦截，API 拦不住——持 token 的 admin 绕过对话框（有审计；admin 信任边界）。
- 策略设置前已被清扫的数据无法复活；「永久保存」只救得回还存在的数据。
- settings 值形状从 `{"enabled"}` 泛化到含 `{"days"}` —— settings API 契约必须向后兼容地吸收它（既有 `vip` 键不动）。
- 物化把保留策略耦合进授予路径；钩子必须活过 VIP 授予策略的重构，耦合记录于此与钩子现场。
- 按行 `CASE` cutoff 让 prune SQL 比单时间戳比较更重；在约 0.12 次写/秒的量级下是噪声，且批量化本就限定了锁范围。
