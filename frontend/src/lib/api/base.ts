import { useAuthStore } from "@/stores/auth";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "";

interface RequestOptions extends RequestInit {
  skipAuthRefresh?: boolean;
}

let isRefreshing = false;
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
  if (isRefreshing && refreshPromise) {
    return refreshPromise;
  }

  isRefreshing = true;
  refreshPromise = refreshAccessToken().finally(() => {
    isRefreshing = false;
    refreshPromise = null;
  });

  return refreshPromise;
}

export async function request<T>(
  url: string,
  options: RequestOptions = {}
): Promise<T> {
  const { token, logout } = useAuthStore.getState();
  const { skipAuthRefresh = false, ...fetchOptions } = options;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string> || {}),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE_URL}${url}`, {
    ...fetchOptions,
    headers,
  });

  if (response.status === 401 && !skipAuthRefresh) {
    const refreshed = await handleTokenRefresh();

    if (refreshed) {
      const newToken = useAuthStore.getState().token;
      headers["Authorization"] = `Bearer ${newToken}`;

      const retryResponse = await fetch(`${API_BASE_URL}${url}`, {
        ...fetchOptions,
        headers,
      });

      if (!retryResponse.ok) {
        const error = await retryResponse.json();
        throw new Error(error.message || "Request failed");
      }

      return retryResponse.json();
    } else {
      logout();
      throw new Error("Session expired");
    }
  }

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || "Request failed");
  }

  return response.json();
}
