# Backend Architecture

本文档定义后端的洋葱架构（Clean Architecture）分层、目录结构、依赖方向规则，以及 pkg/ 边界。端点清单见 [api-schema.md](./api-schema.md)；认证流程见 [auth.md](../auth.md)。

## 设计哲学

markpost 后端遵循 Clean Architecture（洋葱架构）的核心原则——依赖反转：外层依赖内层，内层不依赖外层。接口（端口）定义在内核，由外层实现。

参考依据：[Microsoft Clean Architecture 文档](https://learn.microsoft.com/en-us/dotnet/architecture/modern-web-apps-azure/common-web-application-architectures)、[ardalis/CleanArchitecture](https://github.com/ardalis/CleanArchitecture) 参考实现。

适配决策：采用**修正版 3 层**（domain / service / infra + api），而非 ardalis 的 4 层（domain / usecases / infra / web）。理由：markpost 是单体、聚合少（user/post/delivery），usecase 层会极薄；Go 社区的 clean architecture 实现普遍用 3 层 + service 合一。遵循 Clean Architecture 的**内核原则**（依赖反转、domain 纯净、接口在内部层定义、组合根装配），但不照搬特定生态的**实现形态**（usecase 层、CQRS、Mediator）。

## 目录结构

```
backend/
├── cmd/
│   ├── server/              HTTP server 入口（main.go：组合根 — 装配 repo→service→handler）
│   └── buildcss/            CSS 构建期代码生成器
├── internal/
│   ├── domain/              纯领域内核（feature-based，零项目内依赖）
│   │   ├── errors.go        跨域共享 sentinel error（ErrNotFound / ErrConflict / ...）
│   │   ├── user/            user 聚合：User model + Repository / TokenRepository 接口
│   │   ├── post/            post 聚合：Post model + Repository 接口 + 投递端口（DeliveryJob / DeliveryEnqueuer）
│   │   └── delivery/        delivery 聚合：Channel / Attempt model + Repository / AttemptRepository 接口
│   ├── service/             应用逻辑（依赖 domain 接口；包间禁横向 import）
│   │   ├── errors.go        ErrCode struct / Error / FieldDetail 类型 + 共享错误码 + 构造器
│   │   ├── auth/            认证（OAuth / JWT / 密码 / 会话）+ auth/errors.go 域专属码
│   │   ├── post/            文章 CRUD / Markdown 渲染
│   │   ├── delivery/        投递渠道管理 + 重试调度 + filter/
│   │   └── admin/           管理员只读视图（跨聚合）
│   ├── infra/               GORM 仓储实现（实现 domain 接口）+ DB 初始化
│   │   ├── db.go            Database 结构、New(dsn)、AutoMigrate、迁移函数
│   │   ├── helpers.go       泛型 GORM 辅助（findFirst / findMany / existsBy / ...）
│   │   ├── search.go        LIKE 搜索辅助
│   │   └── *_repo.go        每聚合一个仓储实现文件（user_repo / token_repo / post_repo / ...）
│   ├── apierr/              HTTP 错误响应格式化（handler / middleware 的统一错误入口）
│   ├── api/rest/v1/         REST API handler（Gin handler）+ DTO
│   ├── middleware/          Gin 中间件（auth / CORS / rate limiting / panic recovery / post_key）
│   ├── config/              配置加载（Viper + TOML）
│   └── web/                 构建元数据 + 嵌入 CSS 资产
├── pkg/                     仅放零 internal 依赖的可复用包
│   ├── utils/               通用工具（password / token / strings / post_key / generics / oauth）
│   └── httputil/            HTTP 工具（FetchAndDecodeJSON）
├── locales/                 后端 i18n locale 文件（TOML）
├── templates/               文章渲染 HTML 模板
├── tools/                   开发工具（fake data 生成器）
└── docs/                    生成的 Swagger 文档（勿手编）
```

### domain 层：feature-based 组织

domain 按聚合（aggregate）分包，每个聚合的 model 和它的 repository 接口**同居一包**：

```
internal/domain/user/
├── user.go          User / Role / GitHubUser 模型
├── token.go         RefreshToken / TokenBlacklist 模型
└── repository.go    Repository / TokenRepository 接口
```

这与 ardalis/CleanArchitecture 参考实现一致（`ContributorAggregate/` 下同放 model + Specifications + Events + Handlers）。聚合的 model 和它的端口放一起更内聚——改 user 的 model 时，它的接口就在隔壁。

**不采用** layer-based 拆分（`domain/model/` + `domain/port/`）——那会把聚合的 model 和接口拆到两个目录，增加跨目录跳转，且可能引入 port→model 的循环依赖。

domain 根包只放**跨域通用**的 sentinel error（`ErrNotFound`、`ErrConflict` 等），作为跨层错误识别的稳定契约。域特定的业务错误由 service 层识别后转 service.Error。

### service 层：应用逻辑 + 域专属错误码

service 层承担应用编排（调 repo、业务规则、发事件）。按功能分包，每包一个 `Service`：

```
internal/service/
├── errors.go            ErrCode struct（自带 HTTP/i18n 映射）+ 共享错误码
├── auth/
│   ├── errors.go        auth 域专属码（ErrInvalidCredentials / ErrInvalidToken / ...）
│   ├── auth.go          认证 service
│   └── jwt.go           JWT 签发/校验
├── post/                文章 service + 渲染缓存
├── delivery/            投递 service + 调度器 + filter/
└── admin/               管理员 service
```

域专属错误码按域分文件（遵循 [error-handling.md](./error-handling.md) 的"域专属码分文件"原则）。

## 依赖方向（核心规则）

```
cmd/server/main  ──►  infra, service/*, domain, api/v1, middleware, config   （组合根）
api/v1          ──►  service（接口 + DTO）, domain, middleware, apierr, web, pkg/*
middleware      ──►  service, domain, pkg/*
service/*       ──►  domain, config, pkg/*      （service 包之间禁止直接 import）
infra           ──►  domain, config
domain/*        ──►  （仅同层 domain 包）       ✓ 纯净内核
pkg/*           ──►  （仅 stdlib / 外部库）     ✓ 真正可复用
```

依赖只能向内指向：

- **domain** → 零项目内依赖（纯内核，只依赖 stdlib + 外部库如 gorm tags）
- **service** → domain（通过接口）；**service 包之间禁止直接 import**（横向协作通过 domain port）
- **infra** → domain（实现接口）+ config
- **api** → service（接口 + DTO）+ domain + middleware + apierr
- **pkg** → 仅 stdlib / 外部库（零 internal 依赖，真正可复用）
- **cmd/server/main.go**（组合根）→ 全部，负责装配 repo→service→handler

## Dependency Injection

Services 和 handlers 通过构造函数接收依赖。Repository 接口定义在 domain 层，infra 层实现，组合根装配：

```go
// Domain 层定义接口
type Repository interface {
    GetByID(ctx context.Context, id int) (*User, error)
}

// Infra 层实现
type userRepository struct { db *gorm.DB }
func (r *userRepository) GetByID(ctx context.Context, id int) (*User, error) { ... }

// 组合根装配（main.go）
userRepo = infra.NewUserRepository(dbInstance.DB())
authSvc = auth.NewService(userRepo, tokenRepo, oauthConfig, jwtSvc, "markpost")
```

Handlers 接受 service 接口（不是具体 struct），允许测试注入 mock：

```go
func LoginWithUsername(authSvc AuthService) gin.HandlerFunc { ... }
```

## 偏离点修复

以下 4 处偏离 Clean Architecture 的设计已修正：

### 1. apierr 移入 internal/

**问题**：`pkg/apierr` import `internal/service`，导致 `pkg/` 不自包含（反向依赖）。

**修复**：`apierr` 移到 `internal/apierr/`。它是应用特定的错误响应格式化器（service 产生 service.Error、apierr 格式化成 HTTP 响应、middleware 也用它 abort），不是通用可复用库，应待在 `internal/`。

**pkg/ 边界**：移除 apierr 后，`pkg/` 只保留真正零 internal 依赖的包：`pkg/utils`、`pkg/httputil`。

### 2. api 层 import service 包拿 DTO 类型（接受）

**现状**：`api/v1/auth.go` import `service/auth` 拿 `*auth.JWTTokenPair`；`api/v1/delivery.go` import `service/delivery` 拿 `delivery.UpdateChannelParams`。

**决策**：接受。handler 函数签名已用本地接口（`AuthService`、`DeliveryService`），只为了 DTO 类型才 import service 包。这不是"依赖 service 实现"，是"依赖 service 的数据契约"。

**规则**：api 层可以 import service 层拿 DTO 类型，但 handler **必须通过接口**（而非具体 struct）调用 service 逻辑。强解耦（把 DTO 下沉 domain）会污染内核——domain 不该知道 JWT token pair 长什么样。ardalis 的 Web 层也 import UseCases 层的 DTO，这是 Clean Architecture 接受的。

### 3. service/delivery → service/post 横向依赖（已解耦）

**问题**：`service/delivery/dispatcher.go` import `service/post`，只为拿 `DeliveryJob`（struct）和 `DeliveryEnqueuer`（interface）两个类型。

**诊断**：这两个类型是**纯领域数据契约**——`DeliveryJob` 字段全是基础类型（int/string），`DeliveryEnqueuer` 是单方法接口。它们没有任何应用逻辑，完全是"post 聚合向 delivery 暴露的投递契约"。这正是领域端口（domain port）的定义，待在 `service/post` 是放错层了。

**修复**：把 `DeliveryJob` 和 `DeliveryEnqueuer` 下沉到 `domain/post/`（新增 `domain/post/delivery.go`）。`service/post.Service` 依赖 `post.DeliveryEnqueuer`（domain 接口），`Dispatcher`（在 `service/delivery`）实现它。依赖方向从 `service/delivery → service/post` 变成 `service/delivery → domain/post`（正确的向内指向）。

### 4. service/admin 自定义本地接口（接受）

**现状**：`service/admin` 定义 `UserLister` / `PostLister` / `ChannelLister` 等本地接口，而非复用 domain 的 Repository 接口。

**决策**：接受。admin 是跨聚合只读视图，它的查询方法（`GetAllUsers`、`ListAllPosts`）是 domain Repository 接口里没有的。admin 为自己需要的协作定义本地接口（端口）是合理的依赖倒置——它依赖 domain 类型，不依赖其它 service 实现。

## Router Setup

路由在 `cmd/server/main.go` 的 `SetupRoutes` 函数配置。Gin engine 创建后：

1. HTML 模板加载（`templates/*`）
2. Trusted proxies 配置
3. **otelgin**（`otelgin.Middleware`）— OpenTelemetry tracing：每 HTTP 请求自动创建 span（method/path/status/latency），见 [observability.md](./observability.md)
4. i18n 中间件（加载 `./locales` locale 文件）
5. Panic recovery 中间件（`middleware.Fallback`）
6. CORS 中间件
7. 限流中间件
8. Route groups 注册

## Middleware Chain

中间件链执行顺序：

1. **Gin default** — Logging + recovery
2. **otelgin**（`otelgin.Middleware`）— 自动创建 trace span，通过 trace_id 关联日志（见 [observability.md](./observability.md)）
3. **i18n**（`ginI18n.Localize`）— 从 `Accept-Language` 头检测语言
4. **Fallback**（`middleware.Fallback`）— panic recovery，返回 500
5. **CORS**（`cors.New`）— preflight，CORS headers
6. **Rate limiting**（`middleware.RateLimitByIP`）— per-IP 限流（tollbooth）
7. **Auth**（per group）— `middleware.AuthWithBlacklist` 校验 JWT + 查黑名单
8. **Admin**（per group）— `middleware.RequireAdmin` 校验角色
9. **PostKey**（per route）— `middleware.PostKey` 解析 post_key 到 user

Auth 中间件成功时设置 context：`user`、`user_id`、`email`、`username`、`role`、`claims`。

## Route Groups

端点清单见 [api-schema.md](./api-schema.md) 和 [api-design.md](../api-design.md)。

---

## 参考

- [error-handling.md](./error-handling.md) — 分层错误契约、apierr 包
- [api-design.md](../api-design.md) — API 设计规范
- [auth.md](../auth.md) — 认证流程
