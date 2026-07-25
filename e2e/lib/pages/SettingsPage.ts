import type { Page, Locator } from "@playwright/test";

export class SettingsPage {
  readonly page: Page;
  readonly appSettingsHeading: Locator;
  readonly languageLabel: Locator;
  readonly changePasswordHeading: Locator;
  readonly currentPasswordInput: Locator;
  readonly newPasswordInput: Locator;
  readonly confirmPasswordInput: Locator;
  readonly changePasswordButton: Locator;
  readonly localeSelect: Locator;

  constructor(page: Page) {
    this.page = page;
    this.appSettingsHeading = page.getByText("Application Settings", { exact: true });
    this.languageLabel = page.getByText("Language", { exact: true });
    this.changePasswordHeading = page.getByText("Change Password", { exact: true });
    this.currentPasswordInput = page.locator("#current-password");
    this.newPasswordInput = page.locator("#new-password");
    this.confirmPasswordInput = page.locator("#confirm-password");
    this.changePasswordButton = page.getByRole("button", { name: "Save" });
    this.localeSelect = page.locator("#locale-select");
  }

  async goto() {
    await this.page.goto("/settings");
  }

  async fillPasswordForm(current: string, newPass: string, confirm: string) {
    await this.currentPasswordInput.fill(current);
    await this.newPasswordInput.fill(newPass);
    await this.confirmPasswordInput.fill(confirm);
  }

  async clickChangePassword() {
    await this.changePasswordButton.click();
  }

  async getSuccessMessage() {
    return this.page.getByText("Password changed successfully!", { exact: true });
  }

  getAlert() {
    return this.page.locator("[data-slot='alert']");
  }

  async selectLocale(value: string) {
    await this.localeSelect.selectOption(value);
  }
}
