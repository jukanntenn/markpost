# Database Guidelines

> Database patterns and conventions for this project.

---

## Overview

This project uses **GORM** as the ORM with support for both PostgreSQL and SQLite. Database operations are abstracted through the repository pattern.

- **ORM**: GORM v2
- **Supported databases**: PostgreSQL (production), SQLite (development/testing)
- **Migrations**: Auto-migrate on startup

---

## Query Patterns

### Model-Level Operations

Simple CRUD operations are defined as methods on the model:

```go
// models/post.go:22-29
func (model *Post) Create(database *Database) error {
    db := database.DB()
    if err := db.Create(model).Error; err != nil {
        return fmt.Errorf("Post.Create: %w", err)
    }
    return nil
}
```

### Repository-Level Operations

Complex queries and business operations go in repositories:

```go
// repositories/post.go:91-98
func (r *PostRepo) GetPostsByUserID(userID int, offset int, limit int) ([]models.Post, error) {
    posts, err := models.GetPosts(r.database, map[string]any{"user_id": userID}, offset, limit)
    if err != nil {
        return nil, err
    }
    return posts, nil
}
```

### Query Building with Map

Use `map[string]any` for flexible query conditions:

```go
// models/post.go:31-44
func GetPost(database *Database, query map[string]any) (*Post, error) {
    db := database.DB()
    var post Post
    err := db.Take(&post, query).Error
    // ...
}
```

### Pagination

Always use offset/limit for list queries:

```go
// models/post.go:46-57
func GetPosts(database *Database, query map[string]any, offset, limit int) ([]Post, error) {
    var models []Post
    err := db.Where(query).Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error
    // ...
}
```

---

## Migrations

### Auto-Migrate

Migrations are handled automatically on application startup:

```go
// main.go:170-172
if err := database.DB().AutoMigrate(&models.User{}, &models.Post{}, &models.DeliveryChannel{}); err != nil {
    log.Fatalf("Failed to migrate database: %v", err)
}
```

### Adding New Models

1. Define model struct with GORM tags in `models/`
2. Add model to AutoMigrate call in `main.go`
3. GORM will create/update tables on next startup

---

## Naming Conventions

### Table Names

GORM uses snake_case plural by default:
- `User` → `users`
- `Post` → `posts`
- `DeliveryChannel` → `delivery_channels`

### Column Names

Use `gorm:"column:name"` tag to specify column name:

```go
// models/post.go:13
QID string `json:"qid" gorm:"unique;not null;column:qid"`
```

### Index Names

GORM auto-generates index names. Use `gorm:"index"` for simple indexes:

```go
// models/post.go:18
UserID int `json:"user_id" gorm:"index;not null;column:user_id"`
```

---

## Model Definition

Standard model structure:

```go
// models/post.go:11-20
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

---

## Database Wrapper

The `Database` struct wraps GORM DB:

```go
// models/database.go:15-21
type Database struct {
    db *gorm.DB
}

func (d *Database) DB() *gorm.DB {
    return d.db
}
```

### Test Database

Use in-memory SQLite for tests:

```go
// models/database.go:80-95
func NewTestDatabase() (*Database, error) {
    gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    // ...
}
```

---

## Common Mistakes

### 1. Forgetting to wrap errors

```go
// Bad
if err != nil {
    return nil, err
}

// Good
if err != nil {
    return nil, fmt.Errorf("GetPost: %w", err)
}
```

### 2. Not handling ErrRecordNotFound

```go
// models/post.go:36-41
err := db.Take(&post, query).Error
if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, ErrNotFound
    }
    return nil, fmt.Errorf("GetPost: %w", err)
}
```

### 3. Missing cascade delete for relationships

```go
User User `json:"user" gorm:"constraint:OnDelete:CASCADE"`
```
