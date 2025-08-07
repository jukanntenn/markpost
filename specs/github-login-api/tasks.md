# GitHub 登录 API 实施计划

## 任务概览

本计划将 GitHub 登录 API 功能拆分为具体的实施任务，按照依赖关系和优先级进行排序。

## 实施任务

### 阶段 1: 依赖和配置准备

- [ ] 1. 添加项目依赖

  - 添加 `golang.org/x/oauth2` 依赖
  - 添加 `github.com/appleboy/gin-jwt/v2` 依赖
  - 更新 `go.mod` 和 `go.sum` 文件
  - \_需求: 需求 1 - GitHub OAuth2 配置

- [ ] 2. 扩展配置结构

  - 在 `config.go` 中添加 GitHub OAuth2 配置结构
  - 在 `config.go` 中添加 JWT 配置结构
  - 更新配置示例文件
  - \_需求: 需求 1 - GitHub OAuth2 配置

### 阶段 2: 数据模型更新

- [ ] 3. 扩展用户模型

  - 在 `models.go` 中为 User 结构体添加 GitHubID 字段
  - 使用指针类型 `*int64` 允许 NULL 值
  - 添加 `uniqueIndex` 标签
  - \_需求: 需求 2 - 用户模型扩展

### 阶段 3: 核心功能实现

- [ ] 4. 实现 GitHub OAuth2 配置初始化

  - 在 `handlers.go` 中添加 `initGitHubOAuth()` 函数
  - 配置 OAuth2 客户端
  - 设置授权端点和令牌端点
  - \_需求: 需求 1 - GitHub OAuth2 配置

- [ ] 5. 实现 GitHub 用户信息获取

  - 添加 `GitHubUser` 结构体
  - 实现 `exchangeCodeForToken()` 函数
  - 实现 `getUserInfo()` 函数
  - \_需求: 需求 3 - GitHub OAuth2 回调处理器

- [ ] 6. 实现用户查找和创建逻辑

  - 实现 `findOrCreateUser()` 函数
  - 处理现有用户和新用户场景
  - 实现 GitHub ID 唯一性检查
  - \_需求: 需求 2 - 用户模型扩展, 需求 3 - GitHub OAuth2 回调处理器

- [ ] 7. 实现 JWT 令牌生成

  - 添加 `TokenPair` 结构体
  - 实现 `generateTokenPair()` 函数
  - 生成访问令牌和刷新令牌
  - \_需求: 需求 4 - JWT 令牌管理

### 阶段 4: API 接口实现

- [ ] 8. 实现生成授权 URL 接口

  - 实现 `GenerateGitHubAuthURL()` 处理函数
  - 生成包含 state 参数的授权 URL
  - 返回 JSON 格式的响应
  - \_需求: 需求 3 - GitHub OAuth2 回调处理器

- [ ] 9. 实现 GitHub 回调处理接口

  - 实现 `HandleGitHubCallback()` 处理函数
  - 验证授权码和 state 参数
  - 调用用户查找和创建逻辑
  - 生成并返回 JWT 令牌
  - \_需求: 需求 3 - GitHub OAuth2 回调处理器

### 阶段 5: 路由和集成

- [ ] 10. 添加认证路由

  - 在 `routes.go` 中添加 `/auth` 路由组
  - 添加 `/auth/github/url` 路由
  - 添加 `/auth/github/callback` 路由
  - \_需求: 需求 3 - GitHub OAuth2 回调处理器

- [ ] 11. 初始化配置

  - 在 `main.go` 中调用 `initGitHubOAuth()`
  - 确保配置在应用启动时正确加载
  - \_需求: 需求 1 - GitHub OAuth2 配置

### 阶段 6: 错误处理

- [ ] 12. 实现错误处理

  - 添加 OAuth2 相关错误处理
  - 添加 JWT 相关错误处理
  - 添加数据库相关错误处理
  - 返回适当的 HTTP 状态码和错误消息
  - \_需求: 需求 5 - 错误处理

## 任务依赖关系

```
阶段 1 → 阶段 2 → 阶段 3 → 阶段 4 → 阶段 5 → 阶段 6
```

## 预估工作量

- **阶段 1**: 1-2 小时
- **阶段 2**: 1 小时
- **阶段 3**: 4-6 小时
- **阶段 4**: 2-3 小时
- **阶段 5**: 1 小时
- **阶段 6**: 2-3 小时

**总计预估**: 11-16 小时

## 风险点

1. **GitHub API 限制**: 需要处理 GitHub API 的速率限制
2. **数据库迁移**: 现有数据与新模型的兼容性
3. **JWT 安全**: 确保 JWT 密钥的安全性和令牌的有效性
4. **配置管理**: 确保敏感配置信息的安全存储

## 验收标准

- [ ] 用户可以通过 GitHub 账户成功登录
- [ ] 新用户自动创建账户
- [ ] 现有用户正确关联 GitHub ID
- [ ] JWT 令牌正确生成和返回
- [ ] 错误情况得到适当处理
