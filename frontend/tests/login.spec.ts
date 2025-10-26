import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.context().clearCookies();
  await page.goto("login");
  await page.evaluate(() => localStorage.clear());
});

test("renders login page and enables submit when inputs filled", async ({ page }) => {
  const username = page.locator('input[name="username"]');
  const password = page.locator('input[name="password"]');
  const submit = page.locator('button[type="submit"]');
  await expect(username).toBeVisible();
  await expect(password).toBeVisible();
  await expect(submit).toBeDisabled();
  await username.fill("testuser");
  await password.fill("testpass");
  await expect(submit).toBeEnabled();
});

test("keeps submit disabled when only one field is filled", async ({ page }) => {
  const username = page.locator('input[name="username"]');
  const password = page.locator('input[name="password"]');
  const submit = page.locator('button[type="submit"]');
  await username.fill("onlyuser");
  await expect(submit).toBeDisabled();
  await username.fill("");
  await password.fill("onlypass");
  await expect(submit).toBeDisabled();
});

test("shows loading state and disables fields during login", async ({ page }) => {
  await page.route("**/api/auth/login", async (route) => {
    await new Promise((r) => setTimeout(r, 600));
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } }),
    });
  });
  const username = page.locator('input[name="username"]');
  const password = page.locator('input[name="password"]');
  const submit = page.locator('button[type="submit"]');
  await username.fill("tester");
  await password.fill("secret");
  await submit.click();
  await expect(submit.getByText("Signing in...")).toBeVisible();
  await expect(submit.locator(".spinner-border")).toBeVisible();
  await expect(username).toBeDisabled();
  await expect(password).toBeDisabled();
  await page.unrouteAll({ behavior: "ignoreErrors" });
});

test("persists login and redirects to dashboard on success", async ({ page }) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.route("**/api/auth/login", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ access_token: "e2e_access_token", refresh_token: "e2e_refresh_token", user: { id: 1, username: "tester" } }),
    });
  });
  await page.route("**/api/post_key", async (route) => {
    const headers = route.request().headers();
    expect(headers["authorization"]).toBe("Bearer e2e_access_token");
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ post_key: "abc123", created_at: "2024-01-01T00:00:00Z" }),
    });
  });

  await page.evaluate(() => localStorage.removeItem("markpost_dev_login"));

  await page.locator('input[name="username"]').fill("tester");
  await page.locator('input[name="password"]').fill("secret");
  await page.locator('button[type="submit"]').click();
  await page.waitForURL("**/dashboard");
  await expect(page.getByText("Post Key", { exact: true })).toBeVisible({ timeout: 15000 });
});

test("shows error alert when backend returns message", async ({ page }) => {
  await page.route("**/api/auth/login", async (route) => {
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ message: "Invalid credentials" }),
    });
  });
  await page.locator('input[name="username"]').fill("tester");
  await page.locator('input[name="password"]').fill("wrong");
  await page.locator('button[type="submit"]').click();
  await expect(page.locator(".alert-danger")).toContainText("Invalid credentials");
});

test("clears error alert when inputs change", async ({ page }) => {
  await page.route("**/api/auth/login", async (route) => {
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ message: "Invalid credentials" }),
    });
  });
  const username = page.locator('input[name="username"]');
  const password = page.locator('input[name="password"]');
  const submit = page.locator('button[type="submit"]');
  await username.fill("tester");
  await password.fill("wrong");
  await submit.click();
  await expect(page.locator(".alert-danger")).toBeVisible();
  await username.type("x");
  await expect(page.locator(".alert-danger")).toHaveCount(0);
});

test("shows toast when login response shape is invalid", async ({ page }) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.route("**/api/auth/login", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ access_token: "t", user: { id: 1, username: "tester" } }),
    });
  });
  await page.reload();
  await page.locator('input[name="username"]').fill("tester");
  await page.locator('input[name="password"]').fill("secret");
  await page.locator('button[type="submit"]').click();
  await expect(page.getByText("Login failed")).toBeVisible();
  await expect(page.getByText("Login error, please try again later")).toBeVisible();
});

test("shows toast on network error", async ({ page }) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.route("**/api/auth/login", async (route) => {
    await route.abort();
  });
  await page.locator('input[name="username"]').fill("tester");
  await page.locator('input[name="password"]').fill("secret");
  await page.locator('button[type="submit"]').click();
  await expect(page.getByText("Login failed")).toBeVisible();
  await expect(page.getByText("Unknown error")).toBeVisible();
});

test("submits form when pressing Enter in password field", async ({ page }) => {
  await page.route("**/api/auth/login", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } }),
    });
  });
  await page.locator('input[name="username"]').fill("tester");
  const password = page.locator('input[name="password"]');
  await password.fill("secret");
  await Promise.all([
    page.waitForRequest("**/api/auth/login"),
    password.press("Enter"),
  ]);
});

test("uses Accept-Language header with English language", async ({ page }, testInfo) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.route("**/api/auth/login", async (route) => {
    const h = route.request().headers();
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^en-US/);
    }
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } }) });
  });
  await page.reload();
  await page.locator('input[name="username"]').fill("tester");
  await page.locator('input[name="password"]').fill("secret");
  await page.locator('button[type="submit"]').click();
});

test("uses Accept-Language header with Chinese language", async ({ page }, testInfo) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "zh"));
  await page.route("**/api/auth/login", async (route) => {
    const h = route.request().headers();
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^zh-CN/);
    }
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ access_token: "t2", refresh_token: "r2", user: { id: 2, username: "tester2" } }) });
  });
  
  await page.reload();
  
  await expect(page.getByRole("button", { name: "登录", exact: true })).toBeVisible();
  await page.locator('input[name="username"]').fill("tester2");
  await page.locator('input[name="password"]').fill("secret2");
  await page.locator('button[type="submit"]').click();
});

test("redirects to dashboard when already authenticated", async ({ page }) => {
  await page.route("**/api/post_key", async (route) => {
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ post_key: "abc123", created_at: "2024-01-01T00:00:00Z" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.goto("login");
  await page.waitForURL("**/dashboard");
});

test("displays divider text", async ({ page }) => {
  await expect(page.getByText("or", { exact: true })).toBeVisible();
});
