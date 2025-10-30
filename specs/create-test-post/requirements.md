# Requirements Document
# 创建测试 Post 功能需求

## Introduction

为 Markpost 系统添加一个快速创建测试 Post 的功能，允许用户在 Dashboard 页面通过便捷的模态框界面创建临时的 Markdown 内容 Post，无需跳转页面，提升用户体验和工作效率。

## Requirements

### Requirement 1: 界面入口按钮

**User Story:** 作为用户，我希望在 Dashboard 页面的 Post Key 板块有一个醒目的 "Create Test Post" 按钮，以便我能够快速访问创建测试 Post 的功能。

#### Acceptance Criteria

1. WHEN 用户访问 Dashboard 页面 THEN 系统 SHALL 在 Post Key 板块显示 "Create Test Post" 按钮
2. WHEN 按钮被渲染 THEN 按钮 SHALL 具有清晰的视觉样式，区别于其他次要按钮
3. WHEN 用户悬停在按钮上 THEN 系统 SHALL 显示 Tooltip 说明该功能用途
4. WHEN 按钮被点击 THEN 系统 SHALL 弹出创建测试 Post 的模态框

### Requirement 2: 模态框界面

**User Story:** 作为用户，我希望通过一个简洁的模态框界面输入测试 Post 的标题和 Markdown 内容，以便我能够方便地创建和格式化测试内容。

#### Acceptance Criteria

1. WHEN 模态框打开 THEN 系统 SHALL 显示标题输入框（标记为可选）
2. WHEN 模态框打开 THEN 系统 SHALL 显示 Markdown 内容输入区域
3. WHEN 模态框打开 THEN 系统 SHALL 显示 "Cancel" 和 "Create Post" 按钮
4. WHEN 用户点击模态框外部区域 THEN 系统 SHALL 关闭模态框并取消操作
5. WHEN 用户按下 ESC 键 THEN 系统 SHALL 关闭模态框并取消操作
6. WHEN 模态框获得焦点 THEN 系统 SHALL 确保焦点在第一个输入框（标题）上
7. IF 模态框内容超过视口高度 THEN 系统 SHALL 显示滚动条以确保所有内容可见

### Requirement 3: 表单验证

**User Story:** 作为用户，我希望系统能够验证我输入的内容是否符合要求，以便我能够及时修正错误并成功创建 Post。

#### Acceptance Criteria

1. WHEN 用户尝试提交空白内容 THEN 系统 SHALL 显示错误提示："内容不能为空"
2. WHEN 用户输入超过系统限制的 Markdown 内容 THEN 系统 SHALL 显示错误提示："内容过长"
3. WHEN 标题输入超过限制 THEN 系统 SHALL 显示错误提示："标题过长"
4. WHEN 表单验证失败 THEN 系统 SHALL 禁用 "Create Post" 按钮
5. WHEN 表单验证通过 THEN 系统 SHALL 启用 "Create Post" 按钮

### Requirement 4: 创建 Post 逻辑

**User Story:** 作为用户，我希望提交后系统能够向 `/api/:post_key` 接口发送请求创建 Post，以便我能够成功生成测试内容。

#### Acceptance Criteria

1. WHEN 用户点击 "Create Post" 按钮 AND 表单验证通过 THEN 系统 SHALL 向 `/:post_key` 端点发送 POST 请求
2. WHEN 请求发送中 THEN 系统 SHALL 显示加载状态并禁用提交按钮
3. WHEN 请求成功返回 200 状态码 THEN 系统 SHALL 显示成功消息并关闭模态框
4. WHEN 请求返回 400/401/403 等客户端错误 THEN 系统 SHALL 显示相应的错误消息并保持模态框打开
5. WHEN 请求返回 500 等服务器错误 THEN 系统 SHALL 显示 "服务器错误，请稍后重试"
6. WHEN 请求超时 THEN 系统 SHALL 显示 "请求超时，请检查网络连接"
7. WHEN 创建成功 THEN 系统 SHALL 刷新 Recent Posts 板块的内容

### Requirement 5: 用户体验优化

**User Story:** 作为用户，我希望整个创建过程流畅且有清晰的反馈，以便我能够高效地创建多个测试 Post。

#### Acceptance Criteria

1. WHEN 用户开始输入内容 THEN 系统 SHALL 实时禁用提交按钮直到内容有效
2. WHEN 模态框打开 THEN 系统 SHALL 自动聚焦到第一个输入框
3. WHEN 创建成功 THEN 系统 SHALL 显示 Toast 消息："测试 Post 创建成功"
4. WHEN 创建失败 THEN 系统 SHALL 显示错误消息但不关闭模态框
5. WHEN 用户关闭模态框 THEN 系统 SHALL 清空所有输入内容
