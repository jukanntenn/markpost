# E2E 测试架构 Specification

本文档定义了 E2E 测试的标准架构规范，供其他项目参考实现完全一致的测试体系。

## 1. 技术栈与依赖

### 1.1 核心框架

| 依赖 | 版本范围 | 用途 |
|------|----------|------|
| `@playwright/test` | ^1.56.x | E2E 测试核心框架 |
| `@testing-library/jest-dom` | ^6.x | DOM 匹配器扩展 |
| `@testing-library/react` | ^16.x | React 组件测试工具 |
| `@testing-library/user-event` | ^14.x | 用户交互模拟 |

### 1.2 开发依赖

```json
{
  "devDependencies": {
    "@playwright/test": "^1.56.1",
    "@testing-library/jest-dom": "^6.9.1",
    "@testing-library/react": "^16.3.0",
    "@testing-library/user-event": "^14.6.1"
  }
}
```

### 1.3 npm scripts

```json
{
  "scripts": {
    "test:e2e": "playwright test",
    "test:e2e:ui": "playwright test --ui",
    "test:e2e:install": "playwright install"
  }
}
```

## 2. 项目文件组织结构

```
frontend/
├── playwright.config.ts          # Playwright 配置文件
├── package.json
├── tests/                        # E2E 测试目录
│   ├── fixtures.ts               # 自定义 fixtures 和测试配置
│   ├── data/                     # 测试数据目录
│   │   └── mock-data.ts          # Mock 数据定义
│   ├── pages/                    # Page Object Model 目录
│   │   ├── LoginPage.ts
│   │   ├── DashboardPage.ts
│   │   ├── PostsPage.ts
│   │   └── SettingsPage.ts
│   ├── login.spec.ts             # 登录模块测试
│   ├── dashboard.spec.ts         # Dashboard 模块测试
│   ├── posts.spec.ts             # Posts 模块测试
│   ├── settings.spec.ts          # Settings 模块测试
│   ├── token-refresh.spec.ts     # Token 刷新测试
│   └── dashboard-create-post.spec.ts  # 复杂业务流程测试
```

## 3. Playwright 配置规范

### 3.1 标准配置模板

```typescript
// playwright.config.ts
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",           // 测试文件目录
  timeout: 30000,               // 单个测试超时时间
  fullyParallel: true,          // 并行执行测试
  workers: 2,                   // worker 数量
  reporter: [["list"]],         // 报告器配置

  use: {
    baseURL: "http://localhost:5174/ui/",  // 基础 URL
    trace: "on-first-retry",               // 失败重试时记录 trace
  },

  webServer: {
    command: "VITE_PORT=5174 pnpm dev",    // 启动开发服务器命令
    url: "http://localhost:5174/ui/",      // 等待就绪的 URL
    reuseExistingServer: true,             // 复用已有服务器
    timeout: 120000,                       // 服务器启动超时
  },

  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] }, workers: 1 },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
```

### 3.2 配置要点

- **testDir**: 测试文件统一放在 `./tests` 目录
- **timeout**: 单测试 30 秒，适合复杂业务场景
- **fullyParallel**: 启用并行执行提高效率
- **workers**: 限制为 2 个，避免资源竞争
- **trace**: 仅在重试时记录，节省存储空间
- **webServer**: 自动启动开发服务器，支持复用已有实例
- **projects**: 覆盖三大浏览器引擎

## 4. 测试 Fixtures 规范

### 4.1 Fixtures 定义

```typescript
// tests/fixtures.ts
import { test as base, type Page } from "@playwright/test";
import { LoginPage } from "./pages/LoginPage";
import { DashboardPage } from "./pages/DashboardPage";
import { PostsPage } from "./pages/PostsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { mockUsers } from "./data/mock-data";

// 定义 fixtures 类型
export type TestFixtures = {
  loginPage: LoginPage;
  dashboardPage: DashboardPage;
  postsPage: PostsPage;
  settingsPage: SettingsPage;
  authenticatedPage: Page;  // 预认证的页面
};

// 扩展 base test
export const test = base.extend<TestFixtures>({
  // Page Object fixtures
  loginPage: async ({ page }, provide) => {
    await provide(new LoginPage(page));
  },

  dashboardPage: async ({ page }, provide) => {
    await provide(new DashboardPage(page));
  },

  postsPage: async ({ page }, provide) => {
    await provide(new PostsPage(page));
  },

  settingsPage: async ({ page }, provide) => {
    await provide(new SettingsPage(page));
  },

  // 预认证 fixture - 直接注入已认证状态的页面
  authenticatedPage: async ({ page }, provide) => {
    await page.goto("login");
    await page.evaluate((user) => {
      localStorage.setItem("markpost_dev_login", JSON.stringify(user));
      localStorage.setItem("i18nextLng", "en");
    }, mockUsers.e2e);
    await provide(page);
  },
});

// 导出 expect 供测试使用
export { expect } from "@playwright/test";
```

### 4.2 Fixtures 使用规范

所有测试文件必须从 `./fixtures` 导入 `test` 和 `expect`：

```typescript
// 正确用法
import { test, expect } from "./fixtures";

// 错误用法 - 不要直接从 @playwright/test 导入
import { test, expect } from "@playwright/test";
```

## 5. Page Object Model 规范

### 5.1 Page Object 结构模板

```typescript
// tests/pages/LoginPage.ts
import type { Page, Locator } from "@playwright/test";

export class LoginPage {
  // 1. 页面引用
  readonly page: Page;

  // 2. Locator 定义 - 每个可交互元素独立定义
  readonly usernameInput: Locator;
  readonly passwordInput: Locator;
  readonly submitButton: Locator;
  readonly alertDanger: Locator;

  // 3. 构造函数 - 初始化所有 Locator
  constructor(page: Page) {
    this.page = page;
    this.usernameInput = page.locator('input[name="username"]');
    this.passwordInput = page.locator('input[name="password"]');
    this.submitButton = page.locator('button[type="submit"]');
    this.alertDanger = page.locator(".alert-danger");
  }

  // 4. 导航方法
  async goto() {
    await this.page.goto("login");
  }

  // 5. 业务操作方法
  async login(username: string, password: string) {
    await this.usernameInput.fill(username);
    await this.passwordInput.fill(password);
    await this.submitButton.click();
  }

  // 6. 辅助查询方法
  async getErrorMessage() {
    return this.alertDanger;
  }

  async isSubmitDisabled() {
    return await this.submitButton.isDisabled();
  }
}
```

### 5.2 Locator 选择策略优先级

1. **语义化角色** (首选): `page.getByRole("button", { name: "Submit" })`
2. **文本内容**: `page.getByText("Welcome", { exact: true })`
3. **标签关联**: `page.getByLabel("Email")`
4. **占位符**: `page.getByPlaceholder("Enter email")`
5. **测试 ID**: `page.getByTestId("submit-btn")` (最后选择)
6. **CSS 选择器**: `page.locator(".submit-btn")` (尽量避免)

### 5.3 Page Object 方法命名规范

| 方法前缀 | 用途 | 示例 |
|----------|------|------|
| `get*` | 返回 Locator | `getErrorMessage()`, `getPostLink(title)` |
| `click*` | 点击操作 | `clickSubmit()`, `clickUserMenu()` |
| `fill*` | 填充表单 | `fillPasswordForm(current, new, confirm)` |
| `is*` | 状态检查 | `isSubmitDisabled()`, `isSubmitEnabled()` |

## 6. Mock 数据规范

### 6.1 数据组织结构

```typescript
// tests/data/mock-data.ts

// 用户 Mock 数据
export const mockUsers = {
  tester: {
    id: 1,
    username: "tester",
    access_token: "test_token",
    refresh_token: "test_refresh",
  },
  admin: {
    id: 2,
    username: "admin",
    access_token: "admin_token",
    refresh_token: "admin_refresh",
  },
  e2e: {
    id: 1,
    username: "tester",
    access_token: "e2e_access_token",
    refresh_token: "e2e_refresh_token",
  },
};

// 业务数据 Mock
export const mockPosts = {
  empty: {
    posts: [],
    pagination: { page: 1, limit: 20, total: 0, total_pages: 0 },
  },
  single: {
    posts: [{ id: "p1", title: "Test Post", created_at: "2024-01-01T00:00:00Z" }],
    pagination: { page: 1, limit: 20, total: 1, total_pages: 1 },
  },
  multiple: {
    posts: [
      { id: "p1", title: "Post One", created_at: "2024-01-01T12:00:00Z" },
      { id: "p2", title: "Post Two", created_at: "2024-01-02T13:00:00Z" },
    ],
    pagination: { page: 1, limit: 20, total: 2, total_pages: 1 },
  },
};

// 特定业务 Mock
export const mockPostKey = {
  post_key: "test_key_abc123",
  created_at: "2024-01-01T00:00:00Z",
};
```

### 6.2 数据设计原则

1. **场景化命名**: `empty`, `single`, `multiple` 便于理解用途
2. **结构一致**: 与后端 API 响应结构完全匹配
3. **类型安全**: 使用 TypeScript 类型约束
4. **隔离性**: 每个测试场景使用独立的 token 避免污染

## 7. 测试编写模式与最佳实践

### 7.1 测试文件结构

每个 spec 文件遵循以下结构：

```typescript
// 1. 导入
import { test, expect } from "./fixtures";

// 2. beforeEach 钩子 - 测试前置条件
test.beforeEach(async ({ page }) => {
  await page.context().clearCookies();
  await page.goto("login");
  await page.evaluate(() => localStorage.clear());
  // 设置通用路由 Mock
  await page.route("**/api/posts**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ posts: [], pagination: { ... } }),
    });
  });
});

// 3. 测试用例
test("description of test case", async ({ page, loginPage }) => {
  // 测试代码
});
```

### 7.2 API Mock 模式

#### 7.2.1 基础 Mock

```typescript
await page.route("**/api/auth/login", async (route) => {
  await route.fulfill({
    status: 200,
    contentType: "application/json",
    body: JSON.stringify({
      access_token: "t",
      refresh_token: "r",
      user: { id: 1, username: "tester" },
    }),
  });
});
```

#### 7.2.2 延迟响应 Mock (测试 Loading 状态)

```typescript
await page.route("**/api/auth/login", async (route) => {
  await new Promise((r) => setTimeout(r, 600));  // 延迟 600ms
  await route.fulfill({
    status: 200,
    contentType: "application/json",
    body: JSON.stringify({ ... }),
  });
});
```

#### 7.2.3 条件响应 Mock

```typescript
let callCount = 0;
await page.route("**/api/posts**", async (route) => {
  callCount++;
  if (callCount === 1) {
    // 第一次调用返回空数据
    await route.fulfill({ status: 200, body: JSON.stringify(mockPosts.empty) });
  } else {
    // 第二次调用返回有数据
    await route.fulfill({ status: 200, body: JSON.stringify(mockPosts.single) });
  }
});
```

#### 7.2.4 网络错误 Mock

```typescript
await page.route("**/api/auth/login", async (route) => {
  await route.abort();  // 模拟网络中断
});
```

#### 7.2.5 请求头验证 Mock

```typescript
await page.route("**/api/post_key", async (route) => {
  const headers = route.request().headers();
  expect(headers["authorization"]).toBe("Bearer e2e_access_token");
  await route.fulfill({ ... });
});
```

#### 7.2.6 清理路由

```typescript
// 测试结束后清理路由
await page.unrouteAll({ behavior: "ignoreErrors" });
```

### 7.3 认证状态设置模式

#### 7.3.1 直接设置 localStorage (快速认证)

```typescript
await page.evaluate((user) => {
  localStorage.setItem("markpost_dev_login", JSON.stringify(user));
  localStorage.setItem("i18nextLng", "en");
}, mockUsers.e2e);
```

#### 7.3.2 使用 authenticatedPage fixture

```typescript
test("authenticated test", async ({ authenticatedPage, dashboardPage }) => {
  // authenticatedPage 已预置认证状态
  await dashboardPage.goto();
  // ...
});
```

### 7.4 国际化测试模式

```typescript
// 测试英文界面
test("English UI", async ({ page, loginPage }) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.reload();
  await expect(page.getByRole("button", { name: "Log in" })).toBeVisible();
});

// 测试中文界面
test("Chinese UI", async ({ page, loginPage }) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "zh"));
  await page.reload();
  await expect(page.getByRole("button", { name: "登录" })).toBeVisible();
});
```

### 7.5 浏览器特定测试

```typescript
test("chromium only test", async ({ page }, testInfo) => {
  await page.route("**/api/auth/login", async (route) => {
    const h = route.request().headers();
    // 仅在 chromium 项目中验证
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^en-US/);
    }
    await route.fulfill({ ... });
  });
});
```

### 7.6 断言模式

#### 7.6.1 可见性断言

```typescript
await expect(element).toBeVisible();
await expect(element).not.toBeVisible();
```

#### 7.6.2 文本内容断言

```typescript
await expect(element).toContainText("expected text");
await expect(element).toHaveText("exact text");
```

#### 7.6.3 表单状态断言

```typescript
await expect(button).toBeDisabled();
await expect(button).toBeEnabled();
await expect(input).toHaveValue("expected value");
```

#### 7.6.4 数量断言

```typescript
await expect(page.locator(".alert")).toHaveCount(0);  // 元素不存在
await expect(page.locator(".item")).toHaveCount(3);   // 精确数量
```

#### 7.6.5 URL 断言

```typescript
await page.waitForURL("**/dashboard");
await expect(page).toHaveURL(/dashboard/);
```

#### 7.6.6 localStorage 断言

```typescript
await expect
  .poll(() => page.evaluate(() => localStorage.getItem("key")))
  .toContain("expected");
```

### 7.7 Clipboard API Mock

```typescript
await page.addInitScript(() => {
  const nav = window.navigator as Navigator & {
    clipboard?: { writeText?: (text: string) => Promise<void> };
  };
  try {
    const clip = nav.clipboard;
    if (clip && typeof clip.writeText === "function") {
      clip.writeText = () => Promise.resolve();
    } else {
      Object.defineProperty(nav, "clipboard", {
        value: { writeText: () => Promise.resolve() },
        configurable: true,
      });
    }
  } catch {
    Object.defineProperty(nav, "clipboard", {
      value: { writeText: () => Promise.resolve() },
      configurable: true,
    });
  }
});
```

## 8. 测试用例分类

### 8.1 按测试类型分类

| 类型 | 描述 | 示例文件 |
|------|------|----------|
| 认证测试 | 登录、登出、Token 刷新 | `login.spec.ts`, `token-refresh.spec.ts` |
| 页面渲染测试 | UI 元素、状态展示 | `dashboard.spec.ts`, `settings.spec.ts` |
| 表单交互测试 | 表单验证、提交 | `settings.spec.ts` |
| 业务流程测试 | 完整业务操作流程 | `dashboard-create-post.spec.ts` |
| 权限控制测试 | 未认证重定向 | 所有 spec 文件 |

### 8.2 测试命名规范

```typescript
// 好的命名 - 描述具体行为和预期结果
test("shows error alert when backend returns message", async () => { ... });
test("redirects to login when unauthenticated", async () => { ... });
test("clears error alert when inputs change", async () => { ... });

// 不好的命名 - 过于模糊
test("login works", async () => { ... });
test("error test", async () => { ... });
```

## 9. 常见测试场景模板

### 9.1 认证保护测试

```typescript
test("redirects to login when unauthenticated", async ({ page }) => {
  await page.goto("dashboard");
  await page.waitForURL("**/login");
});
```

### 9.2 加载状态测试

```typescript
test("shows loading state during fetch", async ({ page, dashboardPage }) => {
  await page.route("**/api/post_key", async (route) => {
    await new Promise((r) => setTimeout(r, 600));
    await route.fulfill({ ... });
  });
  // 设置认证状态...
  await dashboardPage.goto();
  await expect(page.getByText("Loading...")).toBeVisible();
  await page.unrouteAll({ behavior: "ignoreErrors" });
});
```

### 9.3 错误处理测试

```typescript
// 服务器错误
test("shows error on server error", async ({ page, dashboardPage }) => {
  await page.route("**/api/post_key", async (route) => {
    await route.fulfill({ status: 500, body: JSON.stringify({}) });
  });
  // ...
  await expect(page.getByText("Failed to load")).toBeVisible();
});

// 网络错误
test("shows error on network abort", async ({ page, dashboardPage }) => {
  await page.route("**/api/post_key", async (route) => {
    await route.abort();
  });
  // ...
  await expect(page.getByText("Failed to load")).toBeVisible();
});
```

### 9.4 Token 刷新测试

```typescript
test("silently refreshes token on 401", async ({ page, dashboardPage }) => {
  // 设置过期 token
  await page.evaluate(() => {
    localStorage.setItem("markpost_dev_login", JSON.stringify({
      access_token: "expired_access",
      refresh_token: "refresh_1",
      user: { id: 1, username: "tester" },
    }));
  });

  let refreshCount = 0;
  await page.route("**/api/auth/refresh", async (route) => {
    refreshCount++;
    await route.fulfill({
      status: 200,
      body: JSON.stringify({
        access_token: "new_access",
        refresh_token: "new_refresh",
        user: { id: 1, username: "tester" },
      }),
    });
  });

  // 模拟 401 后重试
  await page.route("**/api/post_key", async (route) => {
    const h = route.request().headers();
    if (h["authorization"] === "Bearer expired_access") {
      await route.fulfill({ status: 401 });
      return;
    }
    expect(h["authorization"]).toBe("Bearer new_access");
    await route.fulfill({ status: 200, body: JSON.stringify({ post_key: "abc" }) });
  });

  await dashboardPage.goto();
  expect(refreshCount).toBe(1);
});
```

## 10. 运行与调试

### 10.1 运行命令

```bash
# 安装浏览器
pnpm test:e2e:install

# 运行所有测试
pnpm test:e2e

# UI 模式运行 (推荐调试使用)
pnpm test:e2e:ui

# 运行特定文件
pnpm test:e2e tests/login.spec.ts

# 运行特定浏览器
pnpm test:e2e --project=chromium

# 带 headed 模式
pnpm test:e2e --headed
```

### 10.2 调试技巧

1. **使用 `--ui` 模式**: 可视化查看每一步操作
2. **trace 文件**: 失败后可在 Trace Viewer 中查看
3. **`page.pause()`**: 在代码中插入断点
4. **`console.log`**: 在测试中打印调试信息

## 11. 项目集成检查清单

新项目实现 E2E 测试架构时，需确认以下项目：

- [ ] 安装 `@playwright/test` 及相关依赖
- [ ] 创建 `playwright.config.ts` 配置文件
- [ ] 创建 `tests/` 目录结构
- [ ] 实现 `tests/fixtures.ts` 自定义 fixtures
- [ ] 实现 `tests/data/mock-data.ts` 测试数据
- [ ] 为每个页面创建 Page Object 类
- [ ] 为每个功能模块创建对应的 spec 文件
- [ ] 配置 npm scripts
- [ ] 确保测试在三个浏览器引擎中通过
