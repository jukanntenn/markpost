# Backend Architecture

## Layers

The backend follows a layered architecture with strict dependency direction:

```
API handlers → Service layer → Repository layer → Domain models
```

Each layer only depends on the layer below it. The API layer depends on service interfaces, services depend on repository interfaces, and all layers reference domain models.

## Directory Structure

```
backend/
├── cmd/server/        — HTTP server entry point, router setup, CLI flags
├── internal/
│   ├── api/rest/v1/   — REST API handlers (Gin handlers)
│   ├── config/        — Configuration loading (Viper + TOML)
│   ├── domain/        — Domain models and repository interfaces
│   │   ├── user/      — User, RefreshToken, TokenBlacklist models
│   │   ├── post/      — Post model and PostRepository interface
│   │   └── delivery/  — Delivery channel model and repository interface
│   ├── infra/         — Repository implementations (GORM)
│   ├── middleware/     — Gin middleware (auth, CORS, rate limiting, panic recovery)
│   └── service/       — Business logic packages
│       ├── auth/      — Authentication, JWT, OAuth
│       ├── post/      — Post CRUD, Markdown rendering
│       ├── delivery/  — Delivery channel management
│       └── admin/     — Admin-only operations
├── pkg/               — Shared packages usable by any layer
│   ├── apierr/        — API error response formatting
│   ├── auth/          — JWT claims and validation helpers
│   ├── crypto/        — Password hashing, key generation
│   ├── i18n/          — i18n helpers
│   └── utils/         — General utilities
├── locales/           — Backend i18n locale files (TOML)
├── templates/         — HTML templates for post rendering
├── tools/             — Dev tools (fake data generator)
└── docs/              — Generated Swagger documentation
```

## Dependency Injection

Services and handlers receive their dependencies through constructor functions. Repositories are defined as interfaces in the `domain` package, implemented by structs in the `infra` package.

```go
// Domain layer defines the interface
type Repository interface {
    GetByID(ctx context.Context, id int) (*User, error)
    // ...
}

// Infra layer implements it
type userRepository struct { db *gorm.DB }
func (r *userRepository) GetByID(ctx context.Context, id int) (*User, error) { ... }

// Wiring in main.go
userRepo = infra.NewUserRepository(dbInstance.DB())
authSvc = auth.NewService(userRepo, tokenRepo, oauthConfig, jwtSvc, "markpost")
```

Handlers follow the same pattern — they accept service interfaces, not concrete types:

```go
func LoginWithUsername(authSvc AuthService) gin.HandlerFunc { ... }
```

This allows test files to inject mock implementations.

## Router Setup

Routes are configured in `cmd/server/main.go` via the `SetupRoutes` function. The Gin engine is created with default middleware (logging + recovery), then:

1. HTML templates loaded from `templates/*`
2. Trusted proxies configured
3. i18n middleware attached (loads locale files from `./locales`)
4. Panic recovery middleware (`middleware.Fallback`)
5. CORS middleware
6. Rate limiting middleware
7. Route groups registered

## Middleware Chain

The middleware chain executes in this order for each request:

1. **Gin default** — Logging + recovery
2. **i18n** (`ginI18n.Localize`) — Detects language from `Accept-Language` header
3. **Fallback** (`middleware.Fallback`) — Recovers from panics, returns 500
4. **CORS** (`cors.New`) — Handles preflight requests, sets CORS headers
5. **Rate limiting** (`middleware.RateLimitByIP`) — Per-IP rate limiting via tollbooth
6. **Auth** (per group) — `middleware.AuthWithBlacklist` validates JWT and checks blacklist
7. **Admin** (per group) — `middleware.RequireAdmin` checks user role
8. **PostKey** (per route) — `middleware.PostKey` resolves post_key param to a user

Auth middleware sets these context values on success: `user`, `user_id`, `email`, `username`, `role`, `claims`.

## Route Groups

```
/api/v1
├── /health                          — Public health check
├── /oauth
│   ├── GET  /url                    — GitHub OAuth URL generation
│   └── POST /login                  — GitHub OAuth login
├── /auth
│   ├── POST /login                  — Username/password login
│   └── POST /refresh                — Token refresh
└── (authenticated group)
    ├── GET  /post_key               — Get current user's post key
    ├── POST /auth/logout            — Logout (blacklist token)
    ├── POST /auth/change-password   — Change password
    ├── GET  /posts                  — List user's posts
    ├── /delivery/channels           — CRUD delivery channels
    └── /admin                       — Admin-only endpoints

Root-level routes (outside /api/v1):
├── POST /:post_key                  — Create post (public, validated by PostKey middleware)
└── GET  /:id                        — View post (public, renders HTML or returns raw markdown)
```
