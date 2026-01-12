import { test, expect } from "./fixtures";

test.beforeEach(async ({ page }) => {
  await page.context().clearCookies();
  await page.goto("login");
  await page.evaluate(() => localStorage.clear());
});

test("redirects to login when unauthenticated", async ({ page }) => {
  await page.goto("settings");
  await page.waitForURL("**/login");
});

test("renders settings page in English", async ({ settingsPage }) => {
  await settingsPage.page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await settingsPage.page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await expect(settingsPage.appSettingsHeading).toBeVisible();
  await expect(settingsPage.languageLabel).toBeVisible();
  await expect(settingsPage.changePasswordHeading).toBeVisible();
  await expect(settingsPage.currentPasswordInput).toBeVisible();
  await expect(settingsPage.newPasswordInput).toBeVisible();
  await expect(settingsPage.confirmPasswordInput).toBeVisible();
  await expect(settingsPage.changePasswordButton).toBeVisible();
});

test("switches language to Chinese via toggle", async ({ settingsPage }) => {
  await settingsPage.page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await settingsPage.page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.clickLanguageToggle();
  await settingsPage.page.getByText("中文", { exact: true }).click();

  await expect(settingsPage.page.getByText("应用设置", { exact: true })).toBeVisible();
  await expect(settingsPage.page.getByRole("heading", { name: "修改密码", exact: true })).toBeVisible();
  const lng = await settingsPage.page.evaluate(() => localStorage.getItem("i18nextLng"));
  expect(lng).toBe("zh");
});

test("shows loading state during password change", async ({ page, settingsPage }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await new Promise((r) => setTimeout(r, 600));
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("oldpass", "newpass", "newpass");
  await settingsPage.clickChangePassword();

  await expect(page.getByRole("button", { name: "Changing password..." })).toBeDisabled();
});

test("successfully changes password and resets form", async ({ page, settingsPage }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("oldpass", "newpass", "newpass");
  await settingsPage.clickChangePassword();

  const successMsg = await settingsPage.getSuccessMessage();
  await expect(successMsg).toBeVisible();
  await expect(settingsPage.currentPasswordInput).toHaveValue("");
  await expect(settingsPage.newPasswordInput).toHaveValue("");
  await expect(settingsPage.confirmPasswordInput).toHaveValue("");
});

test("shows server error message field", async ({ page, settingsPage }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 400, contentType: "application/json", body: JSON.stringify({ message: "server error message" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("oldpass", "newvalidpass", "newvalidpass");
  await settingsPage.clickChangePassword();

  const alert = await settingsPage.getAlert();
  await expect(alert).toContainText("server error message");
});

test("shows server error error field", async ({ page, settingsPage }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 400, contentType: "application/json", body: JSON.stringify({ error: "weak password" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("oldpass", "newvalidpass", "newvalidpass");
  await settingsPage.clickChangePassword();

  const alert = await settingsPage.getAlert();
  await expect(alert).toContainText("weak password");
});

test("shows error alert on network abort", async ({ page, settingsPage }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.abort();
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("oldpass", "newpass", "newpass");
  await settingsPage.clickChangePassword();

  await expect(page.getByText("Password change failed, please try again", { exact: false })).toBeVisible();
});

test("clears error and success when input changes", async ({ page, settingsPage }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 400, contentType: "application/json", body: JSON.stringify({ error: "weak password" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("oldpass", "newvalidpass", "newvalidpass");
  await settingsPage.clickChangePassword();

  const alert = await settingsPage.getAlert();
  await expect(alert).toContainText("weak password");

  await settingsPage.currentPasswordInput.fill("changed");
  await expect(page.locator(".alert.alert-danger")).toHaveCount(0);
});

test("client validation requires current password", async ({ page, settingsPage }) => {
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm(" ", "newpass", "newpass");
  await settingsPage.clickChangePassword();

  const alert = await settingsPage.getAlert();
  await expect(alert).toContainText("Password change failed, please try again");
});

test("client validation new password min length and valid strength", async ({ page, settingsPage }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 400, contentType: "application/json", body: JSON.stringify({ error: "weak password" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("oldpass", "123", "123");
  await settingsPage.clickChangePassword();

  await expect(page.locator('input[name="new_password"] ~ .invalid-feedback')).toContainText("Password must be at least 6 characters");

  await settingsPage.newPasswordInput.fill("123456");
  await settingsPage.clickChangePassword();

  await expect(page.locator('input[name="new_password"] ~ .valid-feedback')).toContainText("Password strength is valid");
});

test("client validation confirm not match then match", async ({ page, settingsPage }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 400, contentType: "application/json", body: JSON.stringify({ error: "weak password" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("oldpass", "abcdef", "ghijkl");
  await settingsPage.confirmPasswordInput.press("Enter");

  await expect(page.locator('input[name="confirm_password"] ~ .invalid-feedback')).toContainText("Passwords do not match");

  await settingsPage.confirmPasswordInput.fill("abcdef");
  await settingsPage.clickChangePassword();

  await expect(page.locator('input[name="confirm_password"] ~ .valid-feedback')).toContainText("Passwords match");
});

test("client validation new password same as current", async ({ page, settingsPage }) => {
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("abcdef", "abcdef", "abcdef");
  await settingsPage.clickChangePassword();

  await expect(page.getByText("New password cannot be the same as current password", { exact: true })).toBeVisible();
});

test("submits by pressing Enter key on confirm field", async ({ page, settingsPage }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("oldpass", "newpass", "newpass");
  await settingsPage.clickChangePasswordByEnter();

  const successMsg = await settingsPage.getSuccessMessage();
  await expect(successMsg).toBeVisible();
});

test("uses Accept-Language header with English language on submit", async ({ page, settingsPage }, testInfo) => {
  await page.route("**/api/auth/change-password", async (route) => {
    const h = route.request().headers();
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^en-US/);
    }
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t_en", refresh_token: "r_en", user: { id: 1, username: "tester" } })));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("oldpass", "newpass", "newpass");
  await settingsPage.clickChangePassword();
});

test("uses Accept-Language header with Chinese language on submit", async ({ page, settingsPage }, testInfo) => {
  await page.route("**/api/auth/change-password", async (route) => {
    const h = route.request().headers();
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^zh-CN/);
    }
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("i18nextLng", "zh"));
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t_zh", refresh_token: "r_zh", user: { id: 2, username: "tester2" } })));
  await settingsPage.goto();

  await page.locator('input[name="current_password"]').fill("oldpass");
  await page.locator('input[name="new_password"]').fill("newpass");
  await page.locator('input[name="confirm_password"]').fill("newpass");
  await page.getByRole("button", { name: "修改密码" }).click();
});

test("includes Authorization header when changing password", async ({ page, settingsPage }) => {
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "e2e_access_token", refresh_token: "e2e_refresh_token", user: { id: 1, username: "tester" } })));
  await page.route("**/api/auth/change-password", async (route) => {
    const headers = route.request().headers();
    expect(headers["authorization"]).toBe("Bearer e2e_access_token");
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await settingsPage.goto();

  await settingsPage.fillPasswordForm("oldpass", "newpass", "newpass");
  await settingsPage.clickChangePassword();
});
