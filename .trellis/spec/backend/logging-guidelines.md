# Logging Guidelines

> How logging is done in this project.

---

## Overview

This project uses Go's standard `log` package for logging. Logging is minimal and focused on key application events.

- **Logging library**: Go standard `log` package
- **Format**: Plain text with timestamp
- **Output**: stdout/stderr

---

## Log Levels

The project uses Go's standard `log` package without explicit log levels. Convention:

| Log Function | When to Use |
|--------------|-------------|
| `log.Printf` | Info-level events (startup, configuration) |
| `log.Fatalf` | Fatal errors that should terminate the application |
| `log.Printf` with "error"/"failed" in message | Error conditions |

---

## Structured Logging

### Startup Events

Log important initialization steps:

```go
// main.go:195-198
log.Printf("Initializing first admin user: %s", cfg.Admin.InitialUsername)
if err := authSvc.InitializeFirstAdmin(cfg.Admin.InitialUsername); err != nil {
    log.Fatalf("Failed to initialize first admin: %v", err)
}
```

```go
// main.go:227-233
log.Printf("Initializing rate limiting for create post...")
log.Println("Server starting...")
log.Println("Visit http://" + visitHost + ":" + strconv.FormatUint(uint64(cfg.Server.Port), 10))
```

### Database Events

```go
// models/database.go:74
log.Printf("initialized user: %s", username)
```

### Error Context

Include context when logging errors:

```go
// errors/errors.go:143-144
log.Printf("unexpected error: %v", err)

// errors/errors.go:150
log.Printf("unknown service error code: %s detail=%s err=%v", se.Code, se.Description, se.Err)
```

---

## What to Log

### DO Log
- Application startup and configuration
- Initialization of critical components (database, services)
- Admin user creation
- Server listen address
- Unexpected errors with context
- Unknown error codes (for debugging)

### DO NOT Log
- User passwords or password hashes
- JWT tokens or API keys
- Post content (user data)
- Personal identifiable information (PII)
- Request bodies with sensitive data

---

## Logging Patterns

### Startup Sequence

```go
// main.go:152-237
func serve(configPath string) {
    // 1. Config loading
    if err := conf.LoadConfig(configPath); err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 2. Database initialization
    dbInstance, err := models.NewDatabase(cfg.DB.DSN)
    if err != nil {
        log.Fatalf("Failed to init database: %v", err)
    }

    // 3. Service initialization
    log.Printf("Initializing first admin user: %s", cfg.Admin.InitialUsername)

    // 4. Server ready
    log.Println("Server starting...")
    log.Println("Visit http://" + visitHost + ":" + strconv.FormatUint(uint64(cfg.Server.Port), 10))
}
```

### Error Logging Pattern

```go
// errors/errors.go:140-152
func RespondError(c *gin.Context, err error) {
    se, ok := services.AsServiceError(err)
    if !ok {
        log.Printf("unexpected error: %v", err)
        writeInternalError(c)
        return
    }

    mapping, ok := serviceErrorMappings[se.Code]
    if !ok {
        log.Printf("unknown service error code: %s detail=%s err=%v", se.Code, se.Description, se.Err)
        writeInternalError(c)
        return
    }
    // ...
}
```

---

## Common Mistakes

### 1. Using fmt.Println instead of log

```go
// Bad
fmt.Println("Server starting")

// Good
log.Println("Server starting")
```

### 2. Not including context in error logs

```go
// Bad
log.Printf("Error: %v", err)

// Good
log.Printf("Failed to initialize first admin: %v", err)
```

### 3. Logging sensitive information

```go
// Bad - logs user password
log.Printf("User login: %s, password: %s", username, password)

// Good - only logs username
log.Printf("User logged in: %s", username)
```

### 4. Using log.Fatal in library code

```go
// Bad - terminates the application
func (s *Service) DoSomething() {
    if err != nil {
        log.Fatalf("error: %v", err)
    }
}

// Good - return error to caller
func (s *Service) DoSomething() error {
    if err != nil {
        return fmt.Errorf("operation failed: %w", err)
    }
    return nil
}
```
