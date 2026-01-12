import { get } from "./storage";

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

    const loginData = get<LoginResponse | null>("login");
    const accessToken = loginData?.access_token;

    if (accessToken) {
      config.headers.Authorization = `Bearer ${accessToken}`;
    }

    return config;
  },
  (error) => {
    return Promise.reject(error);
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
