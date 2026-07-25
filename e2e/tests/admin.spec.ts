import { test, expect } from "../lib/fixtures";

test("unauthenticated user cannot access /admin and is redirected to login", async ({
  page,
}) => {
  await page.goto("/admin");
  await page.waitForURL("**/login");
});

test("admin user can access /admin and is redirected to /admin/users", async ({
  authenticatedPage,
}) => {
  await authenticatedPage.goto("/admin");
  await authenticatedPage.waitForURL("**/admin/users");
});

test("admin navigation links work correctly", async ({
  authenticatedPage: page,
}) => {
  await page.goto("/admin/users");
  await page.waitForURL("**/admin/users");

  // Scope link lookups to the admin sidebar so they don't match the top nav.
  const sidebar = page.locator("aside");

  await sidebar.getByRole("link", { name: "Posts" }).click();
  await page.waitForURL("**/admin/posts");

  await sidebar.getByRole("link", { name: "Channels" }).click();
  await page.waitForURL("**/admin/delivery/channels");

  await sidebar.getByRole("link", { name: "Delivery History" }).click();
  await page.waitForURL("**/admin/delivery/history");

  await sidebar.getByRole("link", { name: "Users" }).click();
  await page.waitForURL("**/admin/users");
});
