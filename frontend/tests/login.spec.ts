import { test, expect } from "./fixtures";

test.beforeEach(async ({ page }) => {
  await page.context().clearCookies();
  await page.goto("login");
  await page.evaluate(() => localStorage.clear());
  await page.route("**/api/posts**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        posts: [],
        pagination: { page: 1, limit: 10, total: 0, total_pages: 0 },
      }),
    });
  });
});

test("renders login page and enables submit when inputs filled", async ({
  loginPage,
}) => {
  await expect(loginPage.usernameInput).toBeVisible();
  await expect(loginPage.passwordInput).toBeVisible();
  await expect(loginPage.submitButton).toBeDisabled();

  await loginPage.usernameInput.fill("testuser");
  await loginPage.passwordInput.fill("testpass");

  await expect(loginPage.submitButton).toBeEnabled();
});

test("keeps submit disabled when only one field is filled", async ({
  loginPage,
}) => {
  await loginPage.usernameInput.fill("onlyuser");
  await expect(loginPage.submitButton).toBeDisabled();

  await loginPage.usernameInput.fill("");
  await loginPage.passwordInput.fill("onlypass");

  await expect(loginPage.submitButton).toBeDisabled();
});

test("shows loading state and disables fields during login", async ({
  page,
  loginPage,
}) => {
  await page.route("**/api/auth/login", async (route) => {
    await new Promise((r) => setTimeout(r, 600));
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

  await loginPage.usernameInput.fill("tester");
  await loginPage.passwordInput.fill("secret");
  await loginPage.submitButton.click();

  await expect(loginPage.submitButton.getByText("Signing in...")).toBeVisible();
  await expect(loginPage.submitButton.locator(".spinner-border")).toBeVisible();
  await expect(loginPage.usernameInput).toBeDisabled();
  await expect(loginPage.passwordInput).toBeDisabled();

  await page.unrouteAll({ behavior: "ignoreErrors" });
});

test("persists login and redirects to dashboard on success", async ({
  page,
  loginPage,
  dashboardPage,
}) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.route("**/api/auth/login", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        access_token: "e2e_access_token",
        refresh_token: "e2e_refresh_token",
        user: { id: 1, username: "tester" },
      }),
    });
  });
  await page.route("**/api/post_key", async (route) => {
    const headers = route.request().headers();
    expect(headers["authorization"]).toBe("Bearer e2e_access_token");
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        post_key: "abc123",
        created_at: "2024-01-01T00:00:00Z",
      }),
    });
  });

  await page.evaluate(() => localStorage.removeItem("markpost_dev_login"));
  await loginPage.login("tester", "secret");

  await page.waitForURL("**/dashboard");
  await expect(dashboardPage.postKeyHeading).toBeVisible({ timeout: 15000 });
});

test("shows error alert when backend returns message", async ({
  page,
  loginPage,
}) => {
  await page.route("**/api/auth/login", async (route) => {
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ message: "Invalid credentials" }),
    });
  });

  await loginPage.login("tester", "wrong");
  const error = await loginPage.getErrorMessage();
  await expect(error).toContainText("Invalid credentials");
});

test("clears error alert when inputs change", async ({ page, loginPage }) => {
  await page.route("**/api/auth/login", async (route) => {
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ message: "Invalid credentials" }),
    });
  });

  await loginPage.login("tester", "wrong");
  const error = await loginPage.getErrorMessage();
  await expect(error).toBeVisible();

  await loginPage.usernameInput.fill("x");
  await expect(page.locator(".alert-danger")).toHaveCount(0);
});

test("shows toast when login response shape is invalid", async ({
  page,
  loginPage,
}) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.route("**/api/auth/login", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        access_token: "t",
        user: { id: 1, username: "tester" },
      }),
    });
  });
  await page.reload();

  await loginPage.login("tester", "secret");

  await expect(page.getByText("Login failed")).toBeVisible();
  await expect(
    page.getByText("Login error, please try again later")
  ).toBeVisible();
});

test("shows toast on network error", async ({ page, loginPage }) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.route("**/api/auth/login", async (route) => {
    await route.abort();
  });

  await loginPage.login("tester", "secret");

  await expect(page.getByText("Login failed")).toBeVisible();
  await expect(page.getByText("Unknown error")).toBeVisible();
});

test("submits form when pressing Enter in password field", async ({
  page,
  loginPage,
}) => {
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

  await loginPage.usernameInput.fill("tester");
  await loginPage.passwordInput.fill("secret");
  await loginPage.submitByPressingEnter();
});

test("uses Accept-Language header with English language", async ({
  page,
  loginPage,
}, testInfo) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.route("**/api/auth/login", async (route) => {
    const h = route.request().headers();
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^en-US/);
    }
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
  await page.reload();
  await expect(
    page.getByRole("button", { name: "Log in", exact: true })
  ).toBeVisible();
  await loginPage.login("tester", "secret");
});

test("uses Accept-Language header with Chinese language", async ({
  page,
  loginPage,
}, testInfo) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "zh"));
  await page.route("**/api/auth/login", async (route) => {
    const h = route.request().headers();
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^zh-CN/);
    }
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        access_token: "t2",
        refresh_token: "r2",
        user: { id: 2, username: "tester2" },
      }),
    });
  });

  await page.reload();
  await expect(
    page.getByRole("button", { name: "登录", exact: true })
  ).toBeVisible();
  await loginPage.login("tester2", "secret2");
});

test("redirects to dashboard when already authenticated", async ({
  page,
  dashboardPage,
}) => {
  await page.route("**/api/post_key", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        post_key: "abc123",
        created_at: "2024-01-01T00:00:00Z",
      }),
    });
  });
  await page.evaluate(() =>
    localStorage.setItem(
      "markpost_dev_login",
      JSON.stringify({
        access_token: "t",
        refresh_token: "r",
        user: { id: 1, username: "tester" },
      })
    )
  );
  await page.goto("login");
  await page.waitForURL("**/dashboard");
  await expect(dashboardPage.postKeyHeading).toBeVisible();
});

test("displays divider text", async ({ page }) => {
  await expect(page.getByText("or", { exact: true })).toBeVisible();
});
