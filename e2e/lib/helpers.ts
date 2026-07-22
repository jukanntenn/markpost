import { APIRequestContext } from "@playwright/test";

const BACKEND_URL = process.env.BACKEND_URL || process.env.BASE_URL || "https://localhost:2053";
const ADMIN_USERNAME = process.env.ADMIN_USERNAME || "markpost";
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || "markpost";
const WEBHOOK_MOCK_URL = process.env.WEBHOOK_MOCK_URL || "http://localhost:3002";
const OAUTH_MOCK_URL = process.env.OAUTH_MOCK_URL || "http://localhost:3001";

interface AuthResponse {
  user: { id: number; username: string; role: string };
  token: string;
  refresh_token: string;
}

export async function apiLogin(
  request: APIRequestContext,
  username = ADMIN_USERNAME,
  password = ADMIN_PASSWORD,
  retries = 5,
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
  timeoutMs = 120000,
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  let lastError: unknown;
  while (Date.now() < deadline) {
    try {
      const resp = await request.get(`${BACKEND_URL}/api/v1/health`);
      if (resp.ok()) return;
      lastError = new Error(`Health check returned ${resp.status()}: ${await resp.text()}`);
    } catch (e) {
      lastError = e;
      console.log(`Backend not ready yet: ${e instanceof Error ? e.message : String(e)}`);
    }
    await new Promise((r) => setTimeout(r, 2000));
  }
  throw new Error(`Backend not ready after ${timeoutMs}ms. Last error: ${lastError instanceof Error ? lastError.message : String(lastError)}`);
}

export async function getPostKey(
  request: APIRequestContext,
  token: string,
): Promise<string> {
  const resp = await request.get(`${BACKEND_URL}/api/v1/post-key`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!resp.ok()) {
    throw new Error(`Get post key failed: ${resp.status()} ${await resp.text()}`);
  }
  const data = await resp.json();
  return data.post_key;
}

export async function createPost(
  request: APIRequestContext,
  token: string,
  postKey: string,
  title: string,
  body: string,
): Promise<{ id: string; qid: string }> {
  const resp = await request.post(`${BACKEND_URL}/${postKey}`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { title, body },
  });
  if (!resp.ok()) {
    throw new Error(`Create post failed: ${resp.status()} ${await resp.text()}`);
  }
  return resp.json();
}

export async function createDeliveryChannel(
  request: APIRequestContext,
  token: string,
  data: { name: string; kind: string; configuration: Record<string, string>; keywords?: string },
): Promise<{ id: number }> {
  const resp = await request.post(`${BACKEND_URL}/api/v1/delivery/channels`, {
    headers: { Authorization: `Bearer ${token}` },
    data,
  });
  if (!resp.ok()) {
    throw new Error(`Create channel failed: ${resp.status()} ${await resp.text()}`);
  }
  return resp.json();
}

export async function clearWebhooks(request: APIRequestContext): Promise<void> {
  await request.post(`${WEBHOOK_MOCK_URL}/webhooks/clear`);
}

export async function getWebhooks(request: APIRequestContext): Promise<unknown[]> {
  const resp = await request.get(`${WEBHOOK_MOCK_URL}/webhooks`);
  return resp.json();
}

export async function clearOAuthRequests(request: APIRequestContext): Promise<void> {
  await request.post(`${OAUTH_MOCK_URL}/requests/clear`);
}

export async function getOAuthRequests(request: APIRequestContext): Promise<unknown[]> {
  const resp = await request.get(`${OAUTH_MOCK_URL}/requests`);
  return resp.json();
}

export async function deleteAllPosts(request: APIRequestContext, token: string): Promise<void> {
  const resp = await request.get(`${BACKEND_URL}/api/v1/posts`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!resp.ok()) return;
  const data = await resp.json();
  const posts = data.items || [];
  for (const post of posts) {
    await request.delete(`${BACKEND_URL}/api/v1/posts/${post.qid}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  }
}

export async function deleteAllDeliveryChannels(request: APIRequestContext, token: string): Promise<void> {
  const resp = await request.get(`${BACKEND_URL}/api/v1/delivery/channels`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (resp.ok()) {
    const data = await resp.json();
    const channels = data.items || [];
    for (const channel of channels) {
      await request.delete(`${BACKEND_URL}/api/v1/delivery/channels/${channel.id}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
    }
  }
}

export { BACKEND_URL, ADMIN_USERNAME, ADMIN_PASSWORD, WEBHOOK_MOCK_URL, OAUTH_MOCK_URL };
