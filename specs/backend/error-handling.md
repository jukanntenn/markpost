# Error Handling

## 设计原则

- **错误只在边界转换**：error 在产生它的地方保持纯净，跨层时才转换形态。不层层包装、不层层加文字。
- **不吞错**：每个 error 都要有归宿——要么转成已知错误码，要么记日志后兜底。
- **不重复包装**：error 的语义码（Code）由最了解上下文的方法一次性确定，上层直接上抛，不再二次包装。
- **防御性编程**：handler 层传入的参数应已合理；service 及以下适度校验，但即使漏校验也不触发底层裸错误。
- **永不 panic**：error 处理链路保证零 panic，客户端永远收到格式正确的 ErrorResponse（见末尾"四层兜底"）。

## 分层错误流

```
infra (GORM)  ──裸透传 sentinel──▶  domain  ──sentinel──▶  service  ──service.Error──▶  handler  ──▶  apierr  ──▶  HTTP
```

每层的职责与规则见下方"分层契约"。

## 分层契约

### infra 层：GORM error 隔离，裸透传

**核心规则**：
- 开启 GORM 的 `TranslateError: true`（`gorm.Config`），驱动自动把数据库特定错误码翻译成 GORM 通用 sentinel：
  - PostgreSQL `23505`（唯一键冲突）→ `gorm.ErrDuplicatedKey`
  - PostgreSQL `23503`（外键冲突）→ `gorm.ErrForeignKeyViolated`
  - PostgreSQL `23514`（check 约束）→ `gorm.ErrCheckConstraintViolated`
  - 记录不存在 → `gorm.ErrRecordNotFound`
- infra 的 helper 函数**裸透传** GORM 返回的 error，**不加 label 参数**（不写 `fmt.Errorf("create post: %w", err)`）。
- **为什么不要 label**：操作上下文（"是在哪个操作出错的"）由 OTel span 的 name 承载（见 [observability.md](./observability.md)）。error 本身只表达"是什么错"，不混入调试文字。已知 sentinel 靠 `errors.Is` 识别，无需 label；意外 error 靠 trace_id → span 调用链定位（比单层 label 更精确）。

```go
// infra helper 无 label 参数
func findFirst[T any](ctx context.Context, query *gorm.DB) (*T, error) {
    var result T
    if err := query.WithContext(ctx).First(&result).Error; err != nil {
        return nil, err   // 裸透传，已是 GORM sentinel
    }
    return &result, nil
}
```

### domain 层：通用 sentinel

**核心规则**：
- domain 只定义**跨域通用**的 sentinel error（`ErrNotFound`、`ErrConflict`、`ErrAlreadyExists` 等），作为跨层错误识别的稳定契约。
- repository 接口返回这些 sentinel，**透传不包装**。
- 域特定的业务错误（如"投稿 qid 重复"、"频道不存在"）**不**在 domain 定义 sentinel，由 service 层根据业务上下文识别后转 service.Error。

### service 层：最轻量领域隔离

**核心规则**：
- service 调用 infra/repository 后，用 `errors.Is` 判定 sentinel，转成 `service.Error`：

```go
user, err := r.userRepo.FindByID(ctx, id)
switch {
case errors.Is(err, gorm.ErrRecordNotFound):
    return service.New(service.ErrNotFound, "user not found")
case errors.Is(err, gorm.ErrDuplicatedKey):
    return service.New(service.ErrConflict, "email already taken")
case err != nil:
    return nil, err   // 意外 error 裸上抛，由 apierr 记日志 + 500
}
```

- **调用内部方法时，内部方法保证返回 `service.Error`**，外层直接上抛**不重复包装**。理由：错误码（Code）由最了解上下文的内部方法一次性确定，外层无需再区分 error 种类。
- service 层 error **不逐个记日志**——上抛到边界（handler/apierr）才记。

### handler 层：binding + 转发

**核心规则**：
- handler 的 error 基本都来自 service。
- handler 只做三件事：(1) binding 校验失败 → 转 FieldDetail → `service.Error{Code: ErrValidation}`；(2) 调 service；(3) 把 error 交给 `apierr.RespondError`。
- **handler 不重复包装 service error**。

```go
func SomeHandler(svc SomeService) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req SomeRequest
        if !bindJSON(c, &req) { return }   // binding 失败已由 handleBindingError 处理
        result, err := svc.DoSomething(c.Request.Context(), req)
        if err != nil {
            apierr.RespondError(c, err)
            return
        }
        c.JSON(http.StatusOK, result)
    }
}
```

### apierr 层：客户端错误响应唯一入口

**核心规则**：
- `apierr.RespondError(c, err)` 是 handler 和 middleware 返回客户端错误响应的**唯一入口**。
- 输入是一个 `error`，内部处理：
  - 非 `service.Error` → `slog.Error` 记录（带 trace 字段）+ 兜底 500
  - `service.Error` → 用 ErrCode 自带的 HTTP 状态码 + i18n Message 渲染 ErrorResponse

### middleware 层

**核心规则**：
- 与 handler 同模式，但必须 `c.Abort()` 后再 `RespondError`（中止中间件链）。
- 详见下方"middleware 错误处理"。

## service.Error 结构

```go
type Error struct {
    Code        *ErrCode       // 指向错误码单例（自带 HTTP/i18n 映射）
    Description string         // 领域语义描述（给开发者看，不进客户端响应）
    Err         error          // 底层原始 error（如 repository 返回的）
    Details     []FieldDetail  // 字段级校验错误（仅用于表单 binding）
}

type FieldDetail struct {
    Field string   // json 字段名
    Code  *ErrCode // 字段错误码（required/min/max/...）
    Param string   // 规则参数值（如 min 的 "6"），供 i18n 模板渲染
}
```

**方法**：实现 `Error()`、`Unwrap()`（返回 `Err`），支持 `errors.As`/`errors.Is`。

**构造器**：
- `New(code *ErrCode, description string) *Error`
- `Wrap(code *ErrCode, description string, err error) *Error`
- `WithDetails(code *ErrCode, description string, details []FieldDetail) *Error`
- `NewValidation(details []FieldDetail) *Error`（便捷：Code=ErrValidation）

## ErrCode struct 自带映射

**核心设计**：`ErrCode` 是一个 struct 实例（非 string 常量），自带 HTTP 状态码、i18n Message、模板占位符、动态参数提供器。消除传统的三张映射 map。

```go
type ErrCode struct {
    Value         string         // 错误码字符串（进客户端响应的 code 字段）
    HTTP          int            // 映射的 HTTP 状态码
    Message       *i18n.Message  // i18n 消息模板（英文 DefaultMessage，权威兜底）
    Placeholder   string         // 字段校验码的模板占位符名（如 "Min"、"Max"），可选
    ParamProvider func() string  // 自定义规则的动态阈值提供器，可选
}
```

**为什么这样设计**：
- 消除 `httpStatuses`、`errorCodeMessages`、`validationFieldMessages` 三张全局 map。
- **域完全自治**：auth 的错误码 + httpStatus + i18n message + 占位符全在 `auth/errors.go` 一个文件里定义。
- **零副作用**：无 `init()` 注册、无全局 merge、无注册函数。纯声明式，静态可分析。
- **ParamProvider** 解决自定义 validator 规则的 `Param()` 失效问题（见 validate 校验章节）。

**使用方式**：错误码是 package-level `var` 单例，`*ErrCode` 指针传递，约定不复制。比较时用 `.Value`（字符串值）或 `.HTTP`（状态码），apierr 里不写大 switch——直接 `code.HTTP` / `code.Message` 取值。

**无本质缺陷**：指针比较的所谓问题，彻底重写下不存在——正确的比较范式是按值或按属性，序列化由 `MarshalJSON` 自动处理，apierr 直接用 ErrCode 自带映射。

## 错误码组织（按域分文件）

```
internal/service/
├── errors.go        # ErrCode/Error/FieldDetail 类型 + 构造器 + 共享错误码
├── auth/errors.go   # auth 域专属码
├── post/errors.go   # post 域专属码
├── delivery/errors.go
└── admin/errors.go
```

### 共享错误码（service/errors.go）

所有域通用的错误码 + 字段校验通用码：

| ErrCode | Value | HTTP | 含义 |
|---|---|---|---|
| `ErrInternal` | `internal` | 500 | 意外的服务器内部错误 |
| `ErrValidation` | `validation` | **422** | 请求参数校验失败（表单 binding）——字段校验失败是"语义上无法处理"（RFC 4918 422），见 [api-design.md](../api-design.md) §3.1 |
| `ErrInvalidRequest` | `invalid_request` | 400 | 请求格式错误（JSON 反序列化失败、空 body 等） |
| `ErrNotFound` | `not_found` | 404 | 资源不存在 |
| `ErrUnauthorized` | `unauthorized` | 401 | 未认证 |
| `ErrForbidden` | `forbidden` | 403 | 无权限 |
| `ErrConflict` | `conflict` | 409 | 资源冲突（重复创建等） |
| `ErrRateLimited` | `rate_limited` | 429 | 触发限流 |

字段校验通用码（均 422，理由同 `ErrValidation`）：

| ErrCode | Value | HTTP | Placeholder | 含义 |
|---|---|---|---|---|
| `ErrRequired` | `required` | 422 | — | 字段必填 |
| `ErrMinLength` | `min_length` | 422 | `Min` | 未达最小长度 |
| `ErrMaxLength` | `max_length` | 422 | `Max` | 超过最大长度 |
| `ErrLength` | `length` | 422 | `Len` | 长度不符 |
| `ErrEmail` | `invalid_email` | 422 | — | 邮箱格式错误 |
| `ErrOneOf` | `not_one_of` | 422 | `OneOf` | 值不在允许范围内 |
| `ErrFieldViolation` | `field_violation` | 422 | `Param` | 通用兜底（未知 validator tag） |

### 域专属码示例（service/auth/errors.go）

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

## API 错误响应（GitHub 风格）

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

### 响应示例

**简单错误（无 errors）**：
```json
{
  "code": "invalid_credentials",
  "message": "Invalid username or password"
}
```

**表单校验错误（有 errors 数组）**：
```json
{
  "code": "validation",
  "message": "Request validation failed",
  "errors": [
    {"field": "new_password", "code": "min_length", "message": "new_password must be at least 6 characters"},
    {"field": "current_password", "code": "required", "message": "current_password is required"}
  ]
}
```

### 前端处理范式

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
    err.errors.forEach(e => {
      if (e.field) {
        setError(e.field, { type: e.code, message: e.message });  // 字段错误标红
      } else {
        toast.error(e.message);  // 非字段错误顶部提示
      }
    });
  } else {
    toast.error(err.message);  // 简单错误整体提示
  }
}
```

## apierr 包契约

`pkg/apierr` 是 handler/middleware 返回客户端错误响应的统一入口。

### RespondError 逻辑

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

### i18n 渲染（含兜底）

```go
func renderMessage(c *gin.Context, code *service.ErrCode, data map[string]any) string {
    msg := ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
        DefaultMessage: code.Message,   // 代码内英文兜底
        TemplateData:   data,           // {{.Field}} {{.Min}} 等
    })
    if msg == "" {
        return code.Message.Other       // 终极兜底：英文原文，永不失败
    }
    return msg
}
```

### 命名约定

- `writeXxxError`：写入 ErrorResponse
- `writeXxxResponse`：写入成功 Response

### 便捷函数定位

`RespondError` 是默认入口。若 handler 或 middleware 对 error 响应有特殊要求（如自定义 header、附加字段），可自行实现，但**响应体结构必须仍是 ErrorResponse**。

## validate 校验

### binding error 的三类来源

gin 的 `ShouldBindJSON`/`ShouldBindQuery`/`ShouldBindHeader` 等返回的 error 分三类（基于 gin 源码 `binding/json.go`、`binding/form.go`）：

| 阶段 | error 类型 | 处理 |
|---|---|---|
| **JSON 反序列化** | `*json.SyntaxError` / `*json.UnmarshalTypeError` / `io.EOF`（空 body） | → `ErrInvalidRequest`（400，无法定位字段，笼统"请求格式错误"） |
| **结构体校验** | `validator.ValidationErrors`（`[]FieldError`） | → `ErrValidation`（422，带 `errors[]` 字段级详情） |
| **切片校验** | `binding.SliceValidationError`（`[]error`） | → 递归扁平化为 `[]FieldDetail`（当前无数组 body 接口，保留防御） |

**语义区分的理由**：`ErrInvalidRequest` 表示"整个请求体解析不了"（前端检查 Content-Type / body 构造）；`ErrValidation` 表示"能解析但字段值不合规"（前端回填表单标红）。

### 统一入口 handleBindingError

```go
func handleBindingError(c *gin.Context, req any, err error) {
    var se *service.Error
    switch {
    // 切片校验（递归扁平化）
    case errors.As(err, new(binding.SliceValidationError)):
        details := flattenSliceErrors(err.(binding.SliceValidationError))
        se = service.NewValidation(details)

    // 字段校验
    case errors.As(err, new(validator.ValidationErrors)):
        details := make([]FieldDetail, 0)
        for _, fe := range err.(validator.ValidationErrors) {
            details = append(details, fieldErrorToDetail(fe))
        }
        se = service.NewValidation(details)

    // JSON 类型不匹配（可定位字段）
    case errors.As(err, new(*json.UnmarshalTypeError)):
        ute := err.(*json.UnmarshalTypeError)
        se = service.New(service.ErrInvalidRequest,
            fmt.Sprintf("type mismatch on field %s", ute.Field))

    // JSON 语法错 / 空 body / 其他
    default:
        se = service.New(service.ErrInvalidRequest, err.Error())
    }
    apierr.RespondError(c, se)
}
```

### RegisterTagNameFunc（取代手写反射）

启动时注册，让 validator 的 `fe.Field()` 直接返回 json tag 名（含嵌套结构体）：

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
        return f.Name  // 兜底：Go 字段名
    })
    v.RegisterValidation("titlesize", validateTitleLength)
    v.RegisterValidation("bodysize", validateBodySize)
}
```

注册后 handler 不再需要手写 `resolveFieldName` 反射逻辑。

### tagRegistry（方案 A：显式占位符映射）

定义 validator tag → ErrCode + 占位符的映射表。新增规则只需加一行：

```go
var tagRegistry = map[string]struct {
    code        *ErrCode
    placeholder string  // 模板占位符名，空表示无参数
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
    // 新增规则在这里加一行
}

func fieldErrorToDetail(fe validator.FieldError) FieldDetail {
    spec, ok := tagRegistry[fe.Tag()]
    if !ok {
        return FieldDetail{Field: fe.Field(), Code: ErrFieldViolation, Param: fe.Param()}
    }
    param := fe.Param()
    if param == "" && spec.code.ParamProvider != nil {
        param = spec.code.ParamProvider()  // 自定义规则取动态阈值
    }
    return FieldDetail{Field: fe.Field(), Code: spec.code, Param: param}
}
```

apierr 渲染时构造 TemplateData：

```go
func buildTemplateData(fd FieldDetail) map[string]any {
    data := map[string]any{
        "Field": fd.Field,
        "Param": fd.Param,  // 通用兜底占位符，永远存在
    }
    if fd.Code.Placeholder != "" {
        data[fd.Code.Placeholder] = fd.Param  // 语义占位符（Min/Max/...）
    }
    return data
}
```

### 明确弃用 validator 自带 translation

validator 自带的 `ValidationErrors.Translate(translator)` 方案不采用。理由：
1. **职责越界**：把 error→message 绑死在 validator 包内，与统一 i18n 体系冲突。
2. **绕过错误码**：直接生成字符串 message，不经过 ErrCode 映射，客户端拿不到结构化的 code/field/param。
3. **Translator 初始化复杂**：与已有的 go-i18n locale 文件体系重叠冲突。
4. **自定义规则仍需手动**：titlesize/bodysize 仍要自己注册 transFn，没省事。

所有 validator 错误统一走"FieldError → ErrCode + FieldDetail → i18n 渲染"链路。

### ParamProvider 机制

自定义 validator 规则（如 `titlesize`/`bodysize`）通过 `RegisterValidation` 注册，其 `fe.Param()` 永远返回空字符串（validator 只对内置带参规则提供 Param）。阈值来自运行时配置时，用 ErrCode 的 `ParamProvider` 字段提供：

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

`fieldErrorToDetail` 发现 `fe.Param()==""` 但 `code.ParamProvider != nil` 时，调用它取值。`config.Get()` 用 `sync.Once` 保证安全，返回值类型不 panic，最坏返回零值（文案瑕疵，非崩溃）。

## middleware 错误处理

### panic recovery（fallback middleware）

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

### 限流（tollbooth）

429 + `ErrRateLimited` + tollbooth 自动设置的 `X-Rate-Limit-Limit` / `X-Rate-Limit-Duration` 头。**不发 `Retry-After`**（tollbooth 不提供，客户端从头中剩余额度判断重试时机）。

## 故障定位（流派 2：无 label，靠 trace）

error 不携带操作上下文（无 label），定位靠可观测性体系：

**场景 1：已知 sentinel**（如 not_found）—— error 码直接映射，无需排查。

**场景 2：infra 意外 error**（如连接池耗尽）—— `app.jsonl` 记录 trace_id → `traces.jsonl` 查 span 调用链：
```
POST /:post_key → post.Create → posts.Insert（span 的 err 属性记录原始错误）
```
span name 提供层级化操作上下文，比单层 label 更精确。

**场景 3：service 内部 error**（如渲染失败）—— `service.Error.Description`（领域语义"render post failed"）+ span 调用链双重定位。

详见 [observability.md](./observability.md)。

## 四层兜底（防御性审查结论）

```
第一层：tagRegistry 兜底
        未知 validator tag → ErrFieldViolation + {{.Param}} 通用占位符

第二层：RegisterTagNameFunc 兜底
        字段无 json/form tag → 返回 Go 字段名

第三层：i18n 渲染兜底
        ginI18n.MustGetMessage 返回空 → code.Message.Other 英文原文（永不失败）

第四层：panic recovery
        fallback middleware，任何 panic → slog 记录 + 500
```

**审查结论**：0 panic 风险。
- `MustGetMessage` 不 panic（源码确认：`message, _ := GetMessage()` 丢弃 error 返回空串）
- `config.Get()` 不 panic（`sync.Once`，返回零值）
- validator 接口方法纯读取不 panic
- `errors.As` 返回 bool 不 panic

程序永不因 error 处理而崩溃，客户端永远收到格式正确的 ErrorResponse（即使 message 可能降级为英文原文）。

## 测试约定

- service 测试：断言 `errors.As(err, &service.Error{})` 后检查 `Code.Value` / `Code.HTTP`
- handler 测试：断言 HTTP 状态码 + ErrorResponse JSON 结构（code/message/errors 字段）
- binding 测试：覆盖各类 validator tag + JSON 反序列化错误 + 嵌套结构体字段名解析
