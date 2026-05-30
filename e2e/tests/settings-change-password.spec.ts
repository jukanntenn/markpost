import { test, expect } from "../lib/fixtures";

test("successfully changes password and resets form", async ({
  authenticatedPage,
  settingsPage,
}) => {
  await settingsPage.goto();
  await expect(settingsPage.appSettingsHeading).toBeVisible({ timeout: 15000 });

  await settingsPage.fillPasswordForm("markpost", "newpassword123", "newpassword123");
  await settingsPage.clickChangePassword();

  const successMsg = await settingsPage.getSuccessMessage();
  await expect(successMsg).toBeVisible();
  await expect(settingsPage.currentPasswordInput).toHaveValue("");
  await expect(settingsPage.newPasswordInput).toHaveValue("");
  await expect(settingsPage.confirmPasswordInput).toHaveValue("");
});
