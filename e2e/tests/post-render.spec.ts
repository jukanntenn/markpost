import { test, expect } from "../lib/fixtures";
import {
  waitForBackend,
  apiLogin,
  deleteAllPosts,
  getPostKey,
  createPost,
} from "../lib/helpers";

test.beforeEach(async ({ page, request }) => {
  await waitForBackend(request);
  const auth = await apiLogin(request);
  await deleteAllPosts(request, auth.token);
  await page.context().clearCookies();
});

// The public render page is the anti-hotlink surface: every <img> must carry
// referrerpolicy=no-referrer so forum image beds with Referer allowlists
// (which 403 cross-site referrals but allow an empty Referer) still load.
// The raw <img> case also asserts a source-supplied referrerpolicy value is
// replaced, not duplicated.
test("public post page stamps no-referrer on images", async ({
  page,
  request,
}) => {
  const auth = await apiLogin(request);
  const postKey = await getPostKey(request, auth.token);
  const { id } = await createPost(
    request,
    auth.token,
    postKey,
    "No Referrer E2E",
    '![logo](https://img.invalid/e2e.png)\n\n<img src="https://img.invalid/raw.png" referrerpolicy="origin">',
  );

  await page.goto(`/${id}`);

  await expect(page.locator("h1.post-title")).toHaveText("No Referrer E2E");
  const imgs = page.locator(".content img");
  await expect(imgs).toHaveCount(2);
  await expect(imgs.first()).toHaveAttribute("referrerpolicy", "no-referrer");
  await expect(imgs.nth(1)).toHaveAttribute("referrerpolicy", "no-referrer");
});
