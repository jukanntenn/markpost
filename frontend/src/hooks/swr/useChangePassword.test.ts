import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { server } from "../../mocks/server";
import { useChangePassword } from "./useChangePassword";
import { setMockAuth, createWrapper } from "../../test/utils";
import { http, HttpResponse } from "msw";

describe("useChangePassword", () => {
  beforeEach(() => {
    setMockAuth({
      access_token: "test_token",
      refresh_token: "test_refresh",
      user: { id: 1, username: "testuser" },
    });
  });

  it("changes password successfully", async () => {
    const { result } = renderHook(() => useChangePassword(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.trigger({
        current_password: "oldpass",
        new_password: "newpass",
      });
    });

    await waitFor(() => {
      expect(result.current.data).toBeDefined();
    });
  });

  it("sets isMutating during request", async () => {
    const { result } = renderHook(() => useChangePassword(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      const promise = result.current.trigger({
        current_password: "oldpass",
        new_password: "newpass",
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
      http.post("/api/auth/change-password", () => {
        return HttpResponse.json(
          { error: "Current password is incorrect" },
          { status: 400 }
        );
      })
    );

    const { result } = renderHook(() => useChangePassword(), {
      wrapper: createWrapper(),
    });

    let hasError = false;
    try {
      await act(async () => {
        await result.current.trigger({
          current_password: "wrongpass",
          new_password: "newpass",
        });
      });
    } catch {
      hasError = true;
    }

    expect(hasError || result.current.error).toBeTruthy();
  });

  it("refreshes token and retries mutation after 401", async () => {
    setMockAuth({
      access_token: "expired_access",
      refresh_token: "refresh_1",
      user: { id: 1, username: "testuser" },
    });

    let refreshCount = 0;
    server.use(
      http.post("/api/auth/refresh", () => {
        refreshCount += 1;
        return HttpResponse.json({
          access_token: "new_access",
          refresh_token: "new_refresh",
          user: { id: 1, username: "testuser" },
        });
      }),
      http.post("/api/auth/change-password", ({ request }) => {
        const auth = request.headers.get("authorization");
        if (auth !== "Bearer new_access") {
          return HttpResponse.json({ message: "Unauthorized" }, { status: 401 });
        }
        return HttpResponse.json({ ok: true });
      })
    );

    const { result } = renderHook(() => useChangePassword(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.trigger({
        current_password: "oldpass",
        new_password: "newpass",
      });
    });

    await waitFor(() => {
      expect(result.current.data).toBeDefined();
    });

    expect(refreshCount).toBe(1);
  });
});
