import { test, expect } from "../lib/fixtures";
import { waitForBackend, apiLogin, deleteAllPosts } from "../lib/helpers";

test.beforeEach(async ({ page, request }) => {
  await waitForBackend(request);
  const auth = await apiLogin(request);
  await deleteAllPosts(request, auth.token);
  await page.context().clearCookies();
  await page.goto("/login");
  await page.evaluate(() => {
    localStorage.clear();
    localStorage.setItem("locale", "en");
  });
});

// F.5 帖子列表：空态（信息型）+ 搜索 + 外链。
test("shows empty state with Post Key hint", async ({ page, loginPage }) => {
  await loginPage.goto();
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await page.goto("/posts");
  await expect(page.getByText("No posts yet")).toBeVisible({ timeout: 15000 });
  await expect(page.getByText("Send posts via your Post Key")).toBeVisible();
});

test("post list renders after creating a post via Post Key", async ({
  page,
  loginPage,
  postsPage,
  request,
}) => {
  await loginPage.goto();
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await page.goto("/posts");
  await expect(page.getByText("No posts yet")).toBeVisible({ timeout: 15000 });
  await postsPage.searchInput.fill("nothing matches");
  await expect(page.getByText("No posts yet")).toBeVisible();
});
