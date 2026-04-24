# Backend Error Handling

> Structured error handling system with ServiceError, error codes, and i18n support.

## Architecture

```
Repository Layer → returns raw errors
Service Layer    → wraps with ServiceError
Handler Layer    → uses apperrors.RespondError
HTTP Response    → JSON with i18n message
```

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| `ErrCode` | `services/errors.go` | Error code constants |
| `ServiceError` | `services/errors.go` | Domain error type |
| `ErrorResponse` | `errors/errors.go` | API response structure |
| `RespondError` | `errors/errors.go` | Error response handler |

## ServiceError Structure

```go
type ServiceError struct {
    Code        ErrCode        // Error code for programmatic handling
    Description string         // Human-readable description
    Err         error          // Wrapped underlying error
    Details     []ServiceError // Nested errors for validation
}
```

## Error Codes

```go
type ErrCode string

const (
    // Authentication
    ErrInvalidCredentials ErrCode = "invalid_credentials"
    ErrInvalidPassword    ErrCode = "invalid_password"
    ErrUnauthorized       ErrCode = "unauthorized"
    ErrInvalidToken       ErrCode = "invalid_token"

    // Request
    ErrValidation         ErrCode = "validation"
    ErrNotFound           ErrCode = "not_found"
    ErrInvalidRequest     ErrCode = "invalid_request"
    ErrMissingStateParam  ErrCode = "missing_state_param"
    ErrMissingCode        ErrCode = "missing_code"

    // Authorization
    ErrMissingAuthorizationHeader ErrCode = "missing_authorization_header"
    ErrInvalidPostKey             ErrCode = "invalid_post_key"

    // Internal
    ErrInternal           ErrCode = "internal"
    ErrFailedGetUser      ErrCode = "failed_get_user"

    // Validation detail
    ErrRequired       ErrCode = "required"
    ErrMinLength      ErrCode = "min_length"
    ErrFieldViolation ErrCode = "validation"
)
```

## Creating ServiceErrors

### Simple Error (no underlying cause)

```go
NewServiceError(code ErrCode, description string) *ServiceError
```

### Wrapped Error (preserves stack trace)

```go
NewServiceErrorWrap(code ErrCode, description string, err error) *ServiceError
```

### Validation Error with Field Details

```go
NewServiceErrorWithDetails(code ErrCode, description string, details []ServiceError) *ServiceError
```

## Error Handling Rules

### In Services

- Always wrap errors with `ServiceError` — never return raw errors
- Check for specific error types (`models.ErrNotFound`) before defaulting to `ErrInternal`
- Include context in descriptions (relevant IDs, names, values)
- Include field-level details for validation errors

### In Handlers

- Always use `apperrors.RespondError(c, err)` for all error responses
- Never write custom JSON error responses directly
- Never handle service errors in handlers — propagate them

## API Error Response Format

```go
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

## Error Code → HTTP Status Mapping

| Error Code | HTTP Status |
|------------|-------------|
| `ErrInvalidCredentials` | 401 |
| `ErrInvalidPassword` | 400 |
| `ErrUnauthorized` | 401 |
| `ErrNotFound` | 404 |
| `ErrValidation` | 400 |
| `ErrInternal` | 500 |
| `ErrInvalidToken` | 401 |
| `ErrMissingAuthorizationHeader` | 401 |
| `ErrInvalidPostKey` | 403 |
| `ErrMissingStateParam` | 400 |
| `ErrMissingCode` | 400 |
| `ErrInvalidRequest` | 400 |
| `ErrFailedGetUser` | 500 |

## i18n Support

Client sends `Accept-Language` header. `RespondError` uses `ginI18n.MustGetMessage` to get localized message from `locales/active.{lang}.toml`.

### Adding a New Error Code (3 places)

1. **Define code** in `services/errors.go`
2. **Add mapping** in `errors/errors.go` `serviceErrorMappings` with HTTP status and i18n message ID
3. **Add translations** in `locales/active.en-us.toml` and `locales/active.zh-hans.toml`

## Checking ServiceError

```go
func AsServiceError(err error) (*ServiceError, bool) {
    var se *ServiceError
    if errors.As(err, &se) { return se, true }
    return nil, false
}
```

## Common Mistakes

### 1. Returning Raw Errors from Services

```go
// Bad — loses context and error code
return "", err

// Good — wraps with ServiceError
return "", NewServiceErrorWrap(ErrInternal, "create post failed", err)
```

### 2. Not Using RespondError in Handlers

```go
// Bad
c.JSON(500, gin.H{"error": err.Error()})

// Good
apperrors.RespondError(c, err)
```

### 3. Wrong Error Code for HTTP Semantics

```go
// Bad — 500 for not found
return NewServiceErrorWrap(ErrInternal, "user not found", err)

// Good — correct 404
return NewServiceErrorWrap(ErrNotFound, "user not found", err)
```

### 4. Missing i18n Message

When adding a new error code, you must also add the mapping in `serviceErrorMappings` and translations in locale files — otherwise it falls through to an internal error.

### 5. Not Checking Specific Error Types

```go
// Bad — treats all DB errors the same
post, err := s.postRepo.GetPostByQID(qid)
if err != nil { return "", NewServiceErrorWrap(ErrInternal, "get post failed", err) }

// Good — distinguishes not-found from other errors
post, err := s.postRepo.GetPostByQID(qid)
if err != nil {
    if err == models.ErrNotFound {
        return "", NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("post with qid %s not found", qid), err)
    }
    return "", NewServiceErrorWrap(ErrInternal, fmt.Sprintf("get post with qid %s failed", qid), err)
}
```

## Best Practices

1. **Layered handling**: Repository returns raw → Service wraps with ServiceError → Handler uses RespondError
2. **Specific descriptions**: Include relevant IDs/names (`"post with qid %s not found"`), not generic (`"not found"`)
3. **Secure descriptions**: Never expose passwords, tokens, or internal details
4. **Validation details**: Always include field-level details for validation errors
5. **Preserve stack traces**: Always use `NewServiceErrorWrap` (not `NewServiceError`) when an underlying error exists

## Checklist

- [ ] All service errors wrapped with `ServiceError`
- [ ] All handler errors use `apperrors.RespondError`
- [ ] Error codes match HTTP status semantics
- [ ] New error codes have i18n translations
- [ ] Error descriptions are specific and helpful
- [ ] Validation errors include field details
