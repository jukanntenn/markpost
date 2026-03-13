# Backend Development Guidelines

> Best practices for backend development in this project.

---

## Overview

Guidelines for backend development using Go with Gin framework and GORM ORM.

---

## Guidelines Index

| Guide | Description |
|-------|-------------|
| [Directory Structure](./directory-structure.md) | Module organization |
| [Layered Architecture](./layered-architecture.md) | Handler → Service → Repository → Domain |
| [Database Guidelines](./database-guidelines.md) | GORM patterns |
| [Error Handling](./error-handling.md) | Error handling patterns |
| [Quality Guidelines](./quality-guidelines.md) | Testing, Swagger |
| [Logging Guidelines](./logging-guidelines.md) | Logging patterns |

---

## Quick Reference

### Technology Stack
- **Framework**: Gin
- **ORM**: GORM v2
- **Database**: PostgreSQL (production), SQLite (dev/test)
- **Validation**: go-playground/validator
- **i18n**: gin-contrib/i18n
- **CLI**: urfave/cli

### Architecture Pattern
```
Request → Handler → Service → Repository → Domain
                ↓
         Error wrapping
                ↓
         JSON Response
```

### Key Conventions
1. **Handlers** - HTTP layer only
2. **Services** - Business logic
3. **Repositories** - Data access with interfaces
4. **Domain** - GORM models

### Common Commands
```bash
cd backend
go run cmd/server/main.go    # Run server
go test ./...                # Run tests
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```
