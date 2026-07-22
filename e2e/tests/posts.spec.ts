import { test, expect } from "../lib/fixtures";

test("redirects to login when unauthenticated", async ({ page }) => {
  await page.goto("/posts");
  await page.waitForURL("**/login");
});

test("renders posts page with heading", async ({ authenticatedPage, postsPage }) => {
  await postsPage.goto();
  await expect(postsPage.allPostsHeading).toBeVisible({ timeout: 15000 });
});
