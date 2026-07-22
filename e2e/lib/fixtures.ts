import { test as base, expect, type Page, type APIRequestContext } from "@playwright/test";
import { LoginPage } from "./pages/LoginPage";
import { DashboardPage } from "./pages/DashboardPage";
import { PostsPage } from "./pages/PostsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { AdminUsersPage } from "./pages/AdminUsersPage";
import { AdminPostsPage } from "./pages/AdminPostsPage";
import { AdminChannelsPage } from "./pages/AdminChannelsPage";
import { AdminDeliveryHistoryPage } from "./pages/AdminDeliveryHistoryPage";
import { OAuthCallbackPage } from "./pages/OAuthCallbackPage";
import { apiLogin, waitForBackend, deleteAllPosts, deleteAllDeliveryChannels, clearWebhooks, clearOAuthRequests } from "./helpers";

type TestFixtures = {
  loginPage: LoginPage;
  dashboardPage: DashboardPage;
  postsPage: PostsPage;
  settingsPage: SettingsPage;
  adminUsersPage: AdminUsersPage;
  adminPostsPage: AdminPostsPage;
  adminChannelsPage: AdminChannelsPage;
  adminDeliveryHistoryPage: AdminDeliveryHistoryPage;
  oauthCallbackPage: OAuthCallbackPage;
  authenticatedPage: Page;
  authToken: { token: string; refreshToken: string; user: { id: number; username: string; role: string } };
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

  adminUsersPage: async ({ page }, use) => {
    await use(new AdminUsersPage(page));
  },

  adminPostsPage: async ({ page }, use) => {
    await use(new AdminPostsPage(page));
  },

  adminChannelsPage: async ({ page }, use) => {
    await use(new AdminChannelsPage(page));
  },

  adminDeliveryHistoryPage: async ({ page }, use) => {
    await use(new AdminDeliveryHistoryPage(page));
  },

  oauthCallbackPage: async ({ page }, use) => {
    await use(new OAuthCallbackPage(page));
  },

  request: async ({ playwright }, use) => {
    const request = await playwright.request.newContext({
      ignoreHTTPSErrors: true,
    });
    await use(request);
    await request.dispose();
  },

  authToken: async ({ request }, use) => {
    await waitForBackend(request);
    const auth = await apiLogin(request);
    await use({
      token: auth.token,
      refreshToken: auth.refresh_token,
      user: { id: auth.user.id, username: auth.user.username, role: auth.user.role },
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
      localStorage.setItem("locale", "en");
    }, authToken);
    await page.reload();
    await use(page);
  },
});

export { expect };

export async function cleanupTestData(request: APIRequestContext, token: string): Promise<void> {
  try {
    await deleteAllPosts(request, token);
    await deleteAllDeliveryChannels(request, token);
    await clearWebhooks(request);
    await clearOAuthRequests(request);
  } catch (e) {
    console.log(`Cleanup warning: ${e instanceof Error ? e.message : String(e)}`);
  }
}
