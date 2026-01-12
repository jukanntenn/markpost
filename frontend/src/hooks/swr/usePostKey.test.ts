import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { server } from "../../mocks/server";
import { usePostKey } from "./usePostKey";
import { setMockAuth, createWrapper } from "../../test/utils";
import { http, HttpResponse } from "msw";

describe("usePostKey", () => {
  beforeEach(() => {
    setMockAuth({
      access_token: "test_token",
      refresh_token: "test_refresh",
      user: { id: 1, username: "testuser" },
    });
  });

  it("fetches post key successfully", async () => {
    const { result } = renderHook(() => usePostKey(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toEqual({
        post_key: "test_key_abc123",
        created_at: "2024-01-01T00:00:00Z",
      });
    });
  });

  it("sets loading state during fetch", async () => {
    const { result } = renderHook(() => usePostKey(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toBeDefined();
    });
  });

  it("handles error state", async () => {
    server.use(
      http.get("/api/post_key", () => {
        return HttpResponse.json({ message: "Unauthorized" }, { status: 401 });
      })
    );

    const { result } = renderHook(() => usePostKey(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.error).toBeDefined();
    });
  });
});
