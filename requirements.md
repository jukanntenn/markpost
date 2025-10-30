# Posts展示功能需求文档

## 引言

本需求文档描述了为Markpost系统实现posts展示功能的需求。该功能允许用户在Dashboard页面快速查看最新创建的posts，并通过"查看全部"按钮访问完整的posts列表页面，以分页形式浏览所有posts。

## 需求

### 需求1：Dashboard页面显示最新Posts

**用户故事：** 作为登录用户，我希望在Dashboard页面快速查看我最新创建的10篇posts，以便我能够快速回顾最近的内容创作。

#### 验收标准

1. WHEN 用户访问Dashboard页面 THEN 系统 SHALL 显示用户最新创建的10篇posts
2. IF 用户没有posts THEN 系统 SHALL 显示空状态提示信息
3. WHEN 显示posts列表时 THEN 系统 SHALL 按创建时间倒序排列（最新的在前）
4. WHEN 显示post时 THEN 系统 SHALL 显示post标题、创建时间和内容摘要（限制长度）
5. IF post标题超过50个字符 THEN 系统 SHALL 显示前50个字符并添加省略号

### 需求2：查看全部按钮导航

**用户故事：** 作为登录用户，我希望在Dashboard有一个"查看全部"按钮，以便导航到完整的posts列表页面。

#### 验收标准

1. WHEN Dashboard页面显示posts列表时 THEN 系统 SHALL 显示"查看全部"按钮
2. WHEN 用户点击"查看全部"按钮 THEN 系统 SHALL 导航到`/ui/posts`页面
3. IF 用户未登录 THEN 系统 SHALL 重定向到登录页面

### 需求3：Posts列表页面分页显示

**用户Story：** 作为登录用户，我希望在`/ui/posts`页面以分页形式查看所有posts，以便能够浏览大量内容。

#### 验收标准

1. WHEN 用户访问`/ui/posts`页面 THEN 系统 SHALL 显示所有posts的分页列表
2. WHEN 用户首次访问posts页面 THEN 系统 SHALL 显示第1页，每页显示20篇posts
3. WHEN 用户点击分页导航按钮时 THEN 系统 SHALL 切换到对应页面并显示该页的posts
4. WHEN 用户点击页码时 THEN 系统 SHALL 跳转到指定页面
5. IF 当前页不是第一页 THEN 系统 SHALL 显示"上一页"按钮
6. IF 当前页不是最后一页 THEN 系统 SHALL 显示"下一页"按钮
7. WHEN 显示分页信息时 THEN 系统 SHALL 显示"共X篇posts，第Y页/共Z页"
8. WHEN 显示posts列表时 THEN 系统 SHALL 按创建时间倒序排列

### 需求4：Posts列表页面内容展示

**用户Story：** 作为登录用户，我希望在posts页面清晰查看每篇post的详细信息。

#### 验收标准

1. WHEN 显示posts列表时 THEN 系统 SHALL 显示每篇post的标题、创建时间和内容摘要
2. WHEN 用户点击post标题或摘要时 THEN 系统 SHALL 导航到对应的post查看页面
3. IF 用户没有posts THEN 系统 SHALL 显示空状态提示信息和引导创建提示
4. WHEN 页面加载时 THEN 系统 SHALL 显示loading状态

### 需求5：权限控制

**用户Story：** 作为系统，我希望确保只有登录用户才能访问posts相关功能。

#### 验收标准

1. IF 用户未登录访问Dashboard THEN 系统 SHALL 重定向到`/ui/login`
2. IF 用户未登录访问`/ui/posts` THEN 系统 SHALL 重定向到`/ui/login`
3. WHEN 用户访问posts功能时 THEN 系统 SHALL 验证用户身份并只显示该用户自己的posts