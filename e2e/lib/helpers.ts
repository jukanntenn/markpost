import { APIRequestContext } from "@playwright/test";

const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:7330";
const ADMIN_USERNAME = process.env.ADMIN_USERNAME || "markpost";
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || "markpost";

interface AuthResponse {
  user: { id: number; username: string; role: string };
  token: string;
  refresh_token: string;
}

export async function apiLogin(
  request: APIRequestContext,
  username = ADMIN_USERNAME,
  password = ADMIN_PASSWORD,
  retries = 3,
): Promise<AuthResponse> {
  for (let i = 0; i < retries; i++) {
    const resp = await request.post(`${BACKEND_URL}/api/v1/auth/login`, {
      data: { username, password },
    });
    if (resp.ok()) return resp.json();
    if (i === retries - 1) {
      throw new Error(`Login failed: ${resp.status()} ${await resp.text()}`);
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  throw new Error("Login failed: unreachable");
}

export async function waitForBackend(
  request: APIRequestContext,
  timeoutMs = 60000,
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const resp = await request.get(`${BACKEND_URL}/api/v1/health`);
      if (resp.ok()) return;
    } catch {}
    await new Promise((r) => setTimeout(r, 2000));
  }
  throw new Error(`Backend not ready after ${timeoutMs}ms`);
}

export async function createPost(
  request: APIRequestContext,
  token: string,
  postKey: string,
  title: string,
  body: string,
): Promise<{ id: string }> {
  const resp = await request.post(`${BACKEND_URL}/${postKey}`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { title, body },
  });
  if (!resp.ok()) {
    throw new Error(`Create post failed: ${resp.status()} ${await resp.text()}`);
  }
  return resp.json();
}

export { BACKEND_URL, ADMIN_USERNAME, ADMIN_PASSWORD };
