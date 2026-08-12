import { test, expect } from "../lib/fixtures";
import { waitForBackend, apiLogin } from "../lib/helpers";

// B1.11 改密：用独立账号 pwuser 测试，避免污染 markpost 的密码状态。
test.beforeEach(async ({ page, request }) => {
  await waitForBackend(request);
  const admin = await apiLogin(request);
  // 幂等：删除旧 pwuser 后用已知密码重建（避免密码状态污染）。
  const list = await request.get(`https://localhost:2053/api/v1/admin/users`, {
    headers: { Authorization: `Bearer ${admin.token}` },
  });
  const users = (await list.json()).items ?? [];
  const existing = users.find((u: { username: string }) => u.username === "pwuser");
  if (existing) {
    await request.delete(`https://localhost:2053/api/v1/admin/users/${existing.id}`, {
      headers: { Authorization: `Bearer ${admin.token}` },
    });
  }
  const created = await request.post(`https://localhost:2053/api/v1/admin/users`, {
    headers: { Authorization: `Bearer ${admin.token}` },
    data: { username: "pwuser", password: "password123" },
  });
  if (!created.ok()) {
    throw new Error(`create pwuser failed: ${created.status()} ${await created.text()}`);
  }
  await page.context().clearCookies();
  await page.goto("/login");
  await page.evaluate(() => {
    localStorage.clear();
    localStorage.setItem("locale", "en");
  });
});

test("changes password with valid input and can re-login", async ({
  page,
  loginPage,
  settingsPage,
}) => {
  await loginPage.login("pwuser", "password123");
  await page.waitForURL("**/dashboard", { timeout: 30000 });
  await settingsPage.goto();
  await settingsPage.fillPasswordForm("password123", "newpass123", "newpass123");
  await settingsPage.clickChangePassword();
  // 成功 toast（C2.2 吊销联动：返回新 token 对，无重登）
  const toast = page.getByText("Password changed successfully");
  await expect(toast).toBeVisible({ timeout: 15000 });
  // 等第一次 toast 消失，避免第二次断言命中残留 toast。
  await expect(toast).toHaveCount(0, { timeout: 15000 });
  // 恢复初始密码（UI 改回），保持幂等。
  await settingsPage.fillPasswordForm("newpass123", "password123", "password123");
  await settingsPage.clickChangePassword();
  await expect(toast).toBeVisible({ timeout: 15000 });
  // 清空本地状态后重载页面，重置内存 store，再用原密码重新登录验证。
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await loginPage.goto();
  await loginPage.login("pwuser", "password123");
  await page.waitForURL("**/dashboard", { timeout: 30000 });
});

test("rejects short new password with inline error", async ({ page, loginPage, settingsPage }) => {
  await loginPage.login("pwuser", "password123");
  await page.waitForURL("**/dashboard", { timeout: 30000 });
  await settingsPage.goto();
  await settingsPage.fillPasswordForm("password123", "abc", "abc");
  await settingsPage.clickChangePassword();
  await expect(page.getByText("At least 8 characters")).toBeVisible({
    timeout: 15000,
  });
});

test("shows mismatch error when confirmation differs", async ({
  page,
  loginPage,
  settingsPage,
}) => {
  await loginPage.login("pwuser", "password123");
  await page.waitForURL("**/dashboard", { timeout: 30000 });
  await settingsPage.goto();
  await settingsPage.fillPasswordForm("password123", "newpass123", "different");
  await settingsPage.clickChangePassword();
  await expect(page.getByText("Passwords do not match")).toBeVisible({
    timeout: 15000,
  });
});
