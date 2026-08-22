# Error Handling

English | [中文](error-handling.zh.md)

## Design Principles

- **Errors transform only at boundaries**: an error stays pure where it originates and changes shape only when crossing layers. No wrapping at every level, no text added at every level.
- **Never swallow an error**: every error has a destination — either a known error code, or a logged fallback.
- **No double wrapping**: the semantic code of an error is determined once, by the method with the most context; upper layers re-raise directly and wrap nothing.
- **Defensive programming**: parameters reaching a handler are already expected to be sound; service layers and below validate moderately, but even a missed check never triggers a raw low-level error.
- **Never panic**: the error-handling chain guarantees zero panics; the client always receives a well-formed ErrorResponse (see "The four fallback layers" at the end).

## The Layered Error Flow

```
infra (GORM)  ──bare sentinel pass-through──▶  domain  ──sentinel──▶  service  ──service.Error──▶  handler  ──▶  apierr  ──▶  HTTP
```

Each layer's responsibilities and rules are in "Layer contracts" below.

## Layer Contracts

### The infra layer: GORM error isolation, bare pass-through

**Core rules**:

- GORM's `TranslateError: true` is enabled (`gorm.Config`); the driver automatically translates database-specific error codes into GORM's generic sentinels:
  - PostgreSQL `23505` (unique key violation) → `gorm.ErrDuplicatedKey`
  - PostgreSQL `23503` (foreign key violation) → `gorm.ErrForeignKeyViolated`
  - PostgreSQL `23514` (check constraint) → `gorm.ErrCheckConstraintViolated`
  - Record not found → `gorm.ErrRecordNotFound`
- Infra helper functions pass GORM's returned error through **bare**, with **no label parameter** (no `fmt.Errorf("create post: %w", err)`).
- **Why no label**: the operation context ("which operation failed") is carried by the OTel span's name (see [observability.md](./observability.md)). The error itself expresses only "what went wrong", with no debugging text mixed in. Known sentinels are identified via `errors.Is`, needing no label; unexpected errors are located through trace_id → the span call chain (more precise than a single-layer label).

```go
// infra helper has no label parameter
func findFirst[T any](ctx context.Context, query *gorm.DB) (*T, error) {
    var result T
    if err := query.WithContext(ctx).First(&result).Error; err != nil {
        return nil, err   // bare pass-through: already a GORM sentinel
    }
    return &result, nil
}
```

### The domain layer: universal sentinels

**Core rules**:

- The domain defines only **cross-domain** sentinel errors (`ErrNotFound`, `ErrConflict`, `ErrAlreadyExists`, and the like) as the stable contract for cross-layer error identification.
- Repository interfaces return these sentinels, **passing them through unwrapped**.
- Domain-specific business errors (such as "duplicate submission qid" or "channel does not exist") get **no** sentinel in the domain; the service layer recognizes them from business context and converts them to `service.Error`.

### The service layer: minimal domain isolation

**Core rules**:

- After calling an infra/repository method, the service checks sentinels with `errors.Is` and converts them to `service.Error`:

```go
user, err := r.userRepo.FindByID(ctx, id)
switch {
case errors.Is(err, gorm.ErrRecordNotFound):
    return service.New(service.ErrNotFound, "user not found")
case errors.Is(err, gorm.ErrDuplicatedKey):
    return service.New(service.ErrConflict, "email already taken")
case err != nil:
    return nil, err   // unexpected error passes through raw; apierr logs it + 500
}
```

- **When calling internal methods, those methods guarantee a `service.Error` return**; the outer layer re-raises directly **without re-wrapping**. Rationale: the error code is determined once by the inner method that has the context; the outer layer has no reason to re-classify the error.
- Service-layer errors are **not logged one by one** — logging happens at the boundary (handler / apierr) where the error surfaces.

### The handler layer: binding + forwarding

**Core rules**:

- A handler's errors come almost entirely from the service.
- A handler does exactly three things: (1) binding failure → convert to FieldDetail → `service.Error{Code: ErrValidation}`; (2) call the service; (3) hand the error to `apierr.RespondError`.
- **Handlers do not re-wrap service errors**.

```go
func SomeHandler(svc SomeService) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req SomeRequest
        if !bindJSON(c, &req) { return }   // binding failure already handled by handleBindingError
        result, err := svc.DoSomething(c.Request.Context(), req)
        if err != nil {
            apierr.RespondError(c, err)
            return
        }
        c.JSON(http.StatusOK, result)
    }
}
```

### The apierr layer: the single entry point for client error responses

**Core rules**:

- `apierr.RespondError(c, err)` is the **single entry point** through which handlers and middleware return client error responses.
- The input is an `error`; internally:
  - not a `service.Error` → logged via `slog.Error` (with trace fields) + fallback 500
  - a `service.Error` → the ErrCode's own HTTP status code + i18n Message render the ErrorResponse

### The middleware layer

**Core rules**:

- The same pattern as handlers, except it must `c.Abort()` before calling `RespondError` (to halt the middleware chain).
- Details in "Middleware error handling" below.

## The service.Error Struct

```go
type Error struct {
    Code        *ErrCode       // points at the error-code singleton (carries its own HTTP/i18n mappings)
    Description string         // domain-semantic description (for developers; never reaches the client response)
    Err         error          // the underlying raw error (e.g. as returned by the repository)
    Details     []FieldDetail  // field-level validation errors (form binding only)
}

type FieldDetail struct {
    Field string   // the json field name
    Code  *ErrCode // the field error code (required/min/max/...)
    Param string   // the rule parameter value (e.g. "6" for min), consumed by i18n template rendering
}
```

**Methods**: implements `Error()` and `Unwrap()` (returns `Err`), supporting `errors.As` / `errors.Is`.

**Constructors**:

- `New(code *ErrCode, description string) *Error`
- `Wrap(code *ErrCode, description string, err error) *Error`
- `WithDetails(code *ErrCode, description string, details []FieldDetail) *Error`
- `NewValidation(details []FieldDetail) *Error` (convenience: Code=ErrValidation)

## The ErrCode Struct Carries Its Own Mappings

**Core design**: `ErrCode` is a struct instance (not a string constant) carrying its own HTTP status code, i18n Message, template placeholder, and dynamic parameter provider. This eliminates the traditional three mapping maps.

```go
type ErrCode struct {
    Value         string         // the error-code string (lands in the response's code field)
    HTTP          int            // the mapped HTTP status code
    Message       *i18n.Message  // i18n message template (English DefaultMessage, the authoritative fallback)
    Placeholder   string         // template placeholder name for field validation codes (e.g. "Min", "Max"), optional
    ParamProvider func() string  // dynamic threshold provider for custom rules, optional
}
```

**Why this design**:

- Eliminates the three global maps `httpStatuses`, `errorCodeMessages`, and `validationFieldMessages`.
- **Fully autonomous domains**: auth's error codes + httpStatus + i18n message + placeholder are all defined in the single file `auth/errors.go`.
- **Zero side effects**: no `init()` registration, no global merge, no registration functions. Purely declarative and statically analyzable.
- **ParamProvider** solves the `Param()`-returns-empty problem for custom validator rules (see the request-validation section).

**Usage**: error codes are package-level `var` singletons passed by `*ErrCode` pointer, with copying avoided by convention. Compare via `.Value` (the string) or `.HTTP` (the status); apierr writes no big switch — it reads `code.HTTP` / `code.Message` directly.

**No inherent flaw**: the alleged problem with pointer comparison does not exist in this design — the correct comparison idiom is by value or by attribute, serialization is handled automatically by `MarshalJSON`, and apierr consumes the ErrCode's own mappings directly.

## Error Code Organization (per-domain files)

```
internal/service/
├── errors.go        # ErrCode/Error/FieldDetail types + constructors + shared error codes
├── auth/errors.go   # auth domain-specific codes
├── post/errors.go   # post domain-specific codes
├── delivery/errors.go
└── admin/errors.go
```

### Shared error codes (service/errors.go)

Codes common to all domains + the universal field-validation codes:

| ErrCode             | Value             | HTTP    | Meaning                                                                                                                                                                    |
| ------------------- | ----------------- | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ErrInternal`       | `internal`        | 500     | Unexpected internal server error                                                                                                                                           |
| `ErrValidation`     | `validation`      | **422** | Request parameter validation failure (form binding) — a failed field validation is "semantically unprocessable" (RFC 4918 422); see [api-design.md](../api-design.md) §3.1 |
| `ErrInvalidRequest` | `invalid_request` | 400     | Malformed request (JSON deserialization failure, empty body, etc.)                                                                                                         |
| `ErrNotFound`       | `not_found`       | 404     | Resource does not exist                                                                                                                                                    |
| `ErrUnauthorized`   | `unauthorized`    | 401     | Not authenticated                                                                                                                                                          |
| `ErrForbidden`      | `forbidden`       | 403     | Insufficient permission                                                                                                                                                    |
| `ErrConflict`       | `conflict`        | 409     | Resource conflict (duplicate creation, etc.)                                                                                                                               |
| `ErrRateLimited`    | `rate_limited`    | 429     | Rate limit triggered                                                                                                                                                       |

Universal field-validation codes (all 422, same rationale as `ErrValidation`):

| ErrCode             | Value             | HTTP | Placeholder | Meaning                                  |
| ------------------- | ----------------- | ---- | ----------- | ---------------------------------------- |
| `ErrRequired`       | `required`        | 422  | —           | Field is required                        |
| `ErrMinLength`      | `min_length`      | 422  | `Min`       | Below the minimum length                 |
| `ErrMaxLength`      | `max_length`      | 422  | `Max`       | Above the maximum length                 |
| `ErrLength`         | `length`          | 422  | `Len`       | Length does not match                    |
| `ErrEmail`          | `invalid_email`   | 422  | —           | Malformed email address                  |
| `ErrOneOf`          | `not_one_of`      | 422  | `OneOf`     | Value outside the allowed set            |
| `ErrFieldViolation` | `field_violation` | 422  | `Param`     | Generic fallback (unknown validator tag) |

### Domain-specific code example (service/auth/errors.go)

```go
var (
    ErrInvalidCredentials = &ErrCode{
        Value:   "invalid_credentials",
        HTTP:    401,
        Message: &i18n.Message{ID: "error.invalid_credentials", Other: "Invalid username or password"},
    }
    ErrUserDisabled = &ErrCode{
        Value:   "user_disabled",
        HTTP:    403,
        Message: &i18n.Message{ID: "error.user_disabled", Other: "User account is disabled"},
    }
    ErrInvalidToken = &ErrCode{
        Value:   "invalid_token",
        HTTP:    401,
        Message: &i18n.Message{ID: "error.invalid_token", Other: "Invalid or expired token"},
    }
)
```

## API Error Responses (GitHub style)

```go
type ErrorResponse struct {
    Code    string       `json:"code"`
    Message string       `json:"message"`
    Errors  []FieldError `json:"errors,omitempty"`
}

type FieldError struct {
    Field   string `json:"field,omitempty"`
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

### Response examples

**Simple error (no errors)**:

```json
{
  "code": "invalid_credentials",
  "message": "Invalid username or password"
}
```

**Form validation error (with errors array)**:

```json
{
  "code": "validation",
  "message": "Request validation failed",
  "errors": [
    {
      "field": "new_password",
      "code": "min_length",
      "message": "new_password must be at least 6 characters"
    },
    {
      "field": "current_password",
      "code": "required",
      "message": "current_password is required"
    }
  ]
}
```

### The frontend handling pattern

```typescript
interface ErrorResponse {
  code: string;
  message: string;
  errors?: Array<{
    field?: string;
    code: string;
    message: string;
  }>;
}

try {
  await api.changePassword(data);
} catch (error) {
  const err = error.response.data as ErrorResponse;
  if (err.errors) {
    err.errors.forEach((e) => {
      if (e.field) {
        setError(e.field, { type: e.code, message: e.message }); // mark the field error red
      } else {
        toast.error(e.message); // top notice for non-field errors
      }
    });
  } else {
    toast.error(err.message); // whole-form notice for simple errors
  }
}
```

## The apierr Package Contract

`internal/apierr` is the unified entry point through which handlers and middleware return client error responses.

### RespondError logic

```go
func RespondError(c *gin.Context, err error) {
    se, ok := service.AsError(err)
    if !ok {
        slog.Error("unexpected error", "trace_id", traceID(c), "error", err,
            "method", c.Request.Method, "path", c.Request.URL.Path)
        writeError(c, service.ErrInternal, nil)
        return
    }
    writeError(c, se.Code, buildTemplateData(se))
}

func writeError(c *gin.Context, code *service.ErrCode, data map[string]any) {
    c.JSON(code.HTTP, ErrorResponse{
        Code:    code.Value,
        Message: renderMessage(c, code, data),
    })
}
```

### i18n rendering (with fallbacks)

```go
func renderMessage(c *gin.Context, code *service.ErrCode, data map[string]any) string {
    msg := ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
        DefaultMessage: code.Message,   // in-code English fallback
        TemplateData:   data,           // {{.Field}} {{.Min}} and friends
    })
    if msg == "" {
        return code.Message.Other       // last-resort fallback: the English original, never fails
    }
    return msg
}
```

### Naming conventions

- `writeXxxError`: writes an ErrorResponse
- `writeXxxResponse`: writes a success Response

### Where convenience functions live

`RespondError` is the default entry point. A handler or middleware with special error-response needs (custom headers, extra fields) may implement its own path, but **the response body structure must still be ErrorResponse**.

## Request Validation

### The three sources of binding errors

The errors returned by gin's `ShouldBindJSON` / `ShouldBindQuery` / `ShouldBindHeader` (and siblings) fall into three categories (based on gin's `binding/json.go` and `binding/form.go`):

| Stage                    | Error type                                                               | Handling                                                                                      |
| ------------------------ | ------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------- |
| **JSON deserialization** | `*json.SyntaxError` / `*json.UnmarshalTypeError` / `io.EOF` (empty body) | → `ErrInvalidRequest` (400; no field to point at, a blanket "malformed request")              |
| **Struct validation**    | `validator.ValidationErrors` (`[]FieldError`)                            | → `ErrValidation` (422, with `errors[]` field-level details)                                  |
| **Slice validation**     | `binding.SliceValidationError` (`[]error`)                               | → recursively flattened into `[]FieldDetail` (defensive; no array-body endpoints exist today) |

**Why the semantic split**: `ErrInvalidRequest` says "the whole request body cannot be parsed" (the frontend checks Content-Type / body construction); `ErrValidation` says "parses, but a field value is invalid" (the frontend re-fills the form and marks fields red).

### The unified entry: handleBindingError

```go
func handleBindingError(c *gin.Context, req any, err error) {
    var se *service.Error
    switch {
    // slice validation (recursive flattening)
    case errors.As(err, new(binding.SliceValidationError)):
        details := flattenSliceErrors(err.(binding.SliceValidationError))
        se = service.NewValidation(details)

    // field validation
    case errors.As(err, new(validator.ValidationErrors)):
        details := make([]FieldDetail, 0)
        for _, fe := range err.(validator.ValidationErrors) {
            details = append(details, fieldErrorToDetail(fe))
        }
        se = service.NewValidation(details)

    // JSON type mismatch (field locatable)
    case errors.As(err, new(*json.UnmarshalTypeError)):
        ute := err.(*json.UnmarshalTypeError)
        se = service.New(service.ErrInvalidRequest,
            fmt.Sprintf("type mismatch on field %s", ute.Field))

    // JSON syntax error / empty body / other
    default:
        se = service.New(service.ErrInvalidRequest, err.Error())
    }
    apierr.RespondError(c, se)
}
```

### RegisterTagNameFunc (replaces hand-written reflection)

Registered at startup so validator's `fe.Field()` returns the json tag name directly (nested structs included):

```go
func RegisterValidators() {
    v := binding.Validator.Engine().(*validator.Validate)
    v.RegisterTagNameFunc(func(f reflect.StructField) string {
        jsonTag := f.Tag.Get("json")
        if jsonTag != "-" {
            if name := strings.Split(jsonTag, ",")[0]; name != "" {
                return name
            }
        }
        if formTag := f.Tag.Get("form"); formTag != "" {
            if name := strings.Split(formTag, ",")[0]; name != "" {
                return name
            }
        }
        return f.Name  // fallback: the Go field name
    })
    v.RegisterValidation("titlesize", validateTitleLength)
    v.RegisterValidation("bodysize", validateBodySize)
}
```

With this registered, handlers need no hand-written `resolveFieldName` reflection logic.

### tagRegistry (option A: explicit placeholder mapping)

A mapping table from validator tag → ErrCode + placeholder. Adding a rule is one line:

```go
var tagRegistry = map[string]struct {
    code        *ErrCode
    placeholder string  // template placeholder name; empty means no parameter
}{
    "required":   {ErrRequired, ""},
    "min":        {ErrMinLength, "Min"},
    "max":        {ErrMaxLength, "Max"},
    "len":        {ErrLength, "Len"},
    "email":      {ErrEmail, ""},
    "oneof":      {ErrOneOf, "OneOf"},
    "eq":         {ErrEq, "Eq"},
    "ne":         {ErrNe, "Ne"},
    "contains":   {ErrContains, "Contains"},
    "titlesize":  {ErrTitleSize, "Max"},
    "bodysize":   {ErrBodySize, "Max"},
    // add one line here per new rule
}

func fieldErrorToDetail(fe validator.FieldError) FieldDetail {
    spec, ok := tagRegistry[fe.Tag()]
    if !ok {
        return FieldDetail{Field: fe.Field(), Code: ErrFieldViolation, Param: fe.Param()}
    }
    param := fe.Param()
    if param == "" && spec.code.ParamProvider != nil {
        param = spec.code.ParamProvider()  // custom rule takes its dynamic threshold
    }
    return FieldDetail{Field: fe.Field(), Code: spec.code, Param: param}
}
```

apierr builds the TemplateData when rendering:

```go
func buildTemplateData(fd FieldDetail) map[string]any {
    data := map[string]any{
        "Field": fd.Field,
        "Param": fd.Param,  // generic fallback placeholder, always present
    }
    if fd.Code.Placeholder != "" {
        data[fd.Code.Placeholder] = fd.Param  // semantic placeholder (Min/Max/...)
    }
    return data
}
```

### Why the validator's built-in translation is not used

The validator's own `ValidationErrors.Translate(translator)` mechanism is not adopted. Reasons:

1. **Scope violation**: binding error→message inside the validator package conflicts with the unified i18n system.
2. **Bypasses error codes**: it produces a string message directly, skipping the ErrCode mapping, so the client gets no structured code/field/param.
3. **Translator initialization complexity**: overlaps and conflicts with the existing go-i18n locale file system.
4. **Custom rules still need manual work**: titlesize/bodysize still require registering a transFn — nothing saved.

All validator errors flow through the "FieldError → ErrCode + FieldDetail → i18n rendering" chain.

### The ParamProvider mechanism

Custom validator rules (like `titlesize` / `bodysize`) are registered via `RegisterValidation`, and their `fe.Param()` always returns the empty string (validator provides Param only for built-in parameterized rules). When the threshold comes from runtime configuration, the ErrCode's `ParamProvider` field supplies it:

```go
var ErrTitleSize = &ErrCode{
    Value:   "title_too_long",
    HTTP:    400,
    Message: &i18n.Message{ID: "error.validation_titlesize", Other: "{{.Field}} exceeds the maximum of {{.Max}} characters"},
    Placeholder: "Max",
    ParamProvider: func() string {
        return strconv.Itoa(config.Get().Post.TitleMaxLength)
    },
}
```

When `fieldErrorToDetail` finds `fe.Param()==""` but `code.ParamProvider != nil`, it calls the provider. `config.Get()` is guarded by `sync.Once` and its return types never panic; the worst case is a zero value (a wording blemish, not a crash).

## Middleware Error Handling

### panic recovery (the fallback middleware)

```go
func Fallback() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if r := recover(); r != nil {
                slog.Error("panic recovered", "error", r, "trace_id", traceID(c),
                    "path", c.Request.URL.Path)
                apierr.RespondError(c, service.New(service.ErrInternal, "internal server error"))
                c.Abort()
            }
        }()
        c.Next()
    }
}
```

### Rate limiting (tollbooth)

429 + `ErrRateLimited` + the `X-Rate-Limit-Limit` / `X-Rate-Limit-Duration` headers tollbooth sets automatically. **No `Retry-After` is sent** (tollbooth does not provide one; clients judge retry timing from the remaining allowance in the headers).

## Fault Localization (label-free, trace-based)

Errors carry no operation context (no labels); localization relies on the observability stack:

**Scenario 1: a known sentinel** (e.g. not_found) — the error code maps directly; nothing to chase.

**Scenario 2: an unexpected infra error** (e.g. connection pool exhaustion) — `app.jsonl` records the trace_id → `traces.jsonl` shows the span call chain:

```
POST /:post_key → post.Create → posts.Insert (the span's err attribute records the original error)
```

The span names provide hierarchical operation context, more precise than a single-layer label.

**Scenario 3: an error internal to a service** (e.g. render failure) — `service.Error.Description` (the domain semantics, "render post failed") plus the span call chain localize it twice over.

See [observability.md](./observability.md) for the full picture.

## The Four Fallback Layers (defensive review conclusion)

```
Layer 1: tagRegistry fallback
         unknown validator tag → ErrFieldViolation + the generic {{.Param}} placeholder

Layer 2: RegisterTagNameFunc fallback
         field without json/form tags → the Go field name

Layer 3: i18n rendering fallback
         ginI18n.MustGetMessage returns empty → code.Message.Other, the English original (never fails)

Layer 4: panic recovery
         the fallback middleware: any panic → slog record + 500
```

**Review conclusion**: zero panic risk.

- `MustGetMessage` does not panic (source-verified: `message, _ := GetMessage()` discards the error and returns an empty string)
- `config.Get()` does not panic (`sync.Once`, returns a zero value)
- validator interface methods are pure reads and do not panic
- `errors.As` returns bool and does not panic

The program never crashes over error handling; the client always receives a well-formed ErrorResponse (even if the message degrades to the English original).

## Testing Conventions

- Service tests: assert `errors.As(err, &service.Error{})` then check `Code.Value` / `Code.HTTP`
- Handler tests: assert the HTTP status code + the ErrorResponse JSON structure (code/message/errors fields)
- Binding tests: cover every kind of validator tag + JSON deserialization errors + nested-struct field-name resolution
