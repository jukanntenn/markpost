# MRFC: User VIP flag

Status: proposed

[English](2026-08-23-user-vip-flag.md) | 中文

## Problem

VIP 运营策略（issue #10）需要一个逐用户、可变的 vip 状态，其生命周期独立于策略自身的开关：策略开启期间作出的授予必须在管理员关闭后存续（「首批保留」），且管理员必须能无视登录活动、逐用户覆盖 vip。今天的 `users` 表带着治理状态（`role`、`is_active`），但没有任何「会员标记」类的东西。三个数据形态问题迫使决策：vip 存什么、放在哪、哪些读取方看得到。

## Proposal

通过版本化迁移为 `users` 增加布尔列 `vip`（`BOOLEAN NOT NULL DEFAULT FALSE`）——`000008_user_vip`，沿 `000005_token_version` 的 `ALTER TABLE` 模式——down 侧为 `DROP COLUMN`；存量行全部落在 `false`。领域模型在 `backend/internal/domain/user/user.go` 增加 `User.VIP bool`，所有写入经仓库 setter 走既有的 `updateByID` map 模式（`backend/internal/infra/user_repo.go`）。该值在授予时即持久化：vip 在授予时写入，此后绝不从登录 provider 加策略当前状态重新推导。

对读取方，`vip` 进入两个用户 DTO——`UserResponse`（登录响应携带当前用户）与 `AdminUserItem`（管理列表/详情），均在 `backend/internal/api/rest/v1/types.go`，swagger 再生成。它刻意不进 JWT claims：auth 中间件逐请求从数据库重载完整用户行（`backend/internal/middleware/auth.go`），每个 handler 天然看到当前 vip；不进 token 意味着管理员翻转 vip 永远不需要使会话失效。

## Alternatives considered

**读时推导 vip（`github_id IS NOT NULL` 且策略开启）。** 零 schema 变更——但恰恰破坏策略赖以存在的两个要求：管理员关闭策略的瞬间推导值翻转、首批用户静默失去 vip；管理员的逐用户覆盖也没有容身之处。这一列的意义就在于一个独立于策略当前状态的可变逐用户事实。

**用分级枚举代替布尔。** 预设了没人要过的 VIP 等级，并把每个消费方拓宽成要去解释一个等级；将来真出现等级需求时再加一次迁移加回填——那正是该需求第一次变真的时刻。

**独立的会员表（`user_memberships`）。** 为未来的多策略状态做了归一化，但一个策略的一个布尔值不值得在每次用户读取上付出一次 join；若真出现第二个会员维度再重议。

**把 vip 放进 JWT claims。** 省掉每请求读列——但中间件本来就读整行，而除非每次 vip 变更都 bump `token_version`、逼用户为一个荣誉标记重新登录，claims 会处处过期。

## Acceptance criteria

`markpost migrate up` 后 `down` 再 `up` 干净应用；列存在、默认 `false`、存量行不受影响。`UserResponse` 与 `AdminUserItem` 暴露 `vip` 且 swagger 反映。handler 从请求上下文用户（数据库加载）读 vip，绝不从 token claims 读。

## Risks

布尔冻结了词汇表——将来引入等级意味着一次带回填的迁移，这是刻意接受的代价，以换取现在不去猜测未知的等级语义。`ALTER TABLE … ADD COLUMN … NOT NULL DEFAULT` 取一次短暂的排它锁；在 markpost 的用户量级下无关紧要，记录在此以便未来大表加列时有意识地重访。
