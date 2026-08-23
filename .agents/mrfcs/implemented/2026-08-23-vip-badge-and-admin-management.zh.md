# MRFC: VIP badge and admin management surface

Status: implemented

[English](2026-08-23-vip-badge-and-admin-management.md) | 中文

## Problem

vip 既已入库（[标志 MRFC](2026-08-23-user-vip-flag.zh.md)）、又由策略授予（[授予 MRFC](2026-08-23-github-login-vip-grant-strategy.zh.md)），策略却仍没有面向用户的表面：VIP 用户看不到自己的身份——而这份可见性正是增长策略的全部产品——管理员也没有逐用户杠杆与界面开关。前端在仪表盘欢迎语与应用外壳用户菜单两处渲染当前用户的用户名，管理端在用户列表与详情页渲染用户名；仓库有自己的 Badge 组件与可照抄的治理对话框范式。徽章写什么、出现在哪、管理员如何驱动两个杠杆，就是本层的决策。

## Decision

徽章是仓库自有 `Badge`（`frontend/src/components/ui/badge.tsx`，`variant="accent"`）封装成的 `VipBadge`（`frontend/src/components/ui/vip-badge.tsx`），文案为语言无关的 `VIP`，紧贴用户名之后渲染于四处：仪表盘欢迎语（`DashboardPage`）、应用外壳用户菜单（`AppShell`）、管理端用户列表与用户详情页；非 VIP 用户看不到任何多余之物。公开帖子页维持在外——那里不渲染作者用户名。

逐用户管理是 `PATCH /api/v1/admin/users/:id/vip`，体 `{"vip": <bool>}`，端到端镜像 `/active` 端点：管理 REST 层的 handler 带审计动作 `user.set_vip`（值为元数据）、响应 `AdminUserItem`，`UserMutator` 端口之后的 service `SetUserVIP`，经 `updateByID` 的仓库 setter。与 `/active` 的两处刻意偏离：不设自我操作守卫（管理员设置自己的 vip 不破坏任何不变量）、不 bump `token_version`（vip 不进任何 claim；逐请求重读整行使翻转即刻可见）。管理 UI 把该动作加进逐行治理菜单（`UserGovernance`），沿用同一确认对话框与失效刷新范式，对自己也开放。

策略开关是管理端用户列表页头部的一个开关（`AdminUsersPage` 里的 `VipStrategyToggle`），调用 `PUT /admin/settings/vip`——v1 不做独立设置页。四份 locale 文件同步携带治理字符串、开关标签与审计叙事（`admin.users.vip*`、`admin.users.vipStrategy.*`、`admin.audit.action.user.vipGrant/vipRevoke`、`admin.audit.action.setting.*`）；`audit-action-text` 映射两个新动作；MSW handler 覆盖新端点。

## Alternatives considered

**用 @base-ui/react 的 Badge。** 所装版本没有 Badge 组件；仓库自有 Badge 是既定组件，且已按 `variant` 分型。

**专门的 vip service 或嵌套资源端点。** 为一个布尔值造更多「RESTful」表面；单个 PATCH 照抄的是评审与测试都已熟悉的成熟范式。

**vip 变更时 bump `token_version`。** 把荣誉标记当安全状态对待；会让每个被管理的用户强制重新登录，而实际上没有任何权限变化。

**独立的管理设置页承载开关。** 为仅有一项的设置预设一个表面；用户列表页头部把杠杆放在运营者已经在的地方，等设置真正多起来再抽页。

**公开帖子页徽章。** 那里今天不渲染作者用户名；为了挂徽章而新增作者展示，是在发明策略从没要求过的功能。

## Consequences

VIP 用户在欢迎语与用户菜单自己的用户名旁看到标记，管理员在同一屏看到并驱动每个用户的 vip 与策略开关；两个杠杆都有带本地化叙事的审计。代价：四语言同步靠手工（实施层逐键列出，评审者能一眼对 diff 四个文件）；徽章会诱使范围向「意味着什么」蔓延——若 vip 将来授予权限，那次变更自带新 MRFC，在那之前文案保持纯荣誉。开关寄放在用户列表页使它视觉上与管理耦合；在它仍是唯一策略时可接受，第二个策略进入 `settings` 时重议。验证方式：handler 测试覆盖授予/撤销/404；前端套件覆盖徽章渲染与开关；新端点的 swagger 已再生成。
