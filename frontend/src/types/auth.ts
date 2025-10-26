// User interface
export interface User {
  id: number;
  username: string;
}

// GitHub auth URL response interface
export interface GitHubAuthUrlResponse {
  auth_url: string;
  state: string;
}

// Auth response interface
export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  message: string;
}

// Error response interface
export interface ErrorResponse {
  error: string;
}

// API response union type
export type ApiResponse<T> = T | ErrorResponse;

export type LoginResponse = {
  access_token: string;
  refresh_token: string;
  user: User;
};

export type OAuthUrlResponse = {
  url: string;
};
