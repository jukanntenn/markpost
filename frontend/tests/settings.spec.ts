import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.context().clearCookies();
  await page.goto("login");
  await page.evaluate(() => localStorage.clear());
});

test("redirects to login when unauthenticated", async ({ page }) => {
  await page.goto("settings");
  await page.waitForURL("**/login");
});

test("renders settings page in English", async ({ page }) => {
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await expect(page.getByText("Application Settings", { exact: true })).toBeVisible();
  await expect(page.getByText("Language", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Change Password", exact: true })).toBeVisible();
  await expect(page.getByPlaceholder("Enter your current password", { exact: true })).toBeVisible();
  await expect(page.getByPlaceholder("At least 6 characters", { exact: true })).toBeVisible();
  await expect(page.getByPlaceholder("Enter new password again", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Change Password" })).toBeVisible();
});

test("switches language to Chinese via toggle", async ({ page }) => {
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByRole("button", { name: "Change Language" }).click();
  await page.getByText("中文", { exact: true }).click();
  await expect(page.getByText("应用设置", { exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "修改密码", exact: true })).toBeVisible();
  const lng = await page.evaluate(() => localStorage.getItem("i18nextLng"));
  expect(lng).toBe("zh");
});

test("shows loading state during password change", async ({ page }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await new Promise((r) => setTimeout(r, 600));
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("oldpass");
  await page.getByPlaceholder("At least 6 characters").fill("newpass");
  await page.getByPlaceholder("Enter new password again").fill("newpass");
  await page.getByRole("button", { name: "Change Password" }).click();
  await expect(page.getByRole("button", { name: "Changing password..." })).toBeDisabled();
});

test("successfully changes password and resets form", async ({ page }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("oldpass");
  await page.getByPlaceholder("At least 6 characters").fill("newpass");
  await page.getByPlaceholder("Enter new password again").fill("newpass");
  await page.getByRole("button", { name: "Change Password" }).click();
  await expect(page.getByText("Password changed successfully!", { exact: true })).toBeVisible();
  await expect(page.getByPlaceholder("Enter your current password")).toHaveValue("");
  await expect(page.getByPlaceholder("At least 6 characters")).toHaveValue("");
  await expect(page.getByPlaceholder("Enter new password again")).toHaveValue("");
});

test("shows server error message field", async ({ page }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 400, contentType: "application/json", body: JSON.stringify({ message: "server error message" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("oldpass");
  await page.getByPlaceholder("At least 6 characters").fill("newvalidpass");
  await page.getByPlaceholder("Enter new password again").fill("newvalidpass");
  await page.getByRole("button", { name: "Change Password", exact: true }).click();
  await expect(page.locator(".alert.alert-danger")).toContainText("server error message");
});

test("shows server error error field", async ({ page }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 400, contentType: "application/json", body: JSON.stringify({ error: "weak password" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("oldpass");
  await page.getByPlaceholder("At least 6 characters").fill("newvalidpass");
  await page.getByPlaceholder("Enter new password again").fill("newvalidpass");
  await page.getByRole("button", { name: "Change Password", exact: true }).click();
  await expect(page.locator(".alert.alert-danger")).toContainText("weak password");
});

test("shows error alert on network abort", async ({ page }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.abort();
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("oldpass");
  await page.getByPlaceholder("At least 6 characters").fill("newpass");
  await page.getByPlaceholder("Enter new password again").fill("newpass");
  await page.getByRole("button", { name: "Change Password" }).click();
  await expect(page.getByText("Password change failed, please try again", { exact: false })).toBeVisible();
});

test("clears error and success when input changes", async ({ page }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 400, contentType: "application/json", body: JSON.stringify({ error: "weak password" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("oldpass");
  await page.getByPlaceholder("At least 6 characters").fill("newvalidpass");
  await page.getByPlaceholder("Enter new password again").fill("newvalidpass");
  await page.getByRole("button", { name: "Change Password", exact: true }).click();
  await expect(page.locator(".alert.alert-danger")).toContainText("weak password");
  await page.getByPlaceholder("Enter your current password").fill("changed");
  await expect(page.locator(".alert.alert-danger")).toHaveCount(0);
});

test("client validation requires current password", async ({ page }) => {
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill(" ");
  await page.getByPlaceholder("At least 6 characters").fill("newpass");
  await page.getByPlaceholder("Enter new password again").fill("newpass");
  await page.getByRole("button", { name: "Change Password", exact: true }).click();
  await expect(page.locator(".alert.alert-danger")).toContainText("Please enter current password");
});

test("client validation new password min length and valid strength", async ({ page }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 400, contentType: "application/json", body: JSON.stringify({ error: "weak password" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("oldpass");
  await page.getByPlaceholder("At least 6 characters").fill("123");
  await page.getByPlaceholder("Enter new password again").fill("123");
  await page.getByRole("button", { name: "Change Password", exact: true }).click();
  await expect(page.locator('input[name="new_password"] ~ .invalid-feedback')).toContainText("Password must be at least 6 characters");
  await page.getByPlaceholder("At least 6 characters").fill("123456");
  await page.getByRole("button", { name: "Change Password", exact: true }).click();
  await expect(page.locator('input[name="new_password"] ~ .valid-feedback')).toContainText("Password strength is valid");
});

test("client validation confirm not match then match", async ({ page }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 400, contentType: "application/json", body: JSON.stringify({ error: "weak password" }) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("oldpass");
  await page.getByPlaceholder("At least 6 characters").fill("abcdef");
  const confirmInput = page.getByPlaceholder("Enter new password again");
  await confirmInput.fill("ghijkl");
  await confirmInput.press("Enter");
  await expect(page.locator('input[name="confirm_password"] ~ .invalid-feedback')).toContainText("Passwords do not match");
  await page.getByPlaceholder("Enter new password again").fill("abcdef");
  await page.getByRole("button", { name: "Change Password", exact: true }).click();
  await expect(page.locator('input[name="confirm_password"] ~ .valid-feedback')).toContainText("Passwords match");
});

test("client validation new password same as current", async ({ page }) => {
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("abcdef");
  await page.getByPlaceholder("At least 6 characters").fill("abcdef");
  await page.getByPlaceholder("Enter new password again").fill("abcdef");
  await page.getByRole("button", { name: "Change Password" }).click();
  await expect(page.getByText("New password cannot be the same as current password", { exact: true })).toBeVisible();
});

test("submits by pressing Enter key on confirm field", async ({ page }) => {
  await page.route("**/api/auth/change-password", async (route) => {
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t", refresh_token: "r", user: { id: 1, username: "tester" } })));
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("oldpass");
  await page.getByPlaceholder("At least 6 characters").fill("newpass");
  const confirm = page.getByPlaceholder("Enter new password again");
  await confirm.fill("newpass");
  await confirm.press("Enter");
  await expect(page.getByText("Password changed successfully!", { exact: true })).toBeVisible();
});

test("uses Accept-Language header with English language on submit", async ({ page }, testInfo) => {
  await page.route("**/api/auth/change-password", async (route) => {
    const h = route.request().headers();
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^en-US/);
    }
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t_en", refresh_token: "r_en", user: { id: 1, username: "tester" } })));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("oldpass");
  await page.getByPlaceholder("At least 6 characters").fill("newpass");
  await page.getByPlaceholder("Enter new password again").fill("newpass");
  await page.getByRole("button", { name: "Change Password" }).click();
});

test("uses Accept-Language header with Chinese language on submit", async ({ page }, testInfo) => {
  await page.route("**/api/auth/change-password", async (route) => {
    const h = route.request().headers();
    if (testInfo.project.name === "chromium") {
      expect(h["accept-language"]).toMatch(/^zh-CN/);
    }
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("i18nextLng", "zh"));
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "t_zh", refresh_token: "r_zh", user: { id: 2, username: "tester2" } })));
  await page.goto("settings");
  await page.getByPlaceholder("当前密码", { exact: false }).fill("oldpass");
  await page.getByPlaceholder("至少6个字符", { exact: true }).fill("newpass");
  await page.getByPlaceholder("再次输入新密码", { exact: true }).fill("newpass");
  await page.getByRole("button", { name: "修改密码" }).click();
});

test("includes Authorization header when changing password", async ({ page }) => {
  await page.evaluate(() => localStorage.setItem("markpost_dev_login", JSON.stringify({ access_token: "e2e_access_token", refresh_token: "e2e_refresh_token", user: { id: 1, username: "tester" } })));
  await page.route("**/api/auth/change-password", async (route) => {
    const headers = route.request().headers();
    expect(headers["authorization"]).toBe("Bearer e2e_access_token");
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
  });
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));
  await page.goto("settings");
  await page.getByPlaceholder("Enter your current password").fill("oldpass");
  await page.getByPlaceholder("At least 6 characters").fill("newpass");
  await page.getByPlaceholder("Enter new password again").fill("newpass");
  await page.getByRole("button", { name: "Change Password" }).click();
});
