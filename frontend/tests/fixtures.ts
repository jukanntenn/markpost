import { test as base, type Page } from "@playwright/test";
import { LoginPage } from "./pages/LoginPage";
import { DashboardPage } from "./pages/DashboardPage";
import { PostsPage } from "./pages/PostsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { mockUsers } from "./data/mock-data";

export type TestFixtures = {
  loginPage: LoginPage;
  dashboardPage: DashboardPage;
  postsPage: PostsPage;
  settingsPage: SettingsPage;
  authenticatedPage: Page;
};

export const test = base.extend<TestFixtures>({
  loginPage: async ({ page }, provide) => {
    await provide(new LoginPage(page));
  },

  dashboardPage: async ({ page }, provide) => {
    await provide(new DashboardPage(page));
  },

  postsPage: async ({ page }, provide) => {
    await provide(new PostsPage(page));
  },

  settingsPage: async ({ page }, provide) => {
    await provide(new SettingsPage(page));
  },

  authenticatedPage: async ({ page }, provide) => {
    await page.goto("login");
    await page.evaluate((user) => {
      localStorage.setItem("markpost_dev_login", JSON.stringify(user));
      localStorage.setItem("i18nextLng", "en");
    }, mockUsers.e2e);
    await provide(page);
  },
});

export { expect } from "@playwright/test";
