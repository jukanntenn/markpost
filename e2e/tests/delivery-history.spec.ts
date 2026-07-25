import { test, expect, cleanupTestData } from "../lib/fixtures";
import { createDeliveryChannel, createPost, getPostKey, clearWebhooks } from "../lib/helpers";

test.beforeEach(async ({ request, authToken, authenticatedPage, deliveryPage }) => {
  await cleanupTestData(request, authToken.token);
  await deliveryPage.goto();
  await expect(deliveryPage.heading).toBeVisible({ timeout: 15000 });
});

test.afterEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});

test("settings page no longer contains delivery sections", async ({ authenticatedPage, page, settingsPage }) => {
  await settingsPage.goto();
  await expect(settingsPage.appSettingsHeading).toBeVisible({ timeout: 15000 });

  await expect(page.getByText("Delivery Channels", { exact: true })).toHaveCount(0);
  await expect(page.getByText("Delivery History", { exact: true })).toHaveCount(0);
});

test("delivery history page shows delivered records after a post", async ({
  request,
  authToken,
  authenticatedPage,
  page,
  deliveryPage,
}) => {
  await clearWebhooks(request);
  await createDeliveryChannel(request, authToken.token, {
    name: "History Channel",
    kind: "feishu",
    configuration: { webhook_url: "http://webhook-mock:3002/webhook" },
  });

  const postKey = await getPostKey(request, authToken.token);
  const postTitle = `History Post ${Date.now()}`;
  await createPost(request, authToken.token, postKey, postTitle, "body for delivery");

  await new Promise((r) => setTimeout(r, 5000));

  await deliveryPage.gotoHistory();
  await expect(page.getByRole("heading", { name: "Delivery History", exact: true })).toBeVisible({ timeout: 15000 });

  const historyTable = page.getByRole("table").first();
  await expect(historyTable.getByText(postTitle, { exact: true })).toBeVisible({ timeout: 15000 });
  await expect(historyTable.getByText("Delivered", { exact: true }).first()).toBeVisible();
});
