# 编码原则

[English](PRINCIPLES.md) | 中文

已被 [AGENTS.md § Conventions](AGENTS.md#conventions) 取代为规则的活的家；作为归档保留至迁移完成，不再更新。

面向 agent 的行为约束。每一条都是 agent 不被告知就会做错的规则。生产安全与数据完整性凌驾于这里所有原则之上 —— 包括从零重造的许可；与单纯的默认值或风格规则相抵时，原则获胜。

<a id="ground-every-conclusion-in-fact"></a>

## 每个结论都落到事实

库的事实、API 与协议必须先读源码或文档再依其行动 —— 训练数据是盲区，不是来源。每个结论都要在场验证：逻辑用 `file:line`，UI 用 Playwright，数据用 `docker exec markpost-postgres psql`，行为用对 dev 的只读 HTTP —— 取模型实际可执行的手段。纯算法或语法知识可以使用训练知识。

`docker compose run` 旗标事件是做错这件事的形状：ansible 的 `interactive=false` 会发出 `--no-interactive`，一个 docker compose 根本没有的旗标 —— 只因依赖前对照了 compose 文档（v5.1.3）才被发现。

<a id="defer-to-community-convention"></a>

## 遵循社区约定

当约定或最佳实践不确定时，先问"社区/官方约定是什么"，再对照权威开源源码验证，而不是依赖训练记忆（例如 `fmt`/`lint` 在这里是否是 prek 的组名、golangci-lint v2 要求怎样的配置 —— 两者都可对照工具自身的 schema 验证）。

与 _每个结论都落到事实_ 的分野：那条管的是你正在集成的库的事实；这条管的是约定与最佳实践决策。

<a id="converge-before-you-implement"></a>

## 实现前先收敛

spec 或计划必须自足、完整、无歧义 —— 一个没有品味的执行者也能机械落地，没有即兴发挥的余地。实现前解决每一个悬而未决的点；不要靠一份半熟的计划开工。

<a id="fix-the-root-cause-not-the-symptom"></a>

## 修根因，不修症状

你选择的解法必须最自然最优 —— 不是覆在症状上的补丁，也不被既有实现困住。当根因修复需要时，可以甩掉全部遗留、从零开始。

当 formatter/lint 的覆盖面在工具间泄漏时，修复不是给每个 AI 钩子单独配置，而是委托 `prek.toml` 作为唯一事实 —— prek 之外不存在平行的 formatter 或 lint 定义。

<a id="design-from-first-principles"></a>

## 从第一性原理设计

从业务本质推导设计；每个前提都可打破；优雅的方案胜过继承的方案。与 _修根因，不修症状_ 的分野：那条讲的是如何 _修_ 一个问题（根因，非补丁）；这条讲的是如何 _设计_ 一个系统（重推导、质疑前提）。彻底放弃 sqlite/mysql 而不是永远背着三个驱动，就是这条原则的应用。

<a id="single-source-of-truth"></a>

## 单一事实源

每一类信息恰有一个权威来源：schema 的事实是内嵌的 golang-migrate SQL 文件（与每次 GORM tag 变更配对），API 的事实是 `backend/docs/` 里生成的 Swagger（重新生成，绝不手编），UI 文案在 locale 文件里，部署配置在 ansible group_vars + 模板里。前端负责渲染；它不做决定。

<a id="naming-is-part-of-the-api"></a>

## 命名是 API 的一部分

名字就是 API 面。如果一个名字不合它的业务含义，不要硬用 —— 列举候选让用户选择，防止语义漂移。

<a id="degrade-gracefully-never-silently"></a>

## 优雅降级，绝不静默

失败必须被处理、被可观测地记录，且不得阻塞下游工作 —— 但静默失败永远是错的。一次失败的投递（webhook、飞书）不得阻塞帖子，但它必须落入投递历史，而不是消失；配置校验在启动时大声失败，而不是悄悄回退到没人知道的默认值。

<a id="minimal-mock-maximal-real"></a>

## 最小 mock，最大真实

只 mock 请求边界，绝不 mock 整个服务。后端测试经 testcontainers-go 跑在真实 PostgreSQL 容器上 —— CI 里的 repository mock 会藏住 SQL 漂移；e2e 经公开 URL 驱动真实容器镜像；本地与 CI 尽可能跑同一套件。
