# Posts展示功能设计文档

## 概览

本文档描述了Markpost系统中posts展示功能的设计方案。该功能允许用户在Dashboard页面快速查看最新创建的10篇posts，并通过"查看全部"按钮访问完整的posts列表页面，支持分页浏览。

## 架构

### 整体架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │    Backend      │    │   Database      │
│   (React)       │    │   (Golang)      │    │  (PostgreSQL/   │
│                 │    │                 │    │   SQLite)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │  HTTP/REST            │  GORM                │
         │  (JSON)               │  (SQL)               │
         │                       │                       │
    ┌────▼────┐             ┌────▼────┐            ┌────▼────┐
    │  Pages  │             │ Handlers│            │  Data   │
    │         │             │         │            │         │
    │Dashboard│             │ Posts   │            │  Posts  │
    │Posts    │             │ API     │            │  Table  │
    └────┬────┘             └────┬────┘            └────┬────┘
         │                       │                       │
         └───────────┬───────────┴───────────────────────┘
                     │
              ┌──────▼──────┐
              │  Services   │
              │             │
              │ PostService │
              └─────────────┘
```

### 后端架构

采用三层架构模式：

1. **Handlers层** - HTTP请求处理，参数验证，响应格式化
2. **Services层** - 业务逻辑处理，事务管理
3. **Repository层** - 数据访问，数据库操作

## 组件和接口

### 后端API设计

#### 1. 获取用户所有Posts（分页）

Dashboard页面使用此接口获取最新10篇posts：`GET /api/posts?page=1&limit=10`

**端点：** `GET /api/posts?page=1&limit=20`

**请求参数：**
- `page` (int, query, optional): 页码，默认1
- `limit` (int, query, optional): 每页数量，默认20，最大100

**响应格式：**
```json
{
  "posts": [
    {
      "id": "string",
      "title": "string",
      "created_at": "string (ISO 8601 format)"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

#### 2. 错误响应格式

**未授权（401）：**
```json
{
  "code": "unauthorized",
  "message": "error.unauthorized"
}
```

**参数无效（400）：**
```json
{
  "code": "bad_request",
  "message": "error.invalid_parameters"
}
```

**内部错误（500）：**
```json
{
  "code": "internal",
  "message": "error.internal"
}
```

**未找到（404）：**
```json
{
  "code": "not_found",
  "message": "error.not_found"
}
```

### 前端组件设计

#### 1. Dashboard页面改造

**新增组件：**
- `RecentPosts` - 显示最新10篇posts的组件
- `ViewAllButton` - "查看全部"按钮

**布局结构：**
```
Dashboard (Container)
├── Post Key Section (现有)
├── API Docs Section (现有)
└── Recent Posts Section (新增)
    ├── Header with "Recent Posts" title
    ├── Posts List (最多10条)
    │   ├── PostItem (每条)
    │   │   ├── Clickable Title (链接到 /:post_id)
    │   │   └── Created Date
    │   └── Empty State (if no posts)
    └── View All Button (链接到 /ui/posts)
```

#### 2. Posts列表页面

**页面路径：** `/ui/posts`

**组件结构：**
```
Posts (Container)
├── Page Header
│   └── Title: "All Posts"
├── Posts List Section
│   ├── Posts List
│   │   ├── PostItem (每条)
│   │   │   ├── Clickable Title (链接到 /:post_id)
│   │   │   └── Created Date
│   │   └── Empty State (if no posts)
│   └── Loading State
└── Pagination
    ├── Previous Button (disabled if page=1)
    ├── Page Numbers
    ├── Next Button (disabled if last page)
    └── Page Info Text
```

**分页组件特性：**
- 显示格式："共X篇posts，第Y页/共Z页"
- 页码按钮最多显示5个（当前页前后各2个）
- 支持直接跳转到指定页码
- 每页默认20篇posts，可调整但不超过100

#### 3. 数据类型定义

**PostListItem接口：**
```typescript
interface PostListItem {
  id: string;
  title: string;
  created_at: string;
}

interface PostsResponse {
  posts: PostListItem[];
}

interface PostsPaginatedResponse {
  posts: PostListItem[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}
```

## 数据模型

### 现有Post模型

```go
type Post struct {
  ID        string    `json:"id" gorm:"primaryKey"`
  Title     string    `json:"title" gorm:"not null"`
  Body      string    `json:"body" gorm:"not null"`
  CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
  UserID    *int      `json:"user_id" gorm:"index;foreignKey:ID;references:users"`
}
```

### 新增Repository方法

#### PostRepository扩展

```go
// 获取用户最新N篇posts（按创建时间倒序）
func (r *postRepository) GetRecentPostsByUserID(userID int, limit int) ([]Post, error)

// 获取用户所有posts（分页，按创建时间倒序）
func (r *postRepository) GetPostsByUserIDPaginated(userID int, page, limit int) ([]Post, int64, error)
```

### 数据库查询策略

- 使用现有`UserID`索引进行关联查询
- 使用LIMIT和OFFSET进行分页
- 使用COUNT(*)计算总数量
- 按`created_at DESC`排序

## 错误处理

### 后端错误处理

1. **JWT认证失败** - 返回401未授权
2. **用户无posts** - 返回空数组，状态码200
3. **分页参数无效** - 返回400 Bad Request
4. **数据库查询失败** - 返回500内部错误

### 前端错误处理

1. **网络错误** - 显示错误提示，允许重试
2. **认证失效** - 自动重定向到登录页
3. **空数据状态** - 显示友好的空状态提示
4. **加载状态** - 显示loading spinner

## 测试策略

### 端到端测试

1. **Dashboard页面posts展示**
   - 验证显示最新10篇posts
   - 验证点击"查看全部"按钮跳转
   - 验证空状态显示

2. **Posts页面分页浏览**
   - 验证posts列表正确显示
   - 验证分页导航功能（上一页、下一页、页码跳转）
   - 验证总页数计算正确

3. **Post详情页跳转**
   - 验证点击post标题在新tab页打开正确页面
   - 验证post内容正确渲染

4. **权限控制**
   - 验证未登录用户重定向到登录页
   - 验证用户只能看到自己的posts

## 设计决策说明

### 1. 分页实现选择

**决策：** 服务端分页（offset/limit）

**原因：**
- 用户posts数量可能很大，服务端分页避免一次加载所有数据
- 减少网络传输和前端内存占用
- 数据库索引支持高效的offset查询

**备选方案：**
- 客户端分页：不适合大量数据
- 游标分页：更复杂，实现成本高

### 2. API响应格式

**决策：** 统一返回包含posts和pagination字段的对象

**原因：**
- 保持API一致性
- 便于前端处理和错误处理
- 扩展性强（如添加排序、筛选等）

### 3. 前端状态管理

**决策：** 使用React Hooks (useState, useEffect)

**原因：**
- 符合项目现有技术栈
- 组件逻辑简单，无需复杂状态管理
- 降低学习成本和维护复杂度

### 4. 新标签页打开

**决策：** 点击post标题在新tab页打开`/:post_id`

**原因：**
- 用户体验更好，不丢失当前页面状态
- 符合常规web应用习惯
- 便于用户对比和参考多个posts

### 5. 标题显示

**决策：** 标题完整显示，不截断

**原因：**
- 用户能够看到完整标题信息
- Dashboard页面可以通过CSS控制显示行数
- Posts列表页面空间充足，可以显示完整标题