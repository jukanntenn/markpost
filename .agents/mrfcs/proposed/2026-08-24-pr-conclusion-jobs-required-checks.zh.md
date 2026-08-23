# MRFC: PR conclusion jobs and required checks

Status: proposed

[English](2026-08-24-pr-conclusion-jobs-required-checks.md) | 中文

## Problem

门禁 3——交付批准——没有平台强制:`main` 的分支保护今天不要求任何检查,带红检查的 PR 可以合并。2026-08-23 实测:PR #23(`chore: release v0.2.0-rc.6`)在 `Issue policy` 失败的状态下合并,在无人盯守的时刻破了闭环的 never-merge-red 规则。

required checks 也不能直接指向今天的 job 名。五个质量 workflow——Lint、Test、Build、E2E、Docs——都在 workflow 级对 `pull_request` 做路径过滤:Lint/Test/Build/E2E 用 `paths-ignore`(`**/*.md`、`docs/**`、`.gitignore`、`LICENSE`),Docs 用正向 `paths` 清单。改动落在某个 workflow 路径集之外的 PR 根本不触发它,其检查会永远停在 "Expected — waiting for status" 并阻塞合并。本仓库实测:纯文档 PR #24 只跑了 Docs 与 Issue policy,Lint、Test、Build、E2E 均未启动。

[开发闭环 MRFC](../implemented/2026-08-22-agent-driven-development-loop.zh.md) 把平台强制留给了维护者的设置步骤;本记录设计该步骤要开启的东西。

## Proposal

五个 workflow 全部去掉 `pull_request` 上的 workflow 级路径过滤(`push: main` 的过滤原样保留——required checks 只关乎 PR,main 推送的算力经济不受损)。路径选择性下沉到 job 级:Lint 与 Docs 补上 Test、Build、E2E 已有的 `changes` job(dorny/paths-filter),所有真实 job 照旧按它条件执行。

每个 workflow 增加一个 `conclusion` job:`needs` 列出全部真实 job,`if: always()` 使其在跳过与失败之后照常运行,单一步骤仅在任一 need 结果为 `failure` 时失败——被跳过的 job 记为成功。由此每个 PR 恒定产出五个稳定检查:`Lint / conclusion`、`Test / conclusion`、`Build / conclusion`、`E2E / conclusion`、`Docs / conclusion`。

维护者随后把这五个设为 `main` 的 required checks。`Issue policy` 刻意不在首个集合里:它无条件跑在每个 PR 上(名字稳定),但天然地让 release PR 失败——release 豁免问题由其配套提案先解决,之后再加入集合。

## Alternatives considered

**直接把 required 指向今天的 job 名。** 零改造,但路径外的 PR 不触发被过滤的 workflow(#24 实测),required check 卡死在 "Expected — waiting for status";且过滤一旦下沉到 job 级,required 集合就必须枚举每个条件 job,未来新增 job 而未同步保护设置便悄然脱管——conclusion job 存在的意义就是把这张清单收敛为每个 workflow 一个名字。

**单一伞形 gate workflow。** 跨 workflow 的 `needs` 不存在,单文件意味着把所有 job 定义复制进一个 workflow,抛弃与 prek 结构同构的分领域文件和 main 推送的路径算力经济——为每个 workflow 两个小 job 就能解决的问题重写整个 CI。

**用 merge queue 取代 required checks。** merge queue 以 required checks 为前提,把合并串行化进排队组并跑全量 CI;它是叠加在本决策之上的新鲜度机制,不是替代品——还替单维护者添了看 queue 的负担。

**只靠约定(什么都不做)。** 实测已失败:#23 带红合并且无平台信号;而闭环的停止规则约束的是 agent,不是发版途中赶时间的人类。

## Acceptance criteria

纯文档 PR 上,五个 conclusion 检查全部运行且为绿,所有路径门控的真实 job 为 skipped。全量 PR 上,任一矩阵 job 失败恰好使其所在 workflow 的 conclusion 变红。required checks 生效后,任一 conclusion 为红的 PR 无法合并(设置生效前先用临时分支验证)。`push: main` 行为不变:过滤与跳过与今天一致。开发闭环 MRFC 的分支保护步骤补上指向本记录的链接。

## Risks

每个 PR 都会启动全部五个 workflow,路径外 PR 各多付一个 `changes` 和一个 `conclusion` 微 job——有界:两者都不装工具链,路径内行为不变。conclusion 的 `needs` 清单是显式的;GitHub Actions 没有"全部 job"通配符,未来新增 job 而未扩展 `needs` 会脱离该 workflow 的门控——由验收测试与评审惯例承担该职责。required checks 需要工作流落地后的维护者设置步骤;在设置完成前 conclusion 仅为参考,门禁 3 仍只靠约定,与今天相同。过滤从 workflow 级移到 job 级也会改变运行页的样子:路径外 PR 会列出被跳过的真实 job 而非整段无运行——行数变多,结论不变。
