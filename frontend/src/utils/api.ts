import { get, remove, set } from "./storage";

import type { LoginResponse } from "../types/auth";
import axios from "axios";
import { getAcceptLanguageHeader } from "../utils/i18n";

import type { InternalAxiosRequestConfig } from "axios";

const addLanguageHeader = (config: InternalAxiosRequestConfig) => {
  const acceptLanguage = getAcceptLanguageHeader();
  config.headers = (config.headers || {}) as InternalAxiosRequestConfig["headers"];
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

auth.interceptors.request.use(
  (config) => {
    config = addLanguageHeader(config);

    try {
      const loginData = get<LoginResponse | null>("login");
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
type PendingItem = { resolve: (token: string) => void; reject: (err: unknown) => void };
let pendingQueue: PendingItem[] = [];

const processQueue = (error: unknown, token: string | null) => {
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
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };
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

type ErrorItem = { field?: string; code?: string; message?: string };
type ErrorResponseData = { code?: string; message?: string; errors?: ErrorItem[]; error?: string };
type AxiosErrorLike = { response?: { data?: unknown } };

function isErrorResponseData(d: unknown): d is ErrorResponseData {
  if (!d || typeof d !== "object") return false;
  const anyObj = d as Record<string, unknown>;
  if (typeof anyObj.message === "string") return true;
  if (typeof anyObj.error === "string") return true;
  if (Array.isArray(anyObj.errors)) return true;
  return false;
}

export function getErrorMessage(error: unknown, fallback: string): string {
  const data = (error as AxiosErrorLike)?.response?.data;
  if (!isErrorResponseData(data)) return fallback;
  if (data.message) return data.message;
  if (Array.isArray(data.errors) && data.errors.length > 0) {
    const first = data.errors[0];
    if (first?.message) return first.message;
  }
  if (data.error) return data.error;
  return fallback;
}
