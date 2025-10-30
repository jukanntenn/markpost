# Posts展示功能实现任务规划

## 任务列表

- [ ] 1. 扩展PostRepository接口，添加分页查询方法
  - 在repositories.go中添加GetPostsByUserIDPaginated方法
  - 实现分页查询逻辑，支持按user_id过滤和created_at倒序排序
  - 返回posts列表和总数
  - _Requirements: 需求3-分页显示，需求4-内容展示，需求5-权限控制_

- [ ] 2. 扩展PostService，添加posts列表业务逻辑
  - 在services.go中添加GetPostsByUserIDPaginated方法
  - 调用PostRepository的GetPostsByUserIDPaginated方法
  - 封装错误处理逻辑
  - _Requirements: 需求3-分页显示，需求5-权限控制_

- [ ] 3. 创建Posts API Handler
  - 在handlers.go中添加ListPostsHandler函数
  - 实现JWT认证中间件验证
  - 解析page和limit查询参数（默认值：page=1, limit=20）
  - 参数验证：page >= 1, limit <= 100
  - 调用PostService获取posts数据
  - 返回标准格式JSON响应（posts数组 + pagination对象）
  - 实现错误处理：401未认证、400参数无效、500内部错误
  - _Requirements: 需求3-分页显示，需求5-权限控制_

- [ ] 4. 注册/posts API路由
  - 在routes.go中的jwtAuth分组下添加GET /posts路由
  - 映射到ListPostsHandler
  - _Requirements: 需求3-分页显示_

- [ ] 5. 在组件内直接实现API请求逻辑
  - 在RecentPosts组件内使用auth.get发送GET请求到/api/posts
  - 在Posts组件内使用auth.get发送GET请求到/api/posts
  - 实现分页参数支持（page, limit）
  - 使用auth实例自动添加JWT token
  - _Requirements: 需求3-分页显示_

- [ ] 6. 创建PostListItem TypeScript类型
  - 在frontend/src/types/目录下创建posts.ts文件
  - 定义PostListItem接口（id, title, created_at）
  - 定义PostsPaginatedResponse接口（posts, pagination）
  - 导出类型供组件使用
  - _Requirements: 需求4-内容展示_

- [ ] 7. 创建RecentPosts组件
  - 在frontend/src/components/下创建RecentPosts.tsx
  - 实现获取最新10篇posts的逻辑（调用getPosts，page=1, limit=10）
  - 显示post列表，每个条目包含：标题（可点击）、创建时间
  - 点击标题在新tab页打开/:post_id
  - 实现加载状态和空状态显示
  - 标题完整显示，不截断
  - _Requirements: 需求1-显示最新Posts，需求4-内容展示_

- [ ] 8. 集成RecentPosts到Dashboard页面
  - 修改frontend/src/pages/Dashboard.tsx
  - 在现有两个卡片下方添加RecentPosts组件
  - 添加"查看全部"按钮，点击导航到/ui/posts
  - 确保响应式布局
  - _Requirements: 需求1-显示最新Posts，需求2-查看全部按钮_

- [ ] 9. 创建Pagination组件
  - 在frontend/src/components/下创建Pagination.tsx
  - 实现上一页、下一页按钮
  - 显示页码（最多5个：当前页前后各2个）
  - 显示页码信息文本："共X篇posts，第Y页/共Z页"
  - 按钮禁用逻辑：第一页无上一页，最后一页无下一页
  - _Requirements: 需求3-分页显示_

- [ ] 10. 创建Posts页面组件
  - 在frontend/src/pages/下创建Posts.tsx
  - 实现posts列表显示（调用getPosts API）
  - 集成Pagination组件
  - 实现页面切换逻辑（更新URL query参数）
  - 显示加载状态和空状态
  - 点击post标题在新tab页打开/:post_id
  - 标题完整显示，不截断
  - _Requirements: 需求3-分页显示，需求4-内容展示_

- [ ] 11. 添加Posts页面路由
  - 修改frontend/src/App.tsx
  - 在ProtectedRoute下添加/ui/posts路由
  - 映射到Posts组件
  - _Requirements: 需求3-分页显示_

- [ ] 12. 创建端到端测试
  - 在frontend/cypress/e2e/下创建posts.cy.ts
  - 测试Dashboard页面显示最新10篇posts
  - 测试"查看全部"按钮跳转
  - 测试Posts页面分页功能
  - 测试点击post标题在新tab页打开
  - 验证权限控制（未登录重定向）
  - 验证用户只能看到自己的posts
  - _Requirements: 需求1-5全部_

## 实现顺序说明

1. 步骤1-4完成后端API基础设施
2. 步骤5-6完成前端类型定义和API调用
3. 步骤7-8完成Dashboard页面改造
4. 步骤9-11完成Posts页面和分页功能
5. 步骤12完成端到端测试验证

每个步骤完成后都可通过相关API或页面进行独立验证，确保增量式开发。