import { test, expect } from "../lib/fixtures";

test("redirects to login when unauthenticated", async ({ page }) => {
  await page.goto("/settings");
  await page.waitForURL("**/login");
});

test("renders settings page in English", async ({
  authenticatedPage,
  settingsPage,
}) => {
  await settingsPage.goto();

  await expect(settingsPage.appSettingsHeading).toBeVisible({ timeout: 15000 });
  await expect(settingsPage.languageLabel).toBeVisible();
  await expect(settingsPage.changePasswordHeading).toBeVisible();
  await expect(settingsPage.currentPasswordInput).toBeVisible();
  await expect(settingsPage.newPasswordInput).toBeVisible();
  await expect(settingsPage.confirmPasswordInput).toBeVisible();
  await expect(settingsPage.changePasswordButton).toBeVisible();
});

test("switches language to Chinese via select", async ({
  authenticatedPage,
  settingsPage,
}) => {
  await settingsPage.goto();
  await expect(settingsPage.appSettingsHeading).toBeVisible({ timeout: 15000 });

  await settingsPage.selectLocale("zh");

  await expect(authenticatedPage.getByText("应用设置", { exact: true })).toBeVisible();
  await expect(
    authenticatedPage.getByText("修改密码", { exact: true })
  ).toBeVisible();
  const lng = await authenticatedPage.evaluate(() => localStorage.getItem("locale"));
  expect(lng).toBe("zh");
});

test("shows error on wrong current password", async ({
  authenticatedPage,
  settingsPage,
}) => {
  await settingsPage.goto();
  await expect(settingsPage.appSettingsHeading).toBeVisible({ timeout: 15000 });

  await settingsPage.fillPasswordForm("wrongpassword", "newpass123", "newpass123");
  await settingsPage.clickChangePassword();

  const alert = settingsPage.getAlert();
  await expect(alert).toBeVisible();
});

test("client validation: new password min length", async ({
  authenticatedPage,
  settingsPage,
}) => {
  await settingsPage.goto();
  await expect(settingsPage.appSettingsHeading).toBeVisible({ timeout: 15000 });

  await settingsPage.fillPasswordForm("markpost", "123", "123");
  await settingsPage.clickChangePassword();

  await expect(settingsPage.getAlert()).toContainText("Password must be at least 6 characters");
});

test("client validation: passwords must match", async ({
  authenticatedPage,
  settingsPage,
}) => {
  await settingsPage.goto();
  await expect(settingsPage.appSettingsHeading).toBeVisible({ timeout: 15000 });

  await settingsPage.fillPasswordForm("markpost", "abcdef", "ghijkl");
  await settingsPage.clickChangePassword();

  await expect(settingsPage.getAlert()).toContainText("Passwords do not match");
});
