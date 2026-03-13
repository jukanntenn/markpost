# Directory Structure

> How backend code is organized in this project.

---

## Overview

The backend follows **Go standard project layout** with clean architecture:

- **Domain-driven design** - Business logic in `internal/domain/`
- **Layered architecture** - Handler → Service → Repository → Domain
- **Standard Go layout** - `cmd/`, `internal/`, `pkg/`

---

## Directory Layout

```
backend/
├── cmd/                    # CLI commands
│   ├── server/             # HTTP server
│   │   ├── main.go         # Entry point
│   │   └── validators.go   # Custom validators
│   ├── import_fake_posts.go
│   └── prune_expired_posts.go
├── internal/               # Internal packages
│   ├── api/rest/v1/        # REST API handlers
│   ├── config/             # Configuration
│   ├── domain/             # Domain models
│   │   ├── post/           # Post entity
│   │   ├── user/           # User entity
│   │   └── delivery/       # Delivery entity
│   ├── infra/database/     # Database repositories
│   ├── middleware/         # Gin middlewares
│   └── service/            # Business services
│       ├── auth/           # Auth service
│       └── post/           # Post service
├── pkg/                    # Public packages
│   ├── apierr/             # API error types
│   ├── auth/               # JWT utilities
│   ├── crypto/             # Crypto utilities
│   ├── i18n/               # i18n loader
│   └── utils/              # General utilities
├── docs/                   # Swagger docs
└── tools/                  # Dev tools
```

---

## Directory Responsibilities

### cmd/

CLI commands and entry points:

```
cmd/
├── server/
│   ├── main.go             # HTTP server entry
│   └── validators.go       # Gin validators
├── import_fake_posts.go    # Import test data
└── prune_expired_posts.go  # Cleanup job
```

### internal/api/rest/v1/

HTTP handlers (Handler layer):

```
internal/api/rest/v1/
├── auth.go                 # Auth handlers
├── auth_test.go
├── post.go                 # Post handlers
├── post_test.go
├── health.go               # Health check
└── common.go               # Shared utilities
```

### internal/domain/

Domain entities (Model layer):

```
internal/domain/
├── post/
│   ├── post.go             # Post entity
│   └── repository.go       # Repository interface
├── user/
│   ├── user.go             # User entity
│   └── repository.go
└── delivery/
    ├── delivery.go         # Delivery entity
    └── repository.go
```

### internal/infra/database/

Repository implementations:

```
internal/infra/database/
├── database.go             # DB connection
├── post_repository.go
├── post_repository_test.go
├── user_repository.go
├── user_repository_test.go
└── delivery_repository.go
```

### internal/service/

Business services:

```
internal/service/
├── auth/                   # Auth service
│   ├── auth.go
│   ├── auth_test.go
│   ├── jwt.go
│   ├── jwt_test.go
│   └── errors.go
├── post/                   # Post service
│   ├── post.go
│   ├── post_test.go
│   └── errors.go
└── errors.go               # Common service errors
```

### internal/middleware/

Gin middlewares:

```
internal/middleware/
├── auth.go                 # JWT auth
├── admin.go                # Admin check
├── rate_limit.go           # Rate limiting
├── post_key.go             # Post key validation
└── fallback.go             # Fallback handler
```

### internal/config/

Configuration:

```
internal/config/
├── config.go               # Config loading
└── config_test.go
```

### pkg/

Public packages:

```
pkg/
├── apierr/                 # API error types
│   └── apierr.go
├── auth/                   # JWT utilities
│   └── jwt.go
├── crypto/                 # Password hashing
│   ├── password.go
│   └── password_test.go
├── i18n/                   # i18n loader
│   └── loader.go
└── utils/                  # General utilities
    ├── oauth.go
    ├── post_key.go
    └── password.go
```

---

## Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Files | snake_case | `post_repository.go` |
| Packages | lowercase | `database`, `auth` |
| Structs | PascalCase | `PostService` |
| Interfaces | PascalCase | `PostRepository` |
| Functions | PascalCase (exported) | `CreatePost` |

---

## Adding a New Feature

1. Define entity in `internal/domain/{entity}/`
2. Define repository interface in domain
3. Implement repository in `internal/infra/database/`
4. Create service in `internal/service/{entity}/`
5. Create handler in `internal/api/rest/v1/`
6. Add tests alongside source files
