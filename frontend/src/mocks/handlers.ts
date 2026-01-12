import { http, HttpResponse } from "msw";

export interface MockPost {
  id: string;
  title: string;
  created_at: string;
}

export interface MockPostsResponse {
  posts: MockPost[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

export interface MockPostKeyResponse {
  post_key: string;
  created_at: string;
}

export const mockPostKey: MockPostKeyResponse = {
  post_key: "test_key_abc123",
  created_at: "2024-01-01T00:00:00Z",
};

export const mockEmptyPosts: MockPostsResponse = {
  posts: [],
  pagination: { page: 1, limit: 20, total: 0, total_pages: 0 },
};

export const mockPosts: MockPostsResponse = {
  posts: [
    { id: "p1", title: "Test Post 1", created_at: "2024-01-01T12:00:00Z" },
    { id: "p2", title: "Test Post 2", created_at: "2024-01-02T13:00:00Z" },
  ],
  pagination: { page: 1, limit: 20, total: 2, total_pages: 1 },
};

export const handlers = [
  http.get("/api/post_key", () => {
    return HttpResponse.json<MockPostKeyResponse>(mockPostKey);
  }),

  http.get("/api/posts", ({ request }) => {
    const url = new URL(request.url);
    const page = url.searchParams.get("page");

    if (page === "2") {
      return HttpResponse.json<MockPostsResponse>({
        posts: [],
        pagination: { page: 2, limit: 20, total: 2, total_pages: 1 },
      });
    }

    return HttpResponse.json<MockPostsResponse>(mockPosts);
  }),

  http.post("/api/post_key", async () => {
    return HttpResponse.json<MockPostKeyResponse>({
      post_key: "new_key_abc123",
      created_at: "2024-01-01T00:00:00Z",
    });
  }),

  http.post("/api/auth/change-password", async () => {
    return HttpResponse.json({});
  }),

  http.post("/:postKey", async () => {
    return HttpResponse.json<{ id: string }>({ id: "new_post_123" });
  }),

  http.post("/api/auth/login", async () => {
    return HttpResponse.json({
      access_token: "test_access_token",
      refresh_token: "test_refresh_token",
      user: { id: 1, username: "testuser" },
    });
  }),

  http.post("/api/auth/refresh", async () => {
    return HttpResponse.json({
      access_token: "refreshed_access_token",
      refresh_token: "refreshed_refresh_token",
      user: { id: 1, username: "testuser" },
    });
  }),
];
