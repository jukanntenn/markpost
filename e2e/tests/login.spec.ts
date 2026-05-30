import { test, expect } from "../lib/fixtures";
import { waitForBackend } from "../lib/helpers";

test.beforeEach(async ({ page, request }) => {
  await waitForBackend(request);
  await page.context().clearCookies();
  await page.goto("/login");
  await page.evaluate(() => localStorage.clear());
});

test("renders login page and enables submit when inputs filled", async ({ loginPage }) => {
  await expect(loginPage.usernameInput).toBeVisible();
  await expect(loginPage.passwordInput).toBeVisible();
  await expect(loginPage.submitButton).toBeDisabled();

  await loginPage.usernameInput.fill("testuser");
  await loginPage.passwordInput.fill("testpass");

  await expect(loginPage.submitButton).toBeEnabled();
});

test("keeps submit disabled when only one field is filled", async ({ loginPage }) => {
  await loginPage.usernameInput.fill("onlyuser");
  await expect(loginPage.submitButton).toBeDisabled();

  await loginPage.usernameInput.fill("");
  await loginPage.passwordInput.fill("onlypass");
  await expect(loginPage.submitButton).toBeDisabled();
});

test("logs in with valid credentials and redirects to dashboard", async ({
  page,
  loginPage,
  dashboardPage,
}) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));

  await loginPage.login("markpost", "markpost");
  await page.waitForURL("**/dashboard", { timeout: 30000 });
  await expect(dashboardPage.postKeyHeading).toBeVisible({ timeout: 15000 });
});

test("shows error on invalid credentials", async ({ page, loginPage }) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));

  await loginPage.login("markpost", "wrongpassword");
  const error = await loginPage.getErrorMessage();
  await expect(error).toBeVisible();
});

test("clears error alert when inputs change", async ({ page, loginPage }) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));

  await loginPage.login("markpost", "wrongpassword");
  const error = await loginPage.getErrorMessage();
  await expect(error).toBeVisible();

  await loginPage.usernameInput.fill("x");
  await expect(loginPage.errorAlert).toHaveCount(0, { timeout: 10000 });
});

test("submits form when pressing Enter in password field", async ({
  page,
  loginPage,
  dashboardPage,
}) => {
  await page.evaluate(() => localStorage.setItem("i18nextLng", "en"));

  await loginPage.usernameInput.fill("markpost");
  await loginPage.passwordInput.fill("markpost");
  await loginPage.submitByPressingEnter();

  await page.waitForURL("**/dashboard", { timeout: 30000 });
  await expect(dashboardPage.postKeyHeading).toBeVisible({ timeout: 15000 });
});

test("redirects to dashboard when already authenticated", async ({
  authenticatedPage,
  dashboardPage,
}) => {
  await authenticatedPage.goto("/dashboard");
  await expect(dashboardPage.postKeyHeading).toBeVisible({ timeout: 15000 });
});

test("displays divider text", async ({ page }) => {
  await expect(page.getByText("or", { exact: true })).toBeVisible();
});
