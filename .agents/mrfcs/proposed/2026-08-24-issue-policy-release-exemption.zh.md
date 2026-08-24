# MRFC: Issue-policy exemption for release pull requests

Status: proposed

[English](2026-08-24-issue-policy-release-exemption.md) | 中文

## Problem

[`issue-policy` workflow](../../../.github/workflows/issue-policy.yml) 对每个非草稿、非机器的 PR 校验准入契约:conventional-commit 标题、至少一个 `area/*` 标签、至少一个 issue 引用。release PR 天然不满足:它只带版本号与变更日志,不引用任何 issue,历史上也不带 area 标签。2026-08-23 实测:PR #23(`chore: release v0.2.0-rc.6`)以两项错误让 `Issue policy` 变红,且照旧被合并。

一旦 [PR conclusion jobs and required checks](../implemented/2026-08-24-pr-conclusion-jobs-required-checks.zh.md) 使门禁 3 获得平台强制,"红但仍可合并"就不复存在:要么 release PR 被豁免而全绿,要么每次发版都卡在一个对它不适用的检查上。

## Proposal

[`policy.py`](../../../.github/issue-management/policy.py) 的 `pr` 子命令在 PR 的 head 分支匹配 `release/**` 时跳过三项准入检查——标题形式、area 标签、issue 引用;head ref 随事件载荷而来,判定是纯函数分支,无额外 API 调用。其余照旧:release PR 真携带的引用仍被正常校验,issue 侧检查不变。release 全绿后,`Issue policy` 与五个 conclusion 检查一并加入 `main` 的 required checks,准入契约同样获得平台强制。release 分支命名本已刚性(`release/v0.2.0-rc.6`,实测)——归 release 技能所有,不是本记录新造的约定。

## Alternatives considered

**约定:release PR 自带 `area/devops` 与一个长期引用。** 实测反驳:#23 是发版压力下裸报的,且没有可引用的追踪 issue;该约定还会逼每次发版造一个一次性 issue,去满足一个为评审注意力分流而生的检查——它不该管发版。

**让 `Issue policy` 留在 required 集合之外、对 release 保持红。** 策略层免于强制,但它把"发版时永远红"变成常态——这正是门禁 3 平台强制要消除的歧义——而且 never-merge-red 规则第一次被豁免时就再也无法教人。

**对 `release/**` head 直接跳过 issue-policy workflow。** 丢掉了仍然适用的检查——release PR 一旦携带引用,引用校验照跑——并让 workflow 从 release PR 页面上彻底消失;豁免应放在策略的准入规则里,契约住在那里。

## Acceptance criteria

来自 `release/**` 分支、无标签无引用的 PR 通过 `policy.py pr`;非 release PR 缺任一检查仍然失败(两者都进 prek 运行的策略单元测试)。两层落地后,`Issue policy` 加入 `main` 的 required checks,并在一次真实发版 PR 上端到端实测为绿。

## Risks

豁免以分支名为界:从错误命名的分支发版会挂掉 policy,required checks 生效后即阻塞——立即可见,改名即修,命名归 release 技能管。读取 head ref 必须覆盖 workflow 触发的全部事件形态(`pull_request`、`pull_request_review`),由单元测试覆盖。豁免也意味着 release PR 上的 area 误标永远无法被策略捕获——今天没有任何按 area 路由发版的机制,此代价是理论性的。
