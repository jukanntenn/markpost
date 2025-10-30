# Implementation Plan
# 创建测试 Post 功能实施任务列表

- [ ] 1. 实现 API 工具函数
  - [ ] 1.1 在 utils/api.ts 中添加 createTestPost 函数
    - 函数接收 postKey、title、body 参数
    - 调用 POST `/${postKey}` 接口
    - 返回创建的 Post ID
    - 引用设计文档章节：Components and Interfaces - API 工具函数接口
    - Requirements: Requirement 4 - 创建 Post 逻辑

- [ ] 2. 创建 CreateTestPostModal 组件
  - [ ] 2.1 创建 CreateTestPostModal.tsx 组件文件
    - 定义 CreateTestPostModalProps 接口
    - 使用 React Bootstrap Modal 组件
    - 实现状态管理（title、body、loading、error）
    - 引用设计文档章节：Components and Interfaces - CreateTestPostModal 组件接口
    - Requirements: Requirement 2 - 模态框界面

  - [ ] 2.2 实现表单字段和布局
    - 标题输入框（可选）
    - 内容输入框（必填，textarea）
    - 取消和创建按钮
    - 引用设计文档章节：用户界面设计 - 表单布局
    - Requirements: Requirement 2 - 模态框界面

  - [ ] 2.3 实现表单提交逻辑
    - 调用 createTestPost 函数
    - 处理成功响应：关闭模态框、调用 onSuccess
    - 处理错误响应：显示错误信息、保持模态框打开
    - 处理加载状态：提交时禁用按钮
    - 引用设计文档章节：Error Handling - 错误处理策略
    - Requirements: Requirement 4 - 创建 Post 逻辑

  - [ ] 2.4 实现模态框关闭功能
    - 支持点击 X 按钮关闭
    - 支持点击背景遮罩关闭
    - 支持按 ESC 键关闭
    - 关闭时清空表单数据
    - Requirements: Requirement 2 - 模态框界面

- [ ] 3. 集成到 Dashboard 页面
  - [ ] 3.1 修改 Dashboard.tsx 文件
    - 导入 CreateTestPostModal 组件
    - 导入 createTestPost 函数
    - 添加 showCreateModal 状态
    - 引用设计文档章节：Components and Interfaces - Dashboard.tsx 集成接口
    - Requirements: Requirement 1 - 界面入口按钮

  - [ ] 3.2 在 Post Key 板块添加按钮
    - 在 Post Key 卡片底部添加 "Create Test Post" 按钮
    - 使用 variant="primary", size="sm"
    - 点击时设置 showCreateModal 为 true
    - 引用设计文档章节：用户界面设计 - 按钮设计
    - Requirements: Requirement 1 - 界面入口按钮

  - [ ] 3.3 集成 CreateTestPostModal 组件
    - 传递 show、postKey、onHide、onSuccess 属性
    - 在 onSuccess 中调用 loadRecentPosts 刷新 Recent Posts
    - 引用设计文档章节：Components and Interfaces - Dashboard.tsx 集成接口
    - Requirements: Requirement 1 - 界面入口按钮

- [ ] 4. 编写端对端测试
  - [ ] 4.1 创建端对端测试文件
    - 在 frontend/tests/ 目录下创建 dashboard-create-post.e2e.spec.ts
    - 引用设计文档章节：Testing Strategy - 端对端测试
    - Requirements: Requirement 4 - 创建 Post 逻辑, Requirement 1 - 界面入口按钮

  - [ ] 4.2 实现完整创建流程测试
    - 测试点击按钮打开模态框
    - 测试填写表单并提交
    - 测试验证模态框关闭
    - 测试验证 Recent Posts 刷新
    - 引用设计文档章节：Testing Strategy - 完整创建流程测试案例
    - Requirements: Requirement 4 - 创建 Post 逻辑

  - [ ] 4.3 实现验证失败场景测试
    - 测试提交空内容场景
    - 测试提交过长标题或内容场景
    - 验证错误信息显示
    - 验证模态框保持打开
    - 引用设计文档章节：Testing Strategy - 验证失败场景测试案例
    - Requirements: Requirement 3 - 表单验证

  - [ ] 4.4 实现错误处理场景测试
    - 测试服务器 500 错误
    - 测试网络超时错误
    - 验证错误信息显示
    - 验证模态框保持打开
    - 引用设计文档章节：Testing Strategy - 验证错误处理测试案例
    - Requirements: Requirement 4 - 创建 Post 逻辑

  - [ ] 4.5 实现模态框交互测试
    - 测试点击 X 按钮关闭
    - 测试点击背景遮罩关闭
    - 测试按 ESC 键关闭
    - 测试点击取消按钮关闭
    - 验证表单数据清空
    - 引用设计文档章节：Testing Strategy - 验证模态框关闭功能测试案例
    - Requirements: Requirement 2 - 模态框界面
