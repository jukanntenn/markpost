# 技术方案设计

## 架构概述

本方案采用 React + TypeScript + Vite 技术栈，使用 react-bootstrap 作为 UI 组件库，react-router 进行路由管理，axios 处理 HTTP 请求。整体架构采用组件化设计，确保代码的可维护性和可扩展性。

## 技术栈

- **前端框架**: React 19.1.1 + TypeScript
- **构建工具**: Vite 7.1.0
- **UI 组件库**: react-bootstrap
- **路由管理**: react-router-dom
- **HTTP 客户端**: axios
- **状态管理**: React Hooks + localStorage

## 项目结构设计

```
frontend/src/
├── components/
│   ├── LoginPage.tsx          # 登录页面组件
│   ├── GitHubLoginButton.tsx  # GitHub 登录按钮组件
│   └── LoadingSpinner.tsx     # 加载状态组件
├── hooks/
│   ├── useAuth.ts             # 认证相关 Hook
│   └── useApi.ts              # API 调用 Hook
├── services/
│   └── authService.ts         # 认证服务
├── types/
│   └── auth.ts                # 认证相关类型定义
├── utils/
│   └── storage.ts             # 本地存储工具
├── App.tsx                    # 主应用组件
└── main.tsx                   # 应用入口
```

## 核心组件设计

### 1. LoginPage 组件

- **职责**: 登录页面的主容器
- **功能**:
  - 展示登录界面
  - 处理登录逻辑
  - 显示加载状态和错误信息
- **Props**: 无
- **State**:
  - `loading`: 加载状态
  - `error`: 错误信息

### 2. GitHubLoginButton 组件

- **职责**: GitHub 登录按钮
- **功能**:
  - 触发 GitHub 授权流程
  - 显示按钮状态
- **Props**:
  - `onClick`: 点击回调函数
  - `loading`: 加载状态
  - `disabled`: 禁用状态

### 3. LoadingSpinner 组件

- **职责**: 加载状态指示器
- **功能**: 显示加载动画
- **Props**:
  - `size`: 加载器大小
  - `text`: 加载文本

## Hook 设计

### 1. useAuth Hook

```typescript
interface UseAuthReturn {
  login: () => Promise<void>;
  logout: () => void;
  isAuthenticated: boolean;
  user: User | null;
  loading: boolean;
  error: string | null;
}
```

### 2. useApi Hook

```typescript
interface UseApiReturn {
  get: <T>(url: string) => Promise<T>;
  post: <T>(url: string, data: any) => Promise<T>;
  loading: boolean;
  error: string | null;
}
```

## 服务层设计

### AuthService

- **getGitHubAuthUrl()**: 获取 GitHub 授权 URL
- **handleGitHubCallback(code: string)**: 处理 GitHub 回调
- **saveToken(token: string)**: 保存 JWT token
- **getToken()**: 获取 JWT token
- **removeToken()**: 移除 JWT token

## 路由设计

```typescript
const routes = [
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/dashboard",
    element: <DashboardPage />,
    protected: true,
  },
  {
    path: "/",
    element: <Navigate to="/login" replace />,
  },
];
```

## 数据流设计

### 登录流程

1. 用户访问 `/login` 页面
2. 点击 "Login with GitHub" 按钮
3. 调用 `AuthService.getGitHubAuthUrl()` 获取授权 URL
4. 跳转到 GitHub 授权页面
5. 用户授权后，GitHub 重定向到 `/auth/github/callback?code=xxx`
6. 前端检测到 URL 参数，调用 `AuthService.handleGitHubCallback(code)`
7. 保存 JWT token 到 localStorage
8. 跳转到 `/dashboard` 页面

### 错误处理流程

1. API 调用失败时，显示错误信息
2. 网络错误时，显示网络错误提示
3. 授权失败时，显示授权失败信息
4. Token 存储失败时，显示存储错误信息

## 状态管理设计

### 本地状态

- 使用 React useState 管理组件内部状态
- 使用 localStorage 持久化存储 JWT token

### 全局状态

- 用户认证状态通过 Context API 管理
- 错误状态通过组件状态管理

## 安全考虑

1. **Token 存储**: JWT token 存储在 localStorage 中
2. **HTTPS**: 确保所有 API 调用使用 HTTPS
3. **错误信息**: 不暴露敏感信息在错误消息中
4. **状态验证**: 在关键操作前验证用户认证状态

## 性能优化

1. **代码分割**: 使用 React.lazy 进行路由级别的代码分割
2. **组件优化**: 使用 React.memo 优化组件重渲染
3. **API 缓存**: 对授权 URL 进行短期缓存
4. **加载优化**: 使用骨架屏和加载状态提升用户体验
