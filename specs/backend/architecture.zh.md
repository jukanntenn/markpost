# 后端架构

[English](architecture.md) | 中文

本文档定义后端的洋葱架构（Clean Architecture）分层、目录结构、依赖方向规则，以及 pkg/ 边界。端点清单见 [api-schema.zh.md](./api-schema.zh.md)；认证流程见 [auth.md](../auth.zh.md)。

<a id="design-philosophy"></a>

## 设计哲学

markpost 后端遵循 Clean Architecture（洋葱架构）的核心原则——依赖反转：外层依赖内层，内层不依赖外层。接口（端口）定义在内核，由外层实现。

参考依据：[Microsoft Clean Architecture 文档](https://learn.microsoft.com/en-us/dotnet/architecture/modern-web-apps-azure/common-web-application-architectures)、[ardalis/CleanArchitecture](https://github.com/ardalis/CleanArchitecture) 参考实现。

适配决策：采用**修正版 3 层**（domain / service / infra + api），而非 ardalis 的 4 层（domain / usecases / infra / web）。理由：markpost 是单体、聚合少（user / post / delivery），usecase 层会极薄；Go 社区的 clean architecture 实现普遍用 3 层 + service 合一。遵循 Clean Architecture 的**内核原则**（依赖反转、domain 纯净、接口在内部层定义、组合根装配），但不照搬特定生态的**实现形态**（usecase 层、CQRS、Mediator）。

<a id="directory-layout"></a>

## 目录结构

```
backend/
├── cmd/
│   ├── server/              HTTP server entry (main.go: composition root — wires repo→service→handler)
│   └── buildcss/            build-time CSS code generator
├── internal/
│   ├── domain/              pure domain core (feature-based, zero intra-project dependencies)
│   │   ├── errors.go        cross-domain shared sentinel errors (ErrNotFound / ErrConflict / ...)
│   │   ├── user/            user aggregate: User model + Repository / TokenRepository interfaces
│   │   ├── post/            post aggregate: Post model + Repository interface + delivery ports (DeliveryJob / DeliveryEnqueuer)
│   │   └── delivery/        delivery aggregate: Channel / Attempt models + Repository / AttemptRepository interfaces
│   ├── service/             application logic (depends on domain interfaces; lateral imports between its packages forbidden)
│   │   ├── errors.go        ErrCode struct / Error / FieldDetail types + shared error codes + constructors
│   │   ├── auth/            authentication (OAuth / JWT / password / session) + auth/errors.go domain-specific codes
│   │   ├── post/            post CRUD / Markdown rendering
│   │   ├── delivery/        delivery channel management + retry scheduling + filter/
│   │   └── admin/           admin read-only views (cross-aggregate)
│   ├── infra/               GORM repository implementations (implement the domain interfaces) + DB bootstrap
│   │   ├── db.go            Database struct, New(dsn) (opens the connection only; no migration)
│   │   ├── migrate.go       golang-migrate wrapper (MigrateUp/Down/Force/Version)
│   │   ├── migrations/      embedded SQL migration files (NNNNNN_description.up/down.sql)
│   │   ├── helpers.go       generic GORM helpers (findFirst / findMany / existsBy / ...)
│   │   ├── search.go        LIKE search helpers
│   │   └── *_repo.go        one repository implementation file per aggregate (user_repo / token_repo / post_repo / ...)
│   ├── apierr/              HTTP error response formatting (the single error entry point for handler / middleware)
│   ├── api/rest/v1/         REST API handlers (Gin) + DTOs
│   ├── middleware/          Gin middleware (auth / CORS / rate limiting / panic recovery / post_key)
│   ├── config/              configuration loading (Viper + TOML)
│   └── web/                 build metadata + embedded CSS assets
├── pkg/                     reusable packages with zero internal dependencies only
│   ├── utils/               general utilities (password / token / strings / post_key / generics / oauth)
│   └── httputil/            HTTP utilities (FetchAndDecodeJSON)
├── locales/                 backend i18n locale files (TOML)
├── templates/               post-rendering HTML templates
├── tools/                   development tools (fake data generator)
└── docs/                    generated Swagger docs (never edited by hand)
```

<a id="the-domain-layer-feature-based-organization"></a>

### domain 层：feature-based 组织

domain 按聚合（aggregate）分包，每个聚合的 model 和它的 repository 接口**同居一包**：

```
internal/domain/user/
├── user.go          User / Role / GitHubUser models
├── token.go         RefreshToken / TokenBlacklist models
└── repository.go    Repository / TokenRepository interfaces
```

这与 ardalis/CleanArchitecture 参考实现一致（`ContributorAggregate/` 下同放 model + Specifications + Events + Handlers）。聚合的 model 和它的端口放一起更内聚——改 user 的 model 时，它的接口就在隔壁。

**不采用** layer-based 拆分（`domain/model/` + `domain/port/`）——那会把聚合的 model 和接口拆到两个目录，增加跨目录跳转，且可能引入 port→model 的循环依赖。

domain 根包只放**跨域通用**的 sentinel error（`ErrNotFound`、`ErrConflict` 等），作为跨层错误识别的稳定契约。域特定的业务错误由 service 层识别后转 service.Error。

<a id="the-service-layer-application-logic--domain-specific-error-codes"></a>

### service 层：应用逻辑 + 域专属错误码

service 层承担应用编排（调 repo、业务规则、发事件）。按功能分包，每包一个 `Service`：

```
internal/service/
├── errors.go            ErrCode struct (carries its own HTTP/i18n mappings) + shared error codes
├── auth/
│   ├── errors.go        auth domain-specific codes (ErrInvalidCredentials / ErrInvalidToken / ...)
│   ├── auth.go          authentication service
│   └── jwt.go           JWT issuing / verification
├── post/                post service + render cache
├── delivery/            delivery service + scheduler + filter/
└── admin/               admin service
```

域专属错误码按域分文件（遵循 [error-handling.zh.md](./error-handling.zh.md) 的「域专属码分文件」原则）。

<a id="dependency-direction-the-core-rule"></a>

## 依赖方向（核心规则）

```
cmd/server/main  ──►  infra, service/*, domain, api/v1, middleware, config   (composition root)
api/v1          ──►  service (interfaces + DTOs), domain, middleware, apierr, web, pkg/*
middleware      ──►  service, domain, pkg/*
service/*       ──►  domain, config, pkg/*      (direct imports between service packages are forbidden)
infra           ──►  domain, config
domain/*        ──►  (same-layer domain packages only)   ✓ pure kernel
pkg/*           ──►  (stdlib / external libraries only)  ✓ truly reusable
```

依赖只能向内指向：

- **domain** → 零项目内依赖（纯内核，只依赖 stdlib + 外部库如 gorm tags）
- **service** → domain（通过接口）；**service 包之间禁止直接 import**（横向协作通过 domain port）
- **infra** → domain（实现接口）+ config
- **api** → service（接口 + DTO）+ domain + middleware + apierr
- **pkg** → 仅 stdlib / 外部库（零 internal 依赖，真正可复用）
- **cmd/server/main.go**（组合根）→ 全部，负责装配 repo→service→handler

<a id="dependency-injection"></a>

## 依赖注入

Services 和 handlers 通过构造函数接收依赖。Repository 接口定义在 domain 层，infra 层实现，组合根装配：

```go
// The domain layer defines the interface
type Repository interface {
    GetByID(ctx context.Context, id int) (*User, error)
}

// The infra layer implements it
type userRepository struct { db *gorm.DB }
func (r *userRepository) GetByID(ctx context.Context, id int) (*User, error) { ... }

// The composition root assembles (main.go)
userRepo = infra.NewUserRepository(dbInstance.DB())
authSvc = auth.NewService(userRepo, tokenRepo, oauthConfig, jwtSvc, "markpost")
```

Handlers 接受 service 接口（不是具体 struct），允许测试注入 mock：

```go
func LoginWithUsername(authSvc AuthService) gin.HandlerFunc { ... }
```

<a id="deviations-from-the-reference-shape"></a>

## 偏离参考形态之处

四处代码库偏离教科书式 Clean Architecture 形态的地方，以及各自的取舍：

<a id="1-apierr-lives-in-internal"></a>

### 1. apierr 位于 internal/

**为何不放 `pkg/`**：`apierr` import `internal/service`；若放在 `pkg/` 下，会破坏 `pkg/` 的自包含（反向依赖）。

**安排**：`apierr` 位于 `internal/apierr/`。它是应用特定的错误响应格式化器（service 产生 service.Error、apierr 格式化成 HTTP 响应、middleware 也用它 abort）——不是通用可复用库——归属 `internal/`。

**`pkg/` 边界**：`pkg/` 只保留真正零 internal 依赖的包：`pkg/utils`、`pkg/httputil`。

<a id="2-the-api-layer-imports-service-packages-for-dto-types-accepted"></a>

### 2. api 层 import service 包拿 DTO 类型（接受）

**现状**：`api/v1/auth.go` import `service/auth` 拿 `*auth.JWTTokenPair`；`api/v1/delivery.go` import `service/delivery` 拿 `delivery.UpdateChannelParams`。

**决策**：接受。handler 函数签名已用本地接口（`AuthService`、`DeliveryService`），import service 包只为了 DTO 类型。这不是「依赖 service 实现」，是「依赖 service 的数据契约」。

**规则**：api 层可以 import service 层拿 DTO 类型，但 handler **必须通过接口**（而非具体 struct）调用 service 逻辑。强解耦（把 DTO 下沉 domain）会污染内核——domain 不该知道 JWT token pair 长什么样。ardalis 的 Web 层也 import UseCases 层的 DTO，这是 Clean Architecture 接受的做法。

<a id="3-deliveryjob-and-deliveryenqueuer-live-in-domainpost"></a>

### 3. DeliveryJob 和 DeliveryEnqueuer 位于 domain/post

**问题所在**：投递 dispatcher 需要 `DeliveryJob`（struct）和 `DeliveryEnqueuer`（interface）两个类型——横向 import `service/post` 只为拿到它们。

**诊断**：这两个类型是**纯领域数据契约**——`DeliveryJob` 字段全是基础类型（int/string），`DeliveryEnqueuer` 是单方法接口。它们没有任何应用逻辑，完全是「post 聚合向 delivery 暴露的投递契约」——正是领域端口（domain port）的定义。待在 `service/post` 是放错层了。

**安排**：`DeliveryJob` 和 `DeliveryEnqueuer` 位于 `domain/post/`（`domain/post/delivery.go`）。`service/post.Service` 依赖 `post.DeliveryEnqueuer`（domain 接口），由 `Dispatcher`（在 `service/delivery`）实现。依赖方向是 `service/delivery → domain/post`——正确的向内指向。

<a id="4-serviceadmin-defines-its-own-local-interfaces-accepted"></a>

### 4. service/admin 自定义本地接口（接受）

**现状**：`service/admin` 定义 `UserLister` / `PostLister` / `ChannelLister` 等本地接口，而非复用 domain 的 Repository 接口。

**决策**：接受。admin 是跨聚合只读视图，它的查询方法（`GetAllUsers`、`ListAllPosts`）是 domain Repository 接口里没有的。admin 为自己需要的协作定义本地接口（端口）是合理的依赖倒置——它依赖 domain 类型，不依赖其它 service 实现。

<a id="router-setup"></a>

## Router Setup

路由在 `cmd/server/main.go` 的 `SetupRoutes` 函数配置。Gin engine 创建后：

1. HTML 模板加载（`templates/*`）
2. Trusted proxies 配置
3. **otelgin**（`otelgin.Middleware`）— OpenTelemetry tracing：每 HTTP 请求自动创建 span（method/path/status/latency），见 [observability.zh.md](./observability.zh.md)
4. i18n 中间件（加载 `./locales` locale 文件）
5. Panic recovery 中间件（`middleware.Fallback`）
6. CORS 中间件
7. 限流中间件
8. Route groups 注册

<a id="middleware-chain"></a>

## 中间件链

中间件链执行顺序：

1. **Gin default** — Logging + recovery
2. **otelgin**（`otelgin.Middleware`）— 自动创建 trace span，通过 trace_id 关联日志（见 [observability.zh.md](./observability.zh.md)）
3. **i18n**（`ginI18n.Localize`）— 从 `Accept-Language` 头检测语言
4. **Fallback**（`middleware.Fallback`）— panic recovery，返回 500
5. **CORS**（`cors.New`）— preflight，CORS headers
6. **Rate limiting**（`middleware.RateLimitByIP`）— per-IP 限流（tollbooth）
7. **Auth**（per group）— `middleware.AuthWithBlacklist` 校验 JWT + 查黑名单
8. **Admin**（per group）— `middleware.RequireAdmin` 校验角色
9. **PostKey**（per route）— `middleware.PostKey` 解析 post_key 到 user

Auth 中间件成功时设置 context：`user`、`user_id`、`email`、`username`、`role`、`claims`。

<a id="route-groups"></a>

## Route Groups

端点清单见 [api-schema.zh.md](./api-schema.zh.md) 和 [api-design.md](../api-design.zh.md)。

---

<a id="references"></a>

## 参考

- [error-handling.zh.md](./error-handling.zh.md) — 分层错误契约、apierr 包
- [api-design.md](../api-design.zh.md) — API 设计规范
- [auth.md](../auth.zh.md) — 认证流程
