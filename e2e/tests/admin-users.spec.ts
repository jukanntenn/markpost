import { test, expect } from "../lib/fixtures";

test.beforeEach(async ({ authenticatedPage, adminUsersPage }) => {
  await adminUsersPage.goto();
  await expect(adminUsersPage.heading).toBeVisible({ timeout: 15000 });
});

test("displays user list with table", async ({ adminUsersPage }) => {
  await expect(adminUsersPage.table).toBeVisible();
  await expect(adminUsersPage.tableHeader).toBeVisible();
  await expect(adminUsersPage.tableBody).toBeVisible();
});

test("shows admin user in the list", async ({ adminUsersPage }) => {
  const adminRow = adminUsersPage.getUserRow("markpost");
  await expect(adminRow).toBeVisible();
  await expect(adminRow).toContainText("admin");
});
