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
  authenticatedPage,
}) => {
  await authenticatedPage.goto("/admin/users");
  await authenticatedPage.waitForURL("**/admin/users");

  await authenticatedPage.getByRole("link", { name: "Posts" }).click();
  await authenticatedPage.waitForURL("**/admin/posts");

  await authenticatedPage.getByRole("link", { name: "Channels" }).click();
  await authenticatedPage.waitForURL("**/admin/delivery/channels");

  await authenticatedPage.getByRole("link", { name: "Delivery History" }).click();
  await authenticatedPage.waitForURL("**/admin/delivery/history");

  await authenticatedPage.getByRole("link", { name: "Users" }).click();
  await authenticatedPage.waitForURL("**/admin/users");
});
