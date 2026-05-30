import { test, expect } from "../lib/fixtures";

test("redirects to login when unauthenticated", async ({ page }) => {
  await page.goto("/dashboard");
  await page.waitForURL("**/login");
});

test("renders post key masked by default and toggles visibility", async ({
  authenticatedPage,
  dashboardPage,
}) => {
  await dashboardPage.goto();

  await expect(dashboardPage.postKeyHeading).toBeVisible({ timeout: 15000 });
  await expect(dashboardPage.showKeyButton).toBeVisible();

  await dashboardPage.clickShowKey();
  await expect(dashboardPage.hideKeyButton).toBeVisible();
  const postKeyText = await dashboardPage.getPostKeyText();
  await expect(postKeyText).toBeVisible();
  const keyValue = await postKeyText.textContent();

  await dashboardPage.clickHideKey();
  await expect(authenticatedPage.getByText(keyValue!, { exact: true })).not.toBeVisible();
});

test("copies post key and shows temporary success badge", async ({
  authenticatedPage,
  dashboardPage,
}) => {
  await authenticatedPage.addInitScript(() => {
    const nav = window.navigator as Navigator & {
      clipboard?: { writeText?: (text: string) => Promise<void> };
    };
    try {
      const clip = nav.clipboard;
      if (clip && typeof clip.writeText === "function") {
        clip.writeText = () => Promise.resolve();
      } else {
        Object.defineProperty(nav, "clipboard", {
          value: { writeText: () => Promise.resolve() },
          configurable: true,
        });
      }
    } catch {
      Object.defineProperty(nav, "clipboard", {
        value: { writeText: () => Promise.resolve() },
        configurable: true,
      });
    }
  });

  await dashboardPage.goto();
  await expect(dashboardPage.postKeyHeading).toBeVisible({ timeout: 15000 });

  await dashboardPage.clickCopyKey();
  const copiedBadge = await dashboardPage.getCopiedBadge();
  await expect(copiedBadge).toBeVisible();

  await authenticatedPage.waitForTimeout(2200);
  await expect(
    authenticatedPage.getByText("copied to clipboard!", { exact: true })
  ).toHaveCount(0);
});

test("navigates to settings and logs out from user menu", async ({
  authenticatedPage,
  dashboardPage,
  loginPage,
}) => {
  await dashboardPage.goto();
  await expect(dashboardPage.postKeyHeading).toBeVisible({ timeout: 15000 });

  await dashboardPage.clickUserMenu();
  await authenticatedPage.getByText("Settings", { exact: true }).click();
  await authenticatedPage.waitForURL("**/settings");

  await dashboardPage.clickUserMenu();
  await dashboardPage.clickLogout();
  await authenticatedPage.waitForURL("**/login");

  await expect(loginPage.usernameInput).toBeVisible();
});
