import type { Page, Locator } from "@playwright/test";

// D3.1/D3.3 用户列表 + 治理操作（⋮ 菜单 → AlertDialog → 输入用户名确认删除）。
export class AdminUsersPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly addUserButton: Locator;
  readonly emptyMessage: Locator;
  readonly searchInput: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", { name: "Users", exact: true });
    this.addUserButton = page.getByRole("button", { name: "Add user" });
    this.emptyMessage = page.getByText("No data");
    this.searchInput = page.getByPlaceholder("Search username...");
  }

  async goto() {
    await this.page.goto("/admin/users");
  }

  getUserRow(username: string) {
    return this.page.locator("tr", { hasText: username }).first();
  }

  async openDetail(username: string) {
    await this.getUserRow(username).getByRole("link", { name: username }).click();
  }

  async openUserMenu(username: string) {
    const row = this.getUserRow(username);
    await row.getByRole("button", { name: "More actions" }).click();
  }

  async clickMenuItem(name: string) {
    await this.page.getByRole("menuitem", { name }).first().click();
  }

  async confirmDialog() {
    await this.page.getByRole("button", { name: "Confirm", exact: true }).click();
  }

  async confirmDelete(username: string) {
    await this.page.getByPlaceholder("Type username to confirm").fill(username);
    await this.page.getByRole("button", { name: "Delete permanently" }).click();
  }
}
