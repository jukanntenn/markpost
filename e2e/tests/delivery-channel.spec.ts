import { test, expect } from "../lib/fixtures";
import { waitForBackend, apiLogin, deleteAllDeliveryChannels } from "../lib/helpers";

test.beforeEach(async ({ page, request }) => {
  await waitForBackend(request);
  const auth = await apiLogin(request);
  await deleteAllDeliveryChannels(request, auth.token);
  await page.context().clearCookies();
  await page.goto("/login");
  await page.evaluate(() => {
    localStorage.clear();
    localStorage.setItem("locale", "en");
  });
});

test("creates a delivery channel (D5)", async ({ page, loginPage, deliveryPage }) => {
  await loginPage.goto();
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await deliveryPage.goto();
  await deliveryPage.createChannel(
    "Workgroup",
    "https://open.feishu.cn/open-apis/bot/v2/hook/test",
    "mark",
  );
  await expect(deliveryPage.channelRow("Workgroup")).toBeVisible({
    timeout: 15000,
  });
});

test("shows keyword syntax error in the dialog (D5.2)", async ({
  page,
  loginPage,
  deliveryPage,
}) => {
  await loginPage.goto();
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await deliveryPage.goto();
  await deliveryPage.clickAddChannel();
  await deliveryPage.channelKeywordsInput.fill('"unterminated');
  await expect(page.getByText(/Syntax error/)).toBeVisible();
});

test("previews the keyword filter in natural language (D5.6)", async ({
  page,
  loginPage,
  deliveryPage,
}) => {
  await loginPage.goto();
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await deliveryPage.goto();
  await deliveryPage.clickAddChannel();
  await expect(
    page.getByText("Delivers every post (empty — no filtering)"),
  ).toBeVisible();
  await deliveryPage.channelKeywordsInput.fill(
    "prod & (error, warning) & !debug",
  );
  await expect(
    page.getByText(
      "Delivers when the title contains “prod” and (contains “error” or contains “warning”) and does not contain “debug”",
    ),
  ).toBeVisible();
});

test("edit mode exposes test + delete actions (D5.3)", async ({
  page,
  loginPage,
  deliveryPage,
}) => {
  await loginPage.goto();
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await deliveryPage.goto();
  await deliveryPage.createChannel("EditMe", "https://open.feishu.cn/open-apis/bot/v2/hook/ok");
  await deliveryPage.editChannel("EditMe");
  await expect(deliveryPage.dialog.getByRole("button", { name: "Send test" })).toBeVisible();
  await expect(deliveryPage.dialog.getByRole("button", { name: "Delete" })).toBeVisible();
});

test("deletes a channel with confirmation (D5.5)", async ({ page, loginPage, deliveryPage }) => {
  await loginPage.goto();
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await deliveryPage.goto();
  await deliveryPage.createChannel("Doomed", "https://open.feishu.cn/open-apis/bot/v2/hook/ok");
  await deliveryPage.editChannel("Doomed");
  await deliveryPage.clickDeleteInDialog();
  await deliveryPage.confirmDelete();
  await expect(deliveryPage.channelRow("Doomed")).toHaveCount(0, {
    timeout: 15000,
  });
});
