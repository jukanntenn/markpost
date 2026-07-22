import { test, expect } from "../lib/fixtures";
import { apiLogin, BACKEND_URL, deleteAllPosts, deleteAllDeliveryChannels, clearWebhooks } from "../lib/helpers";

test("successfully changes password and can login with new password", async ({
  authenticatedPage,
  settingsPage,
  request,
  authToken,
}) => {
  // Clean up before the test
  await deleteAllPosts(request, authToken.token);
  await deleteAllDeliveryChannels(request, authToken.token);
  await clearWebhooks(request);

  await settingsPage.goto();
  await expect(settingsPage.appSettingsHeading).toBeVisible({ timeout: 15000 });

  const newPassword = "newpassword123";

  await settingsPage.fillPasswordForm("markpost", newPassword, newPassword);
  await settingsPage.clickChangePassword();

  const successMsg = await settingsPage.getSuccessMessage();
  await expect(successMsg).toBeVisible();
  await expect(settingsPage.currentPasswordInput).toHaveValue("");
  await expect(settingsPage.newPasswordInput).toHaveValue("");
  await expect(settingsPage.confirmPasswordInput).toHaveValue("");

  // Logout
  await authenticatedPage.evaluate(() => localStorage.clear());
  await authenticatedPage.goto("/login");

  // Re-login with new password to verify it works
  const auth = await apiLogin(request, "markpost", newPassword);
  expect(auth.token).toBeTruthy();
  expect(auth.user.username).toBe("markpost");

  // Reset password back to original via API
  const resetResp = await request.post(`${BACKEND_URL}/api/v1/auth/change-password`, {
    headers: { Authorization: `Bearer ${auth.token}` },
    data: { current_password: newPassword, new_password: "markpost" },
  });
  expect(resetResp.ok()).toBeTruthy();

  // Clean up after the test (using the new token before reset)
  await deleteAllPosts(request, auth.token);
  await deleteAllDeliveryChannels(request, auth.token);
  await clearWebhooks(request);
});
