# Agent-loop runbook

English | [中文](agent-loop-runbook.md)

以共享机器账号在其他仓库复用 agent 驱动开发闭环时的一次性激活清单与实测平台约束。设计理由见[闭环记录](../.agents/mrfcs/implemented/2026-08-22-agent-driven-development-loop.zh.md)；本页持有操作流程。

## Activation checklist

1. 创建机器账号并邀为仓库 collaborator，生成带以下四个 scope 的 classic PAT，交给 agent 会话作 `GH_TOKEN`。
2. 建标签集：issue 用 `type/idea|feature|bug|research|task`，pull request 用 `area/<领域>`。
3. 落地仓库侧：五张 issue 模板与 `config.yml`（禁空白）、pull request 模板、带单测的 policy 脚本、两个 workflow、skills、根 `AGENTS.md` 章节——经一个 bootstrap pull request。
4. 建看板：个人 GitHub Project，自定义单选 `Loop status` 字段（`Inbox / Backlog / Ready / In progress / In review / Done / No action`）、`Priority` 字段（`P0–P3`），并把机器账号加为项目的 `WRITER` 协作者。
5. 把可写看板的 PAT 存为仓库 secret `MARKPOST_PROJECT_TOKEN`，并把闭环的 `config.json` 指向该项目、置 `requireProject: true`。
6. 在 bootstrap pull request 合并之后（绝不提前）把仓库设为仅 merge commit，并开分支保护：一个 approving review、作废过期批准、要求线程解决、包含管理员、禁止向 `main` force push，并把六个状态检查设为 required——五个 `X conclusion` 检查加 `Issue policy`（[PR conclusion jobs and required checks](../.agents/mrfcs/implemented/2026-08-24-pr-conclusion-jobs-required-checks.zh.md)）。
7. 宿主机装栈工具：`gh extension install github/gh-stack`。

## Measured platform constraints

下表每一行都在本仓库的激活过程中撞上并验证过；每行都值一轮调试周期，这张表就是为省下它们而存在。

| 约束                                                        | 证据                                                                                                                                               | 规避                                                                                                  |
| ----------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `GITHUB_TOKEN` 没有面向用户级 Projects v2 的 scope          | actionlint 的权限清单里不存在此类授权；直接变更失败                                                                                                | 改写看板的 workflow 以 `MARKPOST_PROJECT_TOKEN` secret（classic PAT）认证                             |
| PAT scope 不授予资源可见性                                  | scope 齐全的 token 仍对私有项目得到 `NOT_FOUND`                                                                                                    | 把机器账号加为项目 `WRITER` 协作者（`updateProjectV2Collaborators`，入参 `userId` 加 `role: WRITER`） |
| 项目默认 `Status` 字段不可删除、不可经 API 编辑             | `deleteProjectV2Field` 拒绝非自定义字段；不存在编辑选项的 mutation                                                                                 | 工作流状态放自定义 `Loop status` 字段；视图中隐藏默认字段                                             |
| fine-grained PAT 选不了协作者仓库                           | 仓库选择器只列 token 持有者的仓库                                                                                                                  | 用 classic PAT                                                                                        |
| gh CLI 的内部查询需要 `read:org`                            | 缺该 scope 时 `gh pr edit` 的 GraphQL 前置查询失败                                                                                                 | PAT 带上 `read:org`                                                                                   |
| HTTPS 推送触及 `.github/workflows/**` 需要 `workflow` scope | 远端以缺 scope 为由拒绝推送                                                                                                                        | PAT 带上 `workflow`；SSH 推送不受 scope 限制                                                          |
| issue 评论的创建与更新端点不同                              | 对 `issues/{n}/comments/{id}` 发 `PATCH` 返回 404                                                                                                  | 创建走 `issues/{n}/comments`；更新走扁平的 `issues/comments/{id}`                                     |
| 无人可 approve 自己的 pull request                          | 唯一维护者在 required-approval 加 include-administrators 下被自己署名的 PR 锁死                                                                    | agent 的 PR 以机器账号署名；分支保护只在 bootstrap 合并之后开启                                       |
| bootstrap 期间策略脚本不在 `main` 上                        | 检出默认分支时找不到 `policy.py`                                                                                                                   | workflow 检出事件提交（`github.sha`）而非默认分支                                                     |
| prek 在一个自洽的暂存快照上运行                             | 未暂存的 `prek.toml` 修改中止提交；暂存了重命名却未暂存重写内容会打挂文档门禁                                                                      | 配置与重写内容一起暂存；提交排序保证每个暂存集自洽                                                    |
| workflow 的 `pull_request` action 清单决定策略代码可达性    | lifecycle workflow 只注册了 `[opened, edited, synchronize, reopened]`，`policy.py` 的 review-requested 分支沦为死代码；请求评审从不搬动看板（#15） | `on.pull_request.types` 与 `resolving_command` 处理的 action 保持同步；单测钉住配对                   |
| `gh stack merge` 对多层栈误判批准状态                       | 两次全 APPROVED 的栈被原子合并以 "At least 1 approving review is required" 拒绝（2026-08-23/24）                                                   | 自底向上边界合并：先落底层，等上层重定基到 `main` 且 checks 转绿，再逐层合并                          |
| 落掉下层会作废上层批准                                      | 落 #29 后 #30 被重定基重放，评审显示 `DISMISSED`，合并被阻直至重批                                                                                 | 每落一层，预期余下各层各需一次机械性重批                                                              |
| 栈顶被人工开启 auto-merge 会卡死在 "merging"                | 队列合并从未完成                                                                                                                                   | 先走 GraphQL `disablePullRequestAutoMerge`（不是 `disableAutoMerge`）再正常合并                       |
| 增量文档门禁看不见移动文件断掉的链接                        | 把 MRFC 对移入 `implemented/` 后兄弟文件的同目录链接悬空；本地增量 `doc_sync` 通过而 CI 全量失败（PR #29 首跑）                                    | 提交移动或重命名 Markdown 时本地跑全量 `doc_sync`                                                     |

## Verifying each step

先确认 token 身份与触达（`GH_TOKEN=<pat> gh api user` 输出机器账号名；`gh api repos/<owner>/<repo>` 显示 `push`），再用真实事件验证看板链路：resolving issue 上的任意 pull request 活动都会搬动看板并留下标记评论（`<!-- markpost-lifecycle: <status> by <account>]`），受审的 pull request 会让 Issue policy check 依据真实内容给出绿或红。闭环的首次生产快速通道——issue、机器署名 PR、人类批准、agent 合并、issue 自动关闭、看板 `Done`——即端到端验收。
| prek 的 stash 循环可能静默持有未暂存工作 | keeper 把改动写入 `~/.cache/prek/patches/<ms>-<pid>.patch`、把树 checkout 干净、Drop 时回放;被杀的运行把工作留在树上之外(2026-08-22/23,#16) | 会话开场跑 `python3 scripts/prek_patch_audit.py`;经其预览命令恢复,绝不盲 `git apply`;同一节律跑 `prek cache gc` |
| pre-commit 在 worktree 内误报 | `backend-generate-check` 在 `.local/worktrees/*` 下判根 `docs/*.md` 已删除,同命令手工执行 exit 0(#16 轨迹) | 手工复刻钩子命令,通过后 `--no-verify` 提交,并在 PR 正文披露两者 |
