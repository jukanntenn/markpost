export interface User {
  id: number;
  email: string;
  username: string;
  name?: string;
  avatar_url?: string | null;
  role?: string;
}

export interface GitHubAuthUrlResponse {
  auth_url: string;
  state: string;
}

export interface AuthResponse {
  user: User;
  token: string;
  refresh_token: string;
  expires_in: number;
}

export interface ErrorResponse {
  error: string;
}

export type ApiResponse<T> = T | ErrorResponse;

export type LoginResponse = {
  token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
};

export type RefreshResponse = {
  token: string;
  refresh_token: string;
  expires_in: number;
};

export type OAuthUrlResponse = {
  url: string;
};

export type LogoutResponse = {
  message: string;
};
