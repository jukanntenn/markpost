import { useAuthStore } from "@/stores/auth";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "";

interface FieldError {
  field?: string;
  code: string;
  message: string;
}

interface ApiErrorResponse {
  code?: string;
  message?: string;
  errors?: FieldError[];
}

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

interface RequestOptions extends RequestInit {
  skipAuthRefresh?: boolean;
  params?: Record<string, string | number>;
}

let refreshPromise: Promise<boolean> | null = null;

async function refreshAccessToken(): Promise<boolean> {
  const { refreshToken, setTokens, logout } = useAuthStore.getState();

  if (!refreshToken) {
    logout();
    return false;
  }

  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!response.ok) {
      logout();
      return false;
    }

    const data = await response.json();
    setTokens(data.token, data.refresh_token);
    return true;
  } catch {
    logout();
    return false;
  }
}

export async function handleTokenRefresh(): Promise<boolean> {
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = refreshAccessToken().finally(() => {
    refreshPromise = null;
  });

  return refreshPromise;
}

export function buildUrl(base: string, path: string, params?: Record<string, string | number>): string {
  if (!params || Object.keys(params).length === 0) {
    return `${base}${path}`;
  }
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    searchParams.set(key, String(value));
  }
  return `${base}${path}?${searchParams}`;
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
  const { skipAuthRefresh = false, params, headers: optHeaders, ...fetchOptions } = options;

  const fullUrl = buildUrl(API_BASE_URL, url, params);

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(optHeaders as Record<string, string> || {}),
  };

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
