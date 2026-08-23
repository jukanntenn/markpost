# MRFC: GitHub-login VIP grant strategy

Status: proposed

[English](2026-08-23-github-login-vip-grant-strategy.md) | 中文

## Problem

「经 GitHub 登录的用户成为 VIP」这一策略必须能被管理员当作运营动作关闭——即时生效、无需部署——且其关闭态必须停发而不回收已授予的。两件事悬而未决，因为仓库里不存在类似物：其一，仓库完全没有运行时设置机制（配置是启动时一次加载的 Viper/TOML，`backend/internal/config/config.go`；管理端点今天只有只读指标与逐用户治理）；其二，授予本身需要精确语义——何时触发、对再登录用户做什么、策略关闭后登录还做什么。需求方明说这是*第一个*运营策略，后续策略会走同一条路。

## Proposal

**开关存储——键值设置表。** 新表 `site_settings`（`key TEXT PRIMARY KEY, value JSONB NOT NULL, updated_by INTEGER, updated_at TIMESTAMPTZ`），由其迁移播种 `github-vip-strategy = {"enabled": true}`，策略随上线即开启。这是未来策略的共用归宿，一次买断。管理面：`GET /api/v1/admin/settings` 返回全部行，`PUT /api/v1/admin/settings/:key` 写一行（v1 仅 `github-vip-strategy`，体 `{"enabled": <bool>}`），都在 `RequireAdmin` 之后，审计动作 `site_setting.set` 记录键与值。读取按用直查表——登录是低频路径，本就跑好几条查询；在没有度量之前不加缓存。

**授予语义。** 在 `LoginWithGitHub`（`backend/internal/service/auth/auth.go`）里 `GetOrCreateFromGitHub` 之后：策略开启时，把该用户 vip 置 true——幂等，新建用户与再登录用户一视同仁，存量 GitHub 用户在窗口期内登录即加入首批。策略关闭时，登录对 vip 不做任何事：不授予、不撤销。管理员的手工写入（叠在此层之上的 VIP 徽章与管理层的逐用户 PATCH 端点）是唯一的撤销路径，而[标志本身](2026-08-23-user-vip-flag.zh.md)授予即持久，关闭策略永远不会让首批失去资格。若登录途中设置读取失败，登录照常完成但不授予，错误记日志——向「不授予」一侧失败，因为错误授予的 vip 比漏发的更难收回。

**一个摆在明面上的权衡。** 策略开启期间，管理员对某用户 vip 的撤销会被该用户下一次 GitHub 登录重新覆盖——策略自我重申。想让撤销立住的管理员先关策略、再逐个整理。逐用户豁免位曾被考虑并刻意推迟：没有需求提出过，且它会给每个用户行增加第二个可变事实。

## Alternatives considered

**config.toml/Viper 开关。** 不需要新表新端点——但翻转变成一次发布加部署，增长实验的运营杠杆变成工程动作，且值会在各环境间漂移而不是待在一个可审计的位置。

**单行强类型设置表（一个布尔列）。** 读取形态最简单，但未来的每个设置都要再迁移一次；需求方说了「第一个策略」，键值形态立刻开始摊薄成本。

**仅在建号（首次登录）时授予。** 避开撤销-再授予的拉锯——但抛弃了「首批用户登录即可成为 vip」的本意对象：存量 GitHub 关联用户，且不符合「任何通过 github 登录的用户」的任何读法。

**策略关闭时登录即撤销。** 直接违反「已授予不回收」；一旦关闭，登录必须对 vip 零副作用。

## Acceptance criteria

策略开启时，一次 GitHub 登录——新用户或 `vip=false` 的再登录用户——结束时 vip 为 true；密码登录永远不写 vip。策略关闭时，两个方向的登录都不改变 vip。管理员翻转对下一次登录即刻生效、无需重启；翻转动作以 `site_setting.set` 审计。登录途中设置读取失败时登录仍完成、不授予、错误有日志。

## Risks

通用的 `site_settings` 表容易沦为配置垃圾场；限定它只装运营策略——schema 与行为变更仍走 MRFC 与 Ask-first。不缓存的逐登录读取在该路径多加一条查询；当前量级可忽略，有度量之前不缓存。策略开启下的撤销-再授予拉锯是本决策刻意为之并记录于此；若实践表明管理员需要在策略运行期间做持久撤销，由一份接替 MRFC 增加豁免位，而不是悄悄打补丁。
