import { test, expect } from "../lib/fixtures";

test.beforeEach(async ({ authenticatedPage, settingsPage }) => {
  await settingsPage.goto();
  await expect(settingsPage.deliveryChannelsHeading).toBeVisible({ timeout: 15000 });
});

test("shows empty state when no channels exist", async ({ settingsPage }) => {
  await expect(settingsPage.emptyChannelsMessage).toBeVisible();
});

test("opens create form and closes it via cancel", async ({ settingsPage }) => {
  await settingsPage.addChannelButton.click();
  await expect(settingsPage.channelNameInput).toBeVisible();
  await expect(settingsPage.channelWebhookInput).toBeVisible();
  await expect(settingsPage.channelKeywordsInput).toBeVisible();

  await settingsPage.deliveryChannelsCard.getByRole("button", { name: "Cancel" }).click();
  await expect(settingsPage.channelNameInput).not.toBeVisible();
});

test("creates a delivery channel successfully", async ({ settingsPage }) => {
  await settingsPage.createChannel("Test Channel", "https://example.com/webhook");

  await expect(settingsPage.channelNameInput).not.toBeVisible();
  const row = settingsPage.channelRow("Test Channel");
  await expect(row).toBeVisible();
  await expect(row).toContainText("Test Channel");
  await expect(row).toContainText("feishu");
});

test("creates a channel with keywords", async ({ settingsPage }) => {
  await settingsPage.createChannel("Keyword Channel", "https://example.com/hook", "alert,error");

  const row = settingsPage.channelRow("Keyword Channel");
  await expect(row).toBeVisible();
});

test("edits a channel name", async ({ settingsPage }) => {
  await settingsPage.createChannel("Edit Me", "https://example.com/webhook");

  const row = settingsPage.channelRow("Edit Me");
  await expect(row).toBeVisible();
  await row.getByRole("button", { name: "Edit" }).click();

  await expect(settingsPage.channelNameInput).toBeVisible();
  await settingsPage.channelNameInput.clear();
  await settingsPage.channelNameInput.fill("Edited Name");
  await settingsPage.deliveryChannelsCard.getByRole("button", { name: "Save" }).click();

  await expect(settingsPage.channelRow("Edited Name")).toBeVisible();
});

test("toggles channel enabled/disabled", async ({ settingsPage }) => {
  await settingsPage.createChannel("Toggle Test", "https://example.com/webhook");

  const row = settingsPage.channelRow("Toggle Test");
  await expect(row).toBeVisible();

  const toggle = row.locator("[data-slot='switch']");
  await expect(toggle).toHaveAttribute("data-checked", "");
  await toggle.click();
  await expect(toggle).not.toHaveAttribute("data-checked");
});

test("deletes a channel with confirmation", async ({ settingsPage }) => {
  await settingsPage.createChannel("Delete Me", "https://example.com/webhook");

  const row = settingsPage.channelRow("Delete Me");
  await expect(row).toBeVisible();

  await row.getByRole("button", { name: "Delete" }).click();
  await expect(row.getByRole("button", { name: "Confirm" })).toBeVisible();
  await expect(row.getByRole("button", { name: "Cancel" })).toBeVisible();

  await row.getByRole("button", { name: "Cancel" }).click();
  await expect(row).toBeVisible();

  await row.getByRole("button", { name: "Delete" }).click();
  await row.getByRole("button", { name: "Confirm" }).click();

  await expect(row).not.toBeVisible();
});
