import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.context().clearCookies();
  await page.goto("login");
  await page.evaluate(() => localStorage.clear());
});

test("redirects to login when unauthenticated", async ({ page }) => {
  await page.goto("posts");
  await page.waitForURL("**/login");
});

test("renders posts page with empty state (English)", async ({ page }) => {
  await page.route("**/api/posts**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ posts: [], pagination: { page: 1, limit: 20, total: 0, total_pages: 0 } }),
    });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("posts");
  await expect(page.getByRole("heading", { name: "Posts", exact: true })).toBeVisible();
  await expect(page.getByText("No posts yet", { exact: true })).toBeVisible();
});

test("lists posts and shows pagination", async ({ page }) => {
  await page.route("**/api/posts**", async (route) => {
    const url = new URL(route.request().url());
    const pageParam = url.searchParams.get("page");
    if (pageParam === "1") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          posts: [
            { id: "p1", title: "Post One", created_at: "2024-01-01T12:00:00Z" },
            { id: "p2", title: "Post Two", created_at: "2024-01-02T13:00:00Z" },
          ],
          pagination: { page: 1, limit: 20, total: 2, total_pages: 1 },
        }),
      });
    } else {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ posts: [], pagination: { page: 2, limit: 20, total: 2, total_pages: 1 } }) });
    }
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("posts");
  await expect(page.getByRole("columnheader", { name: "Title" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Post One" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Post Two" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Previous" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Next" })).toBeDisabled();
});

test("uses Authorization header when fetching posts", async ({ page }) => {
  await page.route("**/api/posts**", async (route) => {
    const headers = route.request().headers();
    expect(headers["authorization"]).toBe("Bearer e2e_access_token");
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ posts: [], pagination: { page: 1, limit: 20, total: 0, total_pages: 0 } }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "e2e_access_token", refresh_token: "e2e_refresh_token", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("posts");
});

test("uses Accept-Language header on fetch (English)", async ({ page }, testInfo) => {
  await page.route("**/api/posts**", async (route) => {
    const h = route.request().headers();
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^en-US/);
    }
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ posts: [], pagination: { page: 1, limit: 20, total: 0, total_pages: 0 } }) });
  });
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t_en", refresh_token: "r_en", user: { id: 1, username: "tester" } })));
  await page.goto("posts");
});

