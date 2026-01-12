import type { Page, Locator } from "@playwright/test";

export class PostsPage {
  readonly page: Page;
  readonly allPostsHeading: Locator;
  readonly titleColumnHeader: Locator;
  readonly previousButton: Locator;
  readonly nextButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.allPostsHeading = page.getByRole("heading", { name: "All Posts", exact: true });
    this.titleColumnHeader = page.getByRole("columnheader", { name: "Title" });
    this.previousButton = page.getByRole("button", { name: "Previous" });
    this.nextButton = page.getByRole("button", { name: "Next" });
  }

  async goto() {
    await this.page.goto("posts");
  }

  async getPostLink(title: string) {
    return this.page.getByRole("link", { name: title });
  }

  async getNoPostsMessage() {
    return this.page.getByText("No posts yet", { exact: true });
  }
}
