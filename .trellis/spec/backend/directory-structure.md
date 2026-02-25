# Directory Structure

> How backend code is organized in this project.

---

## Overview

The backend follows a layered architecture pattern with clear separation of concerns:
- **Handlers** - HTTP request handling and validation
- **Services** - Business logic and orchestration
- **Repositories** - Data access layer
- **Models** - Data structures and database schemas

---

## Directory Layout

```
backend/
├── cmd/                    # CLI commands (prune, import, etc.)
├── conf/                   # Configuration loading and management
├── docs/                   # Swagger documentation (generated)
├── errors/                 # Error response handling utilities
├── handlers/               # HTTP request handlers
├── middlewares/            # Gin middlewares (auth, rate limit, etc.)
├── models/                 # Data models and database operations
├── repositories/           # Repository layer for data access
├── services/               # Business logic layer
├── templates/              # Server-side HTML templates
├── tools/                  # Development tools
├── locales/                # i18n translation files
├── main.go                 # Application entrypoint
├── routes.go               # Route definitions
└── validators.go           # Custom request validators
```

---

## Layer Responsibilities

### handlers/ - HTTP Layer
- Parse and validate incoming requests
- Extract user context from gin.Context
- Call service methods
- Format and return HTTP responses
- Define request/response DTOs

**Example**: `handlers/post.go:40-62` - CreatePost handler

### services/ - Business Logic Layer
- Implement business rules and orchestration
- Transform data between layers
- Handle cross-cutting concerns (e.g., delivery dispatch)
- Return ServiceError for error cases

**Example**: `services/post.go:33-47` - CreatePost service method

### repositories/ - Data Access Layer
- Abstract database operations
- Define interfaces for testability
- Handle query building and execution

**Example**: `repositories/post.go:32-50` - CreatePost repository

### models/ - Data Structures
- Define GORM models with tags
- Implement model-level CRUD methods (for simple operations)
- Define domain errors

**Example**: `models/post.go:11-20` - Post model definition

---

## Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Files | lowercase, underscore for multi-word | `delivery_channel.go` |
| Packages | lowercase, single word preferred | `handlers`, `services` |
| Structs | PascalCase | `PostService`, `UserRepoInterface` |
| Interfaces | PascalCase with `Interface` suffix | `PostRepoInterface` |
| Functions | PascalCase (exported), camelCase (private) | `CreatePost`, `newTestRouter` |

---

## Adding a New Feature

1. Define model in `models/` if new entity
2. Create repository interface and implementation in `repositories/`
3. Create service in `services/`
4. Create handler in `handlers/`
5. Register routes in `routes.go`
6. Add tests alongside source files

---

## Examples

- **Complete CRUD feature**: `handlers/delivery_channels.go` + `services/delivery_channels.go` + `repositories/delivery_channel.go`
- **CLI command**: `cmd/prune_expired_posts.go`
- **Middleware**: `middlewares/auth.go`
