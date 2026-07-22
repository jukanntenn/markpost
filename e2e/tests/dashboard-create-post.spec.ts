import { test, expect, cleanupTestData } from "../lib/fixtures";

test.beforeEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});

test.afterEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});

test("creates a test post via modal and refreshes recent posts", async ({
  authenticatedPage,
  dashboardPage,
}) => {
  await dashboardPage.goto();
  await expect(dashboardPage.postKeyHeading).toBeVisible({ timeout: 15000 });

  const quickCreateButton = await dashboardPage.getQuickCreateButton();
  await quickCreateButton.click();
  await expect(authenticatedPage.getByRole("dialog")).toBeVisible();

  const postTitle = `E2E Test Post ${Date.now()}`;
  await authenticatedPage.getByPlaceholder("Enter a title, or leave empty").fill(postTitle);
  await authenticatedPage.getByPlaceholder("Write some Markdown content...").fill("Hello world body");

  await authenticatedPage.getByRole("dialog").getByRole("button", { name: "Create", exact: true }).click();

  await expect(authenticatedPage.getByText("Post Created")).toBeVisible({ timeout: 10000 });

  await expect(authenticatedPage.getByRole("dialog")).toHaveCount(0);

  await expect(dashboardPage.latestPostsHeading).toBeVisible();
  await expect(authenticatedPage.getByText(postTitle, { exact: true })).toBeVisible();
});

test("disables Create when body is empty", async ({
  authenticatedPage,
  dashboardPage,
}) => {
  await dashboardPage.goto();
  await expect(dashboardPage.postKeyHeading).toBeVisible({ timeout: 15000 });

  const quickCreateButton = await dashboardPage.getQuickCreateButton();
  await quickCreateButton.click();
  await expect(
    authenticatedPage.getByRole("dialog").getByRole("button", { name: "Create", exact: true })
  ).toBeDisabled();
});
