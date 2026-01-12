import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import useSWR from "swr";
import { SWRConfig } from "swr";
import { I18nextProvider } from "react-i18next";
import i18n from "../i18n";
import { server } from "../mocks/server";
import { http, HttpResponse } from "msw";
import * as middleware from "./middleware";
import { authFetcher } from "./fetcher";
import { setMockAuth, clearMockAuth } from "../test/utils";
import { auth } from "../utils/api";

function createMiddlewareWrapper() {
  return ({ children }: { children: React.ReactNode }) => (
    <SWRConfig
      value={{
        use: [middleware.authMiddleware],
        fetcher: authFetcher,
        provider: () => new Map(),
        dedupingInterval: 0,
        revalidateOnFocus: false,
        revalidateOnReconnect: false,
        refreshWhenHidden: false,
        refreshInterval: 0,
        shouldRetryOnError: () => false,
      }}
    >
      <I18nextProvider i18n={i18n}>{children}</I18nextProvider>
    </SWRConfig>
  );
}

describe("authMiddleware", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    clearMockAuth();
  });

  it("refreshes token and retries request after 401", async () => {
    setMockAuth({
      access_token: "expired_access",
      refresh_token: "refresh_1",
      user: { id: 1, username: "testuser" },
    });

    let refreshCount = 0;
    server.use(
      http.post("/api/auth/refresh", async () => {
        refreshCount += 1;
        return HttpResponse.json({
          access_token: "new_access",
          refresh_token: "new_refresh",
          user: { id: 1, username: "testuser" },
        });
      }),
      http.get("/api/post_key", ({ request }) => {
        const auth = request.headers.get("authorization");
        if (auth === "Bearer expired_access") {
          return HttpResponse.json({ message: "Unauthorized" }, { status: 401 });
        }
        if (auth === "Bearer new_access") {
          return HttpResponse.json({
            post_key: "test_key_abc123",
            created_at: "2024-01-01T00:00:00Z",
          });
        }
        return HttpResponse.json({ message: "Missing auth" }, { status: 401 });
      })
    );

    const { result } = renderHook(() => useSWR("/api/post_key"), {
      wrapper: createMiddlewareWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toEqual({
        post_key: "test_key_abc123",
        created_at: "2024-01-01T00:00:00Z",
      });
    });

    expect(refreshCount).toBe(1);
    expect(localStorage.getItem("markpost_dev_login")).toBe(
      JSON.stringify({
        access_token: "new_access",
        refresh_token: "new_refresh",
        user: { id: 1, username: "testuser" },
      })
    );
  });

  it("shares a single refresh for concurrent 401 requests", async () => {
    setMockAuth({
      access_token: "expired_access",
      refresh_token: "refresh_1",
      user: { id: 1, username: "testuser" },
    });

    let refreshCount = 0;
    server.use(
      http.post("/api/auth/refresh", async () => {
        refreshCount += 1;
        return HttpResponse.json({
          access_token: "new_access",
          refresh_token: "new_refresh",
          user: { id: 1, username: "testuser" },
        });
      }),
      http.get("/api/a", ({ request }) => {
        const authHeader = request.headers.get("authorization");
        if (authHeader === "Bearer expired_access") {
          return HttpResponse.json({ message: "Unauthorized" }, { status: 401 });
        }
        return HttpResponse.json({ ok: true });
      }),
      http.get("/api/b", ({ request }) => {
        const authHeader = request.headers.get("authorization");
        if (authHeader === "Bearer expired_access") {
          return HttpResponse.json({ message: "Unauthorized" }, { status: 401 });
        }
        return HttpResponse.json({ ok: true });
      })
    );

    await Promise.all([
      middleware.withAuthRefresh(() => auth.get("/api/a")),
      middleware.withAuthRefresh(() => auth.get("/api/b")),
    ]);

    expect(refreshCount).toBe(1);
  });

  it("clears login and redirects when refresh token is missing", async () => {
    setMockAuth({
      access_token: "expired_access",
      refresh_token: "",
      user: { id: 1, username: "testuser" },
    });

    const redirectSpy = vi
      .spyOn(middleware.navigation, "redirectToLogin")
      .mockImplementation(() => {});

    server.use(
      http.get("/api/a", () => {
        return HttpResponse.json({ message: "Unauthorized" }, { status: 401 });
      })
    );

    await expect(middleware.withAuthRefresh(() => auth.get("/api/a"))).rejects.toBeDefined();

    await waitFor(() => {
      expect(redirectSpy).toHaveBeenCalled();
    });

    expect(localStorage.getItem("markpost_dev_login")).toBeNull();
  });
});
