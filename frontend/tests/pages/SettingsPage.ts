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
  readonly languageToggleButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.appSettingsHeading = page.getByText("Application Settings", { exact: true });
    this.languageLabel = page.getByText("Language", { exact: true });
    this.changePasswordHeading = page.getByRole("heading", { name: "Change Password", exact: true });
    this.currentPasswordInput = page.locator('input[name="current_password"]');
    this.newPasswordInput = page.locator('input[name="new_password"]');
    this.confirmPasswordInput = page.locator('input[name="confirm_password"]');
    this.changePasswordButton = page.getByRole("button", { name: "Change Password" });
    this.languageToggleButton = page.getByRole("button", { name: "Change Language" });
  }

  async goto() {
    await this.page.goto("settings");
  }

  async fillPasswordForm(current: string, newPass: string, confirm: string) {
    await this.currentPasswordInput.fill(current);
    await this.newPasswordInput.fill(newPass);
    await this.confirmPasswordInput.fill(confirm);
  }

  async clickChangePassword() {
    await this.changePasswordButton.click();
  }

  async clickChangePasswordByEnter() {
    await this.confirmPasswordInput.press("Enter");
  }

  async getSuccessMessage() {
    return this.page.getByText("Password changed successfully!", { exact: true });
  }

  async getAlert() {
    return this.page.locator(".alert.alert-danger");
  }

  async clickLanguageToggle() {
    await this.languageToggleButton.click();
  }

  async selectLanguage(text: string) {
    await this.page.getByText(text, { exact: true }).click();
  }
}
