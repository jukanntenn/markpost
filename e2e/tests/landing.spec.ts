import { test, expect } from "../lib/fixtures";
import { LandingPage } from "../lib/pages/LandingPage";

// routes.md「Landing Page (/)」冒烟：纯静态营销页（无守卫、无数据请求），
// Masthead / Hero / Colophon 结构齐备；CTA 按会话状态切换 —— 未登录指向
// /login，已登录（useAuthReady 客户端检测，不强制跳转）变为
// Open console → /dashboard。
test.beforeEach(async ({ page }) => {
  await page.goto("/");
  await page.evaluate(() => localStorage.setItem("locale", "en"));
  await page.reload();
});

test("renders masthead, hero and colophon", async ({ landingPage }) => {
  await expect(landingPage.masthead).toBeVisible();
  await expect(landingPage.brandLink).toBeVisible();
  await expect(landingPage.heroHeading).toBeVisible();
  await expect(landingPage.footer).toBeVisible();
});

test("unauthenticated CTAs point to /login", async ({ landingPage }) => {
  await expect(landingPage.headerCta("/login")).toHaveText("Sign in");
  await expect(landingPage.heroCta("/login")).toHaveText("Get started");
  await expect(landingPage.colophonCta("/login")).toBeVisible();
});

test("authenticated CTAs become Open console → /dashboard", async ({ authenticatedPage }) => {
  const landing = new LandingPage(authenticatedPage);
  await landing.goto();

  await expect(landing.headerCta("/dashboard")).toHaveText("Open console");
  await expect(landing.heroCta("/dashboard")).toBeVisible();
  await expect(landing.colophonCta("/dashboard")).toBeVisible();
});
