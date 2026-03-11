# Error Handling

> How errors are handled in this project.

---

## Overview

This project uses a structured error handling system with:
- **ServiceError**: Domain-level errors with error codes
- **Error codes**: Typed error identifiers for consistent API responses
- **i18n support**: Error messages support multiple languages
- **Layered handling**: Services wrap errors, handlers respond uniformly

---

## Architecture

### Error Flow

```
Repository Layer
    ↓ (returns raw errors)
Service Layer
    ↓ (wraps with ServiceError)
Handler Layer
    ↓ (uses apperrors.RespondError)
HTTP Response (JSON with i18n)
```

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| `ErrCode` | `services/errors.go` | Error code constants |
| `ServiceError` | `services/errors.go` | Domain error type |
| `ErrorResponse` | `errors/errors.go` | API response structure |
| `RespondError` | `errors/errors.go` | Error response handler |

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

All error codes are defined as constants in `services/errors.go`:

```go
// services/errors.go:7-30
type ErrCode string

const (
    // Authentication errors
    ErrInvalidCredentials ErrCode = "invalid_credentials"
    ErrInvalidPassword    ErrCode = "invalid_password"
    ErrUnauthorized       ErrCode = "unauthorized"
    ErrInvalidToken       ErrCode = "invalid_token"

    // Request errors
    ErrValidation         ErrCode = "validation"
    ErrNotFound           ErrCode = "not_found"
    ErrInvalidRequest     ErrCode = "invalid_request"
    ErrMissingStateParam  ErrCode = "missing_state_param"
    ErrMissingCode        ErrCode = "missing_code"

    // Authorization errors
    ErrMissingAuthorizationHeader ErrCode = "missing_authorization_header"
    ErrInvalidPostKey             ErrCode = "invalid_post_key"

    // Internal errors
    ErrInternal           ErrCode = "internal"
    ErrFailedGetUser      ErrCode = "failed_get_user"

    // Validation detail errors
    ErrRequired       ErrCode = "required"
    ErrMinLength      ErrCode = "min_length"
    ErrFieldViolation ErrCode = "validation"
)
```

---

## Creating ServiceErrors

### Three Constructor Functions

#### 1. Simple Error

```go
// services/errors.go:63-68
func NewServiceError(code ErrCode, description string) *ServiceError
```

**When to use**: Error without underlying cause

**Example**:
```go
// services/admin.go:49
return NewServiceError(ErrNotFound, fmt.Sprintf("user with ID %d not found", id))
```

#### 2. Wrapped Error

```go
// services/errors.go:70-76
func NewServiceErrorWrap(code ErrCode, description string, err error) *ServiceError
```

**When to use**: Wrapping an underlying error (preserves stack trace)

**Example**:
```go
// services/post.go:36
return "", NewServiceErrorWrap(ErrInternal, "create post failed", err)
```

#### 3. Validation Error with Details

```go
// services/errors.go:78-84
func NewServiceErrorWithDetails(code ErrCode, description string, details []ServiceError) *ServiceError
```

**When to use**: Validation errors with field-level details

**Example**:
```go
// services/admin.go:58-60
return nil, NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
    {Code: ErrFieldViolation, Description: "username"},
})
```

---

## Error Handling Patterns

### In Services

**Rule**: Always wrap errors with context using ServiceError

#### Pattern 1: Database Errors

```go
// services/post.go:49-57
func (s *PostService) RenderPostHTML(qid string) (string, string, error) {
    post, err := s.postRepo.GetPostByQID(qid)
    if err != nil {
        if err == models.ErrNotFound {
            // Specific error type → specific error code
            return "", "", NewServiceErrorWrap(ErrNotFound,
                fmt.Sprintf("post with qid %s not found", qid), err)
        }
        // Generic error → internal error code
        return "", "", NewServiceErrorWrap(ErrInternal,
            fmt.Sprintf("get post with qid %s failed", qid), err)
    }
    // ...
}
```

#### Pattern 2: External Service Errors

```go
// services/auth.go:48-52
func (s *AuthService) LoginWithGitHub(ctx context.Context, code string) (*models.User, *JWTTokenPair, error) {
    token, err := s.oauth.Exchange(ctx, code)
    if err != nil {
        return nil, nil, NewServiceErrorWrap(ErrUnauthorized, "oauth exchange failed", err)
    }
    // ...
}
```

#### Pattern 3: Business Logic Errors

```go
// services/admin.go:54-64
func (s *AdminService) CreateUser(username, password string) (*models.User, error) {
    user, err := s.users.CreateUser(username, password)
    if err != nil {
        if err.Error() == "username is already taken" {
            // Business rule violation → validation error with details
            return nil, NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
                {Code: ErrFieldViolation, Description: "username"},
            })
        }
        return nil, NewServiceErrorWrap(ErrInternal, "create user failed", err)
    }
    return user, nil
}
```

### In Handlers

**Rule**: Use `apperrors.RespondError` for all error responses

#### Pattern 1: Service Error

```go
// handlers/post.go:42-47
func CreatePost(postSvc PostServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        user, ok := ExtractUser(c)
        if !ok {
            err := services.NewServiceErrorWrap(services.ErrFailedGetUser,
                "failed to get user from context", nil)
            apperrors.RespondError(c, err)
            return
        }
        // ...
    }
}
```

#### Pattern 2: Validation Error

```go
// handlers/admin.go:61-68
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        apperrors.RespondError(c, services.NewServiceErrorWithDetails(
            services.ErrValidation,
            "request validation failed",
            []services.ServiceError{
                {Code: services.ErrFieldViolation, Description: "id"},
            }))
        return
    }
    // ...
}
```

#### Pattern 3: Propagate Service Error

```go
// handlers/auth.go:28-35
func GenerateGitHubOAuthURL(authSvc GitHubAuthURLGenerator) gin.HandlerFunc {
    return func(c *gin.Context) {
        url, err := authSvc.GenerateGitHubAuthURL(c.Request.Context())
        if err != nil {
            apperrors.RespondError(c, err)  // Service already wrapped it
            return
        }
        c.JSON(http.StatusOK, gin.H{"url": url})
    }
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

The mapping is defined in `errors/errors.go:31-123`:

| Error Code | HTTP Status | i18n Message ID |
|------------|-------------|-----------------|
| `ErrInvalidCredentials` | 401 | `error.invalid_credentials` |
| `ErrInvalidPassword` | 400 | `error.invalid_current_password` |
| `ErrUnauthorized` | 401 | `error.unauthorized` |
| `ErrNotFound` | 404 | `error.not_found` |
| `ErrValidation` | 400 | `error.validation_failed` |
| `ErrInternal` | 500 | `error.internal` |
| `ErrInvalidToken` | 401 | `error.invalid_token` |
| `ErrMissingAuthorizationHeader` | 401 | `error.missing_authorization_header` |
| `ErrInvalidPostKey` | 403 | `error.invalid_post_key` |
| `ErrMissingStateParam` | 400 | `error.missing_state_param` |
| `ErrMissingCode` | 400 | `error.missing_code` |
| `ErrInvalidRequest` | 400 | `error.invalid_request` |
| `ErrFailedGetUser` | 500 | `error.failed_get_user` |

### Example Responses

#### Simple Error

```json
{
    "code": "not_found",
    "message": "Not Found"
}
```

#### Validation Error with Details

```json
{
    "code": "validation",
    "message": "Request validation failed",
    "errors": [
        {
            "field": "username",
            "code": "validation",
            "message": "Invalid request format"
        }
    ]
}
```

---

## i18n Support

### How It Works

1. Client sends `Accept-Language` header (e.g., `zh-hans`, `en-us`)
2. `RespondError` uses `ginI18n.MustGetMessage` to get localized message
3. Message is looked up in `locales/active.{lang}.toml`

### Adding New Error Code

When adding a new error code, you must update **three places**:

#### 1. Define Error Code

```go
// services/errors.go
const (
    // ... existing codes
    ErrNewErrorCase ErrCode = "new_error_case"
)
```

#### 2. Add Mapping

```go
// errors/errors.go
var serviceErrorMappings = map[services.ErrCode]serviceErrorMapping{
    // ... existing mappings
    services.ErrNewErrorCase: {
        Status: http.StatusBadRequest,
        Message: &i18n.Message{
            ID:    "error.new_error_case",
            Other: "New error case occurred",
        },
    },
}
```

#### 3. Add Translations

```toml
# locales/active.en-us.toml
["error.new_error_case"]
other = "New error case occurred"

# locales/active.zh-hans.toml
["error.new_error_case"]
other = "新错误案例发生"
```

### Validation Field Messages

For validation detail errors, use `validationFieldMessages`:

```go
// errors/errors.go:125-138
var validationFieldMessages = map[services.ErrCode]*i18n.Message{
    services.ErrRequired: {
        ID:    "error.validation_required",
        Other: "This field is required",
    },
    services.ErrMinLength: {
        ID:    "error.validation_min_length",
        Other: "Value does not meet minimum length",
    },
    services.ErrFieldViolation: {
        ID:    "error.invalid_request",
        Other: "Invalid request format",
    },
}
```

---

## Checking ServiceError

### Type Assertion Helper

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

### Usage in RespondError

```go
// errors/errors.go:140-146
func RespondError(c *gin.Context, err error) {
    se, ok := services.AsServiceError(err)
    if !ok {
        log.Printf("unexpected error: %v", err)
        writeInternalError(c)
        return
    }
    // ...
}
```

---

## Common Mistakes

### 1. Returning Raw Errors

```go
// ❌ Bad - loses context and error code
func (s *PostService) CreatePost(title, body string, userID int) (string, error) {
    post, err := s.postRepo.CreatePost(title, body, userID)
    if err != nil {
        return "", err
    }
    // ...
}

// ✅ Good - wraps with ServiceError
func (s *PostService) CreatePost(title, body string, userID int) (string, error) {
    post, err := s.postRepo.CreatePost(title, body, userID)
    if err != nil {
        return "", NewServiceErrorWrap(ErrInternal, "create post failed", err)
    }
    // ...
}
```

### 2. Not Using RespondError in Handlers

```go
// ❌ Bad - inconsistent error format
if err != nil {
    c.JSON(500, gin.H{"error": err.Error()})
    return
}

// ✅ Good - uses RespondError for proper handling
if err != nil {
    apperrors.RespondError(c, err)
    return
}
```

### 3. Wrong Error Code for HTTP Semantics

```go
// ❌ Bad - wrong HTTP status (404 → 500)
return NewServiceErrorWrap(ErrInternal, "user not found", err)

// ✅ Good - correct HTTP status (404 → 404)
return NewServiceErrorWrap(ErrNotFound, "user not found", err)
```

**Error Code → HTTP Status Mapping**:
- `ErrNotFound` → 404 Not Found
- `ErrUnauthorized` → 401 Unauthorized
- `ErrInvalidCredentials` → 401 Unauthorized
- `ErrValidation` → 400 Bad Request
- `ErrInternal` → 500 Internal Server Error
- `ErrInvalidPostKey` → 403 Forbidden

### 4. Missing i18n Message

```go
// ❌ Bad - no i18n message defined
// services/errors.go
const ErrNewCode ErrCode = "new_code"

// errors/errors.go - missing from serviceErrorMappings
// locales/*.toml - missing translations

// ✅ Good - complete i18n setup
// 1. Define code
const ErrNewCode ErrCode = "new_code"

// 2. Add mapping
services.ErrNewCode: {
    Status: http.StatusBadRequest,
    Message: &i18n.Message{
        ID:    "error.new_code",
        Other: "New code error",
    },
}

// 3. Add translations
// locales/active.en-us.toml
["error.new_code"]
other = "New code error"
```

### 5. Not Checking Specific Error Types

```go
// ❌ Bad - treats all database errors the same
post, err := s.postRepo.GetPostByQID(qid)
if err != nil {
    return "", "", NewServiceErrorWrap(ErrInternal, "get post failed", err)
}

// ✅ Good - checks for specific error types
post, err := s.postRepo.GetPostByQID(qid)
if err != nil {
    if err == models.ErrNotFound {
        return "", "", NewServiceErrorWrap(ErrNotFound,
            fmt.Sprintf("post with qid %s not found", qid), err)
    }
    return "", "", NewServiceErrorWrap(ErrInternal,
        fmt.Sprintf("get post with qid %s failed", qid), err)
}
```

---

## Best Practices

### 1. Layered Error Handling

- **Repository**: Return raw errors (e.g., `models.ErrNotFound`)
- **Service**: Wrap with `ServiceError` and add context
- **Handler**: Use `apperrors.RespondError` for uniform response

### 2. Error Description Guidelines

- Be specific: Include relevant IDs, names, or values
- Be helpful: Describe what operation failed
- Be secure: Don't expose sensitive information

```go
// ✅ Good - specific and helpful
NewServiceErrorWrap(ErrNotFound,
    fmt.Sprintf("post with qid %s not found", qid), err)

// ❌ Bad - too generic
NewServiceErrorWrap(ErrNotFound, "not found", err)

// ❌ Bad - exposes sensitive info
NewServiceErrorWrap(ErrUnauthorized,
    fmt.Sprintf("password %s is incorrect", password), err)
```

### 3. Validation Error Details

For validation errors, always include field-level details:

```go
// ✅ Good - includes field details
NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
    {Code: ErrRequired, Description: "title"},
    {Code: ErrMinLength, Description: "body"},
})

// ❌ Bad - no field details
NewServiceError(ErrValidation, "request validation failed")
```

### 4. Error Wrapping

Always wrap underlying errors to preserve stack trace:

```go
// ✅ Good - preserves stack trace
return NewServiceErrorWrap(ErrInternal, "create post failed", err)

// ❌ Bad - loses stack trace
return NewServiceError(ErrInternal, "create post failed")
```

### 5. Logging

`RespondError` automatically logs unexpected errors:

```go
// errors/errors.go:143-145
if !ok {
    log.Printf("unexpected error: %v", err)
    writeInternalError(c)
    return
}
```

For additional logging in services:

```go
// Only log at service layer if needed for debugging
// Handler layer will handle logging via RespondError
```

---

## Testing Error Handling

### Testing Service Errors

```go
// services/post_test.go
func TestCreatePost_DatabaseError(t *testing.T) {
    // ... setup mock repo to return error

    _, err := postSvc.CreatePost("title", "body", 1)

    se, ok := services.AsServiceError(err)
    assert.True(t, ok)
    assert.Equal(t, services.ErrInternal, se.Code)
    assert.Contains(t, se.Description, "create post failed")
}
```

### Testing Handler Responses

```go
// handlers/post_test.go
func TestCreatePost_ValidationError(t *testing.T) {
    // ... setup request with invalid data

    w := httptest.NewRecorder()
    // ... make request

    var resp errors.ErrorResponse
    json.Unmarshal(w.Body.Bytes(), &resp)

    assert.Equal(t, http.StatusBadRequest, w.Code)
    assert.Equal(t, "validation", resp.Code)
    assert.NotEmpty(t, resp.Errors)
}
```

---

## Summary

### Key Principles

1. **Always wrap errors** in Service layer with `ServiceError`
2. **Always use `RespondError`** in Handler layer
3. **Always add i18n** when adding new error codes
4. **Match error codes** to HTTP semantics
5. **Include context** in error descriptions
6. **Preserve stack traces** by wrapping underlying errors

### Error Handling Checklist

Before committing code with error handling:

- [ ] All service errors wrapped with `ServiceError`
- [ ] All handler errors use `apperrors.RespondError`
- [ ] Error codes match HTTP status semantics
- [ ] New error codes have i18n translations
- [ ] Error descriptions are specific and helpful
- [ ] Validation errors include field details
- [ ] Unit tests verify error codes and messages

---

**Language**: All documentation is written in **English**.
