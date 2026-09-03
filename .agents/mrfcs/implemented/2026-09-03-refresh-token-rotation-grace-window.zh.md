# MRFC: Refresh token rotation grace window

Status: implemented

[English](2026-09-03-refresh-token-rotation-grace-window.md) | 中文

## Problem

刷新 token 对的标签页可能在后端完成轮换之后、写回 localStorage 之前死掉,而前端互斥无法封住这个间隙。轮换发生在服务端(`RefreshToken`,`backend/internal/service/auth/auth.go`):先软吊销被提交的 token、签发后继 token 对,客户端此后才持久化后继(`setTokens` → localStorage,`frontend/src/lib/api/base.ts`)。若标签页在该间隙内崩溃,localStorage 里留着的仍是已被吊销的旧 token——而 localStorage 是共享的,每个同级标签页持有同一份过期副本。下一次重放——同级标签页的 401 驱动刷新,或崩溃标签页的重载——就会命中复用检测路径(auth.md §2.3):已吊销 token 被再次提交即判为盗窃,`RevokeAllByUserID` 吊销该用户的全部 refresh token。一次客户端崩溃因此等于在所有设备上登出该用户。Web Locks 互斥(`frontend/src/lib/auth/refresh-lock.ts`)只能串行化存活标签页之间的刷新,帮不上忙:锁随标签页一起死,而服务端轮换已经发生。

复用检测的反应与这个故障模式不相称。窃取者重放一枚已消费的 token 本就一无所获——token 已死,响应也不会泄露后继(token 只以 SHA-256 哈希存储)。严格语义惩罚的是用户,而非攻击者。议题(#38)强迫的决策:是否引入一个有界的轮换宽限窗口,把"重放刚轮换的 token"视为竞态而非盗窃;若引入,窗口时长与窗口内的吊销语义是什么。

## Decision

markpost 运行**拒绝但不全量吊销**(reject-without-revocation)的宽限窗口。

**语义。**重放一枚吊销时间距现在小于窗口的 token 时,请求以 401 `ErrInvalidToken` 拒绝,但家族保持完好——不执行 `RevokeAllByUserID`。重放吊销时间超出窗口的 token、或没有吊销时间戳的存量行,维持严格行为:全量吊销加 reuse-detected 拒绝。每一次重放——无论窗口内外——仍被检测并拒绝,RFC 9700 §4.14.2 的检测要求继续满足;窗口内被推迟的只是吊销这一*反应*,且窗口内重放造不出任何东西,窃取者得不到任何凭证。窗口内重放以 info 级别记录 token 哈希,与 `refresh token reuse detected` 盗窃告警相区分。议题自己的量化——"窃取者在窗口内重放至多得手一次"——描述的是重签语义(见备选项);拒绝但不全量吊销把它改进为零。

**窗口:30 秒**——auth 服务内的 `refreshGraceWindow` 常量(同 `oauthStateTTL`),不是配置项。在拒绝但不全量吊销的语义下,加宽窗口的边际窃取价值为零——窗口内重放得不到任何可以续命的东西——而崩溃→重载间隙的覆盖面随之增长。30 秒与 WorkOS 默认值及议题 10–30 秒区间的上沿一致。

**Schema。**`refresh_tokens` 带有可空的 `revoked_at TIMESTAMPTZ` 列(迁移 `000011_refresh_token_revoked_at`),三处软吊销写入(`backend/internal/infra/token_repo.go` 的 `RevokeRefreshToken`、`RevokeAllByUserID`、`RevokeRefreshTokenByID`)都随 `revoked = true` 一并落时间戳;GORM 模型(`backend/internal/domain/user/token.go`)带有 `RevokedAt *time.Time` 字段。NULL 表示"该列存在之前已吊销",走严格路径——保守方向。`RefreshToken` 的复用分支读取被吊销行(`GetRevokedRefreshToken`,本就返回整行)并按 `withinGraceWindow` 分叉。

**前端零改动。**客户端刷新失败的处理(登出 → 重新登录)不变;崩溃设备重新认证,其他标签页与设备的会话得以幸存——这正是本决策消除的伤害。

文档:`specs/auth.md` §2.2–2.5 记载两列与窗口;`specs/backend/database-schema.md` 带有 `revoked_at` 行与放宽后的只写表表述;中文双胞胎同变更更新。

## Alternatives considered

**维持严格吊销(决策前的行为)。**残余窗口真实存在,而惩罚落在用户头上:轮换间隙内的任何标签页崩溃都会吊销该用户在所有设备上的全部会话,迫使全部重新登录。窃取者重放已消费 token 本就一无所获,严格性没有带来任何面向攻击者的价值——它只是把一次客户端崩溃转化成一次自我拒绝服务。

**返回同一批轮换后的 token(Supabase 的 reuse interval、WorkOS 的 grace period)。**主流答案——窗口内重放幂等、返回后继 token——要求服务端能恢复后继 token 的明文。markpost 只存 SHA-256 哈希(`TokenHash`);返回同一 token 意味着存储可恢复的 refresh token,为一个罕见崩溃场景削弱存储设计。输在约束上,而非优劣上。

**窗口内重签一对新 token(Auth0 的 rotation overlap period,`leeway`)。**哈希存储下可行,但它把一枚全新可用的 token 对交到任何窗口内重放的窃取者手中——宽限窗口由此变成"盗窃可以得手的有限时段"——并且重新打开 Supabase 设计中已被指出的重复重放漏洞(每隔不到 interval 重放一次即可无限铸造 token,supabase/auth#1901)。它还会分叉轮换链:从被重放 token 重新签发,必须孤立或吊销仍存活的后继,恰好重新引入 Web Locks 互斥已经解决的多标签页竞态。

**用内存"近期已吊销"缓存(ristretto)驱动窗口,规避 schema 变更。**免了迁移,却在 `revoked` 列之外建了第二个事实来源:重启后条目清空,窗口静默塌缩回严格路径(失败方向安全,但不一致);吊销时间——这个属于行的事实——寄居在行外。`refresh_tokens` 上的 `revoked_at` 才是这个事实的唯一归宿;迁移就是被点名的 Ask-first 代价。

**把窗口暴露为配置。**没有出现过运维调参的需求,且在拒绝但不全量吊销的语义下该值与性能、容量无关。常量避免了在安全敏感旋钮上开一个误配置面,并与 `oauthStateTTL` 一致。

## Consequences

残余崩溃窗口(轮换已完成、后继未持久化)不再升级为全量吊销:只有崩溃设备需要重新认证,同级标签页与其他设备的会话全部幸存。窗口外的盗窃检测不变。

交换的代价:

- **窗口内盗窃检测变钝**:若攻击者与合法客户端在 30 秒内竞用同一枚被盗 token,受害者的重放不再触发全量吊销,攻击者保住其已到手的后继 token。风险以窗口为界;在拒绝但不全量吊销的语义下,窗口永远不可能为攻击者铸造新凭证。
- **残余窗口是收窄而非消除**:崩溃后重放晚于 30 秒到达仍会触发全量吊销;更宽的窗口以检测延迟换边际恢复覆盖——主流产品的窗口在秒到一分钟量级,理由相同(Auth0 默认关闭)。
- **存量行**(`revoked_at IS NULL`)走严格路径——保守,且无需回填。

验证:testcontainers 测试钉住窗口内分支(重放的 token 被拒、合法后继仍可刷新)、窗口两侧 29 秒/31 秒的边界对、以及 NULL 时间戳存量行(家族被吊销);repository 测试钉住三处吊销写入都落 `revoked_at`。`go test ./...` 全绿,触及包的 `-race` 全绿,golangci-lint 0 问题。
