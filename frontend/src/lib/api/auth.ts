import { request } from "./base";
import type { PostKeyResponse, LoginResponse, RefreshResponse, OAuthUrlResponse, LogoutResponse } from "@/types/auth";

export const authApi = {
  login: (username: string, password: string) =>
    request<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      json: { username, password },
      skipAuthRefresh: true,
    }),

  loginWithGitHub: (code: string, state: string) =>
    request<LoginResponse>("/api/v1/oauth/login", {
      method: "POST",
      json: { code, state },
      skipAuthRefresh: true,
    }),

  getOAuthUrl: () =>
    request<OAuthUrlResponse>("/api/v1/oauth/url", {
      skipAuthRefresh: true,
    }),

  logout: () =>
    request<LogoutResponse>("/api/v1/auth/logout", {
      method: "POST",
    }),

  refreshToken: (refreshToken: string) =>
    request<RefreshResponse>("/api/v1/auth/refresh", {
      method: "POST",
      json: { refresh_token: refreshToken },
      skipAuthRefresh: true,
    }),

  changePassword: (currentPassword: string, newPassword: string) =>
    request<{ message: string }>("/api/v1/auth/change-password", {
      method: "POST",
      json: {
        current_password: currentPassword,
        new_password: newPassword,
      },
    }),

  queryPostKey: () =>
    request<PostKeyResponse>("/api/v1/post-key"),
};
