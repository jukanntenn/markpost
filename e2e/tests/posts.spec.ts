import { test, expect } from "../lib/fixtures";

test("redirects to login when unauthenticated", async ({ page }) => {
  await page.goto("/posts");
  await page.waitForURL("**/login");
});

test("renders posts page with empty state", async ({
  authenticatedPage,
  postsPage,
}) => {
  await postsPage.goto();

  await expect(postsPage.allPostsHeading).toBeVisible({ timeout: 15000 });
  const noPostsMsg = postsPage.getNoPostsMessage();
  await expect(noPostsMsg).toBeVisible({ timeout: 15000 });
});
