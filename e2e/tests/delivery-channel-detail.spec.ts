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

test("channel detail page shows configuration and channel-scoped history", async ({
  request,
  authToken,
  authenticatedPage,
  page,
  deliveryPage,
}) => {
  await clearWebhooks(request);
  await createDeliveryChannel(request, authToken.token, {
    name: "Detail Channel",
    kind: "feishu",
    configuration: { webhook_url: "http://webhook-mock:3002/webhook" },
  });

  const postKey = await getPostKey(request, authToken.token);
  const postTitle = `Detail Post ${Date.now()}`;
  await createPost(request, authToken.token, postKey, postTitle, "body");

  await new Promise((r) => setTimeout(r, 5000));

  await deliveryPage.goto();
  await expect(deliveryPage.channelRow("Detail Channel")).toBeVisible({ timeout: 15000 });

  await deliveryPage.gotoChannelDetail("Detail Channel");

  await expect(page.getByRole("heading", { name: "Detail Channel", exact: true })).toBeVisible({ timeout: 15000 });
  await expect(page.getByText("Configuration", { exact: true })).toBeVisible();
  await expect(page.getByText("Delivery History", { exact: true })).toBeVisible();

  const historyTable = page.getByRole("table").first();
  await expect(historyTable.getByText(postTitle, { exact: true })).toBeVisible({ timeout: 15000 });
});

test("latest delivery column reflects delivery outcome on the list page", async ({
  request,
  authToken,
  page,
  deliveryPage,
}) => {
  await clearWebhooks(request);
  await createDeliveryChannel(request, authToken.token, {
    name: "Latest Channel",
    kind: "feishu",
    configuration: { webhook_url: "http://webhook-mock:3002/webhook" },
  });

  const postKey = await getPostKey(request, authToken.token);
  await createPost(request, authToken.token, postKey, `Latest Post ${Date.now()}`, "body");

  await new Promise((r) => setTimeout(r, 5000));

  await deliveryPage.goto();
  await expect(deliveryPage.channelRow("Latest Channel")).toBeVisible({ timeout: 15000 });

  const latestCell = deliveryPage.getLatestCell("Latest Channel");
  await expect(latestCell.getByText("Delivered", { exact: true })).toBeVisible({ timeout: 15000 });
});

test("channel with no history shows Never in latest column", async ({
  request,
  authToken,
  deliveryPage,
}) => {
  await createDeliveryChannel(request, authToken.token, {
    name: "Silent Channel",
    kind: "feishu",
    configuration: { webhook_url: "http://webhook-mock:3002/webhook" },
  });

  await deliveryPage.goto();
  await expect(deliveryPage.channelRow("Silent Channel")).toBeVisible({ timeout: 15000 });

  const latestCell = deliveryPage.getLatestCell("Silent Channel");
  await expect(latestCell.getByText("Never", { exact: true })).toBeVisible();
});
