import { test, expect, cleanupTestData } from "../lib/fixtures";
import { createDeliveryChannel, clearWebhooks, getWebhooks } from "../lib/helpers";

const TEST_CARD_TITLE = "Markpost test message";

test.beforeEach(async ({ request, authToken, authenticatedPage, deliveryPage }) => {
  await cleanupTestData(request, authToken.token);
  await deliveryPage.goto();
  await expect(deliveryPage.heading).toBeVisible({ timeout: 15000 });
});

test.afterEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});

test("sends a test message to the channel webhook", async ({
  request,
  authToken,
  deliveryPage,
}) => {
  await clearWebhooks(request);
  await createDeliveryChannel(request, authToken.token, {
    name: "Test Target",
    kind: "feishu",
    configuration: { webhook_url: "http://webhook-mock:3002/webhook" },
  });

  await deliveryPage.goto();
  await expect(deliveryPage.channelRow("Test Target")).toBeVisible({
    timeout: 15000,
  });

  await deliveryPage.editChannel("Test Target");
  await deliveryPage.clickTestInDialog();

  await expect(deliveryPage.dialog).toBeVisible({ timeout: 15000 });

  await new Promise((r) => setTimeout(r, 3000));
  const webhooks = await getWebhooks(request);
  const testWebhook = webhooks.find((w: any) =>
    JSON.stringify(w.body ?? {}).includes(TEST_CARD_TITLE),
  );
  expect(testWebhook, "expected a test card to reach the webhook mock").toBeDefined();
});

test("test button is hidden for a brand-new channel (create mode)", async ({ deliveryPage }) => {
  await deliveryPage.clickAddChannel();
  await expect(deliveryPage.dialog).toBeVisible();
  await expect(deliveryPage.dialog.getByRole("button", { name: "Test", exact: true })).toHaveCount(
    0,
  );
});
