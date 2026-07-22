import { test, expect, cleanupTestData } from "../lib/fixtures";
import {
  createDeliveryChannel,
  createPost,
  getPostKey,
  clearWebhooks,
  getWebhooks,
} from "../lib/helpers";

test.afterEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});

test("webhook is called with correct payload when post is created", async ({
  request,
  authToken,
}) => {
  await clearWebhooks(request);

  await createDeliveryChannel(request, authToken.token, {
    name: "Webhook Test Channel",
    kind: "feishu",
    configuration: { webhook_url: "http://webhook-mock:3002/webhook" },
  });

  const postKey = await getPostKey(request, authToken.token);
  const postTitle = `Webhook Test ${Date.now()}`;
  const postBody = "This post should trigger webhook delivery";

  await createPost(request, authToken.token, postKey, postTitle, postBody);

  // Wait for webhook delivery
  await new Promise((r) => setTimeout(r, 5000));

  const webhooks = await getWebhooks(request);
  expect(webhooks.length).toBeGreaterThan(0);

  const webhook = webhooks.find((w: any) => 
    w.body?.msg_type === "interactive" || 
    w.body?.content?.includes(postTitle) ||
    JSON.stringify(w.body).includes(postTitle)
  );

  expect(webhook).toBeDefined();
});
