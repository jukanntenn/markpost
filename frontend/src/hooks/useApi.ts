import { useState, useCallback } from "react";
import axios from "axios";
import type { AxiosRequestConfig } from "axios";

interface UseApiReturn {
  get: <T>(url: string, config?: AxiosRequestConfig) => Promise<T>;
  post: <T>(url: string, data?: any, config?: AxiosRequestConfig) => Promise<T>;
  loading: boolean;
  error: string | null;
  clearError: () => void;
}

export const useApi = (): UseApiReturn => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  const handleRequest = useCallback(
    async <T>(requestFn: () => Promise<T>): Promise<T> => {
      setLoading(true);
      setError(null);

      try {
        const result = await requestFn();
        return result;
      } catch (err) {
        let errorMessage = "An unexpected error occurred";

        if (axios.isAxiosError(err)) {
          if (err.response?.data?.error) {
            errorMessage = err.response.data.error;
          } else if (err.code === "NETWORK_ERROR") {
            errorMessage = "Network error. Please check your connection.";
          } else if (err.code === "ECONNABORTED") {
            errorMessage = "Request timeout. Please try again.";
          } else {
            errorMessage = err.message || "Request failed";
          }
        } else if (err instanceof Error) {
          errorMessage = err.message;
        }

        setError(errorMessage);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    []
  );

  const get = useCallback(
    async <T>(url: string, config?: AxiosRequestConfig): Promise<T> => {
      return handleRequest(async () => {
        const response = await axios.get<T>(url, config);
        return response.data;
      });
    },
    [handleRequest]
  );

  const post = useCallback(
    async <T>(
      url: string,
      data?: any,
      config?: AxiosRequestConfig
    ): Promise<T> => {
      return handleRequest(async () => {
        const response = await axios.post<T>(url, data, config);
        return response.data;
      });
    },
    [handleRequest]
  );

  return {
    get,
    post,
    loading,
    error,
    clearError,
  };
};
