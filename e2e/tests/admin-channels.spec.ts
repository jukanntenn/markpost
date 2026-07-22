import { test, expect, cleanupTestData } from "../lib/fixtures";
import { createDeliveryChannel } from "../lib/helpers";

test.beforeEach(async ({ request, authToken, authenticatedPage, adminChannelsPage }) => {
  await cleanupTestData(request, authToken.token);
  await adminChannelsPage.goto();
  await expect(adminChannelsPage.heading).toBeVisible({ timeout: 15000 });
});

test.afterEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});

test("displays empty state when no channels", async ({ adminChannelsPage }) => {
  await expect(adminChannelsPage.emptyMessage).toBeVisible();
});

test("create a channel via API and verify it appears in admin list", async ({
  request,
  authToken,
  adminChannelsPage,
  page,
}) => {
  await createDeliveryChannel(request, authToken.token, {
    name: "Admin Test Channel",
    kind: "feishu",
    configuration: { webhook_url: "https://example.com/webhook" },
  });

  await page.reload();
  await expect(adminChannelsPage.heading).toBeVisible({ timeout: 15000 });

  const channelRow = adminChannelsPage.getChannelRow("Admin Test Channel");
  await expect(channelRow).toBeVisible({ timeout: 15000 });
});
