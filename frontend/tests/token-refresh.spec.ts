import { test, expect } from "./fixtures";

test.beforeEach(async ({ page }) => {
  await page.context().clearCookies();
  await page.goto("login");
  await page.evaluate(() => localStorage.clear());
});

test("silently refreshes token on 401 and loads dashboard", async ({
  page,
  dashboardPage,
}) => {
  await page.evaluate(() =>
    localStorage.setItem(
      "markpost_dev_login",
      JSON.stringify({
        access_token: "expired_access",
        refresh_token: "refresh_1",
        user: { id: 1, username: "tester" },
      })
    )
  );
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));

  let refreshCount = 0;
  await page.route("**/api/auth/refresh", async (route) => {
    refreshCount += 1;
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        access_token: "new_access",
        refresh_token: "new_refresh",
        user: { id: 1, username: "tester" },
      }),
    });
  });

  await page.route("**/api/post_key", async (route) => {
    const h = route.request().headers();
    if (h["authorization"] === "Bearer expired_access") {
      await route.fulfill({
        status: 401,
        contentType: "application/json",
        body: JSON.stringify({ message: "Unauthorized" }),
      });
      return;
    }
    expect(h["authorization"]).toBe("Bearer new_access");
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        post_key: "abc123",
        created_at: "2024-01-01T00:00:00Z",
      }),
    });
  });

  await page.route("**/api/posts**", async (route) => {
    const h = route.request().headers();
    if (h["authorization"] === "Bearer expired_access") {
      await route.fulfill({
        status: 401,
        contentType: "application/json",
        body: JSON.stringify({ message: "Unauthorized" }),
      });
      return;
    }
    expect(h["authorization"]).toBe("Bearer new_access");
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        posts: [],
        pagination: { page: 1, limit: 10, total: 0, total_pages: 0 },
      }),
    });
  });

  await dashboardPage.goto();

  await expect(dashboardPage.showKeyButton).toBeVisible();
  await dashboardPage.clickShowKey();
  await expect(await dashboardPage.getPostKeyText()).toHaveText("abc123");

  await expect
    .poll(() => page.evaluate(() => localStorage.getItem("markpost_dev_login")))
    .toContain("new_access");
  expect(refreshCount).toBe(1);
});

test("redirects to login when refresh fails", async ({ page, dashboardPage }) => {
  await page.evaluate(() =>
    localStorage.setItem(
      "markpost_dev_login",
      JSON.stringify({
        access_token: "expired_access",
        refresh_token: "refresh_1",
        user: { id: 1, username: "tester" },
      })
    )
  );
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));

  await page.route("**/api/auth/refresh", async (route) => {
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ message: "refresh expired" }),
    });
  });

  await page.route("**/api/post_key", async (route) => {
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ message: "Unauthorized" }),
    });
  });

  await dashboardPage.goto();

  await page.waitForURL("**/login");
  const saved = await page.evaluate(() => localStorage.getItem("markpost_dev_login"));
  expect(saved).toBeNull();
});
