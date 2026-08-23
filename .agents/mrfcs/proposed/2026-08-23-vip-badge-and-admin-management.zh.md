# MRFC: VIP badge and admin management surface

Status: proposed

[English](2026-08-23-vip-badge-and-admin-management.md) | 中文

## Problem

vip 既已入库（[标志 MRFC](../implemented/2026-08-23-user-vip-flag.zh.md)）、又由策略授予（[授予 MRFC](../implemented/2026-08-23-github-login-vip-grant-strategy.zh.md)），策略却仍没有面向用户的表面：VIP 用户看不到自己的身份——而这份可见性正是增长策略的全部产品——管理员也没有逐用户杠杆与界面开关。前端在仪表盘欢迎语与应用外壳用户菜单两处渲染当前用户的用户名，管理端在用户列表与详情页渲染用户名；仓库有自己的 Badge 组件与可照抄的治理对话框范式。徽章写什么、出现在哪、管理员如何驱动两个杠杆，就是本层要决定的。

## Proposal

**徽章。** 复用仓库自有 `Badge`（`frontend/src/components/ui/badge.tsx`，`variant="accent"`），文案在所有语言统一为 `VIP`，紧贴用户名之后渲染于四处：仪表盘欢迎语（`DashboardPage`）、应用外壳用户菜单标签（`AppShell`）、管理端用户列表与用户详情页。非 VIP 用户什么都不看到；徽章纯属荣誉，今天不授予任何权限。公开帖子页不在范围：那里不渲染作者用户名，为此发明一个会把策略撑大到没人要求过的程度。

**逐用户管理。** `PATCH /api/v1/admin/users/:id/vip`，体 `{"vip": <bool>}`，端到端镜像 `/active` 端点：管理 REST 层的 handler（bind、`parseIDParam`、审计动作 `user.set_vip` 以值为元数据、响应 `AdminUserItem`）、`UserMutator` 端口之后的 service `SetUserVIP`、经 `updateByID` 的仓库 setter。与 `/active` 有两处刻意偏离：不设自我操作守卫（管理员设置自己的 vip 不破坏任何不变量）、不 bump `token_version`——vip 不进任何 claim、不带任何权限，而 auth 中间件每请求重读整行，翻转即刻可见，不需要让任何人的会话失效。管理 UI 把该动作加进既有的逐行治理菜单（`UserGovernance`），沿用同一确认对话框与失效刷新范式。

**策略开关 UI。** 管理端用户列表页头部一个开关——「GitHub 登录自动 VIP」开/关——调用 `PUT /admin/settings/vip`；v1 不做独立设置页，一个策略不值一个导航面。

**本地化与 mock。** 四份 locale 文件（`en`、`zh-Hans`、`zh-Hant`、`ja`）同步增加治理字符串与开关标签（徽章文案为语言无关的 `VIP`）；`audit-action-text` 映射 `user.set_vip` 与 `setting.set`；MSW handler 覆盖新端点。

## Alternatives considered

**用 @base-ui/react 的 Badge。** 所装版本没有 Badge 组件；仓库自有 Badge 是既定组件，且已按 `variant` 分型。

**专门的 vip service 或嵌套资源端点。** 为一个布尔值造更多「RESTful」表面；单个 PATCH 照抄的是评审与测试都已熟悉的成熟范式。

**vip 变更时 bump `token_version`。** 把荣誉标记当安全状态对待；会让每个被管理的用户强制重新登录，而实际上没有任何权限变化。

**独立的管理设置页承载开关。** 为仅有一项的设置预设一个表面；用户列表页头部把杠杆放在运营者已经在的地方，等设置真正多起来再抽页。

**公开帖子页徽章。** 那里今天不渲染作者用户名；为了挂徽章而新增作者展示，是在发明策略从没要求过的功能。

## Acceptance criteria

VIP 用户在欢迎语与用户菜单自己的用户名旁看到徽章；非 VIP 用户不见任何多余之物。管理端用户列表与详情展示每个用户的 vip，行菜单提供授予/撤销并带确认、toast 与列表失效刷新。头部开关翻转 `vip` 设置，下一次 GitHub 登录按授予 MRFC 的语义行事。两个新审计动作在审计视图里有本地化文案。四份 locale 文件齐备所有新键，MSW 覆盖新端点。

## Risks

四语言同步靠手工、易漂移；实施层会逐键列出，让评审者能一眼对 diff 四个文件。徽章会诱使范围向「意味着什么」蔓延——若 vip 将来授予权限，那次变更自带新 MRFC，在那之前文案保持纯荣誉。开关寄放在用户列表页使它视觉上与管理耦合；在它仍是唯一策略时可接受，第二个策略进入 `settings` 时重议。
