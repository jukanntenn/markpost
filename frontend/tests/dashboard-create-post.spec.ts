import { test, expect } from "./fixtures";

test.beforeEach(async ({ page }) => {
  await page.context().clearCookies();
  await page.goto("login");
  await page.evaluate(() => localStorage.clear());
});

test("creates a test post via modal and refreshes recent posts", async ({ page, dashboardPage }) => {
  await page.route("**/api/post_key", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ post_key: "abc123", created_at: "2024-01-01T00:00:00Z" }),
    });
  });

  let postsCall = 0;
  await page.route("**/api/posts**", async (route) => {
    postsCall++;
    if (postsCall === 1) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          posts: [],
          pagination: { page: 1, limit: 10, total: 0, total_pages: 0 },
        }),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          posts: [
            { id: "123", title: "Hello world", created_at: "2024-01-01T00:00:00Z" },
          ],
          pagination: { page: 1, limit: 10, total: 1, total_pages: 1 },
        }),
      });
    }
  });

  await page.evaluate(() =>
    localStorage.setItem(
      "markpost_dev_login",
      JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })
    )
  );
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await dashboardPage.goto();

  await expect(dashboardPage.postKeyHeading).toBeVisible();

  const quickCreateButton = await dashboardPage.getQuickCreateButton();
  await quickCreateButton.click();
  await expect(page.getByRole("dialog")).toBeVisible();

  await page.getByPlaceholder("Write some Markdown content...").fill("Hello world body");

  await page.route("**/abc123", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ id: "123" }),
    });
  });

  await page.getByRole("dialog").getByRole("button", { name: "Create", exact: true }).click();

  const toast = page.getByRole("alert").filter({ hasText: "Post Created" }).first();
  await expect(toast).toBeVisible();
  await expect(toast).toContainText("Your test post has been created successfully.");

  await expect(page.getByRole("dialog")).toHaveCount(0);

  await expect(dashboardPage.latestPostsHeading).toBeVisible();
  await expect(page.getByText("Hello world", { exact: true })).toBeVisible();
});

test("disables Create when body is empty", async ({ page, dashboardPage }) => {
  await page.route("**/api/post_key", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ post_key: "abc123", created_at: "2024-01-01T00:00:00Z" }),
    });
  });
  await page.route("**/api/posts**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ posts: [], pagination: { page: 1, limit: 10, total: 0, total_pages: 0 } }),
    });
  });

  await page.evaluate(() =>
    localStorage.setItem(
      "markpost_dev_login",
      JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })
    )
  );
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await dashboardPage.goto();

  const quickCreateButton = await dashboardPage.getQuickCreateButton();
  await quickCreateButton.click();
  await expect(
    page.getByRole("dialog").getByRole("button", { name: "Create", exact: true })
  ).toBeDisabled();
});
