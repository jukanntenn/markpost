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

// D4 审计日志：筛选（URL 同步）+ 行展开元数据 + IP 脱敏。
test("audit logs page renders the table", async ({ page, loginPage }) => {
  await loginPage.goto();
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard");
  await page.goto("/admin/audit-logs");
  await expect(page.getByRole("heading", { name: "Audit logs", exact: true })).toBeVisible();
});

test("filtering by action updates the URL (D4.3)", async ({ page, loginPage, request }) => {
  // 先通过 API 触发一条审计记录（admin 建用户会 RecordAudit），
  // 让"动作"筛选下拉出现 user.create 选项（facets 来自真实计数）。
  const admin = await apiLogin(request);
  await request.post(`https://localhost:2053/api/v1/admin/users`, {
    headers: { Authorization: `Bearer ${admin.token}` },
    data: { username: "auditseed", password: "password123" },
  });
  await page.context().clearCookies();
  await page.goto("/login");
  await page.evaluate(() => {
    localStorage.clear();
    localStorage.setItem("locale", "en");
  });
  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard", { timeout: 30000 });
  await page.goto("/admin/audit-logs");

  // 打开动作筛选并选择 user.create（facet 选项 label 为原始 action 码 + 计数；
  // 等待弹出动画稳定后再点，避免点击落在触发器上）。
  await page.getByRole("combobox").first().click();
  await page.waitForTimeout(400);
  await page.getByRole("option").filter({ hasText: "user.create" }).click();
  // 静态导出下 router.replace 的 SPA 导航不触发 Playwright 的 load/commit
  // 导航事件，改用轮询断言 URL。
  await expect.poll(() => page.url()).toContain("action=user.create");
});
