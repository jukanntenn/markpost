import type { Page, Locator } from "@playwright/test";

export class DashboardPage {
  readonly page: Page;
  readonly postKeyHeading: Locator;
  readonly showKeyButton: Locator;
  readonly hideKeyButton: Locator;
  readonly copyKeyButton: Locator;
  readonly latestPostsHeading: Locator;

  constructor(page: Page) {
    this.page = page;
    this.postKeyHeading = page.getByText("Post Key", { exact: true });
    this.showKeyButton = page.getByTitle("Show key");
    this.hideKeyButton = page.getByTitle("Hide key");
    this.copyKeyButton = page.getByTitle("Copy key");
    this.latestPostsHeading = page.getByText("Latest Posts", { exact: true });
  }

  async goto() {
    await this.page.goto("/dashboard");
  }

  async clickShowKey() {
    await this.showKeyButton.click();
  }

  async clickHideKey() {
    await this.hideKeyButton.click();
  }

  async clickCopyKey() {
    await this.copyKeyButton.click();
    await this.page.getByText("Copy Post Key", { exact: true }).click();
  }

  async getPostKeyText() {
    return this.page.locator(".font-mono");
  }

  async getCopiedBadge() {
    return this.page.getByText("copied to clipboard!", { exact: true });
  }

  async clickUserMenu() {
    await this.page.getByRole("button").filter({ hasText: /markpost/ }).click();
  }

  async clickLogout() {
    await this.page.getByText("Logout", { exact: true }).click();
  }

  async getQuickCreateButton() {
    return this.page.getByTitle("Create Test Post");
  }
}
