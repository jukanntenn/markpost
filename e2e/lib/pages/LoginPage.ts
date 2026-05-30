import type { Page, Locator } from "@playwright/test";

export class LoginPage {
  readonly page: Page;
  readonly usernameInput: Locator;
  readonly passwordInput: Locator;
  readonly submitButton: Locator;
  readonly alertDanger: Locator;
  readonly errorAlert: Locator;

  constructor(page: Page) {
    this.page = page;
    this.usernameInput = page.locator('input[name="username"]');
    this.passwordInput = page.locator('input[name="password"]');
    this.submitButton = page.locator('button[type="submit"]');
    this.alertDanger = page.locator(".alert-danger");
    this.errorAlert = page.locator("[data-slot='alert']");
  }

  async goto() {
    await this.page.goto("/login");
  }

  async login(username: string, password: string) {
    await this.usernameInput.fill(username);
    await this.passwordInput.fill(password);
    await this.submitButton.click();
  }

  async submitByPressingEnter() {
    await this.passwordInput.press("Enter");
  }

  async getErrorMessage() {
    return this.errorAlert;
  }

  async isSubmitDisabled() {
    return await this.submitButton.isDisabled();
  }

  async isSubmitEnabled() {
    return await this.submitButton.isEnabled();
  }
}
