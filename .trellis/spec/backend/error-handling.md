# Error Handling

> How errors are handled in this project.

---

## Overview

This project uses a structured error handling system with:
- **ServiceError**: Domain-level errors with error codes
- **Error codes**: Typed error identifiers for consistent API responses
- **i18n support**: Error messages support multiple languages

---

## Error Types

### ServiceError Structure

```go
// services/errors.go:32-37
type ServiceError struct {
    Code        ErrCode        // Error code for programmatic handling
    Description string         // Human-readable description
    Err         error          // Wrapped underlying error
    Details     []ServiceError // Nested errors for validation
}
```

### Error Codes

Define all error codes as constants:

```go
// services/errors.go:7-30
type ErrCode string

const (
    ErrInvalidCredentials ErrCode = "invalid_credentials"
    ErrInvalidPassword    ErrCode = "invalid_password"
    ErrUnauthorized       ErrCode = "unauthorized"
    ErrInternal           ErrCode = "internal"
    ErrValidation         ErrCode = "validation"
    ErrNotFound           ErrCode = "not_found"
    // ... more codes
)
```

### Creating ServiceErrors

```go
// Simple error
services.NewServiceError(services.ErrNotFound, "post not found")

// Wrapped error (preserve stack trace)
services.NewServiceErrorWrap(services.ErrInternal, "database error", err)

// Validation error with details
services.NewServiceErrorWithDetails(services.ErrValidation, "validation failed", details)
```

---

## Error Handling Patterns

### In Services

Always wrap errors with context:

```go
// services/post.go:33-37
func (s *PostService) CreatePost(title, body string, userID int) (string, error) {
    post, err := s.postRepo.CreatePost(title, body, userID)
    if err != nil {
        return "", NewServiceErrorWrap(ErrInternal, "create post failed", err)
    }
    // ...
}
```

### Checking ServiceError

```go
// services/errors.go:55-61
func AsServiceError(err error) (*ServiceError, bool) {
    var se *ServiceError
    if errors.As(err, &se) {
        return se, true
    }
    return nil, false
}
```

### In Handlers

Use `apperrors.RespondError` to send error response:

```go
// handlers/post.go:54-58
id, err := postSvc.CreatePost(req.Title, req.Body, user.ID)
if err != nil {
    apperrors.RespondError(c, err)
    return
}
```

---

## API Error Responses

### Standard Response Format

```go
// errors/errors.go:14-24
type FieldError struct {
    Field   string `json:"field,omitempty"`
    Code    string `json:"code"`
    Message string `json:"message"`
}

type ErrorResponse struct {
    Code    string       `json:"code"`
    Message string       `json:"message"`
    Errors  []FieldError `json:"errors,omitempty"`
}
```

### Error Code to HTTP Status Mapping

```go
// errors/errors.go:31-123
var serviceErrorMappings = map[services.ErrCode]serviceErrorMapping{
    services.ErrInvalidCredentials: {
        Status: http.StatusUnauthorized,
        Message: &i18n.Message{ID: "error.invalid_credentials", Other: "Invalid username or password"},
    },
    services.ErrNotFound: {
        Status: http.StatusNotFound,
        Message: &i18n.Message{ID: "error.not_found", Other: "Not Found"},
    },
    // ...
}
```

### Example Response

```json
{
    "code": "validation",
    "message": "Request validation failed",
    "errors": [
        {"field": "title", "code": "required", "message": "This field is required"}
    ]
}
```

---

## i18n Support

Errors support internationalization through gin-i18n:

```go
// errors/errors.go:155-157
message := ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
    DefaultMessage: mapping.Message,
})
```

The client's `Accept-Language` header determines the language.

---

## Common Mistakes

### 1. Returning raw errors instead of ServiceError

```go
// Bad
return nil, err

// Good
return nil, NewServiceErrorWrap(ErrInternal, "operation failed", err)
```

### 2. Not checking error type in handlers

```go
// Bad - assumes all errors are the same
if err != nil {
    c.JSON(500, gin.H{"error": err.Error()})
}

// Good - use RespondError for proper handling
if err != nil {
    apperrors.RespondError(c, err)
    return
}
```

### 3. Using wrong error code

Match error codes to HTTP semantics:
- `ErrNotFound` → 404
- `ErrUnauthorized` → 401
- `ErrValidation` → 400
- `ErrInternal` → 500

### 4. Missing i18n message

Always add corresponding i18n message when adding new error code:
1. Add code to `services/errors.go`
2. Add mapping to `errors/errors.go`
3. Add translations to `locales/*.toml`
