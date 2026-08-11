import type { Page, Locator } from "@playwright/test";

// B1.7 登录页（base-ui Form/Field + RHF）：按钮不再因空字段禁用，
// 错误显示在 FormAlert（role="alert"）。
export class LoginPage {
  readonly page: Page;
  readonly usernameInput: Locator;
  readonly passwordInput: Locator;
  readonly submitButton: Locator;
  readonly formAlert: Locator;
  readonly sessionExpiredBanner: Locator;

  constructor(page: Page) {
    this.page = page;
    this.usernameInput = page.locator('input[name="username"]');
    this.passwordInput = page.locator('input[name="password"]');
    this.submitButton = page.getByRole("button", { name: "Sign in" });
    this.formAlert = page.getByRole("alert").filter({ hasText: /./ });
    this.sessionExpiredBanner = page.getByText(
      "Your session has expired. Please sign in again.",
    );
  }

  async goto() {
    await this.page.goto("/login");
  }

  async login(username: string, password: string) {
    await this.usernameInput.fill(username);
    await this.passwordInput.fill(password);
    await this.submitButton.click();
  }

  async getErrorMessage() {
    return this.formAlert;
  }
}
