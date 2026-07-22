import { Page, expect } from "@playwright/test";

export class OAuthCallbackPage {
  constructor(private page: Page) {}

  async waitForCallback() {
    await this.page.waitForURL("**/auth/callback**");
  }

  async waitForRedirectToDashboard() {
    await this.page.waitForURL("**/dashboard");
  }

  async waitForRedirectToLogin() {
    await this.page.waitForURL("**/login");
  }

  async expectLoading() {
    await expect(this.page.getByText("Loading...")).toBeVisible();
  }
}
