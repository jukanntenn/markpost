import { test, expect } from "../lib/fixtures";

test("creates a test post via modal and refreshes recent posts", async ({
  authenticatedPage,
  dashboardPage,
}) => {
  await dashboardPage.goto();
  await expect(dashboardPage.postKeyHeading).toBeVisible({ timeout: 15000 });

  const quickCreateButton = await dashboardPage.getQuickCreateButton();
  await quickCreateButton.click();
  await expect(authenticatedPage.getByRole("dialog")).toBeVisible();

  await authenticatedPage.getByPlaceholder("Enter a title, or leave empty").fill("E2E Test Post");
  await authenticatedPage.getByPlaceholder("Write some Markdown content...").fill("Hello world body");

  await authenticatedPage.getByRole("dialog").getByRole("button", { name: "Create", exact: true }).click();

  await expect(authenticatedPage.getByText("Post Created")).toBeVisible({ timeout: 10000 });

  await expect(authenticatedPage.getByRole("dialog")).toHaveCount(0);

  await expect(dashboardPage.latestPostsHeading).toBeVisible();
  await expect(authenticatedPage.getByText("E2E Test Post", { exact: true })).toBeVisible();
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
