# MRFC: GitHub-login VIP grant strategy

Status: implemented

[English](2026-08-23-github-login-vip-grant-strategy.md) | 中文

## Problem

「经 GitHub 登录的用户成为 VIP」这一策略必须能被管理员当作运营动作关闭——即时生效、无需部署——且其关闭态必须停发而不回收已授予的。两件事曾悬而未决，因为仓库里不存在类似物：其一，仓库完全没有运行时设置机制（配置是启动时一次加载的 Viper/TOML；管理端点只有只读指标与逐用户治理）；其二，授予本身需要精确语义——何时触发、对再登录用户做什么、策略关闭后登录还做什么。需求方明说这是*第一个*运营策略，后续策略会走同一条路。

## Decision

开关存于键值 `settings` 表（`key TEXT PRIMARY KEY, value JSONB NOT NULL, updated_by, updated_at`），由迁移 `000009_settings` 创建并播种 `vip = {"enabled": true}`，策略随上线即开启；未来策略落进同一归宿。域包为 `backend/internal/domain/settings`（`SettingValue` 经 `driver.Valuer`/`sql.Scanner` 承载布尔值），仓库实现位于 `backend/internal/infra/settings_repo.go`（经 `ON CONFLICT (key)` upsert），其 `VIPStrategyEnabled` 读取兼任 auth service 的端口——登录路径直读、不缓存。

管理员经 `GET /api/v1/admin/settings` 与 `PUT /api/v1/admin/settings/:key` 驱动，二者在 `RequireAdmin` 之后（v1 只放行已播种的 `vip` 键；其余一律 400 `unknown_setting`），每次写入以 `setting.set` 审计、记录键与值。授予发生在 `LoginWithGitHub`（`backend/internal/service/auth/auth.go`）里 `GetOrCreateFromGitHub` 之后：开启期间，任意 GitHub 登录——新建或再登录——幂等地把该用户 vip 置 true（已是 true 则跳过写入）；关闭期间，登录对 vip 双向零触碰。设置读取失败向「不授予」一侧失败：登录照常完成、不授予、错误记日志。管理员手工写入仍是唯一撤销路径；策略开启期间被撤销的用户下次 GitHub 登录会重新获得——这是记录在案的权衡：先关策略、再逐个整理。

## Alternatives considered

**config.toml/Viper 开关。** 不需要新表新端点——但翻转变成一次发布加部署，增长实验的运营杠杆变成工程动作，且值会在各环境间漂移而不是待在一个可审计的位置。

**单行强类型设置表（一个布尔列）。** 读取形态最简单，但未来的每个设置都要再迁移一次；需求方说了「第一个策略」，键值形态立刻开始摊薄成本。

**仅在建号（首次登录）时授予。** 避开撤销-再授予的拉锯——但抛弃了「首批用户登录即可成为 vip」的本意对象：存量 GitHub 关联用户，且不符合「任何通过 github 登录的用户」的任何读法。

**策略关闭时登录即撤销。** 直接违反「已授予不回收」；一旦关闭，登录必须对 vip 零副作用。

## Consequences

策略成为纯运营动作：一次可审计的 PUT 翻转、下一次登录即生效、无需部署；键值 JSONB 表为需求方预告的后续策略摊薄成本。接受的代价：通用设置表容易沦为配置垃圾场，由 unknown-key 400（不经迁移播种就进不来）与 schema/行为走 MRFC/Ask-first 的路径钳制；不缓存的逐登录读取在该路径多加一条查询；策略开启下的撤销-再授予拉锯是刻意的——若实践需要，由接替 MRFC 增加豁免位，而非悄悄打补丁。验证方式：迁移 up/down 循环；仓库测试（读回、upsert、未知键 `ErrNotFound`）；授予钩子测试覆盖开启/关闭/读取失败/未接线四态；handler 测试经真实管理链路覆盖列表/upsert/未知键。
