import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.context().clearCookies();
  await page.goto("login");
  await page.evaluate(() => localStorage.clear());
});

test("redirects to login when unauthenticated", async ({ page }) => {
  await page.goto("dashboard");
  await page.waitForURL("**/login");
});

test("shows loading state during post key fetch", async ({ page }) => {
  await page.route("**/api/post_key", async (route) => {
    await new Promise((r) => setTimeout(r, 600));
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ post_key: "abc123", created_at: "2024-01-01T00:00:00Z" }),
    });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("dashboard");
  await expect(page.getByText("Loading your post key...", { exact: true })).toBeVisible();
  await page.unrouteAll({ behavior: "ignoreErrors" });
});

test("renders post key masked by default and toggles visibility", async ({ page }) => {
  await page.route("**/api/post_key", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ post_key: "abc123", created_at: "2024-01-01T00:00:00Z" }),
    });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("dashboard");
  await expect(page.getByText("Post Key", { exact: true })).toBeVisible();
  await expect(page.getByTitle("Show key")).toBeVisible();
  await page.getByTitle("Show key").click();
  await expect(page.getByTitle("Hide key")).toBeVisible();
  await expect(page.getByText("abc123", { exact: true })).toBeVisible();
  await page.getByTitle("Hide key").click();
  await expect(page.getByText("abc123", { exact: true })).toHaveCount(0);
});

test("copies post key and shows temporary success badge", async ({ page }) => {
  await page.addInitScript(() => {
    const nav: Navigator = window.navigator as Navigator;
    try {
      if ("clipboard" in nav && nav.clipboard && typeof nav.clipboard.writeText === "function") {
        nav.clipboard.writeText = (_text: string) => Promise.resolve();
      } else {
        Object.defineProperty(nav, "clipboard", {
          value: { writeText: (_text: string) => Promise.resolve() },
          configurable: true,
        });
      }
    } catch {
      Object.defineProperty(nav, "clipboard", {
        value: { writeText: (_text: string) => Promise.resolve() },
        configurable: true,
      });
    }
  });
  await page.route("**/api/post_key", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ post_key: "abc123", created_at: "2024-01-01T00:00:00Z" }),
    });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("dashboard");
  await page.getByTitle("Copy key").click();
  await expect(page.getByText("copied to clipboard!", { exact: true })).toBeVisible();
  await page.waitForTimeout(2200);
  await expect(page.getByText("copied to clipboard!", { exact: true })).toHaveCount(0);
});

test("shows error alert on network abort", async ({ page }) => {
  await page.route("**/api/post_key", async (route) => {
    await route.abort();
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("dashboard");
  await expect(page.getByText("Failed to load post key. Please try again later.")).toBeVisible();
});

test("shows error alert on server error", async ({ page }) => {
  await page.route("**/api/post_key", async (route) => {
    await route.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("dashboard");
  await expect(page.getByText("Failed to load post key. Please try again later.")).toBeVisible();
});

test("uses Accept-Language header with English language", async ({ page }, testInfo) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t_en", refresh_token: "r_en", user: { id: 1, username: "tester" } })));
  await page.route("**/api/post_key", async (route) => {
    const h = route.request().headers();
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^en-US/);
    }
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ post_key: "abc123", created_at: "2024-01-01T00:00:00Z" }) });
  });
  await page.goto("dashboard");
});

test("uses Accept-Language header with Chinese language", async ({ page }, testInfo) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "zh"));
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t_zh", refresh_token: "r_zh", user: { id: 2, username: "tester2" } })));
  await page.route("**/api/post_key", async (route) => {
    const h = route.request().headers();
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^zh-CN/);
    }
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ post_key: "xyz789", created_at: "2024-01-01T00:00:00Z" }) });
  });
  await page.goto("dashboard");
  await expect(page.getByRole("button", { name: "复制 Post Key", exact: true })).toBeVisible();
});

test("includes Authorization header when fetching post key", async ({ page }) => {
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "e2e_access_token", refresh_token: "e2e_refresh_token", user: { id: 1, username: "tester" } })));
  await page.route("**/api/post_key", async (route) => {
    const headers = route.request().headers();
    expect(headers["authorization"]).toBe("Bearer e2e_access_token");
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ post_key: "abc123", created_at: "2024-01-01T00:00:00Z" }) });
  });
  await page.goto("dashboard");
  await expect(page.getByText("Post Key", { exact: true })).toBeVisible();
});

test("navigates to settings and logs out from user menu", async ({ page }) => {
  await page.route("**/api/post_key", async (route) => {
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ post_key: "abc123", created_at: "2024-01-01T00:00:00Z" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("dashboard");
  const userMenuToggle = page.getByRole("button", { name: "tester" });
  await userMenuToggle.click();
  await page.getByText("Settings", { exact: true }).click();
  await page.waitForURL("**/settings");
  const userMenuToggle2 = page.getByRole("button", { name: "tester" });
  await userMenuToggle2.click();
  await page.getByText("Logout", { exact: true }).click();
  await page.waitForURL("**/login");
});
