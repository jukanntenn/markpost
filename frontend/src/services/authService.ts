import axios from "axios";
import type { GitHubAuthUrlResponse, AuthResponse } from "../types/auth";

// Create axios instance with default configuration
const apiClient = axios.create({
  baseURL: "", // Use relative paths for proxy
  timeout: 10000,
  headers: {
    "Content-Type": "application/json",
  },
});

// Request interceptor to add auth token
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem("auth_token");
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor to handle errors
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Clear auth data on 401 error
      localStorage.removeItem("auth_token");
      localStorage.removeItem("auth_user");
    }
    return Promise.reject(error);
  }
);

export class AuthService {
  /**
   * Get GitHub authorization URL
   */
  static async getGitHubAuthUrl(): Promise<GitHubAuthUrlResponse> {
    try {
      const response = await apiClient.get<GitHubAuthUrlResponse>(
        "/api/oauth/url"
      );
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error)) {
        if (error.response?.data?.error) {
          throw new Error(error.response.data.error);
        }
        throw new Error("Failed to get GitHub authorization URL");
      }
      throw error;
    }
  }

  /**
   * Handle GitHub callback with authorization code
   */
  static async handleGitHubCallback(code: string): Promise<AuthResponse> {
    try {
      const response = await apiClient.get<AuthResponse>("/api/oauth/login", {
        params: { code },
      });
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error)) {
        if (error.response?.data?.error) {
          throw new Error(error.response.data.error);
        }
        throw new Error("Failed to complete GitHub authentication");
      }
      throw error;
    }
  }

  /**
   * Check if user is authenticated
   */
  static isAuthenticated(): boolean {
    const token = localStorage.getItem("auth_token");
    if (!token) return false;

    try {
      // Basic JWT token validation
      const payload = JSON.parse(atob(token.split(".")[1]));
      const currentTime = Math.floor(Date.now() / 1000);
      return payload.exp > currentTime;
    } catch {
      return false;
    }
  }

  /**
   * Get current user from localStorage
   */
  static getCurrentUser(): unknown | null {
    try {
      const userStr = localStorage.getItem("auth_user");
      return userStr ? JSON.parse(userStr) : null;
    } catch (error) {
      return null;
    }
  }

  /**
   * Logout user
   */
  static logout(): void {
    localStorage.removeItem("auth_token");
    localStorage.removeItem("auth_user");
  }
}

export default AuthService;
