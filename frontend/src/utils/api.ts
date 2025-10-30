import { get, remove, set } from "./storage";

import type { CreateTestPostResponse } from "../types/posts";
import type { LoginResponse } from "../types/auth";
import axios from "axios";
import { getAcceptLanguageHeader } from "../utils/i18n";

const addLanguageHeader = (config: any) => {
  const acceptLanguage = getAcceptLanguageHeader();
  config.headers = config.headers || {};
  config.headers["Accept-Language"] = acceptLanguage;
  return config;
};

export const anno = axios.create({
  baseURL: import.meta.env.VITE_BASE_URL || "/",
  timeout: 10000,
});

anno.interceptors.request.use(
  (config) => {
    return addLanguageHeader(config);
  },
  (error) => {
    return Promise.reject(error);
  }
);

export const auth = axios.create({
  baseURL: import.meta.env.VITE_BASE_URL || "/",
  timeout: 10000,
});

// Add request interceptor to include JWT token
auth.interceptors.request.use(
  (config) => {
    config = addLanguageHeader(config);

    try {
      const loginData = get<any>("login");
      const accessToken = loginData?.access_token;

      if (accessToken) {
        config.headers.Authorization = `Bearer ${accessToken}`;
      }
    } catch (error) {
      console.error("Error reading login data from storage:", error);
    }

    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

let isRefreshing = false;
let pendingQueue: Array<{
  resolve: (token: string) => void;
  reject: (err: any) => void;
}> = [];

const processQueue = (error: any, token: string | null) => {
  pendingQueue.forEach(({ resolve, reject }) => {
    if (error) {
      reject(error);
    } else if (token) {
      resolve(token);
    }
  });
  pendingQueue = [];
};

auth.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config as any;
    originalRequest.headers = originalRequest.headers || {};

    if (!error.response) {
      return Promise.reject(error);
    }

    const status = error.response.status;

    if (status !== 401) {
      return Promise.reject(error);
    }

    const loginData = get<LoginResponse | null>("login");
    const refreshToken = loginData?.refresh_token;

    if (!refreshToken) {
      remove("login");
      window.location.href = "/ui/login";
      return Promise.reject(error);
    }

    if (originalRequest._retry) {
      remove("login");
      window.location.href = "/ui/login";
      return Promise.reject(error);
    }

    originalRequest._retry = true;

    if (isRefreshing) {
      return new Promise((resolve, reject) => {
        pendingQueue.push({ resolve, reject });
      })
        .then((token) => {
          originalRequest.headers.Authorization = `Bearer ${token}`;
          return auth.request(originalRequest);
        })
        .catch((err) => {
          return Promise.reject(err);
        });
    }

    isRefreshing = true;

    try {
      const res = await anno.post<LoginResponse>("/api/auth/refresh", {
        refresh_token: refreshToken,
      });
      const data = res.data;

      if (!data?.access_token || !data?.refresh_token) {
        throw new Error("invalid refresh response");
      }

      set("login", data);

      processQueue(null, data.access_token);

      originalRequest.headers.Authorization = `Bearer ${data.access_token}`;
      return auth.request(originalRequest);
    } catch (refreshErr) {
      processQueue(refreshErr, null);
      remove("login");
      window.location.href = "/ui/login";
      return Promise.reject(refreshErr);
    } finally {
      isRefreshing = false;
    }
  }
);

export async function createTestPost(postKey: string, title: string, body: string): Promise<string> {
  const res = await anno.post<CreateTestPostResponse>(`/${postKey}`, { title, body });
  return res.data.id;
}
