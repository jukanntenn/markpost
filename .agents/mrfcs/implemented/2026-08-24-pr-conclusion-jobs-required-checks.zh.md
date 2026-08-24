# MRFC: PR conclusion jobs and required checks

Status: implemented

[English](2026-08-24-pr-conclusion-jobs-required-checks.md) | 中文

## Problem

门禁 3——交付批准——没有平台强制:`main` 的分支保护不要求任何检查,带红检查的 PR 可以合并。2026-08-23 实测:PR #23(`chore: release v0.2.0-rc.6`)在 `Issue policy` 失败的状态下合并,在无人盯守的时刻破了闭环的 never-merge-red 规则。

required checks 也无法直接指向当时的 job 名。五个质量 workflow——Lint、Test、Build、E2E、Docs——都在 workflow 级对 `pull_request` 做路径过滤:Lint/Test/Build/E2E 用 `paths-ignore`(`**/*.md`、`docs/**`、`.gitignore`、`LICENSE`),Docs 用正向 `paths` 清单。改动落在某个 workflow 路径集之外的 PR 根本不触发它,其检查会永远停在 "Expected — waiting for status" 并阻塞合并。本仓库实测:纯文档 PR #24 只跑了 Docs 与 Issue policy,Lint、Test、Build、E2E 均未启动。

[开发闭环 MRFC](2026-08-22-agent-driven-development-loop.zh.md) 把平台强制留给了维护者的设置步骤;本记录设计该步骤要开启的东西。

## Decision

五个质量 workflow 对每个目标为 `main` 的 `pull_request` 运行,不再做 workflow 级路径过滤;`push: main` 的过滤原样保留,main 推送的算力经济不受损。路径选择性位于 job 级:每个 workflow 以一个 `changes` job(dorny/paths-filter)门控真实 job——Lint 与 Docs 新增,Test、Build、E2E 原有——job 级条件在 push 与 dispatch 上的适用方式与 Test、Build、E2E 的既有实践一致。

每个 workflow 以一个 `conclusion` job 收尾:其 `needs` 列出其余全部 job,`if: always()` 使其在跳过与失败之后照常运行;单一步骤仅在任一 need 结果为 `failure` 或 `cancelled` 时失败——被跳过的 job 记为成功。每个 job 带显式名——五个裸 `conclusion` 在分支保护的 required 选择器里无法区分选中——由此每个 PR 恒定产出五个稳定检查:`Lint conclusion`、`Test conclusion`、`Build conclusion`、`E2E conclusion`、`Docs conclusion`。

`main` 的 required checks 即这五个 conclusion 检查,并在 release 豁免([issue-policy 豁免](2026-08-24-issue-policy-release-exemption.zh.md))就位后加上 `Issue policy`;执行设置是维护者的分支保护步骤——workflow 侧随本记录落地。

## Alternatives considered

**直接把 required 指向当时的 job 名。** 零改造,但路径外的 PR 不触发被过滤的 workflow(#24 实测),required check 卡死在 "Expected — waiting for status";且过滤一旦下沉到 job 级,required 集合就必须枚举每个条件 job,未来新增 job 而未同步保护设置便悄然脱管——conclusion job 存在的意义就是把这张清单收敛为每个 workflow 一个名字。

**单一伞形 gate workflow。** 跨 workflow 的 `needs` 不存在,单文件意味着把所有 job 定义复制进一个 workflow,抛弃与 prek 结构同构的分领域文件和 main 推送的路径算力经济——为每个 workflow 两个小 job 就能解决的问题重写整个 CI。

**用 merge queue 取代 required checks。** merge queue 以 required checks 为前提,把合并串行化进排队组并跑全量 CI;它是叠加在本决策之上的新鲜度机制,不是替代品——还替单维护者添了看 queue 的负担。

**只靠约定(什么都不做)。** 实测已失败:#23 带红合并且无平台信号;而闭环的停止规则约束的是 agent,不是发版途中赶时间的人类。

## Consequences

每个 PR 无论改了什么路径都会得到每个 workflow 一行绿/红结论,required checks 可以指向五个永不停挂在 "Expected" 的稳定名字。路径外 PR 每个 workflow 多付两个微 job(`changes` 与 `conclusion`;均不装工具链);运行页会列出被跳过的真实 job 而非整段无运行——行数变多,结论不变。conclusion 的 `needs` 清单是显式的——GitHub Actions 没有"全部 job"通配符——未来新增 job 而未扩展 `needs` 会脱离该 workflow 的门控,由评审承担该职责。在维护者应用分支保护设置之前,conclusion 仅为参考,门禁 3 仍与从前一样只靠约定;设置步骤以一条红 conclusion 的临时分支必须无法合并来验证后方可信任。在落地栈自身的 PR 上——首批运行此形态的 PR——一次文档加 workflow 的改动使五个 workflow 全部运行而真实 job 选择性跳过,这就是既存的验收证据。
