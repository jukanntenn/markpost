// User interface
export interface User {
  id: number;
  username: string;
  post_key: string;
  github_id: number;
}

// Token pair interface
export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

// GitHub auth URL response interface
export interface GitHubAuthUrlResponse {
  auth_url: string;
  state: string;
}

// Auth response interface
export interface AuthResponse {
  success: boolean;
  user: User;
  tokens: TokenPair;
  message: string;
}

// Error response interface
export interface ErrorResponse {
  error: string;
}

// API response union type
export type ApiResponse<T> = T | ErrorResponse;
