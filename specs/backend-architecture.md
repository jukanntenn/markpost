# Backend Architecture

> Project structure, layered architecture, and database patterns.

## Directory Layout

```
backend/
├── cmd/
│   ├── server/             # HTTP server entry (main.go, validators.go)
│   ├── import_fake_posts.go
│   └── prune_expired_posts.go
├── internal/
│   ├── api/rest/v1/        # REST API handlers (HTTP layer)
│   ├── config/             # Configuration
│   ├── domain/             # Domain models + repository interfaces
│   │   ├── post/
│   │   ├── user/
│   │   └── delivery/
│   ├── infra/database/     # Repository implementations
│   ├── middleware/         # Gin middlewares (auth, admin, rate limit, etc.)
│   └── service/            # Business services
│       ├── auth/
│       └── post/
├── pkg/
│   ├── apierr/             # API error types
│   ├── auth/               # JWT utilities
│   ├── crypto/             # Crypto utilities
│   ├── i18n/               # i18n loader
│   └── utils/              # General utilities
├── docs/                   # Swagger docs
└── tools/                  # Dev tools
```

## Layered Architecture

```
┌─────────────────────────────────────────┐
│  Handler Layer (HTTP)                   │
│  - Parse/validate requests              │
│  - Extract context from gin.Context      │
│  - Format responses                     │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Service Layer (Business Logic)         │
│  - Business rules                       │
│  - Orchestrate repository calls         │
│  - Wrap errors with ServiceError        │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Repository Layer (Data Access)         │
│  - Database operations                  │
│  - Define interfaces for testability    │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│  Domain Layer (Entities)                │
│  - GORM models                          │
│  - Domain errors (e.g., ErrNotFound)    │
└─────────────────────────────────────────┘
```

## Layer Responsibilities

### Handler Layer (`internal/api/rest/v1/`)
- Parse and validate incoming requests (`ShouldBindJSON`, `ShouldBindQuery`)
- Extract user context from `gin.Context`
- Call service methods — never access DB directly
- Format HTTP responses or delegate errors to `RespondError`

### Service Layer (`internal/service/`)
- Implement business rules
- Orchestrate repository calls
- Wrap all errors with `ServiceError` (never return raw errors to handlers)

### Repository Layer (`internal/infra/database/`)
- Abstract all database operations behind interfaces
- Return raw errors or domain errors (`models.ErrNotFound`)
- No business logic, no error wrapping with context

### Domain Layer (`internal/domain/`)
- Define GORM model structs
- Define repository interfaces
- Define domain-level errors

## Dependency Injection

```go
// Repository
func NewPostRepository(db *gorm.DB) PostRepository { ... }

// Service
func NewPostService(repo PostRepository) *PostService { ... }

// Handler
func CreatePost(svc *PostService) gin.HandlerFunc { ... }
```

## Layer Violations

| Violation | Correct |
|-----------|---------|
| Handler accessing DB directly | Handler → Service → Repository |
| Service returning raw errors | Service wraps with `ServiceError` |
| Repository containing business logic | Repository returns raw errors |
| Domain layer with HTTP logic | Domain is pure data |

## Error Flow

```
Repository → returns raw error
Service    → wraps with ServiceError
Handler    → uses RespondError(c, err)
HTTP       → JSON response with i18n message
```

## Database Patterns

- **ORM**: GORM v2
- **Production**: PostgreSQL; **Dev/Test**: SQLite (in-memory)
- **Migrations**: Auto-migrate on startup

### Query Organization

| Pattern | When to Use |
|---------|-------------|
| Model-level methods | Simple CRUD (`models.GetPost()`, `models.GetPosts()`) |
| Repository methods | Complex queries, business-specific operations |
| `map[string]any` filters | Flexible query conditions |
| Offset/limit pagination | All list queries |

### Adding New Models

1. Define model struct with GORM tags in `internal/domain/{entity}/`
2. Add to `AutoMigrate()` call in `main.go`
3. GORM creates/updates the table on next startup

### Model Definition

```go
type Post struct {
    ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
    QID       string    `json:"qid" gorm:"unique;not null;column:qid"`
    Title     string    `json:"title" gorm:"not null"`
    Body      string    `json:"body" gorm:"not null;type:text"`
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
    UserID    int       `json:"user_id" gorm:"index;not null;column:user_id"`
    User      User      `json:"user" gorm:"constraint:OnDelete:CASCADE"`
}
```

### Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Files | snake_case | `post_repository.go` |
| Packages | lowercase | `database`, `auth` |
| Structs | PascalCase | `PostService` |
| Interfaces | PascalCase | `PostRepository` |
| Tables | snake_case plural | `users`, `delivery_channels` |
| Columns | snake_case | `user_id`, `created_at` |

### Common Mistakes

1. **Not wrapping errors** — always use `fmt.Errorf("Method: %w", err)`
2. **Not handling `ErrRecordNotFound`** — convert to domain `ErrNotFound`
3. **Missing cascade delete** — use `gorm:"constraint:OnDelete:CASCADE"` on relationships

## Adding a New Feature

1. Define entity in `internal/domain/{entity}/`
2. Define repository interface in domain
3. Implement repository in `internal/infra/database/`
4. Create service in `internal/service/{entity}/`
5. Create handler in `internal/api/rest/v1/`
6. Add tests alongside source files
