import { test, expect } from "../lib/fixtures";
import { waitForBackend, apiLogin, BACKEND_URL, ADMIN_USERNAME } from "../lib/helpers";

test("backend reaches postgres over the shared unix-socket volume", async ({ request }) => {
  await waitForBackend(request);

  const resp = await request.get(`${BACKEND_URL}/api/v1/health`);
  expect(resp.ok()).toBeTruthy();
  const body = await resp.json();
  expect(body.status).toBe("ok");
});

test("admin login round-trips the socket-backed database", async ({ request }) => {
  await waitForBackend(request);

  const auth = await apiLogin(request);
  expect(auth.token).toBeTruthy();
  expect(auth.refresh_token).toBeTruthy();
  expect(auth.user.username).toBe(ADMIN_USERNAME);
});
