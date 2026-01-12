import { auth, anno } from "../utils/api";
import { get } from "../utils/storage";
import type { LoginResponse } from "../types/auth";

export const authFetcher = (url: string) => {
  const loginData = get<LoginResponse | null>("login");
  const accessToken = loginData?.access_token;

  if (!accessToken) {
    return Promise.reject(new Error("No access token available"));
  }

  return auth.get(url).then((res) => res.data);
};

export const annoFetcher = (url: string) => anno.get(url).then((res) => res.data);
