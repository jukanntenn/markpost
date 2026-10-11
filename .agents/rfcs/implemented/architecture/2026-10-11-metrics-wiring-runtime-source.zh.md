# RFC: 指标接线补全、contrib 运行时指标源与生成的指标清单

Status: implemented

[English](2026-10-11-metrics-wiring-runtime-source.md) | 中文

## Problem

指标面存在"已声明但未接线"的空洞与一段手写遗留：

- `markpost.auth.login_success_total` / `login_failure_total` / `token_refresh_total` 在 `internal/observability/metrics.go` 中已声明，但没有任何代码递增过它们——auth 服务根本没有 metrics 端口。
- Go 运行时指标来自一个手写块（`registerRuntimeMetrics`）：四个由 `runtime.ReadMemStats` 回 Callback 采样的 `process.runtime.go.*` gauge。ReadMemStats 是 OTel 项目刻意规避的近乎 stop-the-world 的调用；指标名不遵循任何约定；采样也错过了现代 `runtime/metrics` API 暴露的一切（内存上限、GC goal、处理器上限）。
- 投递时延不可观测：队列是"先入队再扫描"（ticker 认领 attempt 行），没有任何请求时长 span 覆盖入队→派发的间隙，而重试会把该间隙拉长到 36 分钟。
- 指标清单是 [observability.md](../../../../specs/backend/observability.zh.md) 里一张手工维护的表，没有任何机制把它与代码绑定——三个 dead 的 auth 计数器证明了它无法暴露漂移。
- `Dispatcher.Enqueue` 伪造 `context.Background()`，在入队读写中丢掉了创建请求的 trace 上下文——违反 ctx-form 规则的存量缺陷。

## Decision

以下内容随本记录一并落地；路径与名称随代码保持最新。

### 运行时指标源迁移

`go.opentelemetry.io/contrib/instrumentation/runtime` **v0.71.0**——其 go.mod 恰好要求 markpost 已固定的 `otel v1.46.0` 系列，核心依赖零变动——由 `observability.Init`（`internal/observability/otel.go` 的 `startRuntimeMetrics`）在 `otel.SetMeterProvider` 之后立即启动，OTLP 与文件两种模式皆然；文件模式压测/容量管道除指标集更丰富外不变。`Start` 在调用时从全局 provider 解析 meter，因此该调用必须排在 provider 安装之后。手写的 `registerRuntimeMetrics` 块与全部 `process.runtime.go.*` 指标删除。启动失败记录 `slog.Error` 后吞掉：`Init` 中导出器构建失败仍按既有约定致命，但运行时指标注册失败不应杀死应用——没有这些 gauge 进程照常服务。

任何地方都不设置 `OTEL_GO_X_DEPRECATED_RUNTIME_METRICS`，因此不产生已弃用的 `runtime.go.*` 系列，被删除的 `process.runtime.go.*` 名称就此消失。

**GC 可观测性取舍。** 新一代 `go.*` 系列直接暴露堆压力（带 `memory.type` 的 `go.memory.used`、`go.memory.limit`、`go.memory.allocated`、`go.memory.allocations`、`go.memory.gc.goal`），另有 `go.goroutine.count`、`go.processor.limit`、`go.config.gogc`。放弃的是 GC 频率与停顿的直接读数（`NumGC`、pause ns）：新约定不携带 GC 计数/停顿指标，上游迁移指引是用堆压力信号推断 GC 行为（goal 与 allocated 的对比）。接受：markpost 没有 GC 调优工作流，对请求服务型 Go 进程而言堆压力才是可行动的信号。未来真有需求时的升级路径**不是**那个弃用环境变量——而是刻意新增的、基于 `runtime/metrics`（`/gc/cycles:total`、`/sync/mutex/wait:seconds`）的手写 observable 指标。

### Auth 计数器

auth 服务获得标准 metrics 端口（本地 `Metrics` 接口 + `noopMetrics` + `WithMetrics` 选项，镜像 post 与 delivery 服务），由 `cmd/server/metrics_adapters.go` 的 `authMetrics` 适配、在 `main.go` 接线。接线点：

- `LoginWithEmail`：每个错误返回计一次失败——无效凭据**和**账号锁定返回（对调用方而言锁定拒绝就是一次失败登录，即使密码正确）；成功计一次成功。
- `LoginWithGitHub`：每个错误返回计一次失败，成功一次成功。早期的畸形请求拒绝（缺 state/code）也计入——"每个失败返回都计数"是不会随分支列表漂移的规则。
- `RefreshToken`：`token_refresh_total` 只计**完成的刷新**。被拒绝的重放、盗用检测、过期/无效 token 一概不计：那些是各有日志行的安全信号，不是刷新；该计数器度量的是轮换量。

递增位于三个公开入口，绝不在 `completeLogin` 内部，因此 token 刷新不会虚增登录计数器。所有调用使用所在流程的 ctx（ctx-form 规则）。

### `markpost.delivery.duration`

`Float64Histogram`，单位 **`s`**（OTel 语义约定的时长单位，也是 otelgin `http.server.request.duration` 使用的单位），在 `Dispatcher.execute` 开头——派发点——以 `now − Attempt.CreatedAt` 观测。`CreatedAt` 是创建 attempt 行时写入的**持久化入队时间戳**，因此测量可跨重启存活且无需内存 map。**每次派发尝试都观测，含重试**：被重试的投递合理地落在 6/16/36 分钟之外（退避序列为 1m/5m/10m/20m），而这一可见性——重试漂了多远、队列是否在越过其壁厚积压——正是该指标的意义。显式桶边界 `0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300, 600, 1200, 2400, 3600` 覆盖快路径（亚 tick 到数秒）到完整重试阶梯，顶格超过 40 分钟的过期壁；SDK 默认桶止于 10s，会把每次重试投递推进 +Inf。测试经 dispatcher 的 `now` 字段注入时钟。

### 生成的指标清单

[`docs/metrics.zh.md`](../../../../docs/metrics.zh.md)（及其英文对侧）由 `backend/tools/metricsdoc` 生成，经 `internal/observability/gen.go` 的 `//go:generate` 接入 `go generate ./...`。工具用 go/ast 解析 `internal/observability/metrics.go` 与 `internal/middleware/active_requests.go` 的指标声明（名称、类型、单位、描述、桶），解析 `cmd/server/metrics_adapters.go` 得到方法→指标映射与属性键，并用 golang.org/x/tools/go/packages 扫描 `internal/` 下各包的服务层调用点——产出名称、类型、单位、描述、属性、适配器方法、调用点 file:line 的表格。**零调用点的指标标记为 dead**——这正是生成文档存在的目的。otelgin 与 contrib-runtime 章节按**版本锚定**：工具从 go.mod 读取固定版本，遇到未列入已验证表的版本即拒绝运行，因此依赖升级未经再验证会让 `go generate` 失败，而不是把过期事实写进文档。输出确定性、零网络；可选的 `--live <vm-base-url>` 标志附加一列对照 VictoriaMetrics `/api/v1/metadata` 的存在性补充列，从不参与门禁。

漂移门禁落在既有的 `backend-generate-check` pre-commit 钩子（backend/prek.toml）：其 `go generate ./... && git diff --exit-code` 的范围现在也覆盖仓库根的 `docs/metrics.*` 文件，其 `files` 触发条件加入了 `go.mod`/`go.sum` 以便版本升级重跑它。只选一个家是有意为之：backend 钩子已经拥有 Go 工具链调用，生成器是 backend 模块内的工具；根级钩子（仿 ansible 模板检查）会为同一命令产出的文件复制那套机制。CI 运行 `uvx prek run --all-files`，漂移经此钩子使 CI 失败。该对登记在 `.hdsh/pairing.manifest.json` 的 `generated` 列表（英文源免除语言切换行以保持与生成器输出字节一致；中文对侧仍回链），工具自行以计算的 git blob 哈希写出 `.i18n.yaml` 记录——与 `hdsh pairing record` 的输出字节一致。

[observability.md](../../../../specs/backend/observability.zh.md) 的手工表替换为指向生成文档的指针（一个事实一个家）；spec 保留属于规则的内容——命名风格、无属性缓存/清除计数器、首个数据点告诫——其运行时行现在指向生成文档与本记录。

### Enqueue 的 ctx 贯通

`domainpost.DeliveryEnqueuer.Enqueue` 现在接收 `ctx`，`Dispatcher.Enqueue` 使用它而不再伪造 `context.Background()`；`post.CreatePost` 透传其请求 ctx。作业仍比请求长寿——只有入队时的仓储读写与 pending 计数器递增搭载请求的 trace 上下文。

## Alternatives considered

**保留手写运行时指标。** 零新依赖、名称完全可控——但 ReadMemStats 采样正是 contrib 包为取代而生的反模式，四个 gauge 只覆盖 `runtime/metrics` 的一角，且名称不遵循任何存储端认识的约定。

**设置 `OTEL_GO_X_DEPRECATED_RUNTIME_METRICS=true` 换取 `runtime.go.*` 系列。** 能拿回接近旧名的 GC 计数/停顿——但它重新引入迁移本要淘汰的弃用约定、翻倍导出序列，且上游明言其为过渡态。直接拒绝；未来需求走上文记录的受认可升级路径。

**用内存中的入队时间戳观测投递时长。** 不必依赖 `CreatedAt` 的语义——但测量跨重启即失（队列的全部意义就是活过重启）、需要带淘汰的 map，而持久化时间戳本来就在那里、本来就在驱动过期壁。

**保留手工维护的清单表。** 比生成器便宜——也已被证明无法暴露漂移：三个 dead 的 auth 计数器在一张"当前"表里躺了几周。AST 扫描约 600 行工具代码，换来把清单变成编译产物。

**基于 Prometheus client 的指标栈。** RED/USE 仪表盘的生态默认——但 markpost 已在 OTel SDK 上以 OTLP 与文件导出器双模运行；第二套指标栈会分叉管道、语义约定与压测工具链，换不来任何新能力。

## Consequences

auth 漏斗、投递时延分布与完整 `go.*` 运行时系列在两条管道中皆可查询，清单也无法再对"什么已接线"撒谎——一个 dead 指标只是一个可见的表格单元、一次门禁校验的重生成之遥。代价：两个新的直接依赖（`go.opentelemetry.io/contrib/instrumentation/runtime`，加上本就间接的 `golang.org/x/tools` 转直接）；多一个需要纳入提交纪律的生成物；引用已删除 `process.runtime.go.*` 名称的仪表盘会断（不存在这样的仪表盘——压测分析器不读它们）；GC 频率/停顿不再可直接读取，取舍已记录于上。Enqueue 的 ctx 贯通触及一个领域接口，`DeliveryEnqueuer` 的后续实现者继承 ctx 契约——那是规则，不是代价。
