import type { Page, Locator } from "@playwright/test";

// routes.md「Landing Page (/)」：纯静态营销页，无守卫、无数据请求。
// Masthead / Hero / Colophon 的 CTA 都按会话状态切换（useAuthReady 客户端
// 检测）：未登录 → /login，已登录 → /dashboard。cta() 按 href 目标取各区域
// 唯一的 CTA 链接（hero 的 CTA 是 main 内唯一的内部链接）。
// masthead/footer 用 landmark role 定位：§01 物证（PostPageArtifact）复刻了
// 文章页的 <header>/<footer>，标签选择器会命中两个。
export class LandingPage {
  readonly page: Page;
  readonly masthead: Locator;
  readonly brandLink: Locator;
  readonly heroHeading: Locator;
  readonly footer: Locator;

  constructor(page: Page) {
    this.page = page;
    this.masthead = page.getByRole("banner");
    this.brandLink = page.locator('header a[aria-label="markpost"]');
    this.heroHeading = page.locator("main h1");
    this.footer = page.getByRole("contentinfo");
  }

  headerCta(target: "/login" | "/dashboard"): Locator {
    return this.page.locator(`header a[href="${target}"]`);
  }

  heroCta(target: "/login" | "/dashboard"): Locator {
    return this.page.locator(`main a[href="${target}"]`);
  }

  colophonCta(target: "/login" | "/dashboard"): Locator {
    return this.page.locator(`footer a[href="${target}"]`);
  }

  async goto() {
    await this.page.goto("/");
  }
}
