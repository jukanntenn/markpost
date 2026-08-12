import type { Page, Locator } from "@playwright/test";

// B2 Dashboard：主副栏版式 + 管道状态机 + 活动流 + Post Key 折叠块。
export class DashboardPage {
  readonly page: Page;
  readonly welcomeHeading: Locator;
  readonly postKeyHeading: Locator;
  readonly pipelineStatus: Locator;
  readonly channelHealth: Locator;

  constructor(page: Page) {
    this.page = page;
    this.welcomeHeading = page.getByRole("heading", {
      name: /Welcome, /,
    });
    this.postKeyHeading = page.getByText("Post Key", { exact: true });
    this.pipelineStatus = page.locator('[class*="animate-ping"]').first();
    this.channelHealth = page.getByText("Channel health", { exact: true });
  }

  async goto() {
    await this.page.goto("/dashboard");
  }

  async clickUserMenu(username: string) {
    await this.page.getByRole("button").filter({ hasText: username }).first().click();
  }

  async clickLogout() {
    await this.page.getByText("Sign out", { exact: true }).click();
  }
}
