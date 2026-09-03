# 规格文件

[English](index.md) | 中文

> 为当前任务参考最相关的规格文件。本目录的每个规格都列在此处 —— 新增规格文件意味着同一变更里加上它的行（`scripts/verify_specs_index.py` 负责闸门）。排序：跨切面规格在前，随后后端，最后前端；同一节内，一起阅读的内容保持相邻。规格描述当前状态；决策理由位于 [MRFCs](../.agents/mrfcs/README.zh.md)。

<a id="cross-cutting-specs"></a>

## 跨切面规格

| 文件                                                                   | 何时阅读                                                                                                                                                                                                                              |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [auth.zh.md](./auth.zh.md)                                             | 端到端认证设计：JWT 双令牌（强化 HS256：锁定算法、强制 exp、密钥 ≥32B）、带令牌盗用重用检测的刷新令牌轮换（revoked 软标记、30 秒轮换宽限窗口）、GitHub OAuth 同页重定向（state + PKCE）、密码登录（bcrypt + NIST 800-63B）、登出、前端令牌存储与自动刷新 |
| [api-design.zh.md](./api-design.zh.md)                                 | REST API 设计规则（对齐 GitHub）：URL 命名（kebab-case、复数集合、admin 命名空间）、HTTP 方法语义（PATCH 部分更新、201 创建、204 删除）、400 与 422 的区分、列表包装对象、双轨认证模型、三层限流                                      |
| [three-tier-testing.zh.md](./three-tier-testing.zh.md)                 | 三层测试策略：单元（真实 PostgreSQL 的后端测试、前端组件测试）、集成、e2e —— 每一层负责哪些行为                                                                                                                                       |
| [docker/build-specification.zh.md](./docker/build-specification.zh.md) | 生产镜像的构建方式：钉住的基础镜像、buildx 多平台构建、缓存策略，以及 `docker/` 的目录布局                                                                                                                                            |

<a id="backend-specs"></a>

## 后端规格

| 文件                                                           | 何时阅读                                                                                                                                                                                                                                               |
| -------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [architecture.zh.md](./backend/architecture.zh.md)             | 洋葱架构（Clean Architecture 变体，三层）：domain（按 feature）/ service / infra / api 分层、依赖方向规则、四项认可的偏离、组合根装配                                                                                                                  |
| [configuration.zh.md](./backend/configuration.zh.md)           | 新增或修改配置项、TOML schema、环境变量映射、校验规则、配置文件约定                                                                                                                                                                                    |
| [error-handling.zh.md](./backend/error-handling.zh.md)         | 分层错误契约（infra/domain/service/handler/apierr）、自带 HTTP/i18n 映射的 ErrCode 结构（按领域的错误码文件）、validate 集成（绑定错误分类、tagRegistry、RegisterTagNameFunc）、GitHub 风格 ErrorResponse、四层回退（零 panic）、经 OTel span 定位故障 |
| [testing.zh.md](./backend/testing.zh.md)                       | 编写测试、测试数据库搭建、mock repository 模式、运行指定测试                                                                                                                                                                                           |
| [i18n.zh.md](./backend/i18n.zh.md)                             | locale 文件组织（BCP 47）、模板插值的消息格式、双源结构（代码内嵌 DefaultMessage + locale 翻译）、消息字面量硬约束（goi18n extract AST），以及完整的 extract/merge 翻译工作流                                                                          |
| [observability.zh.md](./backend/observability.zh.md)           | 三大支柱（日志/追踪/指标）全部导出到文件系统（零外部服务）：slog + timberjack 滚动 + 自写 trace 注入 handler、otelgin 自动 span + 手动子 span、语义化命名的 OTel 指标，以及经 trace_id 的 trace↔log 关联                                               |
| [database-schema.zh.md](./backend/database-schema.zh.md)       | 表结构、字段定义、索引、外键、schema 约定、refresh_tokens.revoked 软标记（令牌盗用检测）                                                                                                                                                               |
| [dsn.zh.md](./backend/dsn.zh.md)                               | PostgreSQL 连接 DSN 格式：pgx keyword 与 URL 两种风格、sslmode 取值、Unix domain socket、自动时区注入，以及经环境变量覆盖机制的口令处理                                                                                                                |
| [api-schema.zh.md](./backend/api-schema.zh.md)                 | API 端点参考 —— 每条路由的请求/响应字段、状态码（201/204/422）、列表包装格式、认证要求                                                                                                                                                                 |
| [keyword-filter.zh.md](./backend/keyword-filter.zh.md)         | 渠道关键词过滤表达式语法 —— OR/AND/NOT 运算符、引号规则、多语言匹配、校验与性能                                                                                                                                                                        |
| [delivery-queue.zh.md](./backend/delivery-queue.zh.md)         | 投递队列的数据模型：`delivery_attempts`（热）+ `delivery_history`（冷）、int8 `Status` 枚举、外键动作（CASCADE 与 SET NULL）、部分/复合索引设计、fillfactor + autovacuum 调优、历史读取面、保留期清理                                                  |
| [delivery-scheduler.zh.md](./backend/delivery-scheduler.zh.md) | 投递分发器：容量包络、三层架构（DB 队列 / 1 秒 ticker / pond v2 池）、入队路径、批量过期墙清扫、带 `next_at` 预留的 `FOR UPDATE SKIP LOCKED` 认领、worker 执行、webhook SSRF 防护、指标、`[delivery]` 配置                                             |
| [delivery-retry.zh.md](./backend/delivery-retry.zh.md)         | 投递重试策略：硬编码的 `[1m,5m,10m,20m]` 退避序列、自动计算的 40 分钟过期墙、终态，以及错误分类（可重试与永久类别驱动快速失败与管理员过滤器）                                                                                                          |
| [delivery-recovery.zh.md](./backend/delivery-recovery.zh.md)   | 投递语义与崩溃恢复：产品契约（尽力而为、至少一次、文章创建触发）、重启行为，以及经认领预留防双重认领                                                                                                                                                   |
| [caching.zh.md](./backend/caching.zh.md)                       | 读路径缓存：硬件包络与工作负载、三层缓存（浏览器 / CDN / 源站渲染缓存）、输出哈希 ETag/304 设计、singleflight + ristretto 渲染缓存、HTTP 缓存头、Cloudflare 免费档 + 缓存标签清除契约、删除驱动失效、发布窗口负载分析、自托管兼容性                    |
| [compression.zh.md](./backend/compression.zh.md)               | 传输最小化：Caddy `encode zstd gzip`、CSS 外置 + 压缩 + 内容哈希指纹（`cmd/buildcss`、`go:embed`、`/static` 路由）、渲染期 HTML 压缩，以及实测的字节级分解                                                                                             |
| [rate-limiting.zh.md](./backend/rate-limiting.zh.md)           | 四个 tollbooth 限流器（读 / 公共写 / 认证写 / 登录）及其键维度与默认值、L2 日上限、匿名 429 与健康检查豁免、Retry-After，以及经 XFF + Caddy trusted_proxies + gin SetTrustedProxies 的客户端 IP 还原                                                   |
| [postgres-tuning.zh.md](./backend/postgres-tuning.zh.md)       | Postgres 调优：连接池边界、五项部署模板 GUC、经迁移启用的 lz4 TOAST 压缩、同级容器 + Unix socket 拓扑、Docker 与裸机对比分析、存储估算                                                                                                                 |
| [disaster-recovery.zh.md](./backend/disaster-recovery.zh.md)   | 单实例韧性姿态：当前备份现状、故障/恢复矩阵、源站宕机期间 CDN 读路径存活、成本                                                                                                                                                                         |
| [cloudflare.zh.md](./backend/cloudflare.zh.md)                 | 三种部署模式（SaaS / 自托管 / homelab）、Cloudflare 接入（Full strict + Origin CA）、缓存行为、缓存标签清除契约、免费档限制，以及 XFF/受信代理的客户端 IP 设计                                                                                         |

<a id="cli-specs"></a>

## CLI 规格

| 文件                     | 何时阅读                                                                                                                                           |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| [cli.zh.md](./cli.zh.md) | 独立客户端 `markpost`（`cli/`）：命令集、会话与配置解析（config.toml + MARKPOST_* 环境变量）、401 自动刷新、agent 检测、输出规则与退出码、测试分层 |

<a id="frontend-specs"></a>

## 前端规格

| 文件                                                | 何时阅读                                                                                                                                                                                           |
| --------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [architecture.zh.md](./frontend/architecture.zh.md) | App Router 结构（纯静态导出、无 SSR/代理）、组件组织、状态管理（Zustand + TanStack Query）、API client（直连后端 + Accept-Language header）、路由保护、provider 栈（LocaleProvider 自举）          |
| [build.zh.md](./frontend/build.zh.md)               | 前端构建规则：纯静态导出（`output: "export"`、无 Node 运行时）、Turbopack 默认打包器、API 相对路径 + 反向代理、`out/` 产物、能力边界表                                                             |
| [design.zh.md](./frontend/design.zh.md)             | 构建 UI 组件、色板、排版、间距、阴影层级、圆角、组件模式                                                                                                                                           |
| [testing.zh.md](./frontend/testing.zh.md)           | Vitest + MSW 单元测试、Playwright E2E 测试、测试设置与工具                                                                                                                                         |
| [routes.zh.md](./frontend/routes.zh.md)             | 前端路由表（/auth/callback OAuth 回调）、声明式守卫架构（AuthGate + Public/Protected/AdminRoute + route-configs 纯函数）、安全边界声明（客户端守卫仅是 UX；安全在后端）、水合处理                  |
| [i18n.zh.md](./frontend/i18n.zh.md)                 | 纯客户端 next-intl（无 getRequestConfig/plugin）、BCP 47 四语言（en/zh-Hans/zh-Hant/ja）、locale 文件命名、语言检测（localStorage + 浏览器语言）、到后端的 Accept-Language header、手工维护的 JSON |

<a id="mcp-specs"></a>

## MCP 规格

| 文件                                       | 何时阅读                                                                                                                                                                                                                 |
| ------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [mcp-server.zh.md](./mcp/mcp-server.zh.md) | markpost-mcp 设计：包装 REST API 的独立 Go module（黄金参考 github-mcp-server、官方 go-sdk v1.7.0）、四个工具集（47 个工具、admin 按需开启、只读模式）、凭据登录与刷新轮换、stdio/无状态 http 双传输、三层测试与发布产物 |
