# Error Handling

## Error Flow

Errors flow upward through the layers: repository → service → handler → HTTP response.

```
Repository returns raw error
  → Service wraps it as service.Error with a code
    → Handler passes it to apierr.RespondError()
      → apierr maps the code to HTTP status + i18n message
```

## Service Errors

The `internal/service` package defines a structured error type:

```go
type Error struct {
    Code        ErrCode   // Machine-readable error code
    Description string    // Developer description
    Err         error     // Wrapped underlying error
    Details     []Error   // Sub-errors for validation
}
```

Error codes are defined as constants in `internal/service/errors.go`:

| Code | HTTP Status | Meaning |
|------|-------------|---------|
| `invalid_credentials` | 401 | Wrong username or password |
| `invalid_password` | 400 | Current password doesn't match |
| `unauthorized` | 401 | Not authenticated |
| `forbidden` | 403 | Lacks permission |
| `not_found` | 404 | Resource not found |
| `internal` | 500 | Unexpected server error |
| `failed_get_user` | 500 | Failed to retrieve user data |
| `validation` | 400 | Request validation failed |
| `missing_authorization_header` | 401 | No Authorization header |
| `invalid_token` | 401 | JWT is invalid or expired |
| `invalid_post_key` | 403 | Post key doesn't match any user |
| `user_disabled` | 403 | User account is deactivated |
| `invalid_request` | 400 | Malformed request body |
| `missing_state_param` | 400 | Missing OAuth state parameter |
| `missing_code` | 400 | Missing OAuth authorization code |

Services create errors using constructors:

```go
// Simple error with code and description
service.NewServiceError(service.ErrNotFound, "post not found")

// Wrapping an underlying error
service.NewServiceErrorWrap(service.ErrInternal, "database query failed", err)

// Validation error with field-level details
service.NewServiceErrorWithDetails(service.ErrValidation, "validation failed", fieldErrors)
```

## API Error Responses

The `pkg/apierr` package handles the final error-to-HTTP conversion via `RespondError()`.

Error response format:

```json
{
  "code": "invalid_credentials",
  "message": "Invalid username or password"
}
```

For validation errors, additional field-level details are included:

```json
{
  "code": "validation",
  "message": "Request validation failed",
  "errors": [
    {
      "field": "title",
      "code": "required",
      "message": "This field is required"
    }
  ]
}
```

## Handler Pattern

Handlers follow a consistent pattern for error handling:

```go
func SomeHandler(svc SomeService) gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... parse request ...
        result, err := svc.DoSomething(c.Request.Context(), ...)
        if err != nil {
            apierr.RespondError(c, err)
            return
        }
        c.JSON(http.StatusOK, result)
    }
}
```

Validation errors from request binding are handled automatically by the `bindJSON` and `bindQuery` helpers in `common.go`, which construct field-level error details and call `apierr.RespondError`.

## Middleware Errors

Middleware follows the same pattern but uses `c.Abort()` after responding to stop the chain:

```go
func SomeMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if !valid {
            apierr.RespondError(c, service.NewServiceErrorWrap(...))
            c.Abort()
            return
        }
        c.Next()
    }
}
```

## i18n in Errors

Error messages are internationalized. Each error code maps to an i18n message ID (e.g., `error.invalid_credentials`). The `apierr` package resolves messages using `ginI18n.MustGetMessage`, falling back to English defaults.

Locale files live in `backend/locales/` as TOML files (e.g., `en.toml`, `zh.toml`).
