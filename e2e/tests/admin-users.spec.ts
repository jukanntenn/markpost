import { test, expect } from "../lib/fixtures";
import { waitForBackend, apiLogin } from "../lib/helpers";

test.beforeEach(async ({ page, request }) => {
  await waitForBackend(request);
  await page.context().clearCookies();
  await page.goto("/login");
  await page.evaluate(() => {
    localStorage.clear();
    localStorage.setItem("locale", "en");
  });
});

test("admin sees the user list with role badges (D3.1)", async ({
  page,
  loginPage,
  adminUsersPage,
}) => {
  await loginPage.goto();
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await adminUsersPage.goto();
  await expect(adminUsersPage.heading).toBeVisible();
  await expect(adminUsersPage.getUserRow("markpost")).toBeVisible();
  await expect(page.getByText("Admin", { exact: true }).first()).toBeVisible();
});

test("self-demote is hidden for own account (I.4)", async ({ page, loginPage, adminUsersPage }) => {
  await loginPage.goto();
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await adminUsersPage.goto();
  await adminUsersPage.openUserMenu("markpost");
  // 菜单中不出现对自身的角色切换（防自降级前端隐藏）。
  await expect(page.getByRole("menuitem", { name: /Change role/ })).toHaveCount(0);
});

test("user detail page shows profile and sessions (D3.2)", async ({
  page,
  loginPage,
  adminUsersPage,
}) => {
  await loginPage.goto();
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await adminUsersPage.goto();
  await adminUsersPage.openDetail("markpost");
  await page.waitForURL("**/admin/users?id=**");
  await expect(page.getByText("Profile", { exact: true })).toBeVisible();
  await expect(page.getByText("Actions", { exact: true })).toBeVisible();
  await expect(page.getByText("Active sessions", { exact: true })).toBeVisible();
});

test("delete user requires typed username confirmation (D3.3)", async ({
  page,
  loginPage,
  adminUsersPage,
  request,
}) => {
  // 通过 API 创建删除目标用户（markpost 是自身，删除项被隐藏）。
  const admin = await apiLogin(request);
  await request.post(`https://localhost:2053/api/v1/admin/users`, {
    headers: { Authorization: `Bearer ${admin.token}` },
    data: { username: "deletee", password: "password123" },
  });
  await page.context().clearCookies();
  await page.goto("/login");
  await page.evaluate(() => {
    localStorage.clear();
    localStorage.setItem("locale", "en");
  });
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard", { timeout: 30000 });
  await adminUsersPage.goto();

  // 菜单 → 删除
  await adminUsersPage.openUserMenu("deletee");
  await adminUsersPage.clickMenuItem("Delete user");
  const confirmBtn = page.getByRole("button", { name: "Delete permanently" });
  await expect(confirmBtn).toBeDisabled();
  await page.getByPlaceholder("Type username to confirm").fill("wrongname");
  await expect(confirmBtn).toBeDisabled();
  await page.getByPlaceholder("Type username to confirm").fill("deletee");
  await expect(confirmBtn).toBeEnabled();
});
