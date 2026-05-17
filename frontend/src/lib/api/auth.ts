import { request } from "./base";
import type { User } from "@/stores/auth";
import type { PostKeyResponse } from "@/types/posts";

export interface LoginResponse {
  token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
}

export interface RefreshResponse {
  token: string;
  refresh_token: string;
  expires_in: number;
}

export interface OAuthUrlResponse {
  url: string;
}

export interface LogoutResponse {
  message: string;
}

export const authApi = {
  login: (username: string, password: string) =>
    request<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
      skipAuthRefresh: true,
    }),

  loginWithGitHub: (code: string, state: string) =>
    request<LoginResponse>("/api/v1/oauth/login", {
      method: "POST",
      body: JSON.stringify({ code, state }),
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
      body: JSON.stringify({ refresh_token: refreshToken }),
      skipAuthRefresh: true,
    }),

  changePassword: (currentPassword: string, newPassword: string) =>
    request<{ message: string }>("/api/v1/auth/change-password", {
      method: "POST",
      body: JSON.stringify({
        current_password: currentPassword,
        new_password: newPassword,
      }),
    }),

  queryPostKey: () =>
    request<PostKeyResponse>("/api/v1/post_key"),
};
