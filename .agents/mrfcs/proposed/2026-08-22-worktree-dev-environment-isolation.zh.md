# MRFC: Worktree development-environment isolation

Status: proposed

English | [中文](2026-08-22-worktree-dev-environment-isolation.md)

## Problem

[开发闭环](../implemented/2026-08-22-agent-driven-development-loop.zh.md)把栈的每层一个 git worktree 放在 `.local/worktrees/` 下，但 compose dev 环境无条件命名其容器——`markpost-backend`、`markpost-frontend`、`markpost-postgres`——并绑定单一的共享 `markpost_pgdata` 卷。两个后果：主 checkout 与所有 worktree 之间同时只能跑一套 dev 环境（启动 worktree 的环境要求先停主 checkout 的），闭环 v1 把这个互斥当作已知限制接受了下来，串行化了一切需要运行中环境的验证。单元测试与 testcontainers 支撑的后端测试不受影响——它们不需要 compose 环境——因此这个约束咬住的恰是想要并行做全栈或前端对后端验证的地方。

## Proposal

以环境名参数化 dev 环境。`devops/dev.py` 接受 `--env <名称>`（默认为不带前缀的 `markpost`），据此设置 compose 项目名并为每个容器名与 postgres 数据卷加前缀——`wt-123-markpost-backend`、`wt-123_markpost_pgdata`——经 compose 文件已存在的插值变量传入。主 checkout 逐字节保持今天的不带前缀名称，既有的肌肉记忆、AGENTS.md 的 `docker exec markpost-postgres` 指令、任何按名寻址容器的脚本照常工作；worktree 会话传 `--env wt-<issue>` 即获得完全不相交的环境。宿主端口绑定也必须参数化（每环境一个偏移或显式映射），否则第二套环境的端口声明与第一套冲突；compose 文件获得插值，`dev.py` 获得该开关与一个把占用环境点名、响亮失败的冲突检查。`dev-loop` skill 在 worktree 内驱动验证时自动传以 issue 派生的环境名。

## Alternatives considered

**维持 v1 的串行用法。** 无需工作且闭环可用——但每次全栈验证都与其他一切串行，而定时驱动（phase 2）恰在无人值守停用主环境时放大争用。

**每个 worktree 一份 compose override 文件。** 生成的 `docker-compose.override.yml` 免改主文件，但基础文件与其覆盖之间的生成漂移成为独立的故障面，且覆盖仍要发明名称——同一种参数化换了一条更不可见的路线。

**单一共享环境，仅按库名隔离数据。** 最廉价的数据隔离；backend 与 frontend 容器本身仍是单例，两个 worktree 的代码变更依旧无法共存于一个运行中的环境。

## Acceptance criteria

两套 dev 环境——主 checkout 的不带前缀环境与 `--env wt-<n>` 的 worktree 环境——并发运行而无名称、卷或端口冲突，各自服务自己 checkout 的代码；主 checkout 的容器名、卷名与宿主端口与今天逐字节一致；不带 `--env` 的 `dev.py start` 行为与从前完全相同；端口被占用的环境请求以占用环境点名的错误响亮失败。

## Risks

compose 文件插值给一份被运维当配置阅读的文件增加了间接层；把每个默认值保持为今天的不带前缀名称框住了这项成本。端口参数化在 backend 与 frontend 端口接线不一致时存在漂移风险——冲突检查的存在就是把这种情况响亮地点出来而非离奇地失败。每套并发环境消耗其完整的容器与内存足迹；并行是按 `--env` 的主动选择，绝不是默认。并发运行的环境之间共享同一 postgres 镜像，因此按环境分卷必须被未来的任何迁移工具尊重。
