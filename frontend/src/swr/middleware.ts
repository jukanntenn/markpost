import type { Middleware } from "swr";
import { get, set, remove } from "../utils/storage";
import type { LoginResponse } from "../types/auth";
import { anno } from "../utils/api";

type ErrorWithStatus = { status?: number };
type ErrorWithResponseStatus = { response?: { status?: number } };

const getStatus = (err: unknown): number | undefined => {
  const status = (err as ErrorWithStatus | undefined)?.status;
  if (typeof status === "number") return status;
  const responseStatus = (err as ErrorWithResponseStatus | undefined)?.response?.status;
  if (typeof responseStatus === "number") return responseStatus;
  return undefined;
};

let refreshPromise: Promise<string> | null = null;

export const navigation = {
  redirectToLogin: () => window.location.assign("/ui/login"),
};

export async function refreshAccessToken(): Promise<string> {
  if (refreshPromise) return refreshPromise;

  refreshPromise = (async () => {
    const loginData = get<LoginResponse | null>("login");
    const refreshToken = loginData?.refresh_token;

    if (!refreshToken) {
      remove("login");
      navigation.redirectToLogin();
      throw new Error("missing refresh token");
    }

    try {
      const res = await anno.post<LoginResponse>("/api/auth/refresh", {
        refresh_token: refreshToken,
      });
      const data = res.data;
      if (!data?.access_token || !data?.refresh_token) {
        throw new Error("invalid refresh response");
      }
      set("login", data);
      return data.access_token;
    } catch (err) {
      remove("login");
      navigation.redirectToLogin();
      throw err;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

export async function withAuthRefresh<T>(fn: () => Promise<T> | T): Promise<T> {
  try {
    return await Promise.resolve(fn());
  } catch (err) {
    if (getStatus(err) !== 401) throw err;
    await refreshAccessToken();
    return await Promise.resolve(fn());
  }
}

export const authMiddleware: Middleware =
  (useSWRNext) => (key, fetcher, config) => {
    const resolvedFetcher = fetcher ?? config.fetcher;
    const wrappedFetcher = resolvedFetcher
      ? (...args: Parameters<typeof resolvedFetcher>) =>
          withAuthRefresh(() => resolvedFetcher(...args))
      : null;

    return useSWRNext(key, wrappedFetcher, config);
  };
