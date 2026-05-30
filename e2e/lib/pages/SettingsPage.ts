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

  readonly deliveryChannelsCard: Locator;
  readonly deliveryChannelsHeading: Locator;
  readonly addChannelButton: Locator;
  readonly channelNameInput: Locator;
  readonly channelWebhookInput: Locator;
  readonly channelKeywordsInput: Locator;
  readonly channelSubmitButton: Locator;
  readonly emptyChannelsMessage: Locator;

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

    this.deliveryChannelsCard = page.getByTestId("delivery-channels-card");
    this.deliveryChannelsHeading = this.deliveryChannelsCard.getByText("Delivery Channels", { exact: true });
    this.addChannelButton = this.deliveryChannelsCard.getByRole("button", { name: "Add Feishu channel" });
    this.channelNameInput = this.deliveryChannelsCard.locator("#channel-name");
    this.channelWebhookInput = this.deliveryChannelsCard.locator("#channel-webhook");
    this.channelKeywordsInput = this.deliveryChannelsCard.locator("#channel-keywords");
    this.channelSubmitButton = this.deliveryChannelsCard.getByRole("button", { name: "Create" });
    this.emptyChannelsMessage = this.deliveryChannelsCard.getByText("No delivery channels yet", { exact: true });
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

  channelRow(name: string) {
    return this.deliveryChannelsCard.locator(".rounded-lg.border", { hasText: name }).first();
  }

  async createChannel(name: string, webhookUrl: string, keywords?: string) {
    await this.addChannelButton.click();
    await this.channelNameInput.fill(name);
    await this.channelWebhookInput.fill(webhookUrl);
    if (keywords) {
      await this.channelKeywordsInput.fill(keywords);
    }
    await this.channelSubmitButton.click();
  }
}
