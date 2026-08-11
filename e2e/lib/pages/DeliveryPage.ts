import type { Page, Locator } from "@playwright/test";

// F.4 渠道列表 + D5 渠道编辑 Dialog（RHF + zod + 测试状态机 + 删除 AlertDialog）。
export class DeliveryPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly table: Locator;
  readonly emptyState: Locator;
  readonly dialog: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", {
      name: "Delivery channels",
      exact: true,
    });
    this.table = page.getByRole("table").first();
    this.emptyState = page.getByText("No delivery channels yet", {
      exact: true,
    });
    this.dialog = page.getByRole("dialog").first();
  }

  async goto() {
    await this.page.goto("/delivery/channels");
  }

  async gotoHistory() {
    await this.page.goto("/delivery/history");
  }

  async clickAddChannel() {
    await this.page.getByRole("button", { name: "Add channel" }).first().click();
  }

  get channelNameInput(): Locator {
    return this.dialog.getByLabel("Channel name");
  }

  get channelWebhookInput(): Locator {
    return this.dialog.getByLabel("Webhook URL");
  }

  get channelCardLinkInput(): Locator {
    return this.dialog.getByLabel("Card link URL");
  }

  get channelKeywordsInput(): Locator {
    return this.dialog.getByPlaceholder("mark, post");
  }

  async submitCreate() {
    await this.dialog
      .getByRole("button", { name: "Create", exact: true })
      .click();
  }

  async submitSave() {
    await this.dialog
      .getByRole("button", { name: "Save", exact: true })
      .click();
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
    await this.dialog
      .getByRole("button", { name: "Send test", exact: true })
      .click();
  }

  async clickDeleteInDialog() {
    await this.dialog.getByRole("button", { name: "Delete" }).first().click();
  }

  async confirmDelete() {
    await this.page
      .getByRole("button", { name: "Delete", exact: true })
      .last()
      .click();
  }

  getLatestCell(name: string): Locator {
    return this.channelRow(name).getByRole("cell").nth(4);
  }

  async gotoChannelDetail(name: string) {
    await this.channelRow(name).getByRole("link", { name }).click();
  }
}
