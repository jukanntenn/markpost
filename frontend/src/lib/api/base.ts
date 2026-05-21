import { authApi } from "./auth";
import { useAuthStore } from "@/stores/auth";
import type { FieldError, ApiErrorResponse } from "@/types/api";

export class ApiError extends Error {
  readonly code?: string;
  readonly fieldErrors?: FieldError[];

  constructor(response: ApiErrorResponse) {
    super(response.message || "Request failed");
    this.name = "ApiError";
    this.code = response.code;
    this.fieldErrors = response.errors;
  }
}

interface RequestOptions extends Omit<RequestInit, "headers"> {
  skipAuthRefresh?: boolean;
  params?: Record<string, string | number>;
  json?: unknown;
  headers?: Record<string, string>;
}

let refreshPromise: Promise<boolean> | null = null;

async function refreshAccessToken(): Promise<boolean> {
  const { refreshToken, setTokens, logout } = useAuthStore.getState();

  if (!refreshToken) {
    logout();
    return false;
  }

  try {
    const data = await authApi.refreshToken(refreshToken);
    setTokens(data.token, data.refresh_token);
    return true;
  } catch {
    logout();
    return false;
  }
}

async function handleTokenRefresh(): Promise<boolean> {
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = refreshAccessToken().finally(() => {
    refreshPromise = null;
  });

  return refreshPromise;
}

export function paginationParams(page?: number, limit?: number): Record<string, string | number> {
  const params: Record<string, string | number> = {};
  if (page != null) params.page = page;
  if (limit != null) params.limit = limit;
  return params;
}

export function buildUrl(base: string, path: string, params?: Record<string, string | number>): string {
  const normalizedBase = base.endsWith('/') ? base.slice(0, -1) : base;
  if (!params || Object.keys(params).length === 0) {
    return `${normalizedBase}${path}`;
  }
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    searchParams.set(key, String(value));
  }
  return `${normalizedBase}${path}?${searchParams}`;
}

async function throwApiError(response: Response): Promise<never> {
  let body: ApiErrorResponse;
  try {
    body = await response.json();
  } catch {
    body = {
      message: response.statusText || `Request failed with status ${response.status}`,
    };
  }
  throw new ApiError(body);
}

async function parseResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    await throwApiError(response);
  }
  return response.json();
}

async function attemptRetry<T>(
  response: Response,
  skipRefresh: boolean,
  retry: () => Promise<Response>,
): Promise<T | undefined> {
  if (response.status !== 401 || skipRefresh) return undefined;
  const refreshed = await handleTokenRefresh();
  if (!refreshed) throw new Error("Session expired");
  return parseResponse<T>(await retry());
}

export async function request<T>(
  url: string,
  options: RequestOptions = {}
): Promise<T> {
  const { token } = useAuthStore.getState();
  const { skipAuthRefresh = false, params, json, headers: optHeaders, ...fetchOptions } = options;

  const fullUrl = buildUrl(process.env.NEXT_PUBLIC_API_URL || "", url, params);

  const headers: Record<string, string> = { ...optHeaders };

  if (json !== undefined) {
    fetchOptions.body = JSON.stringify(json);
    headers["Content-Type"] = "application/json";
  }

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(fullUrl, {
    ...fetchOptions,
    headers,
  });

  const retried = await attemptRetry<T>(response, skipAuthRefresh, () => {
    headers["Authorization"] = `Bearer ${useAuthStore.getState().token}`;
    return fetch(fullUrl, { ...fetchOptions, headers });
  });
  if (retried !== undefined) return retried;

  return parseResponse<T>(response);
}
