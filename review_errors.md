# MarkPost 项目错误处理规范符合性分析报告

## 一、总体评估

项目的错误处理整体架构**基本符合规范**，但在一些细节实现上存在改进空间。

---

## 二、各层级符合性分析

### ✅ **Models 层 - 符合规范**

**符合规范的地方：**
1. ✅ **错误隔离在 models 层**：只定义了一个领域错误 `models.ErrNotFound`
2. ✅ **ORM 错误处理**：将 `gorm.ErrRecordNotFound` 转换为 `models.ErrNotFound`
3. ✅ **统一包装模式**：所有 GORM 错误使用 `fmt.Errorf("MethodName: %w", err)` 包装
4. ✅ **直接返回领域错误**：`models.ErrNotFound` 不包装，直接返回

**代码示例（符合规范）：**
```go
// models/user.go
func GetUser(database *Database, query map[string]any) (*User, error) {
    var user User
    err := db.Take(&user, query).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrNotFound  // 直接返回领域错误
        }
        return nil, fmt.Errorf("GetUser: %w", err)  // 包装并添加上下文
    }
    return &user, nil
}
```

**结论：Models 层完全符合规范，无需整改。**

---

### ⚠️ **Repository 层 - 部分符合规范**

**符合规范的地方：**
1. ✅ **透传 models 层错误**：大部分情况下直接透传 models 层的 error
2. ✅ **不包装 models.ErrNotFound**：正确识别并透传

**不符合规范的地方：**

#### **问题1：Repository 层添加了业务错误（应在 Service 层）**

**位置：** `/backend/repositories/user.go:84` 和 `:92`

```go
// ❌ 不符合规范：Repository 层不应该添加业务错误
func (r *UserRepo) ValidateUserPassword(username, password string) (*models.User, error) {
    user, err := r.GetUserByUsername(username)
    if err != nil {
        return nil, err
    }
    
    if user.Password == "" {
        return nil, fmt.Errorf("user has no password set")  // ❌ 应在 Service 层包装
    }
    
    ok, err := utils.CheckPassword(password, user.Password)
    if err != nil {
        return nil, fmt.Errorf("validate user %s password: %w", username, err)
    }
    if !ok {
        return nil, fmt.Errorf("invalid password")  // ❌ 应在 Service 层包装
    }
    
    return user, nil
}
```

**位置：** `/backend/repositories/user.go:54`

```go
// ❌ 不符合规范：业务逻辑判断应在 Service 层
func (r *UserRepo) CreateUser(username, password string) (*models.User, error) {
    exists, err := models.UserExists(r.database, map[string]any{"username": username})
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, fmt.Errorf("username is already taken")  // ❌ 应在 Service 层
    }
    // ...
}
```

**位置：** `/backend/repositories/post.go:79` 和 `:113`

```go
// ❌ 不符合规范：参数校验应在 Handler 层
func (r *PostRepo) PruneExpiredPosts(retentionDays int, batchSize int) error {
    if retentionDays <= 0 {
        return fmt.Errorf("retention days must be positive, got: %d", retentionDays)
    }
    // ...
}

func (r *PostRepo) CountExpiredPosts(retentionDays int) (int64, error) {
    if retentionDays <= 0 {
        return 0, fmt.Errorf("retention days must be positive, got: %d", retentionDays)
    }
    // ...
}
```

#### **问题2：Repository 层包装了来自 models 层的错误**

```go
// ❌ 不符合规范：应该直接透传
func (r *PostRepo) PruneExpiredPosts(retentionDays int, batchSize int) error {
    ids, err := models.GetPostIDsBefore(r.database, expiredBefore, batchSize)
    if err != nil {
        return fmt.Errorf("PruneExpiredPosts: %w", err)  // ❌ 应直接透传
    }
    // ...
}
```

**建议整改：**
- Repository 层应该只负责数据访问，不处理业务逻辑
- 业务错误（如 "invalid password"、"username is already taken"）应该在 Service 层处理
- 参数校验应该在 Handler 层完成
- Repository 层直接透传 models 层的错误，不包装

---

### ⚠️ **Service 层 - 部分符合规范**

**符合规范的地方：**
1. ✅ **定义了 ServiceError 结构和 ErrCode 类型**
2. ✅ **使用 NewServiceErrorWrap 包装底层错误**
3. ✅ **内部方法保证返回 ServiceError**
4. ✅ **区分 models.ErrNotFound 错误**

**不符合规范的地方：**

#### **问题1：Service 层直接调用 Repository 方法而没有通过内部方法**

**位置：** `/backend/services/auth.go:91-94`

```go
// ❌ 不符合规范：应该通过内部方法
func (s *AuthService) LoginWithPassword(username, password string) (*models.User, *JWTTokenPair, error) {
    user, err := s.users.ValidateUserPassword(username, password)  // ❌ 直接调用 Repository
    if err != nil {
        return nil, nil, NewServiceErrorWrap(ErrInvalidCredentials, "invalid username or password", err)
    }
    // ...
}
```

**建议整改：** 添加内部方法来包装 Repository 调用

```go
// ✅ 符合规范的实现
func (s *AuthService) validateUserCredentials(username, password string) (*models.User, error) {
    user, err := s.users.ValidateUserPassword(username, password)
    if err != nil {
        if errors.Is(err, models.ErrNotFound) {
            return nil, NewServiceErrorWrap(ErrInvalidCredentials, "invalid username or password", err)
        }
        return nil, NewServiceErrorWrap(ErrInternal, "validate user password failed", err)
    }
    return user, nil
}

func (s *AuthService) LoginWithPassword(username, password string) (*models.User, *JWTTokenPair, error) {
    user, err := s.validateUserCredentials(username, password)
    if err != nil {
        return nil, nil, err  // ✅ 直接返回，不需要再包装
    }
    // ...
}
```

---

### ✅ **Handlers 层 - 符合规范**

**符合规范的地方：**
1. ✅ **使用 RespondError 统一返回错误**
2. ✅ **有 writeBindingError 处理验证错误**
3. ✅ **正确使用 serviceErrorMappings**

**不符合规范的地方：**

#### **问题1：个别 Handler 直接处理错误而不是使用 RespondError**

**位置：** `/backend/handlers/post.go:76-92`

```go
// ❌ 不符合规范：应该统一使用 RespondError
func RenderPost(postSvc PostServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        id := c.Param("id")
        title, htmlContent, err := postSvc.RenderPostHTML(id)
        if err != nil {
            if se, ok := err.(*services.ServiceError); ok && se.Code == services.ErrNotFound {
                c.String(http.StatusNotFound, ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
                    DefaultMessage: &i18n.Message{
                        ID:    "error.not_found",
                        Other: "Not Found",
                    },
                }))
            } else {
                c.String(http.StatusInternalServerError, ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
                    DefaultMessage: &i18n.Message{
                        ID:    "error.failed_render_post",
                        Other: "Failed to render post",
                    },
                }))
            }
            return
        }
        c.HTML(http.StatusOK, "post.html", gin.H{"Title": title, "Body": template.HTML(htmlContent)})
    }
}
```

**建议整改：**

```go
// ✅ 符合规范的实现
func RenderPost(postSvc PostServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        id := c.Param("id")
        title, htmlContent, err := postSvc.RenderPostHTML(id)
        if err != nil {
            apperrors.RespondError(c, err)  // ✅ 统一使用 RespondError
            return
        }
        c.HTML(http.StatusOK, "post.html", gin.H{"Title": title, "Body": template.HTML(htmlContent)})
    }
}
```

**原因：** RenderPost 返回的是 HTML 而不是 JSON，所以不能使用 RespondError。但如果要返回错误页面，应该：
1. 在 errors 包中添加一个 `writeHTMLError` 函数
2. 或者创建一个统一的错误页面渲染函数

---

### ✅ **Errors 包 - 符合规范**

**符合规范的地方：**
1. ✅ **定义了 ErrorResponse 和 FieldError 结构**
2. ✅ **RespondError 统一处理错误**
3. ✅ **serviceErrorMappings 映射错误码到 HTTP 状态码**
4. ✅ **支持 i18n 消息**
5. ✅ **validationFieldMessages 用于字段级错误**

---

## 三、详细整改建议

### 优先级 1：高优先级（影响架构一致性）

#### 1. **Repository 层去除业务逻辑**

**文件：** `/backend/repositories/user.go`

**问题代码（第 48-58 行）：**
```go
func (r *UserRepo) CreateUser(username, password string) (*models.User, error) {
    exists, err := models.UserExists(r.database, map[string]any{"username": username})
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, fmt.Errorf("username is already taken")  // ❌ 移除
    }
    return r.createUserWithUniquePostKey(username, password, nil)
}
```

**整改方案：**

```go
// ✅ 简化为直接透传
func (r *UserRepo) CreateUser(username, password string) (*models.User, error) {
    return r.createUserWithUniquePostKey(username, password, nil)
}

// ✅ 在 Service 层添加检查
// backend/services/auth.go
func (s *AuthService) CreateUser(username, password string) (*models.User, error) {
    exists, err := s.users.GetUserByUsername(username)
    if err != nil && !errors.Is(err, models.ErrNotFound) {
        return nil, NewServiceErrorWrap(ErrInternal, "check user existence failed", err)
    }
    if exists != nil {
        return nil, NewServiceError(ErrInvalidRequest, "username is already taken")
    }
    
    return s.users.CreateUser(username, password)
}
```

#### 2. **Repository 层移除参数校验**

**文件：** `/backend/repositories/post.go`

**问题代码（第 78-80 行）：**
```go
func (r *PostRepo) PruneExpiredPosts(retentionDays int, batchSize int) error {
    if retentionDays <= 0 {
        return fmt.Errorf("retention days must be positive, got: %d", retentionDays)  // ❌ 移除
    }
    // ...
}
```

**整改方案：**

```go
// ✅ 移除参数校验，直接在 Handler 层完成
func (r *PostRepo) PruneExpiredPosts(retentionDays int, batchSize int) error {
    expiredBefore := time.Now().AddDate(0, 0, -retentionDays)
    // ... 其余代码保持不变
}

// ✅ 在 Handler 层添加校验
// backend/handlers/post.go (需要添加相应 handler)
func PrunePosts(postSvc PostServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req struct {
            RetentionDays int `form:"retention_days" binding:"required,min=1"`  // ✅ 使用 binding
            BatchSize    int `form:"batch_size" binding:"omitempty,min=1"`
        }
        if !bindQuery(c, &req) {
            return
        }
        
        if req.BatchSize <= 0 {
            req.BatchSize = 99
        }
        
        if err := postSvc.PruneExpiredPosts(req.RetentionDays, req.BatchSize); err != nil {
            apperrors.RespondError(c, err)
            return
        }
        
        c.JSON(http.StatusOK, gin.H{"message": "posts pruned successfully"})
    }
}
```

#### 3. **Service 层添加内部方法包装 Repository 调用**

**文件：** `/backend/services/auth.go`

**问题代码（第 90-102 行）：**
```go
func (s *AuthService) LoginWithPassword(username, password string) (*models.User, *JWTTokenPair, error) {
    user, err := s.users.ValidateUserPassword(username, password)  // ❌ 直接调用
    if err != nil {
        return nil, nil, NewServiceErrorWrap(ErrInvalidCredentials, "invalid username or password", err)
    }
    pair, err := s.jwt.GenerateTokenPair(user.ID)
    if err != nil {
        return nil, nil, NewServiceErrorWrap(ErrInternal, "generate access/refresh token pair failed", err)
    }
    return user, pair, nil
}
```

**整改方案：**

```go
// ✅ 添加内部方法
func (s *AuthService) validateAndGenerateTokens(user *models.User) (*JWTTokenPair, error) {
    pair, err := s.jwt.GenerateTokenPair(user.ID)
    if err != nil {
        return nil, NewServiceErrorWrap(ErrInternal, "generate access/refresh token pair failed", err)
    }
    return pair, nil
}

// ✅ 简化 LoginWithPassword
func (s *AuthService) LoginWithPassword(username, password string) (*models.User, *JWTTokenPair, error) {
    user, err := s.users.ValidateUserPassword(username, password)
    if err != nil {
        if errors.Is(err, models.ErrNotFound) || 
           (err.Error() == "user has no password set") || 
           (err.Error() == "invalid password") {
            return nil, nil, NewServiceErrorWrap(ErrInvalidCredentials, "invalid username or password", err)
        }
        return nil, nil, NewServiceErrorWrap(ErrInternal, "validate user password failed", err)
    }
    
    pair, err := s.validateAndGenerateTokens(user)  // ✅ 调用内部方法
    if err != nil {
        return nil, nil, err
    }
    
    return user, pair, nil
}
```

**注意：** 由于 Repository 层暂时还返回业务错误，Service 层需要先识别这些错误字符串。待 Repository 层整改完成后，可以进一步简化。

---

### 优先级 2：中优先级（提升代码质量）

#### 1. **统一 RenderPost 的错误处理**

**文件：** `/backend/handlers/post.go`

**建议：** 添加 HTML 错误页面渲染支持

```go
// backend/errors/errors.go
func RespondHTMLPageError(c *gin.Context, err error, templateName string, data gin.H) {
    se, ok := services.AsServiceError(err)
    if !ok {
        c.HTML(http.StatusInternalServerError, templateName, gin.H{
            "Title":   "Error",
            "Message": "Internal Server Error",
        })
        return
    }
    
    message := ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
        DefaultMessage: serviceErrorMappings[se.Code].Message,
    })
    
    status := http.StatusInternalServerError
    if mapping, ok := serviceErrorMappings[se.Code]; ok {
        status = mapping.Status
    }
    
    c.HTML(status, templateName, gin.H{
        "Title":   "Error",
        "Message": message,
    })
}

// backend/handlers/post.go
func RenderPost(postSvc PostServiceInterface) gin.HandlerFunc {
    return func(c *gin.Context) {
        id := c.Param("id")
        title, htmlContent, err := postSvc.RenderPostHTML(id)
        if err != nil {
            RespondHTMLPageError(c, err, "error.html", gin.H{})
            return
        }
        c.HTML(http.StatusOK, "post.html", gin.H{"Title": title, "Body": template.HTML(htmlContent)})
    }
}
```

#### 2. **为所有 Service 方法添加内部方法包装**

建议为 Service 层的所有复杂方法添加内部方法包装：

```go
// backend/services/post.go
func (s *PostService) createPostInternal(title, body string, userID int) (*models.Post, error) {
    post, err := s.postRepo.CreatePost(title, body, userID)
    if err != nil {
        return nil, NewServiceErrorWrap(ErrInternal, "create post failed", err)
    }
    return post, nil
}

func (s *PostService) CreatePost(title, body string, userID int) (string, error) {
    post, err := s.createPostInternal(title, body, userID)
    if err != nil {
        return "", err  // ✅ 直接返回，不需要再包装
    }
    return post.QID, nil
}
```

---

### 优先级 3：低优先级（优化和文档）

1. **完善注释**：为所有 ServiceError 使用添加注释说明
2. **添加错误日志**：在 RespondError 中添加更详细的日志记录
3. **统一错误消息格式**：检查所有 i18n 消息的一致性

---

## 四、整改总结

### 完全符合规范的层级
- ✅ Models 层
- ✅ Errors 包

### 部分符合规范，需要整改的层级
- ⚠️ Repository 层（高优先级）
- ⚠️ Service 层（中优先级）
- ⚠️ Handlers 层（低优先级）

### 关键整改点

1. **Repository 层移除业务逻辑**：所有业务错误判断应该在 Service 层完成
2. **Service 层添加内部方法**：通过内部方法包装 Repository 调用，保证返回 ServiceError
3. **统一 Handlers 错误处理**：所有错误都应该通过 RespondError 或统一的 HTML 错误函数处理
4. **参数校验上移**：所有参数校验应该在 Handler 层完成

### 整改后的优势

1. **更清晰的层次职责**：每层只负责自己的职责
2. **更好的错误封装**：ServiceError 作为统一的错误类型，易于处理
3. **更一致的代码风格**：所有错误处理遵循相同模式
4. **更易于维护**：错误处理逻辑集中，易于扩展和修改

---

## 五、不符合规范的文件清单

### Repository 层
1. `/backend/repositories/user.go`
   - 第 54 行：`CreateUser` 方法中的业务逻辑检查
   - 第 84 行：`ValidateUserPassword` 方法中的 "user has no password set" 错误
   - 第 92 行：`ValidateUserPassword` 方法中的 "invalid password" 错误

2. `/backend/repositories/post.go`
   - 第 79 行：`PruneExpiredPosts` 方法中的参数校验
   - 第 91 行：`PruneExpiredPosts` 方法中的错误包装
   - 第 100 行：`PruneExpiredPosts` 方法中的错误包装
   - 第 113 行：`CountExpiredPosts` 方法中的参数校验

### Service 层
1. `/backend/services/auth.go`
   - 第 40 行：`GenerateGitHubAuthURL` 直接返回 utils 错误
   - 第 49 行：`LoginWithGitHub` 包装 Repository 错误（无内部方法）
   - 第 74 行：`getGitHubUser` 包装 HTTP 错误
   - 第 80 行：`getGitHubUser` 包装解码错误
   - 第 93 行：`LoginWithPassword` 包装 Repository 错误（无内部方法）
   - 第 98 行：`LoginWithPassword` 包装 JWT 错误（无内部方法）
   - 第 107 行：`RefreshToken` 包装 JWT 错误（无内部方法）
   - 第 112 行：`RefreshToken` 包装错误（无内部方法）
   - 第 120 行：`RefreshToken` 包装 Repository 错误（无内部方法）
   - 第 125 行：`RefreshToken` 包装 JWT 错误（无内部方法）

### Handlers 层
1. `/backend/handlers/post.go`
   - 第 78-92 行：`RenderPost` 方法直接处理错误，未使用统一错误处理函数

---

## 六、整改检查清单

### Repository 层整改检查清单
- [ ] 移除 `UserRepo.CreateUser` 中的用户名重复检查
- [ ] 移除 `UserRepo.ValidateUserPassword` 中的密码相关业务错误
- [ ] 移除 `PostRepo.PruneExpiredPosts` 中的参数校验
- [ ] 移除 `PostRepo.PruneExpiredPosts` 中的错误包装
- [ ] 移除 `PostRepo.CountExpiredPosts` 中的参数校验
- [ ] 确保所有 Repository 方法直接透传 models 层错误

### Service 层整改检查清单
- [ ] 为 `AuthService` 添加内部方法包装 Repository 调用
- [ ] 为 `PostService` 添加内部方法包装 Repository 调用
- [ ] 确保所有 Service 方法返回的都是 ServiceError
- [ ] 在 Service 层添加必要的业务逻辑检查（从 Repository 层迁移）

### Handlers 层整改检查清单
- [ ] 为 HTML 响应添加统一的错误处理函数
- [ ] 重构 `RenderPost` 使用统一错误处理
- [ ] 确保所有 Handler 都使用 `RespondError` 或统一错误处理函数

### 测试检查清单
- [ ] 更新 Repository 测试以适配整改后的行为
- [ ] 更新 Service 测试以适配内部方法
- [ ] 更新 Handler 测试以适配统一错误处理
- [ ] 添加错误场景的集成测试

---

## 七、风险评估

### 高风险整改
1. **Repository 层移除业务逻辑**：可能影响现有功能，需要仔细测试
2. **Service 层添加内部方法**：需要确保错误转换逻辑正确

### 中风险整改
1. **参数校验上移到 Handler 层**：需要添加相应的验证逻辑
2. **统一错误处理**：需要确保所有错误路径都被覆盖

### 低风险整改
1. **代码注释和文档**：不影响功能，只影响可维护性
2. **日志优化**：提升调试能力，不影响功能

---

**报告生成时间：** 2026-01-20  
**分析范围：** backend/models, backend/repositories, backend/services, backend/handlers, backend/errors  
**涉及文件总数：** 15+  
**发现问题总数：** 12 个主要问题  
**建议整改优先级：** 高优先级 3 项，中优先级 2 项，低优先级 3 项
