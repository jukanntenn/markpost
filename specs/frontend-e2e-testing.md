# Frontend E2E Testing

> E2E testing architecture with Playwright.

## Tech Stack

| Dependency | Version | Purpose |
|------------|---------|---------|
| `@playwright/test` | ^1.56.x | E2E test framework |
| `@testing-library/jest-dom` | ^6.x | DOM matchers |
| `@testing-library/react` | ^16.x | React testing |
| `@testing-library/user-event` | ^14.x | User interaction |

## File Organization

```
frontend/
├── playwright.config.ts
├── tests/
│   ├── fixtures.ts               # Custom fixtures and test config
│   ├── data/
│   │   └── mock-data.ts          # Mock data
│   ├── pages/                    # Page Object Models
│   │   ├── LoginPage.ts
│   │   ├── DashboardPage.ts
│   │   ├── PostsPage.ts
│   │   └── SettingsPage.ts
│   ├── login.spec.ts
│   ├── dashboard.spec.ts
│   ├── posts.spec.ts
│   ├── settings.spec.ts
│   ├── token-refresh.spec.ts
│   └── dashboard-create-post.spec.ts
```

## Playwright Config

```typescript
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  timeout: 30000,
  fullyParallel: true,
  workers: 2,
  reporter: [["list"]],
  use: {
    baseURL: "http://localhost:5174/ui/",
    trace: "on-first-retry",
  },
  webServer: {
    command: "VITE_PORT=5174 pnpm dev",
    url: "http://localhost:5174/ui/",
    reuseExistingServer: true,
    timeout: 120000,
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] }, workers: 1 },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
```

Key settings:
- `trace: "on-first-retry"` — record trace only on failure to save storage
- `reuseExistingServer: true` — use already-running dev server
- 3 browser engines: Chromium, Firefox, WebKit

## Fixtures

All tests must import from `./fixtures`, not directly from `@playwright/test`:

```typescript
// tests/fixtures.ts
import { test as base } from "@playwright/test";

export const test = base.extend<{ loginPage: LoginPage; /* ... */ }>({
  loginPage: async ({ page }, provide) => { await provide(new LoginPage(page)); },
  authenticatedPage: async ({ page }, provide) => {
    await page.goto("login");
    await page.evaluate((user) => {
      localStorage.setItem("markpost_dev_login", JSON.stringify(user));
      localStorage.setItem("i18nextLng", "en");
    }, mockUsers.e2e);
    await provide(page);
  },
});
export { expect } from "@playwright/test";
```

## Page Object Model

Standard structure:

```typescript
export class LoginPage {
  readonly page: Page;
  readonly usernameInput: Locator;
  readonly passwordInput: Locator;
  readonly submitButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.usernameInput = page.locator('input[name="username"]');
    this.passwordInput = page.locator('input[name="password"]');
    this.submitButton = page.locator('button[type="submit"]');
  }

  async goto() { await this.page.goto("login"); }
  async login(username: string, password: string) {
    await this.usernameInput.fill(username);
    await this.passwordInput.fill(password);
    await this.submitButton.click();
  }
  async getErrorMessage() { return this.page.locator(".alert-danger"); }
}
```

### Locator Strategy Priority

1. **Semantic role**: `page.getByRole("button", { name: "Submit" })`
2. **Text content**: `page.getByText("Welcome", { exact: true })`
3. **Label association**: `page.getByLabel("Email")`
4. **Placeholder**: `page.getByPlaceholder("Enter email")`
5. **Test ID**: `page.getByTestId("submit-btn")` (last resort)
6. **CSS selector**: `page.locator(".submit-btn")` (avoid)

### POM Method Naming

| Prefix | Purpose | Example |
|--------|---------|---------|
| `get*` | Return Locator | `getErrorMessage()` |
| `click*` | Click action | `clickSubmit()` |
| `fill*` | Fill form | `fillPasswordForm(current, new, confirm)` |
| `is*` | State check | `isSubmitDisabled()` |

## Mock Data

- **Scenario-based naming**: `mockUsers.tester`, `mockPosts.empty`, `mockPosts.single`
- **Match API structure**: mock data must match backend response format exactly
- **TypeScript typed**: define with proper types
- **Token isolation**: each test scenario uses independent tokens

## API Mock Patterns

| Pattern | Use Case |
|---------|----------|
| Basic mock | Standard API response |
| Delayed response | Test loading states (`setTimeout` 600ms) |
| Conditional response | Test state changes (empty → has data) |
| Network error | `route.abort()` for offline scenarios |
| Header validation | Verify auth headers are sent |

## Auth State Setup

- **Direct localStorage**: `page.evaluate()` to set auth data (fast, for most tests)
- **authenticatedPage fixture**: pre-configured page with auth state

## i18n Testing

Set locale via localStorage before reload:

```typescript
await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
await page.reload();
await expect(page.getByRole("button", { name: "Log in" })).toBeVisible();
```

## Assertion Patterns

```typescript
await expect(element).toBeVisible();
await expect(element).toContainText("expected text");
await expect(button).toBeDisabled();
await expect(input).toHaveValue("expected value");
await expect(page.locator(".item")).toHaveCount(3);
await page.waitForURL("**/dashboard");
```

## Test Naming

```typescript
// Good — describes specific behavior
test("shows error alert when backend returns message", ...);
test("redirects to login when unauthenticated", ...);

// Bad — too vague
test("login works", ...);
test("error test", ...);
```

## Running & Debugging

```bash
pnpm test:e2e:install    # Install browsers
pnpm test:e2e            # Run all tests
pnpm test:e2e:ui         # UI mode (recommended for debugging)
pnpm test:e2e tests/login.spec.ts  # Single file
pnpm test:e2e --project=chromium   # Single browser
pnpm test:e2e --headed             # Headed mode
```

Debug techniques:
- `--ui` mode for visual step-by-step
- `page.pause()` for breakpoints
- Trace files auto-generated on failure

## Integration Checklist

- [ ] `@playwright/test` installed
- [ ] `playwright.config.ts` configured
- [ ] `tests/fixtures.ts` with custom fixtures
- [ ] `tests/data/mock-data.ts` with test data
- [ ] Page Object classes for each page
- [ ] Spec files for each feature module
- [ ] npm scripts configured
- [ ] Tests pass in all 3 browser engines
