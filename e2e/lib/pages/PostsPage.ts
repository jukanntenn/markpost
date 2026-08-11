import type { Page, Locator } from "@playwright/test";

// F.5 帖子列表：搜索 + 分页（页码/总条数）。
export class PostsPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly searchInput: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", { name: "Posts", exact: true });
    this.searchInput = page.getByPlaceholder("Search titles...");
  }

  async goto() {
    await this.page.goto("/posts");
  }

  async getPostLink(title: string) {
    return this.page.getByRole("link", { name: title });
  }
}
