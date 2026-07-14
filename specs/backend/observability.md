# Observability

可观测性的三支柱（Logs / Traces / Metrics）规范。日志（Logs）作为可观测性的一部分，与 traces、metrics 统一在此文档描述。

## 技术栈与硬约束

### 硬约束

**三支柱全部 export 到本地文件系统，不引入外部服务**（no Jaeger / Prometheus / Loki / OTLP collector）。所有可观测性产物（日志/spans/metrics）落盘为 JSONL 文件，`jq` 分析。

### 路线 A：slog + 自写 trace Handler

日志这条支柱采用 `log/slog`（Go 标准库）+ 自写 slog Handler 从 ctx 提取 trace_id 注入每条日志，**不采用 OTel Logs SDK**。

**明确不采用 `slog-otel`**（维护不活跃）—— 改为自写 ~20 行 slog Handler 实现 trace↔log 关联。保留此决策记录。

### 三支柱落盘

| 支柱 | 采集 | 落盘 |
|---|---|---|
| **Logs** | `log/slog`，自写 Handler 从 ctx 提取 trace_id/span_id 注入每条日志 | timberjack → `app-*.jsonl` |
| **Traces** | OTel Go SDK + `otelgin.Middleware`（自动 HTTP span）+ 业务手动子 span | `stdouttrace.New(WithWriter(timberjack))` → `traces-*.jsonl` |
| **Metrics** | OTel Go metric SDK（counter/gauge/histogram）+ runtime 自动采集 | `stdoutmetric.New(WithWriter(timberjack))` → `metrics-*.jsonl` |

### 技术可行性依据

- timberjack 的 `Logger` 实现 `io.Writer`（`timberjack.go` 的 `Write(p []byte)` 方法），可直接作为日志和 exporter 的输出 sink。
- OTel 三个 stdout exporter 都提供 `WithWriter(io.Writer)` 选项（`stdouttrace/config.go`、`stdoutmetric/config.go`、`stdoutlog/config.go`），所以 timberjack 实例可直接喂给 exporter，实现 trace/metric 各自独立滚动落盘。

## 文件布局与滚动

### 三文件模型

```
/var/log/markpost/
├── app-2026-07-14.jsonl          业务事件 + HTTP 访问 + 错误（slog）
├── app-2026-07-14T00-00-00.000-time.jsonl.zst   零点滚动归档
├── traces-2026-07-14.jsonl       OTel span
├── metrics-2026-07-14.jsonl      OTel metric 数据点
└── ...
```

三文件通过 `trace_id` 串联：app 发现异常 → traces 看调用链 → metrics 看当时指标。

### timberjack 滚动配置（混合策略，三文件共用）

| 配置项 | 值 | 说明 |
|---|---|---|
| `RotateAt` | `["00:00"]` | 每天零点滚动（为主） |
| `MaxSize` | 100 MB | 故障日中途切兜底（单文件不过大） |
| `MaxBackups` | 14 | 保留 14 个旧文件（约两周） |
| `MaxAge` | 30 | 30 天前的删除（与 MaxBackups 取严） |
| `Compression` | `"zstd"` | 旧文件 zstd 压缩 |
| `BackupTimeFormat` | `"2006-01-02T15-04-05.000"` | 毫秒格式，避免 size 二次切时重名 |

**混合策略说明**：以每天零点切为主，但若某天日志暴增（故障风暴），100MB 会中途切一次，那一天有 2 个文件。`BackupTimeFormat` 用毫秒格式确保 size 二次切时不重名。如果看重"严格一天一文件"，可去掉 MaxSize 改纯日期切，但失去单文件大小控制。

## Logs（slog）

### 日志级别规范

- **Error**：意外错误、panic、非 service.Error 的边界错误、未知错误码
- **Warn**：可恢复异常（限流、降级、重试）
- **Info**：生命周期事件（启动/关闭/配置加载）、关键业务事件（post 创建、登录、delivery 派发）
- **Debug**：开发期细节，生产默认关闭

### 何时记日志

- **启动生命周期**：config loaded / db init / server start / listening address
- **边界意外错误**：`apierr.RespondError` 遇到非 service.Error 或未知错误码时，**用 `slog.Error` 带 trace 字段**（不用 `log.Printf`）
- **panic recovery**：fallback middleware recover 后 `slog.Error` 记录（带 trace_id、path、error）
- **关键业务事件**：post 创建、登录、delivery 派发等，带结构化字段（user_id、post_id、session_id 等）

**service 层 error 不逐个记日志** —— 上抛到边界（handler/apierr）才记。

### 绝不记录的敏感数据

- 密码（明文或哈希）
- JWT token（access 或 refresh）
- OAuth client secret
- Post key 值（生产日志中）
- 完整请求体（可能含用户内容）

### fatal 日志

**统一用 `slog.Error` + `os.Exit(1)`，废弃 `log.Fatalf`**。理由：保证 fatal 也进结构化日志（app.jsonl）、带 trace 等字段。

仅启动期不可恢复错误使用 fatal（进程无法继续）：
- Config 文件加载失败
- 数据库连接失败
- Admin 用户初始化失败
- Trusted proxy 配置失败
- Server bind 失败

### trace↔log 关联（自写 slog Handler）

自写约 20 行 slog Handler，从 `ctx` 提取 trace 信息注入每条日志：

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

API：`trace.SpanFromContext(ctx).SpanContext()` → `.TraceID()` / `.SpanID()`（来自 `go.opentelemetry.io/otel/trace`）。

## Traces（OTel）

### 自动建 span（otelgin 中间件）

`otelgin.Middleware(serviceName, opts...)` 注册为中间件，每个 HTTP 请求自动建 span：

```go
r.Use(otelgin.Middleware("markpost"))
```

自动记录：HTTP method、path、status code、latency。

### 手动子 span

业务关键操作用 `tracer.Start(ctx, "operation.name")` 建子 span：

| 操作 | span name |
|---|---|
| DB 写事务（创建 post、delivery 派发） | `post.Create`、`delivery.Dispatch` |
| Markdown 渲染 | `post.RenderHTML` |
| delivery 调度循环 | `delivery.Schedule` |
| 外部调用（OAuth 回调 GitHub） | `auth.GitHubCallback` |

子 span 通过 `trace.SpanFromContext(ctx)` 继承父 span 的 trace_id，形成调用链。**错误时在 span 上记录 error 属性**：`span.SetStatus(codes.Error, msg); span.RecordError(err)`。

### 采样策略

`ParentBased(AlwaysOn)` —— 默认全采。

理由：单服务、不涉及跨服务传播，traces 文件量可控。后续若 QPS 增长，改为 `ParentBased(TraceIDRatioBased(0.1))` 即可（预留配置项）。

## Metrics（OTel）

### Reader

`PeriodicReader(stdoutmetricExporter, metric.WithInterval(60*time.Second))` —— 每 60 秒 export 一次到 metrics 文件。

### 命名风格

遵循 OTel 语义约定 semconv（点号分隔，如 `http.server.request.duration`），**非**下划线风格（`http_request_duration_seconds`）。

### 指标清单

暂采纳以下指标，后续按需扩展：

| 层 | 指标 | 类型 | 标签 | 说明 |
|---|---|---|---|---|
| HTTP | `http.server.request.duration` | histogram | method, path, status | 接口级性能（otelgin 自动 + 补充） |
| HTTP | `http.server.active_requests` | gauge | — | 当前在途请求数 |
| 业务 | `markpost.posts.created_total` | counter | — | 投稿创建数 |
| 业务 | `markpost.auth.login_total` | counter | result=success/failure | 登录成功/失败分布 |
| 业务 | `markpost.auth.token_refresh_total` | counter | — | token 刷新次数 |
| 业务 | `markpost.delivery.pending` | gauge | — | 待派发数 |
| 业务 | `markpost.delivery.dispatched_total` | counter | — | 已派发数 |
| 业务 | `markpost.delivery.failed_total` | counter | reason | 派发失败数（按原因） |
| 系统 | runtime metrics | — | — | OTel Go runtime 自动采集（goroutine 数、GC、mem） |

### 日志关联字段

每条业务日志自动带 `trace_id`、`span_id`，以及业务相关字段（`user_id`、`post_id` 等如适用）。

## 初始化装配（cmd/server/main.go）

启动时按顺序装配：

1. **创建三个 timberjack Logger**（app/traces/metrics），配置滚动参数
2. **构造 exporter**：
   - `stdouttrace.New(stdouttrace.WithWriter(appTracesLogger))`
   - `stdoutmetric.New(stdoutmetric.WithWriter(appMetricsLogger))`
3. **装配 Provider**：
   - `sdktrace.NewTracerProvider/sdktrace.WithBatcher(traceExporter)` → `otel.SetTracerProvider`
   - `sdkmetric.NewMeterProvider/sdkmetric.WithReader(metric.NewPeriodicReader(metricExporter))` → `otel.SetMeterProvider`
4. **注册 otelgin 中间件**：`r.Use(otelgin.Middleware("markpost"))`
5. **装配自写 slog Handler**（注入 trace_id），`slog.SetDefault`
6. **优雅关闭**：`Shutdown(ctx)` flush exporter + `Close()` 三个 timberjack

## 输出格式

三文件均为 JSON Lines（JSONL），每行一个 JSON 对象，`jq` 可分析：

```bash
# 按 trace_id 串联三文件
jq 'select(.trace_id=="a1b2c3d4...")' /var/log/markpost/app-*.jsonl
jq 'select(.trace_id=="a1b2c3d4...")' /var/log/markpost/traces-*.jsonl
jq 'select(.trace_id=="a1b2c3d4...")' /var/log/markpost/metrics-*.jsonl
```

stdout exporter 默认输出 JSON，metrics 的 stdoutmetric 输出较冗长（每数据点一行），文件会比 traces 大。这是可接受的默认格式。
