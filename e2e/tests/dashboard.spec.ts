import { test, expect } from "../lib/fixtures";
import {
  waitForBackend,
  apiLogin,
  deleteAllPosts,
  deleteAllDeliveryChannels,
} from "../lib/helpers";

test.beforeEach(async ({ page, request }) => {
  await waitForBackend(request);
  const auth = await apiLogin(request);
  await deleteAllPosts(request, auth.token);
  await deleteAllDeliveryChannels(request, auth.token);
  await page.context().clearCookies();
  await page.goto("/login");
  await page.evaluate(() => {
    localStorage.clear();
    localStorage.setItem("locale", "en");
  });
});

// B2/I.9：新用户（无渠道无帖子）→ 三步 onboarding；创建渠道后进入正常态。
test("new user sees the onboarding guide (I.9)", async ({ page, loginPage }) => {
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await expect(page.getByText("Getting started with markpost")).toBeVisible({
    timeout: 15000,
  });
  await expect(page.getByText("Create a delivery channel", { exact: true })).toBeVisible();
});

test("dashboard renders pipeline status and post key block", async ({
  page,
  loginPage,
  dashboardPage,
}) => {
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await expect(dashboardPage.welcomeHeading).toBeVisible({ timeout: 15000 });
  await expect(dashboardPage.postKeyHeading).toBeVisible();
});

test("logout returns to login (B1 场景E)", async ({ page, loginPage, dashboardPage }) => {
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await dashboardPage.clickUserMenu("markpost");
  await dashboardPage.clickLogout();
  await page.waitForURL("**/login**");
  await expect(loginPage.submitButton).toBeVisible();
});
