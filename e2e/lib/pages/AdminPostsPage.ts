import type { Page, Locator } from "@playwright/test";

export class AdminPostsPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly table: Locator;
  readonly tableHeader: Locator;
  readonly tableBody: Locator;
  readonly searchInput: Locator;
  readonly emptyMessage: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", { name: "Post Management" });
    this.table = page.locator("table");
    this.tableHeader = page.locator("thead");
    this.tableBody = page.locator("tbody");
    this.searchInput = page.locator('input[placeholder="Search title..."]');
    this.emptyMessage = page.getByText("No posts found");
  }

  async goto() {
    await this.page.goto("/admin/posts");
  }

  async search(term: string) {
    await this.searchInput.fill(term);
  }

  getPostRow(title: string) {
    return this.page.locator("tr", { hasText: title });
  }
}
