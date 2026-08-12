import type { Page, Locator } from "@playwright/test";

// F.2 设置页：偏好（base-ui Select）+ 安全（改密 + 我的会话）。
export class SettingsPage {
  readonly page: Page;
  readonly preferencesHeading: Locator;
  readonly changePasswordHeading: Locator;
  readonly currentPasswordInput: Locator;
  readonly newPasswordInput: Locator;
  readonly confirmPasswordInput: Locator;
  readonly changePasswordButton: Locator;
  readonly localeTrigger: Locator;
  readonly sessionsHeading: Locator;

  constructor(page: Page) {
    this.page = page;
    this.preferencesHeading = page.getByText("Preferences", { exact: true });
    this.changePasswordHeading = page.getByText("Change password", {
      exact: true,
    });
    this.currentPasswordInput = page.getByPlaceholder("Enter current password");
    this.newPasswordInput = page.getByPlaceholder("Enter new password", { exact: true });
    this.confirmPasswordInput = page.getByPlaceholder("Re-enter new password");
    this.changePasswordButton = page.getByRole("button", {
      name: "Change password",
    });
    this.localeTrigger = page.getByRole("combobox");
    this.sessionsHeading = page.getByText("My sessions", { exact: true });
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
}
