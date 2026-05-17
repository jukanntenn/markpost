import { http, HttpResponse } from "msw";
import type { PostsPaginatedResponse, PostKeyResponse } from "@/types/posts";

export const mockPostKey: PostKeyResponse = {
  post_key: "test_key_abc123",
  created_at: "2024-01-01T00:00:00Z",
};

export const mockEmptyPosts: PostsPaginatedResponse = {
  posts: [],
  pagination: { page: 1, limit: 20, total: 0, total_pages: 0 },
};

export const mockPosts: PostsPaginatedResponse = {
  posts: [
    { id: 1, qid: "p-qid-1", title: "Test Post 1", created_at: "2024-01-01T12:00:00Z" },
    { id: 2, qid: "p-qid-2", title: "Test Post 2", created_at: "2024-01-02T13:00:00Z" },
  ],
  pagination: { page: 1, limit: 20, total: 2, total_pages: 1 },
};

export const handlers = [
  http.get("/api/v1/post_key", () => {
    return HttpResponse.json<PostKeyResponse>(mockPostKey);
  }),

  http.get("/api/v1/posts", ({ request }) => {
    const url = new URL(request.url);
    const page = url.searchParams.get("page");

    if (page === "2") {
      return HttpResponse.json<PostsPaginatedResponse>({
        posts: [],
        pagination: { page: 2, limit: 20, total: 2, total_pages: 1 },
      });
    }

    return HttpResponse.json<PostsPaginatedResponse>(mockPosts);
  }),

  http.post("/api/v1/post_key", async () => {
    return HttpResponse.json<PostKeyResponse>({
      post_key: "new_key_abc123",
      created_at: "2024-01-01T00:00:00Z",
    });
  }),

  http.post("/api/v1/auth/change-password", async () => {
    return HttpResponse.json({ message: "Password changed successfully" });
  }),

  http.post("/:postKey", async () => {
    return HttpResponse.json<{ id: string }>({ id: "new_post_123" });
  }),

  http.post("/api/v1/auth/login", async () => {
    return HttpResponse.json({
      token: "test_access_token",
      access_token: "test_access_token",
      refresh_token: "test_refresh_token",
      expires_in: 86400,
      user: { id: 1, username: "testuser", email: "test@example.com" },
    });
  }),

  http.post("/api/v1/auth/refresh", async () => {
    return HttpResponse.json({
      token: "refreshed_access_token",
      access_token: "refreshed_access_token",
      refresh_token: "refreshed_refresh_token",
      expires_in: 86400,
    });
  }),

  http.post("/api/v1/auth/logout", async () => {
    return HttpResponse.json({ message: "Logged out successfully" });
  }),

  http.get("/api/v1/oauth/url", async () => {
    return HttpResponse.json({ url: "https://github.com/login/oauth/authorize?mock=true" });
  }),
];
