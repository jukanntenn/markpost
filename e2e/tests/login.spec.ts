import { test, expect } from "../lib/fixtures";
import { waitForBackend, apiLogin } from "../lib/helpers";

test.beforeEach(async ({ page, request }) => {
  await waitForBackend(request);
  await page.context().clearCookies();
  await page.goto("/login");
  await page.evaluate(() => localStorage.clear());
});

// B1.7 登录：成功跳转 /dashboard；失败 FormAlert；会话过期横幅。
test("renders login page with username and password fields", async ({
  loginPage,
}) => {
  await expect(loginPage.usernameInput).toBeVisible();
  await expect(loginPage.passwordInput).toBeVisible();
  await expect(loginPage.submitButton).toBeVisible();
});

test("logs in with valid credentials and redirects to dashboard", async ({
  page,
  loginPage,
  dashboardPage,
}) => {
  await page.evaluate(() => localStorage.setItem("locale", "en"));

  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard", { timeout: 30000 });
  await expect(dashboardPage.welcomeHeading).toBeVisible({ timeout: 15000 });
});

test("shows error on invalid credentials", async ({ page, loginPage }) => {
  await page.evaluate(() => localStorage.setItem("locale", "en"));

  await loginPage.login("markpost", "wrongpassword");
  await expect(loginPage.formAlert).toContainText(
    "Incorrect username or password",
  );
});

test("account locks after 5 failed attempts (C2.1)", async ({
  page,
  loginPage,
  request,
}) => {
  await page.evaluate(() => localStorage.setItem("locale", "en"));

  // 用独立账号锁定，避免把 markpost 锁死 15 分钟污染后续测试。
  const auth = await apiLogin(request);
  await request.post(`https://localhost:2053/api/v1/admin/users`, {
    headers: { Authorization: `Bearer ${auth.token}` },
    data: { username: "locktarget", password: "password123" },
  });

  for (let i = 0; i < 5; i++) {
    await loginPage.login("locktarget", "wrongpassword");
    await expect(loginPage.formAlert).toBeVisible();
  }
  // 锁定期间正确密码也返回 account_locked（429 文案带分钟）。
  await loginPage.login("locktarget", "password123");
  await expect(loginPage.formAlert).toContainText(/Too many attempts/);
});

test("keeps next= target after login (K.3)", async ({
  page,
  loginPage,
}) => {
  await page.evaluate(() => localStorage.setItem("locale", "en"));
  await page.goto("/posts");
  await page.waitForURL("**/login?next=**");
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/posts", { timeout: 30000 });
});
