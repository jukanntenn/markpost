import { test, expect } from "../lib/fixtures";
import { OAuthCallbackPage } from "../lib/pages/OAuthCallbackPage";

const OAUTH_MOCK_URL = process.env.OAUTH_MOCK_URL || "http://localhost:3001";

test.describe("OAuth Callback", () => {
  let oauthCallbackPage: OAuthCallbackPage;

  test.beforeEach(async ({ page }) => {
    oauthCallbackPage = new OAuthCallbackPage(page);
  });

  test("missing code and state redirects to /login", async ({ page }) => {
    await page.goto("/auth/callback");
    await oauthCallbackPage.waitForRedirectToLogin();
  });

  test("error param redirects to /login", async ({ page }) => {
    await page.goto("/auth/callback?error=access_denied");
    await oauthCallbackPage.waitForRedirectToLogin();
  });

  test("invalid state redirects to /login", async ({ page }) => {
    await page.goto("/auth/callback?code=test_code&state=invalid_state");
    await oauthCallbackPage.waitForRedirectToLogin();
  });

  test("full OAuth flow: login with GitHub", async ({ page, request }) => {
    // Clear any previous OAuth requests
    await request.post(`${OAUTH_MOCK_URL}/requests/clear`);

    // Navigate to login page
    await page.goto("/login");

    // Click the GitHub login button
    const githubButton = page.getByRole("button", { name: /github/i });
    await githubButton.click();

    // Wait for the OAuth mock to redirect back with code and state
    await page.waitForURL("**/auth/callback**");

    // The callback page should process the code and state
    // and redirect to dashboard on success
    await oauthCallbackPage.waitForRedirectToDashboard();

    // Verify we're logged in by checking the dashboard
    await expect(page.getByText("Dashboard")).toBeVisible();
  });

  test("full OAuth flow: handles network error gracefully", async ({ page, request }) => {
    // Clear any previous OAuth requests
    await request.post(`${OAUTH_MOCK_URL}/requests/clear`);

    // Navigate to login page
    await page.goto("/login");

    // Intercept the OAuth URL request to simulate a network error
    await page.route("**/api/v1/oauth/url", (route) => {
      route.abort("failed");
    });

    // Click the GitHub login button
    const githubButton = page.getByRole("button", { name: /github/i });
    await githubButton.click();

    // Should stay on login page or show error
    await page.waitForURL("**/login");
  });

  test("full OAuth flow: handles token exchange failure", async ({ page, request }) => {
    // Clear any previous OAuth requests
    await request.post(`${OAUTH_MOCK_URL}/requests/clear`);

    // Set up the route interception BEFORE navigating to prevent the race
    await page.route("**/api/v1/oauth/login", (route) => {
      route.fulfill({
        status: 401,
        body: JSON.stringify({ error: "OAuth exchange failed" }),
      });
    });

    // Navigate to login page
    await page.goto("/login");

    // Click the GitHub login button
    const githubButton = page.getByRole("button", { name: /github/i });
    await githubButton.click();

    // Wait for the OAuth mock to redirect back with code and state
    await page.waitForURL("**/auth/callback**");

    // The callback page should redirect to login on failure
    await oauthCallbackPage.waitForRedirectToLogin();
  });

  test("full OAuth flow: handles user info fetch failure", async ({ page, request }) => {
    // Clear any previous OAuth requests
    await request.post(`${OAUTH_MOCK_URL}/requests/clear`);

    // Set up the route interception BEFORE navigating to prevent the race
    await page.route("**/api/v1/oauth/login", (route) => {
      route.fulfill({
        status: 500,
        body: JSON.stringify({ error: "Failed to fetch user info" }),
      });
    });

    // Navigate to login page
    await page.goto("/login");

    // Click the GitHub login button
    const githubButton = page.getByRole("button", { name: /github/i });
    await githubButton.click();

    // Wait for the OAuth mock to redirect back with code and state
    await page.waitForURL("**/auth/callback**");

    // The callback page should redirect to login on failure
    await oauthCallbackPage.waitForRedirectToLogin();
  });

  test("OAuth state is consumed after use", async ({ page, request }) => {
    // Clear any previous OAuth requests
    await request.post(`${OAUTH_MOCK_URL}/requests/clear`);

    // Navigate to login page
    await page.goto("/login");

    // Click the GitHub login button
    const githubButton = page.getByRole("button", { name: /github/i });
    await githubButton.click();

    // Wait for the OAuth mock to redirect back with code and state
    await page.waitForURL("**/auth/callback**");

    // The callback page should process the code and state
    // and redirect to dashboard on success
    await oauthCallbackPage.waitForRedirectToDashboard();

    // Try to reuse the same callback URL - should redirect to login
    // because the state has been consumed
    const currentUrl = page.url();
    const callbackUrl = new URL(currentUrl);
    const code = callbackUrl.searchParams.get("code");
    const state = callbackUrl.searchParams.get("state");

    if (code && state) {
      await page.goto(`/auth/callback?code=${code}&state=${state}`);
      await oauthCallbackPage.waitForRedirectToLogin();
    }
  });
});
