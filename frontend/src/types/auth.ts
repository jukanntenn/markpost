export interface User {
  id: number;
  email: string;
  username: string;
  name?: string;
  avatar_url?: string | null;
}

export interface GitHubAuthUrlResponse {
  auth_url: string;
  state: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  message: string;
}

export interface ErrorResponse {
  error: string;
}

export type ApiResponse<T> = T | ErrorResponse;

export type LoginResponse = {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
};

export type RefreshResponse = {
  access_token: string;
  refresh_token: string;
  expires_in: number;
};

export type OAuthUrlResponse = {
  url: string;
};

export type LogoutResponse = {
  message: string;
};
