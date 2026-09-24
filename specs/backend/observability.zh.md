# 可观测性

[English](observability.md) | 中文

可观测性三支柱（Logs / Traces / Metrics）规范。日志（Logs）作为可观测性的一部分，与 traces、metrics 统一在本文件描述。

## 技术栈与硬约束

### 硬约束

**遥测经 OTLP 发往外部部署的观测栈；本地文件是回退模式与崩溃通道。** 设置 `OTEL_EXPORTER_OTLP_ENDPOINT` 后，三支柱全部以 OTLP/HTTP + gzip 导出至 collector（bearer token 认证）。未设置时进程回退到仅文件管道（stdout 导出器 → JSONL）——压测/容量栈仍使用该模式，`scripts/loadtest/capacity/analyze.py` 因此零改动。观测栈本身、其传输与暴露架构由 [OTLP 观测栈 MRFC](../../.agents/mrfcs/proposed/2026-09-19-otlp-observability-stack.zh.md) 与 [frp 传输 MRFC](../../.agents/mrfcs/proposed/2026-09-19-otlp-transport-frp-exposure.zh.md) 承载；本规范只描述 markpost 的生产方一侧。

### 路线 A：slog + 手写 trace Handler

日志支柱使用 `log/slog`（Go 标准库）加一个手写 slog Handler，把 trace_id 从 ctx 注入每条日志。OTLP 模式下默认 logger 是扇出：timberjack 文件 handler **加上** `otelslog` bridge 进 OTLP 日志管道（bridge 从 ctx 提取 trace 上下文的行为与文件 handler 完全一致）。**OTel Logs SDK 不被直接使用**——只经 bridge 间接使用。

**不使用 `slog-otel`**（维护不活跃）——trace↔log 关联由手写 Handler 实现。该决策记录保留。

### 三支柱

| 支柱        | 采集                                                          | OTLP 模式                                          | 文件模式（回退）                   |
| ----------- | ------------------------------------------------------------- | -------------------------------------------------- | ---------------------------------- |
| **Logs**    | `log/slog`，手写 Handler 从 ctx 注入 trace_id/span_id         | 扇出：timberjack 文件 + `otelslog` → OTLP logs     | timberjack → `app-*.jsonl`         |
| **Traces**  | OTel Go SDK + `otelgin.Middleware`（自动 HTTP span）          | `otlptracehttp` → collector                        | `stdouttrace` → `traces-*.jsonl`   |
| **Metrics** | OTel Go metric SDK（counter/gauge/histogram）+ 自动运行时采集 | `otlpmetrichttp` → collector（60s PeriodicReader） | `stdoutmetric` → `metrics-*.jsonl` |

导出器选择完全由环境变量驱动（`OTEL_EXPORTER_OTLP_ENDPOINT`、携带 bearer token 的 `OTEL_EXPORTER_OTLP_HEADERS`、`OTEL_EXPORTER_OTLP_COMPRESSION`、`OTEL_SERVICE_NAME`）；两种模式下埋点完全一致。

## 文件布局与轮转

OTLP 模式下只写应用日志文件（崩溃通道：collector/隧道中断时现场仍在）。traces 与 metrics 文件仅存在于文件模式。

```
/app/data/logs/
├── app-2026-07-14.jsonl          business events + HTTP access + errors (slog)
├── app-2026-07-14T00-00-00.000-time.jsonl.zst   midnight rotation archive
├── traces-2026-07-14.jsonl       OTel spans (file mode only)
└── metrics-2026-07-14.jsonl      OTel metric data points (file mode only)
```

### timberjack 轮转配置（混合策略，所有文件共用）

| 设置               | 值                          | 用途                                      |
| ------------------ | --------------------------- | ----------------------------------------- |
| `RotateAt`         | `["00:00"]`                 | 每日午夜轮转（主策略）                    |
| `MaxSize`          | 100 MB                      | 事故日午中的兜底切割（保证单文件有界）    |
| `MaxBackups`       | 14                          | 保留 14 个旧文件（约两周）                |
| `MaxAge`           | 30                          | 30 天后删除（MaxBackups/MaxAge 取更严者） |
| `Compression`      | `"zstd"`                    | 旧文件 zstd 压缩                          |
| `BackupTimeFormat` | `"2006-01-02T15-04-05.000"` | 毫秒精度；避免第二次按大小切割时重名      |

## 日志（slog）

### 日志级别约定

- **Error**：意外错误、panic、非 service.Error 的边界错误、未知错误码
- **Warn**：可恢复异常（限流、降级、重试）
- **Info**：生命周期事件（启动/停机/配置加载）、关键业务事件（文章创建、登录、投递派发）
- **Debug**：开发期细节，生产默认关闭

### 记录时机

- **启动生命周期**：配置加载 / db 初始化 / server 启动 / 监听地址
- **意外边界错误**：`apierr.RespondError` 遇到非 service.Error 或未知错误码时，**以 `slog.Error` 加 trace 字段记录**（不用 `log.Printf`）
- **panic 恢复**：fallback 中间件恢复后以 `slog.Error` 记录（带 trace_id、path、error）
- **关键业务事件**：文章创建、登录、投递派发等，带结构化字段（user_id、post_id、session_id……）

**服务层错误不逐条记录**——在它们浮出的边界（handler / apierr）记录。

**所有请求作用域的日志调用必须用 `*Context` 形式**（`slog.InfoContext(ctx, ...)`、`s.logger().InfoContext(ctx, ...)`）；不带 ctx 的调用会同时破坏两个通道的 trace 关联，属于缺陷。

### 永不记录的敏感数据

- 密码（明文或哈希）
- JWT token（access 或 refresh）
- OAuth client secret
- post key 值（生产日志中）
- 完整请求体（可能含用户内容）

### Fatal 日志

**Fatal 统一为 `slog.Error` + `os.Exit(1)`；不使用 `log.Fatalf`。** 理由：fatal 记录落入带 trace 字段的结构化日志（app.jsonl）。

Fatal 保留给不可恢复的启动错误（进程无法继续）：

- 配置文件加载失败
- 数据库连接失败
- 管理员用户初始化失败
- 信任代理配置失败
- 端口绑定失败

### trace↔log 关联

手写 slog Handler 从 ctx 提取 span 上下文注入每条记录；OTLP 模式下 `otelslog` bridge 对导出副本做同样的事。VictoriaLogs 将 `trace_id` 作为一等索引字段，日志行里的 `trace_id` 可直接反查 Jaeger 调用链。

```go
func (h *traceHandler) Handle(ctx context.Context, r slog.Record) slog.Record {
    spanCtx := trace.SpanContextFromContext(ctx)
    if spanCtx.IsValid() {
        r.AddAttrs(
            slog.String("trace_id", spanCtx.TraceID().String()),
            slog.String("span_id", spanCtx.SpanID().String()),
        )
    }
    return r
}
```

## 链路（OTel）

### 自动 span（otelgin 中间件）

`otelgin.Middleware(serviceName)` 注册为中间件，为每个 HTTP 请求自动创建 span，记录 HTTP 方法、路由、状态码与延迟。**这些自动 HTTP span 是 markpost 当前唯一的 span 来源。**

### 手写子 span（已预留、未实现）

数据库事务、渲染与投递循环的子 span（`post.Create`、`post.RenderHTML`、`delivery.Schedule`、`auth.GitHubCallback`）已设计但**未实现**——代码库中不存在任何 `tracer.Start` 调用。在补齐之前，每条 trace 恰好只有一个 server span。补齐时：`tracer.Start(ctx, "operation.name")`，经 ctx 继承父级，出错时 `span.SetStatus(codes.Error, msg); span.RecordError(err)`。

### 采样策略

`ParentBased(AlwaysOn)`——默认全采样。

理由：单服务、无跨服务传播，量级可控。QPS 增长后切 `ParentBased(TraceIDRatioBased(0.1))` 只需一行（配置槽已预留）；再往上走 collector 侧 tail sampling。

## 指标（OTel）

### Reader

`PeriodicReader(exporter, metric.WithInterval(60*time.Second))`——每 60 秒冲刷至 collector（OTLP 模式）或指标文件（文件模式）。

### 命名风格

OTel 语义约定（semconv），点分风格如 `http.server.request.duration`——**不是**下划线风格（`http_request_duration_seconds`）。存储端摄取时可能做净化（VictoriaMetrics 保留点分形式）。

### 指标清单

当前已采用的指标，按需扩展：

| 层   | 指标                                 | 类型      | 标签                  | 用途                                               |
| ---- | ------------------------------------ | --------- | --------------------- | -------------------------------------------------- |
| HTTP | `http.server.request.duration`       | histogram | method, route, status | 端点级性能（otelgin 自动）                         |
| HTTP | `http.server.active_requests`        | gauge     | —                     | 在途请求数                                         |
| 业务 | `markpost.posts.created_total`       | counter   | —                     | 文章创建数                                         |
| 业务 | `markpost.auth.login_success_total`  | counter   | —                     | 登录成功数                                         |
| 业务 | `markpost.auth.login_failure_total`  | counter   | —                     | 登录失败数                                         |
| 业务 | `markpost.auth.token_refresh_total`  | counter   | —                     | token 刷新数                                       |
| 业务 | `markpost.delivery.pending`          | gauge     | —                     | 待派发数                                           |
| 业务 | `markpost.delivery.dispatched_total` | counter   | —                     | 已派发数                                           |
| 业务 | `markpost.delivery.failed_total`     | counter   | error_category        | 派发失败数（按原因）                               |
| 业务 | `markpost.render_cache.hit_total`    | counter   | —                     | 渲染缓存命中数                                     |
| 业务 | `markpost.render_cache.miss_total`   | counter   | —                     | 渲染缓存未命中、进入 singleflight 的请求数         |
| 业务 | `markpost.cdn.purge_success_total`   | counter   | —                     | CDN cache-tag 清除完成数（HTTP < 300）             |
| 业务 | `markpost.cdn.purge_failure_total`   | counter   | —                     | CDN 清除尝试失败数（marshal/构建/传输/HTTP ≥ 300） |
| 业务 | `markpost.cdn.purge_skipped_total`   | counter   | —                     | 未尝试的 CDN 清除（no-op purger/未配置）           |
| 系统 | 运行时指标                           | —         | —                     | OTel Go 运行时自动采集（goroutines、GC、内存）     |

五个渲染缓存/CDN 清除计数器无属性——每种结局一个序列，命中率与清除尝试可聚合导出（决策记录：[缓存/清除可观测 MRFC](../../.agents/mrfcs/implemented/2026-09-03-cache-purge-observability.zh.md)；对照 `CF-Cache-Status` 阅读：[`caching.md`](./caching.zh.md)）。

计数器在首次递增前不产生数据点——仪表盘与告警规则不得假设首个业务事件前序列已存在。

### 日志关联字段

每条业务日志自动携带 `trace_id` 与 `span_id`，适用时附业务字段（`user_id`、`post_id` 等）。

## 初始化接线（cmd/server/main.go）

启动时依次：

1. **创建三个 timberjack Logger**（app / traces / metrics），带轮转配置
2. **构建管道**（`observability.Init`）：OTLP 模式从环境变量构建 `otlptracehttp` / `otlpmetrichttp` / `otlploghttp` 导出器加共享 resource（`service.name`）；文件模式照旧构建 stdout 导出器
3. **接线上 providers**：tracer/meter providers → `otel.SetTracerProvider` / `SetMeterProvider`
4. **注册 otelgin 中间件**：`r.Use(otelgin.Middleware("markpost"))`
5. **安装默认 slog logger**（`providers.InstallSlogDefault`）：带 trace 关联的文件 handler，OTLP 模式下扇出到 OTLP 日志管道
6. **优雅停机**：`Shutdown(ctx)` 冲刷 trace/metric/log 导出器 + `Close()` 三个 timberjack logger

## 遥测的消费

查询接口是各存储的 HTTP JSON API（也是 AI agent 的消费面），人类用 Grafana：

```bash
# metrics — Prometheus querying API on VictoriaMetrics
curl -s 'http://<vm>:8428/api/v1/query?query=markpost.posts.created_total'
# logs — LogsQL on VictoriaLogs (trace_id is an indexed field)
curl -s 'http://<vl>:9428/select/logsql/query?query={service.name="markpost"}'
# traces — Jaeger REST API
curl -s 'http://<jaeger>:16686/api/traces?service=markpost&limit=1'
```

文件格式（文件模式）保持 JSONL，每行一个 JSON 对象，压测复盘时仍可用 `jq` 分析。
