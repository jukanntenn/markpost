import { test, expect, cleanupTestData } from "../lib/fixtures";

test.beforeEach(async ({ request, authToken, authenticatedPage, settingsPage }) => {
  await cleanupTestData(request, authToken.token);
  await settingsPage.goto();
  await expect(settingsPage.deliveryChannelsHeading).toBeVisible({ timeout: 15000 });
});

test.afterEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});

test("shows empty state when no channels exist", async ({ settingsPage }) => {
  await expect(settingsPage.emptyChannelsMessage).toBeVisible();
});

test("opens create form and closes it via cancel", async ({ settingsPage }) => {
  await settingsPage.addChannelButton.click();
  await expect(settingsPage.channelNameInput).toBeVisible();
  await expect(settingsPage.channelWebhookInput).toBeVisible();

  await settingsPage.deliveryChannelsCard.getByRole("button", { name: "Cancel" }).click();
  await expect(settingsPage.channelNameInput).not.toBeVisible();
});

test("creates a delivery channel successfully", async ({ settingsPage, page }) => {
  await settingsPage.createChannel("Test Channel", "https://example.com/webhook");
  await expect(settingsPage.channelNameInput).not.toBeVisible();

  await page.reload();
  await expect(settingsPage.deliveryChannelsHeading).toBeVisible({ timeout: 15000 });

  const row = settingsPage.channelRow("Test Channel");
  await expect(row).toBeVisible({ timeout: 15000 });
  await expect(row).toContainText("Test Channel");
  await expect(row).toContainText("feishu");
});

test("creates a channel with keywords", async ({ settingsPage, page }) => {
  await settingsPage.createChannel("Keyword Channel", "https://example.com/hook", "alert,error");
  await page.reload();
  await expect(settingsPage.deliveryChannelsHeading).toBeVisible({ timeout: 15000 });

  const row = settingsPage.channelRow("Keyword Channel");
  await expect(row).toBeVisible({ timeout: 15000 });
});

test("edits a channel name", async ({ settingsPage, page }) => {
  await settingsPage.createChannel("Edit Me", "https://example.com/webhook");
  await page.reload();
  await expect(settingsPage.deliveryChannelsHeading).toBeVisible({ timeout: 15000 });

  const row = settingsPage.channelRow("Edit Me");
  await expect(row).toBeVisible({ timeout: 15000 });
  await row.getByRole("button", { name: "Edit" }).click();

  await expect(settingsPage.channelNameInput).toBeVisible();
  await settingsPage.channelNameInput.clear();
  await settingsPage.channelNameInput.fill("Edited Name");
  await settingsPage.deliveryChannelsCard.getByRole("button", { name: "Save" }).click();

  await page.waitForTimeout(1000);
  await page.reload();
  await expect(settingsPage.deliveryChannelsHeading).toBeVisible({ timeout: 15000 });
  await expect(settingsPage.channelRow("Edited Name")).toBeVisible({ timeout: 15000 });
});

test("toggles channel enabled/disabled", async ({ settingsPage, page }) => {
  await settingsPage.createChannel("Toggle Test", "https://example.com/webhook");
  await page.reload();
  await expect(settingsPage.deliveryChannelsHeading).toBeVisible({ timeout: 15000 });

  const row = settingsPage.channelRow("Toggle Test");
  await expect(row).toBeVisible({ timeout: 15000 });

  const toggle = row.locator("[data-slot='switch']");
  // Channel should start enabled (data-checked present)
  await expect(toggle).toHaveAttribute("data-checked", "");
  await toggle.click();

  await page.waitForTimeout(1500);
  await page.reload();
  await expect(settingsPage.deliveryChannelsHeading).toBeVisible({ timeout: 15000 });

  const refreshedRow = settingsPage.channelRow("Toggle Test");
  const refreshedToggle = refreshedRow.locator("[data-slot='switch']");
  // After toggle, should be disabled (data-unchecked present)
  await expect(refreshedToggle).toHaveAttribute("data-unchecked", "");
});

test("deletes a channel with confirmation", async ({ settingsPage, page }) => {
  await settingsPage.createChannel("Delete Me", "https://example.com/webhook");
  await page.reload();
  await expect(settingsPage.deliveryChannelsHeading).toBeVisible({ timeout: 15000 });

  const row = settingsPage.channelRow("Delete Me");
  await expect(row).toBeVisible({ timeout: 15000 });

  await row.getByRole("button", { name: "Delete" }).click();
  await expect(row.getByRole("button", { name: "Confirm" })).toBeVisible();
  await expect(row.getByRole("button", { name: "Cancel" })).toBeVisible();

  await row.getByRole("button", { name: "Cancel" }).click();
  await expect(row).toBeVisible();

  await row.getByRole("button", { name: "Delete" }).click();
  await row.getByRole("button", { name: "Confirm" }).click();

  await page.waitForTimeout(1000);
  await page.reload();
  await expect(settingsPage.deliveryChannelsHeading).toBeVisible({ timeout: 15000 });
  await expect(settingsPage.channelRow("Delete Me")).not.toBeVisible();
});
