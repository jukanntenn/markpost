import { test, expect, cleanupTestData } from "../lib/fixtures";
import { createPost, getPostKey } from "../lib/helpers";

test.afterEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});

test.beforeEach(async ({ request, authToken, authenticatedPage, adminPostsPage }) => {
  await cleanupTestData(request, authToken.token);
  await adminPostsPage.goto();
  await expect(adminPostsPage.heading).toBeVisible({ timeout: 15000 });
});

test("displays posts table", async ({ adminPostsPage }) => {
  await expect(adminPostsPage.table).toBeVisible();
});

test("create a post via API and verify it appears in admin list", async ({
  request,
  authToken,
  adminPostsPage,
  page,
}) => {
  const postKey = await getPostKey(request, authToken.token);
  const post = await createPost(request, authToken.token, postKey, "Admin Test Post", "Test body content");

  await page.reload();
  await expect(adminPostsPage.heading).toBeVisible({ timeout: 15000 });

  const postRow = adminPostsPage.getPostRow("Admin Test Post");
  await expect(postRow).toBeVisible({ timeout: 15000 });
});

test("search for posts", async ({
  request,
  authToken,
  adminPostsPage,
  page,
}) => {
  const postKey = await getPostKey(request, authToken.token);
  await createPost(request, authToken.token, postKey, "Searchable Post", "Body");
  await createPost(request, authToken.token, postKey, "Another Post", "Body");

  await page.reload();
  await expect(adminPostsPage.heading).toBeVisible({ timeout: 15000 });

  await adminPostsPage.search("Searchable");
  await expect(adminPostsPage.getPostRow("Searchable Post")).toBeVisible();
  await expect(adminPostsPage.getPostRow("Another Post")).not.toBeVisible();
});
