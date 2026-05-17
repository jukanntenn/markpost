# Frontend Testing

## Unit Tests

### Framework

- **Vitest** — Test runner with jsdom environment
- **Testing Library** — Component testing utilities
- **MSW** (Mock Service Worker) — API request mocking
- **@testing-library/jest-dom/vitest** — DOM assertions

### Test File Placement

Test files are co-located with source files:

```
src/components/ui/Button.tsx
src/components/ui/Button.test.tsx
```

Test files use `.test.ts` or `.test.tsx` extensions (or `.spec.ts` / `.spec.tsx`).

### Test Setup

`src/test/setup.ts` configures MSW to intercept API requests during tests:

```typescript
import { beforeAll, afterEach, afterAll } from "vitest";
import "@testing-library/jest-dom/vitest";
import { server } from "../mocks/server";

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
```

- MSW starts before all tests and listens for unhandled requests (throws errors)
- Handlers reset between tests for isolation
- Server shuts down after all tests

### Test Utilities

`src/test/utils.tsx` provides a custom render function that wraps components with necessary providers (QueryClient, NextIntl).

### MSW Handlers

`src/mocks/handlers.ts` defines mock API responses:

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

### Running Tests

```bash
pnpm test          # Watch mode
pnpm test:run      # Single run (CI)
```

## E2E Tests

### Framework

- **Playwright** — Browser automation
- Browsers: Chromium, Firefox, WebKit

### Test File Placement

E2E tests are in the `tests/` directory at the frontend root.

### Running E2E Tests

```bash
pnpm test:e2e
```

## Coverage

Vitest uses V8 coverage provider. Coverage configuration is in `vitest.config.ts`.
