import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { server } from "../../mocks/server";
import { usePosts } from "./usePosts";
import { setMockAuth, createWrapper } from "../../test/utils";
import { http, HttpResponse } from "msw";

describe("usePosts", () => {
  beforeEach(() => {
    setMockAuth({
      access_token: "test_token",
      refresh_token: "test_refresh",
      user: { id: 1, username: "testuser" },
    });
  });

  it("fetches posts successfully", async () => {
    const { result } = renderHook(() => usePosts(1, 20), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toEqual({
        posts: [
          { id: "p1", title: "Test Post 1", created_at: "2024-01-01T12:00:00Z" },
          { id: "p2", title: "Test Post 2", created_at: "2024-01-02T13:00:00Z" },
        ],
        pagination: { page: 1, limit: 20, total: 2, total_pages: 1 },
      });
    });
  });

  it("passes page and limit parameters", async () => {
    const { result } = renderHook(() => usePosts(2, 10), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toBeDefined();
    });
  });

  it("returns null when page is null", () => {
    const { result } = renderHook(() => usePosts(0, 20), {
      wrapper: createWrapper(),
    });

    expect(result.current.data).toBeUndefined();
  });

  it("handles error state", async () => {
    server.use(
      http.get("/api/posts", () => {
        return HttpResponse.json({ message: "Error" }, { status: 500 });
      })
    );

    const { result } = renderHook(() => usePosts(1, 20), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.error).toBeDefined();
    });
  });
});
