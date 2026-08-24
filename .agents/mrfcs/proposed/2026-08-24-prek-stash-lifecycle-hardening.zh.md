# MRFC: prek stash-lifecycle incident hardening

Status: proposed

[English](2026-08-24-prek-stash-lifecycle-hardening.md) | 中文

## Problem

prek 0.4.14 的 `WorkTreeKeeper`(`crates/prek/src/cli/run/keeper.rs`)把未暂存改动写入 `~/.cache/prek/patches/<ms>-<pid>.patch`,用 `git checkout --` 从工作树抹掉,待进程 Drop 时重新 apply。2026-08-22/23 事故(#16)实测到三个后果:

clean 与 restore 之间的中断——进程被杀、harness 终止;Ctrl-C 清理路径不覆盖这些——让工作树静默缺失未暂存工作,它们只存在于 patch 文件里(8-22 23:49:16 文件部署状态仅存活于一枚 64.5KB patch,树上只剩 2 文件子集)。

成功回放的 patch 永不删除、也不带"已消费"标记,缓存无界增长(7 月至 8 月 19MB),每枚陈旧 patch 都是貌似合理的恢复来源。

prek 自身绝不跨会话 apply 旧 patch(全源码唯一的 `git apply` 调用点只回放当前运行自己的 patch),恢复只能手工进行——而手工 apply 陈旧 patch 会把一天前的状态静默叠加到当前工作上,2026-08-23 16:07 实测:完整部署状态以一枚新 87KB stash 的形态重现。同一 keeper 窗口也解释了 #16 披露轨迹中的 worktree pre-commit 误报:钩子在被打扫过的树上运行。

## Proposal

防丢失:`scripts/prek_patch_audit.py`(stdlib、带单测)标记孤儿 patch——归属 pid 已死、龄超宽限窗——作为 dev-loop skill 的会话开场分诊步骤运行,先于任何工作叠上被静默打扫的树。

防复活:恢复走审计脚本的预览路径(`git apply --check` 加 diff 检视),绝不盲 `git apply` 缓存 patch;runbook 以约束行持有该规程。

保留期:`prek cache gc`——上游的陈旧 patch 保留期,30 天——加入同一分诊节律,agent 工作区另排每周执行;守卫落地后,"勿清缓存"警告退役。

上游:根因修复——删除或标记已消费 patch、对死 pid 的孤儿 patch 告警——属于 prek 本体;markpost 向上游提 issue(维护者动作),并在采用修复版本时重估仓库侧守卫。[prek MRFC](2026-08-12-prek-single-source-of-format-and-lint.zh.md) 继续持有工具决策;本记录持有事故加固。

## Alternatives considered

**只修上游,不加仓库侧守卫。** 长期正确的归属,但受制于 markpost 控制外的上游发布周期,而实测的事故类别在此期间保持敞开。

**禁用 stash 循环。** prek 0.4.14 无 `--no-stash` 或等价配置(源码证实);包装 `git` 伪造它比要消除的危害更重。

**提交时门禁替代会话分诊。** 提交钩子恰在 keeper 的打扫窗口内运行——挂在那里的审计者无法可靠看见它该审计的东西;自然的检查点是会话开场,先于工作叠上被打扫的树。

**只保留"勿清缓存"警告。** 8-22/23 时间线显示丢失与复活都已实际发生,而该警告只活在一个会话的记忆里。

## Acceptance criteria

审计脚本标记合成的孤儿 patch(死 pid、超宽限)且对活 pid 与新鲜 patch 保持沉默;其单测套件像 policy 测试一样接入 prek;dev-loop skill 带分诊步骤;runbook 增加生命周期行与 worktree 误报行;`prek cache gc` 记入分诊节律;worktree 隔离提案记录补上点燃证据并保持 `proposed`(处置:推迟到下次需要并行验证时)。

## Risks

pid 存活检查是启发式的(pid 复用两个方向都会误判),由龄阈值兜底;审计为每个会话增加一步快速 stdlib 检查,仅在有孤儿需要关注时非零退出;上游行为可能在守卫之下变化,采用时重估;worktree pre-commit 误报在上游修复前保持——其"手工复刻加 `--no-verify`"对策留在 runbook。
