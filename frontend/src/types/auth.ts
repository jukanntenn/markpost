export type UserRole = "admin" | "user";

export interface User {
  id: number;
  email: string;
  username: string;
  name?: string;
  avatar_url?: string | null;
  role?: UserRole;
}

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

export interface PostKeyResponse {
  post_key: string;
  created_at: string;
}
