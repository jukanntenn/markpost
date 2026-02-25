# Backend Development Guidelines

> Best practices for backend development in this project.

---

## Overview

This directory contains guidelines for backend development using Go with Gin framework and GORM ORM.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Module organization and file layout | ✓ Filled |
| [Database Guidelines](./database-guidelines.md) | GORM patterns, queries, migrations | ✓ Filled |
| [Error Handling](./error-handling.md) | ServiceError, error codes, i18n support | ✓ Filled |
| [Quality Guidelines](./quality-guidelines.md) | Code standards, testing, Swagger | ✓ Filled |
| [Logging Guidelines](./logging-guidelines.md) | Log levels, patterns, what to log | ✓ Filled |

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
Request → Handler → Service → Repository → Database
                ↓
         ServiceError
                ↓
         ErrorResponse (JSON)
```

### Key Conventions
1. **Handlers** - HTTP layer only, no business logic
2. **Services** - Business logic, returns ServiceError
3. **Repositories** - Data access, interfaces for testability
4. **Models** - GORM models with CRUD methods

### Running Tests
```bash
cd backend
go test ./...
```

### Generating Swagger Docs
```bash
cd backend
swag init -g main.go -o docs --parseDependency --parseInternal
```

---

**Language**: All documentation is written in **English**.
