# 实现计划

- [ ] 1. 更新 Post 数据模型，添加 UserID 外键字段

  - 修改 `models.go` 中的 Post 结构体，添加 UserID 字段
  - 设置 UserID 为可空的外键字段，关联到 users 表
  - 添加 GORM 标签：`gorm:"index;foreignKey:ID;references:users"`
  - _需求: 2.1_

- [ ] 2. 更新数据库迁移逻辑

  - 修改 `db.go` 中的 `InitDB()` 函数，确保 AutoMigrate 正确处理新的外键关系
  - _需求: 2.1_

- [ ] 3. 实现 CreatePostWithUser 函数

  - 在 `db.go` 中创建新的 `CreatePostWithUser(title, body string, userID int) (*Post, error)` 函数
  - 实现创建 post 时自动关联指定用户的功能
  - 添加适当的错误处理，包括外键约束验证
  - _需求: 1.1, 2.1_

- [ ] 4. 实现 GetPostsByUserID 函数

  - 在 `db.go` 中创建新的 `GetPostsByUserID(userID int) ([]Post, error)` 函数
  - 实现通过用户 ID 查询该用户创建的所有 post 的功能
  - 添加适当的错误处理
  - _需求: 2.2_

- [ ] 5. 更新现有的 CreatePost 函数

  - 修改 `db.go` 中的 `CreatePost` 函数，添加可选的 userID 参数
  - 更新函数签名：`CreatePost(title, body string, userID ...int) (*Post, error)`
  - _需求: 2.1_

- [ ] 6. 更新 CreatePostHandler 以支持用户关联

  - 修改 `handlers.go` 中的 `CreatePostHandler` 函数
  - 从请求上下文中获取用户信息：`user, exists := c.Get("user")`
  - 调用更新后的 `CreatePost` 函数，传入用户 ID
  - 添加用户信息验证和错误处理
  - _需求: 1.1_

- [ ] 7. 添加数据库错误处理

  - 在数据库操作函数中添加外键约束违反的错误处理
  - 添加用户不存在时的错误处理
  - 实现统一的错误响应格式
  - _需求: 2.1, 2.2_
