import type { Page, Locator } from "@playwright/test";

export class DeliveryPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly table: Locator;
  readonly emptyState: Locator;
  readonly dialog: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", {
      name: "Delivery Channels",
      exact: true,
    });
    this.table = page.getByRole("table").first();
    this.emptyState = page.getByText("No delivery channels yet", {
      exact: true,
    });
    this.dialog = page.locator("[data-slot='dialog-content']");
  }

  async goto() {
    await this.page.goto("/delivery/channels");
  }

  async gotoHistory() {
    await this.page.goto("/delivery/history");
  }

  async clickAddChannel() {
    await this.page.getByRole("button", { name: "Add Channel" }).first().click();
  }

  get channelNameInput(): Locator {
    return this.dialog.locator("#channel-name");
  }

  get channelWebhookInput(): Locator {
    return this.dialog.locator("#channel-webhook");
  }

  get channelCardLinkInput(): Locator {
    return this.dialog.locator("#channel-card-link-url");
  }

  get channelKeywordsInput(): Locator {
    return this.dialog.locator("#channel-keywords");
  }

  async submitCreate() {
    await this.dialog.getByRole("button", { name: "Create", exact: true }).click();
  }

  async submitSave() {
    await this.dialog.getByRole("button", { name: "Save", exact: true }).click();
  }

  async createChannel(name: string, webhookUrl: string, keywords?: string) {
    await this.clickAddChannel();
    await this.channelNameInput.fill(name);
    await this.channelWebhookInput.fill(webhookUrl);
    if (keywords) {
      await this.channelKeywordsInput.fill(keywords);
    }
    await this.submitCreate();
  }

  channelRow(name: string): Locator {
    return this.table.getByRole("row").filter({ hasText: name }).first();
  }

  async editChannel(name: string) {
    await this.channelRow(name).getByRole("button", { name: "Edit" }).click();
  }

  async clickTestInDialog() {
    await this.dialog.getByRole("button", { name: "Test", exact: true }).click();
  }

  async clickDeleteInDialog() {
    await this.dialog.getByRole("button", { name: "Delete" }).click();
  }

  async confirmDelete() {
    await this.dialog.getByRole("button", { name: "Confirm Delete" }).click();
  }

  getLatestCell(name: string): Locator {
    return this.channelRow(name).getByRole("cell").nth(4);
  }

  async gotoChannelDetail(name: string) {
    await this.channelRow(name).getByRole("link", { name }).click();
  }
}
