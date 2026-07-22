import type { Page, Locator } from "@playwright/test";

export class AdminDeliveryHistoryPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly table: Locator;
  readonly tableHeader: Locator;
  readonly tableBody: Locator;
  readonly emptyMessage: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", { name: "Delivery History" });
    this.table = page.locator("table");
    this.tableHeader = page.locator("thead");
    this.tableBody = page.locator("tbody");
    this.emptyMessage = page.getByText("No delivery history found");
  }

  async goto() {
    await this.page.goto("/admin/delivery/history");
  }

  getHistoryRow(text: string) {
    return this.page.locator("tr", { hasText: text });
  }
}
