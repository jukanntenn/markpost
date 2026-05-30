import { test as base, expect, type Page, type APIRequestContext } from "@playwright/test";
import { LoginPage } from "./pages/LoginPage";
import { DashboardPage } from "./pages/DashboardPage";
import { PostsPage } from "./pages/PostsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { apiLogin, waitForBackend } from "./helpers";

type TestFixtures = {
  loginPage: LoginPage;
  dashboardPage: DashboardPage;
  postsPage: PostsPage;
  settingsPage: SettingsPage;
  authenticatedPage: Page;
  authToken: { token: string; refreshToken: string; user: { id: number; username: string } };
};

export const test = base.extend<TestFixtures>({
  loginPage: async ({ page }, use) => {
    await use(new LoginPage(page));
  },

  dashboardPage: async ({ page }, use) => {
    await use(new DashboardPage(page));
  },

  postsPage: async ({ page }, use) => {
    await use(new PostsPage(page));
  },

  settingsPage: async ({ page }, use) => {
    await use(new SettingsPage(page));
  },

  authToken: async ({ request }, use) => {
    await waitForBackend(request);
    const auth = await apiLogin(request);
    await use({
      token: auth.token,
      refreshToken: auth.refresh_token,
      user: { id: auth.user.id, username: auth.user.username },
    });
  },

  authenticatedPage: async ({ page, authToken }, use) => {
    await page.goto("/login");
    await page.evaluate((auth) => {
      localStorage.setItem(
        "markpost_auth",
        JSON.stringify({
          state: {
            token: auth.token,
            refreshToken: auth.refreshToken,
            user: auth.user,
            _hasHydrated: true,
          },
          version: 0,
        }),
      );
      localStorage.setItem("i18nextLng", "en");
    }, authToken);
    await page.reload();
    await use(page);
  },
});

export { expect };
