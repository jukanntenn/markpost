import { test, expect } from "../lib/fixtures";

test("displays delivery history section on settings page", async ({
  authenticatedPage,
  settingsPage,
}) => {
  await settingsPage.goto();
  await expect(settingsPage.appSettingsHeading).toBeVisible({ timeout: 15000 });

  await expect(settingsPage.deliveryChannelsHeading).toBeVisible();
});
