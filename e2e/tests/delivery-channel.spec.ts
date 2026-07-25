import { test, expect, cleanupTestData } from "../lib/fixtures";

test.beforeEach(async ({ request, authToken, authenticatedPage, deliveryPage }) => {
  await cleanupTestData(request, authToken.token);
  await deliveryPage.goto();
  await expect(deliveryPage.heading).toBeVisible({ timeout: 15000 });
});

test.afterEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});

test("shows empty state when no channels exist", async ({ deliveryPage }) => {
  await expect(deliveryPage.emptyState).toBeVisible();
});

test("creates a delivery channel successfully", async ({ deliveryPage, page }) => {
  await deliveryPage.createChannel("Test Channel", "https://example.com/webhook");
  await expect(deliveryPage.dialog).not.toBeVisible({ timeout: 15000 });

  await page.reload();
  await expect(deliveryPage.heading).toBeVisible({ timeout: 15000 });

  const row = deliveryPage.channelRow("Test Channel");
  await expect(row).toBeVisible({ timeout: 15000 });
  await expect(row).toContainText("Test Channel");
  await expect(row).toContainText("feishu");
});

test("creates a channel with keywords", async ({ deliveryPage, page }) => {
  await deliveryPage.createChannel("Keyword Channel", "https://example.com/hook", "alert,error");
  await expect(deliveryPage.dialog).not.toBeVisible({ timeout: 15000 });

  await page.reload();
  await expect(deliveryPage.channelRow("Keyword Channel")).toBeVisible({ timeout: 15000 });
});

test("edits a channel name", async ({ deliveryPage, page }) => {
  await deliveryPage.createChannel("Edit Me", "https://example.com/webhook");
  await expect(deliveryPage.dialog).not.toBeVisible({ timeout: 15000 });

  await page.reload();
  await expect(deliveryPage.channelRow("Edit Me")).toBeVisible({ timeout: 15000 });

  await deliveryPage.editChannel("Edit Me");
  await expect(deliveryPage.channelNameInput).toBeVisible();
  await deliveryPage.channelNameInput.clear();
  await deliveryPage.channelNameInput.fill("Edited Name");
  await deliveryPage.submitSave();

  await expect(deliveryPage.dialog).not.toBeVisible({ timeout: 15000 });
  await page.reload();
  await expect(deliveryPage.channelRow("Edited Name")).toBeVisible({ timeout: 15000 });
});

test("toggles channel enabled/disabled", async ({ deliveryPage, page }) => {
  await deliveryPage.createChannel("Toggle Test", "https://example.com/webhook");
  await expect(deliveryPage.dialog).not.toBeVisible({ timeout: 15000 });

  await page.reload();
  const row = deliveryPage.channelRow("Toggle Test");
  await expect(row).toBeVisible({ timeout: 15000 });

  const toggle = row.locator("[data-slot='switch']");
  await expect(toggle).toHaveAttribute("data-checked", "");
  await toggle.click();

  await page.waitForTimeout(1500);
  await page.reload();
  await expect(deliveryPage.heading).toBeVisible({ timeout: 15000 });

  const refreshedToggle = deliveryPage.channelRow("Toggle Test").locator("[data-slot='switch']");
  await expect(refreshedToggle).toHaveAttribute("data-unchecked", "");
});

test("deletes a channel with confirmation", async ({ deliveryPage, page }) => {
  await deliveryPage.createChannel("Delete Me", "https://example.com/webhook");
  await expect(deliveryPage.dialog).not.toBeVisible({ timeout: 15000 });

  await page.reload();
  await expect(deliveryPage.channelRow("Delete Me")).toBeVisible({ timeout: 15000 });

  await deliveryPage.editChannel("Delete Me");
  await deliveryPage.clickDeleteInDialog();
  await expect(deliveryPage.dialog.getByRole("button", { name: "Confirm Delete" })).toBeVisible();
  await deliveryPage.confirmDelete();

  await expect(deliveryPage.dialog).not.toBeVisible({ timeout: 15000 });
  await page.reload();
  await expect(deliveryPage.channelRow("Delete Me")).not.toBeVisible();
});
