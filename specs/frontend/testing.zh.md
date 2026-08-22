# 前端测试

[English](testing.md) | 中文

<a id="unit-tests"></a>

## 单元测试

<a id="framework"></a>

### 框架

- **Vitest** —— 带 jsdom 环境的测试运行器
- **Testing Library** —— 组件测试工具
- **MSW**（Mock Service Worker）—— API 请求模拟
- **@testing-library/jest-dom/vitest** —— DOM 断言

<a id="test-file-placement"></a>

### 测试文件位置

测试文件与源文件同目录：

```
src/components/ui/Button.tsx
src/components/ui/Button.test.tsx
```

测试文件使用 `.test.ts` 或 `.test.tsx` 扩展名（或 `.spec.ts` / `.spec.tsx`）。

<a id="test-setup"></a>

### 测试设置

`src/test/setup.ts` 配置 MSW 在测试期间拦截 API 请求：

```typescript
import { beforeAll, afterEach, afterAll } from "vitest";
import "@testing-library/jest-dom/vitest";
import { server } from "../mocks/server";

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
```

- MSW 在所有测试前启动并监听未处理请求（抛错）
- 测试之间重置 handlers 以隔离
- 所有测试结束后关闭 server

<a id="test-utilities"></a>

### 测试工具

`src/test/utils.tsx` 提供自定义 render 函数，用必要的 providers（QueryClient、NextIntl）包裹组件。

<a id="msw-handlers"></a>

### MSW handlers

`src/mocks/handlers.ts` 定义模拟 API 响应：

```typescript
export const handlers = [
  http.get("/api/v1/posts", () => {
    return HttpResponse.json(mockPosts);
  }),
  http.post("/api/v1/auth/login", async () => {
    return HttpResponse.json({ token: "...", user: {...} });
  }),
];
```

<a id="running-tests"></a>

### 运行测试

```bash
pnpm test          # Watch mode
pnpm test:run      # Single run (CI)
```

<a id="e2e-tests"></a>

## E2E 测试

<a id="framework-1"></a>

### 框架

- **Playwright** —— 浏览器自动化
- 浏览器：Chromium、Firefox、WebKit

<a id="test-file-placement-1"></a>

### 测试文件位置

E2E 测试位于前端根目录的 `tests/` 目录。

<a id="running-e2e-tests"></a>

### 运行 E2E 测试

```bash
pnpm test:e2e
```

<a id="coverage"></a>

## 覆盖率

Vitest 使用 V8 coverage provider。覆盖率配置位于 `vitest.config.ts`。
