import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import AuthService from "../services/authService";
import type { User } from "../types/auth";

interface UseAuthReturn {
  login: () => Promise<void>;
  logout: () => void;
  isAuthenticated: boolean;
  user: User | null;
  loading: boolean;
  error: string | null;
  clearError: () => void;
}

export const useAuth = (): UseAuthReturn => {
  const navigate = useNavigate();
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Initialize auth state on mount
  useEffect(() => {
    const initAuth = () => {
      const authenticated = AuthService.isAuthenticated();
      const currentUser = AuthService.getCurrentUser();

      setIsAuthenticated(authenticated);
      setUser(currentUser as User | null);
    };

    initAuth();
  }, []);

  const login = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);

    try {
      const authUrlResponse = await AuthService.getGitHubAuthUrl();

      // Redirect to GitHub authorization page
      window.location.href = authUrlResponse.auth_url;
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : "Failed to start login process";
      setError(errorMessage);
      setLoading(false);
      console.error("Login error:", err);
    }
  }, []);

  const logout = useCallback((): void => {
    AuthService.logout();
    setIsAuthenticated(false);
    setUser(null);
    setError(null);
    navigate("/login", { replace: true });
  }, [navigate]);

  const clearError = useCallback((): void => {
    setError(null);
  }, []);

  return {
    login,
    logout,
    isAuthenticated,
    user,
    loading,
    error,
    clearError,
  };
};
