import { test, expect, cleanupTestData } from "../lib/fixtures";

test.beforeEach(async ({ request, authToken, authenticatedPage, adminDeliveryHistoryPage }) => {
  await cleanupTestData(request, authToken.token);
  await adminDeliveryHistoryPage.goto();
  await expect(adminDeliveryHistoryPage.heading).toBeVisible({ timeout: 15000 });
});

test("displays delivery history table", async ({ adminDeliveryHistoryPage }) => {
  await expect(adminDeliveryHistoryPage.table).toBeVisible();
});
