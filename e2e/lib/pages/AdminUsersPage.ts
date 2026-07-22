import type { Page, Locator } from "@playwright/test";

export class AdminUsersPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly table: Locator;
  readonly tableHeader: Locator;
  readonly tableBody: Locator;
  readonly addUserButton: Locator;
  readonly emptyMessage: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", { name: "User Management" });
    this.table = page.locator("table");
    this.tableHeader = page.locator("thead");
    this.tableBody = page.locator("tbody");
    this.addUserButton = page.getByRole("button", { name: "Add User" });
    this.emptyMessage = page.getByText("No users found");
  }

  async goto() {
    await this.page.goto("/admin/users");
  }

  getUserRow(username: string) {
    return this.page.locator("tr", { hasText: username });
  }

  getCellText(row: Locator, columnIndex: number) {
    return row.locator("td").nth(columnIndex);
  }
}
