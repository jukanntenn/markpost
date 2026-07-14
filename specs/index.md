# Specification Files

> Reference the most relevant spec file(s) for your current task.

## Cross-Cutting Specs

| File | When to Read |
|------|-------------|
| [auth.md](./auth.md) | 端到端认证设计：JWT 双 token（HS256 安全硬化：锁定算法/强制 exp/密钥≥32B）、refresh token 轮转 + token theft 重用检测（revoked 软标记）、OAuth GitHub 同页重定向（state + PKCE 双保险）、密码登录（bcrypt + NIST 800-63B）、登出、前端 token 存储与自动刷新 |
| [api-design.md](./api-design.md) | REST API 设计规范（深度对齐 GitHub）：URL 命名（kebab-case / 复数集合 / admin 命名空间）、HTTP 方法语义（PATCH 部分更新 / 201 创建 / 204 删除）、400 vs 422 区分、列表包裹对象、双轨认证模型、三层限流 |

## Backend Specs

| File | When to Read |
|------|-------------|
| [architecture.md](./backend/architecture.md) | 洋葱架构（Clean Architecture 修正版 3 层）：domain（feature-based）/ service / infra / api 分层、依赖方向规则、4 偏离点修复（apierr 移入 internal、DTO 归 service、delivery→post 用 domain port 解耦、admin 本地接口）、组合根装配 |
| [configuration.md](./backend/configuration.md) | Adding or changing configuration values, TOML schema, env variable mapping, validation rules, config file conventions |
| [dev-environment.md](./backend/dev-environment.md) | Setting up local development, running tests, linting, Swagger docs, database configuration |
| [error-handling.md](./backend/error-handling.md) | Layered error contract (infra/domain/service/handler/apierr), ErrCode struct with self-carried HTTP/i18n mapping (per-domain error code files), validate integration (binding error classification, tagRegistry, RegisterTagNameFunc), GitHub-style ErrorResponse, four-layer fallback (zero panic), and fault localization via OTel spans |
| [testing.md](./backend/testing.md) | Writing tests, test database setup, mock repository patterns, running specific tests |
| [i18n.md](./backend/i18n.md) | Locale file organization (BCP 47), message format with template interpolation, dual-source structure (code-embedded DefaultMessage + locale translations), message literal hard constraint (goi18n extract AST), and the full extract/merge translation workflow |
| [observability.md](./backend/observability.md) | Three pillars (logs/traces/metrics) all exported to filesystem (zero external services): slog + timberjack rolling + self-written trace-injection handler, otelgin auto-span + manual child spans, OTel metrics with semconv naming, and trace↔log correlation via trace_id |
| [database-schema.md](./backend/database-schema.md) | Table structure, field definitions, indexes, foreign keys, schema conventions, refresh_tokens.revoked 软标记（token theft 检测） |
| [dsn.md](./backend/dsn.md) | 数据库连接规范：driver 推断（尽力而为，缺省时从 DSN 推断）、三端 DSN 格式（PostgreSQL pgx keyword/URL + sslmode + Unix socket / MySQL utf8mb4 / SQLite WAL 固定格式）、SQLite 目录自动创建（dirForSQLite 算法 + edge case 表） |
| [api-schema.md](./backend/api-schema.md) | API 端点参考 — 所有路由的请求/响应字段、状态码（201/204/422）、列表包裹对象格式、认证要求 |
| [keyword-filter.md](./backend/keyword-filter.md) | Channel keyword filter expression grammar — OR/AND/NOT operators, quoting rules, multilingual matching, validation, and performance |
| [performance-optimization.md](./backend/performance-optimization.md) | Caching layers, HTTP/CDN cache strategy (cache-tag purge), compression, ETag/304 design (xxhash of rendered output), render caching (singleflight + ristretto), trusted-proxy IP detection, three-limiter rate limiting, CSS externalization + minify, Postgres tuning (lz4/GUC/Unix socket), deletion + CDN invalidation, disaster recovery, and load testing under the 2C/2G/3M hardware envelope |
| [cloudflare.md](./backend/cloudflare.md) | Three deployment modes (SaaS / self-hosted / homelab), Cloudflare onboarding (Full strict + Origin CA), cache behavior, cache-tag purge contract, free-tier limits, and the XFF/trusted-proxy client-IP design |
| [delivery.md](./backend/delivery.md) | Persistent best-effort delivery: three-layer architecture (DB queue / Go scheduler / pond pool), hardcoded backoff sequence with auto-computed expiry wall, crash recovery (at-least-once), GORM data model portable across Postgres/MySQL/SQLite (int8 status enum, per-dialect indexes, delivery_history with FK ON DELETE SET NULL), dialect-safe batched claim/expire/prune SQL, and rejected alternatives (fire-and-forget, external brokers, exactly-once, configurable backoff) |

## Frontend Specs

| File | When to Read |
|------|-------------|
| [architecture.md](./frontend/architecture.md) | App Router 结构（纯静态导出，无 SSR/proxy）、组件组织、状态管理（Zustand + TanStack Query）、API client（直连后端 + Accept-Language 头）、路由保护、Provider stack（LocaleProvider 自举） |
| [build.md](./frontend/build.md) | 前端打包规范：纯静态导出（output: "export"，无 Node 运行时）、Turbopack 默认 bundler（无需 --turbopack）、移除 proxy.ts/health/i18n request、API 相对路径 + 反代、out/ 产物、能力边界表 |
| [design.md](./frontend/design.md) | Building UI components, color palette, typography, spacing, elevation, border radius, component patterns |
| [dev-environment.md](./frontend/dev-environment.md) | Setting up frontend development, pnpm commands, environment variables, dev server |
| [testing.md](./frontend/testing.md) | Unit tests with Vitest and MSW, E2E tests with Playwright, test setup and utilities |
| [routes.md](./frontend/routes.md) | 前端路由表（/auth/callback OAuth 回调）、声明式守卫架构（AuthGate + Public/Protected/AdminRoute + route-configs 纯函数）、安全边界声明（客户端守卫仅 UX，安全保障在后端）、水合处理 |
| [i18n.md](./frontend/i18n.md) | 纯客户端 next-intl（不用 getRequestConfig/plugin）、BCP 47 四语言（en/zh-Hans/zh-Hant/ja）、locale 文件命名、语言检测（localStorage + navigator.language）、Accept-Language 头传后端、手动维护 JSON |

---
