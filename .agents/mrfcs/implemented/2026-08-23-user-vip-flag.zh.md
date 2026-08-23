# MRFC: User VIP flag

Status: implemented

[English](2026-08-23-user-vip-flag.md) | 中文

## Problem

VIP 运营策略（issue #10）需要一个逐用户、可变的 vip 状态，其生命周期独立于策略自身的开关：策略开启期间作出的授予必须在管理员关闭后存续（「首批保留」），且管理员必须能无视登录活动、逐用户覆盖 vip。今天的 `users` 表带着治理状态（`role`、`is_active`），但没有任何「会员标记」类的东西。三个数据形态问题迫使决策：vip 存什么、放在哪、哪些读取方看得到。

## Decision

`users` 带布尔列 `vip`（`BOOLEAN NOT NULL DEFAULT FALSE`），由迁移 `000008_user_vip` 增加（down：`DROP COLUMN`）；存量行落在 `false`。领域模型以 `User.VIP` 暴露（`backend/internal/domain/user/user.go`）——带显式 `column:vip` GORM tag，因为 GORM 命名策略会把这个缩写词推导成 `v_ip`。所有写入经 `Repository.SetUserVIP`（`backend/internal/domain/user/repository.go`），实现位于 `backend/internal/infra/user_repo.go`，走共享的 `updateByID` map 助手。该值在授予时即持久化，此后绝不从登录 provider 加策略状态重新推导。

读取方通过两个用户 DTO 看到 vip——`UserResponse`（登录响应）与 `AdminUserItem`（管理列表/详情），均在 `backend/internal/api/rest/v1/types.go`，swagger 已再生成。它不进 JWT claims：auth 中间件逐请求从数据库重载完整用户行（`backend/internal/middleware/auth.go`），每个 handler 天然看到当前 vip，管理员翻转 vip 永远不使会话失效。

## Alternatives considered

**读时推导 vip（`github_id IS NOT NULL` 且策略开启）。** 零 schema 变更——但恰恰破坏策略赖以存在的两个要求：管理员关闭策略的瞬间推导值翻转、首批用户静默失去 vip；管理员的逐用户覆盖也没有容身之处。这一列的意义就在于一个独立于策略当前状态的可变逐用户事实。

**用分级枚举代替布尔。** 预设了没人要过的 VIP 等级，并把每个消费方拓宽成要去解释一个等级；将来真出现等级需求时再加一次迁移加回填——那正是该需求第一次变真的时刻。

**独立的会员表（`user_memberships`）。** 为未来的多策略状态做了归一化，但一个策略的一个布尔值不值得在每次用户读取上付出一次 join；若真出现第二个会员维度再重议。

**把 vip 放进 JWT claims。** 省掉每请求读列——但中间件本来就读整行，而除非每次 vip 变更都 bump `token_version`、逼用户为一个荣誉标记重新登录，claims 会处处过期。

## Consequences

这个标志买到的正是策略所需：在策略自身关闭后仍存续的状态、独立于登录活动的管理员覆盖、以及无需触碰 token 生命周期即可对每个 handler 即时可见。代价是词汇表被冻结——将来引入等级意味着一次带回填的迁移，这是刻意接受的、不去猜测未知等级语义的代价；`ALTER TABLE … ADD COLUMN … NOT NULL DEFAULT` 取一次短暂的排它锁，在 markpost 的用户量级下无关紧要，大表加列时应重访。GORM 的缩写词怪癖（`VIP` → `v_ip`）由显式列 tag 钉死；未来任何以连续大写命名的字段都需要同样处理。验证方式：`markpost migrate up`/`down`/`up` 干净循环；仓库测试覆盖授予、撤销与未找到（`TestUserRepository_SetUserVIP`）；两个 DTO 在再生成的 swagger 中暴露 `vip`。
