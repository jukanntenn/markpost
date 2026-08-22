# Backend Architecture

English | [中文](architecture.zh.md)

This document defines the backend's onion-architecture (Clean Architecture) layering, directory layout, dependency-direction rules, and the `pkg/` boundary. The endpoint inventory lives in [api-schema.md](./api-schema.md); the authentication flows live in [auth.md](../auth.md).

## Design Philosophy

The markpost backend follows the core Clean Architecture (onion) principle — dependency inversion: outer layers depend on inner layers, never the reverse. Interfaces (ports) are defined in the core and implemented by the outer layers.

Reference points: [Microsoft's Clean Architecture documentation](https://learn.microsoft.com/en-us/dotnet/architecture/modern-web-apps-azure/common-web-application-architectures) and the [ardalis/CleanArchitecture](https://github.com/ardalis/CleanArchitecture) reference implementation.

Adaptation decision: markpost uses a **modified three layers** (domain / service / infra + api) instead of ardalis's four (domain / usecases / infra / web). Rationale: markpost is a monolith with few aggregates (user / post / delivery), so a usecase layer would be paper-thin; Go clean-architecture implementations commonly fold it into a three-layer service. The codebase follows Clean Architecture's **core principles** (dependency inversion, a pure domain, interfaces defined in inner layers, assembly at the composition root) without adopting any particular ecosystem's **implementation shape** (usecase layer, CQRS, Mediator).

## Directory Layout

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

### The domain layer: feature-based organization

The domain is packaged per aggregate, with each aggregate's models and its repository interface **co-located in one package**:

```
internal/domain/user/
├── user.go          User / Role / GitHubUser models
├── token.go         RefreshToken / TokenBlacklist models
└── repository.go    Repository / TokenRepository interfaces
```

This mirrors the ardalis/CleanArchitecture reference implementation (under `ContributorAggregate/` sit the model together with Specifications, Events, and Handlers). Keeping an aggregate's models next to its ports is more cohesive — when you change the user model, its interface is right there.

**Layer-based splitting is not used** (`domain/model/` + `domain/port/`) — it separates an aggregate's models from its interfaces into two directories, adds cross-directory hopping, and invites port→model import cycles.

The domain root package holds only **cross-domain** sentinel errors (`ErrNotFound`, `ErrConflict`, and the like) as the stable contract for cross-layer error identification. Domain-specific business errors are recognized by the service layer and converted to `service.Error` there.

### The service layer: application logic + domain-specific error codes

The service layer carries application orchestration (calling repositories, business rules, emitting events). It is packaged by feature, one `Service` per package:

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

Domain-specific error codes live in per-domain files (following the "domain-specific codes in their own file" principle of [error-handling.md](./error-handling.md)).

## Dependency Direction (the core rule)

```
cmd/server/main  ──►  infra, service/*, domain, api/v1, middleware, config   (composition root)
api/v1          ──►  service (interfaces + DTOs), domain, middleware, apierr, web, pkg/*
middleware      ──►  service, domain, pkg/*
service/*       ──►  domain, config, pkg/*      (direct imports between service packages are forbidden)
infra           ──►  domain, config
domain/*        ──►  (same-layer domain packages only)   ✓ pure kernel
pkg/*           ──►  (stdlib / external libraries only)  ✓ truly reusable
```

Dependencies point inward only:

- **domain** → zero intra-project dependencies (a pure kernel depending only on the stdlib and external libraries such as gorm tags)
- **service** → domain (through interfaces); **direct imports between service packages are forbidden** (lateral collaboration goes through a domain port)
- **infra** → domain (implements its interfaces) + config
- **api** → service (interfaces + DTOs) + domain + middleware + apierr
- **pkg** → stdlib / external libraries only (zero internal dependencies, genuinely reusable)
- **cmd/server/main.go** (composition root) → everything; it wires repo→service→handler

## Dependency Injection

Services and handlers receive their dependencies through constructors. Repository interfaces are defined in the domain layer, implemented in infra, and assembled by the composition root:

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

Handlers take a service interface (not a concrete struct), allowing mock injection in tests:

```go
func LoginWithUsername(authSvc AuthService) gin.HandlerFunc { ... }
```

## Deviations from the Reference Shape

Four points where the codebase departs from the textbook Clean Architecture shape, and the stance each takes:

### 1. apierr lives in internal/

**Why not `pkg/`**: `apierr` imports `internal/service`; a package under `pkg/` with such an import would break `pkg/`'s self-containment (a reverse dependency).

**Arrangement**: `apierr` lives at `internal/apierr/`. It is an application-specific error response formatter (service produces `service.Error`, apierr formats the HTTP response, middleware also aborts through it) — not a generally reusable library — so its home is `internal/`.

**The `pkg/` boundary**: `pkg/` keeps only packages with genuinely zero internal dependencies: `pkg/utils`, `pkg/httputil`.

### 2. The api layer imports service packages for DTO types (accepted)

**Current state**: `api/v1/auth.go` imports `service/auth` for `*auth.JWTTokenPair`; `api/v1/delivery.go` imports `service/delivery` for `delivery.UpdateChannelParams`.

**Decision**: accepted. Handler function signatures use local interfaces (`AuthService`, `DeliveryService`); the service-package import exists only for DTO types. This is "depending on the service's data contract", not "depending on the service implementation".

**Rule**: the api layer may import service packages for DTO types, but handlers **must go through interfaces** (not concrete structs) to invoke service logic. Stronger decoupling (pushing DTOs down into domain) would pollute the kernel — the domain should not know what a JWT token pair looks like. ardalis's Web layer likewise imports UseCases-layer DTOs; this is accepted Clean Architecture practice.

### 3. DeliveryJob and DeliveryEnqueuer live in domain/post

**The concern**: the delivery dispatcher needs the `DeliveryJob` struct and the `DeliveryEnqueuer` interface — two types and nothing more — which a lateral import of `service/post` would provide.

**Diagnosis**: both types are **pure domain data contracts** — `DeliveryJob`'s fields are all basic types (int/string), and `DeliveryEnqueuer` is a single-method interface. They carry no application logic; they are exactly "the delivery contract the post aggregate exposes to delivery" — the definition of a domain port. Their home in `service/post` was the wrong layer.

**Arrangement**: `DeliveryJob` and `DeliveryEnqueuer` live in `domain/post/` (in `domain/post/delivery.go`). `service/post.Service` depends on `post.DeliveryEnqueuer` (a domain interface), which `Dispatcher` (in `service/delivery`) implements. The dependency direction is `service/delivery → domain/post` — correctly inward-pointing.

### 4. service/admin defines its own local interfaces (accepted)

**Current state**: `service/admin` defines local interfaces such as `UserLister` / `PostLister` / `ChannelLister` instead of reusing the domain Repository interfaces.

**Decision**: accepted. admin is a cross-aggregate read-only view, and its query methods (`GetAllUsers`, `ListAllPosts`) exist on no domain Repository interface. Defining the ports it needs is sound dependency inversion — admin depends on domain types, not on other services' implementations.

## Router Setup

Routes are configured in the `SetupRoutes` function of `cmd/server/main.go`. After the Gin engine is created:

1. HTML template loading (`templates/*`)
2. Trusted proxies configuration
3. **otelgin** (`otelgin.Middleware`) — OpenTelemetry tracing: a span is created automatically for every HTTP request (method/path/status/latency); see [observability.md](./observability.md)
4. i18n middleware (loads the `./locales` locale files)
5. Panic recovery middleware (`middleware.Fallback`)
6. CORS middleware
7. Rate limiting middleware
8. Route group registration

## Middleware Chain

The middleware chain executes in this order:

1. **Gin default** — logging + recovery
2. **otelgin** (`otelgin.Middleware`) — creates trace spans automatically and correlates logs through trace_id (see [observability.md](./observability.md))
3. **i18n** (`ginI18n.Localize`) — detects the language from the `Accept-Language` header
4. **Fallback** (`middleware.Fallback`) — panic recovery, returns 500
5. **CORS** (`cors.New`) — preflight, CORS headers
6. **Rate limiting** (`middleware.RateLimitByIP`) — per-IP rate limiting (tollbooth)
7. **Auth** (per group) — `middleware.AuthWithBlacklist` verifies the JWT + checks the blacklist
8. **Admin** (per group) — `middleware.RequireAdmin` verifies the role
9. **PostKey** (per route) — `middleware.PostKey` resolves the post_key to a user

On success the auth middleware sets the context: `user`, `user_id`, `email`, `username`, `role`, `claims`.

## Route Groups

The endpoint inventory lives in [api-schema.md](./api-schema.md) and [api-design.md](../api-design.md).

---

## References

- [error-handling.md](./error-handling.md) — layered error contract, the apierr package
- [api-design.md](../api-design.md) — API design rules
- [auth.md](../auth.md) — authentication flows
