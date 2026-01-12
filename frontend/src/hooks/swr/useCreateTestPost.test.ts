import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { server } from "../../mocks/server";
import { useCreateTestPost } from "./useCreateTestPost";
import { setMockAuth, createWrapper } from "../../test/utils";
import { http, HttpResponse } from "msw";

describe("useCreateTestPost", () => {
  const postKey = "test_key_abc123";

  beforeEach(() => {
    setMockAuth({
      access_token: "test_token",
      refresh_token: "test_refresh",
      user: { id: 1, username: "testuser" },
    });
  });

  it("creates test post successfully", async () => {
    const { result } = renderHook(() => useCreateTestPost(postKey), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.trigger({
        title: "Test Title",
        body: "Test Body",
      });
    });

    await waitFor(() => {
      expect(result.current.data).toEqual({ id: "new_post_123" });
    });
  });

  it("does not fetch when postKey is empty", () => {
    const { result } = renderHook(() => useCreateTestPost(""), {
      wrapper: createWrapper(),
    });

    expect(result.current.data).toBeUndefined();
  });

  it("sets isMutating during request", async () => {
    const { result } = renderHook(() => useCreateTestPost(postKey), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      const promise = result.current.trigger({
        title: "Test Title",
        body: "Test Body",
      });

      expect(result.current.isMutating).toBe(true);

      await promise;
    });

    await waitFor(() => {
      expect(result.current.isMutating).toBe(false);
    });
  });

  it("handles error response", async () => {
    server.use(
      http.post("/:postKey", () => {
        return HttpResponse.json(
          { error: "Failed to create post" },
          { status: 400 }
        );
      })
    );

    const { result } = renderHook(() => useCreateTestPost(postKey), {
      wrapper: createWrapper(),
    });

    let hasError = false;
    try {
      await act(async () => {
        await result.current.trigger({
          title: "Test Title",
          body: "Test Body",
        });
      });
    } catch {
      hasError = true;
    }

    expect(hasError || result.current.error).toBeTruthy();
  });
});
