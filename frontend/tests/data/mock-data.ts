export const mockUsers = {
  tester: {
    id: 1,
    username: "tester",
    access_token: "test_token",
    refresh_token: "test_refresh",
  },
  admin: {
    id: 2,
    username: "admin",
    access_token: "admin_token",
    refresh_token: "admin_refresh",
  },
  e2e: {
    id: 1,
    username: "tester",
    access_token: "e2e_access_token",
    refresh_token: "e2e_refresh_token",
  },
};

export const mockPosts = {
  empty: {
    posts: [],
    pagination: { page: 1, limit: 20, total: 0, total_pages: 0 },
  },
  single: {
    posts: [{ id: "p1", title: "Test Post", created_at: "2024-01-01T00:00:00Z" }],
    pagination: { page: 1, limit: 20, total: 1, total_pages: 1 },
  },
  multiple: {
    posts: [
      { id: "p1", title: "Post One", created_at: "2024-01-01T12:00:00Z" },
      { id: "p2", title: "Post Two", created_at: "2024-01-02T13:00:00Z" },
    ],
    pagination: { page: 1, limit: 20, total: 2, total_pages: 1 },
  },
};

export const mockPostKey = {
  post_key: "test_key_abc123",
  created_at: "2024-01-01T00:00:00Z",
};

export const mockNewPostKey = {
  post_key: "new_key_xyz789",
  created_at: "2024-01-02T00:00:00Z",
};
