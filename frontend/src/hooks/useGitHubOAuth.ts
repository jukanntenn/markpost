import { useState, useEffect, useRef, useCallback } from "react";
import type { User } from "@/types/auth";
import { authApi } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";

interface UseGitHubOAuthReturn {
  startOAuth: () => Promise<void>;
  loading: boolean;
}

function openAuthWindow(url: string): Window | null {
  const width = 600;
  const height = 700;
  const left = (window.innerWidth - width) / 2 + window.screenX;
  const top = (window.innerHeight - height) / 2 + window.screenY;

  return window.open(
    url,
    "github_oauth",
    `width=${width},height=${height},left=${left},top=${top},resizable=yes,scrollbars=yes,status=yes`
  );
}

export function useGitHubOAuth(
  onSuccess: (token: string, user: User, refreshToken: string) => void
): UseGitHubOAuthReturn {
  const [loading, setLoading] = useState(false);
  const authWindowRef = useRef<Window | null>(null);
  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const closeAuthWindow = useCallback(() => {
    if (authWindowRef.current && !authWindowRef.current.closed) {
      authWindowRef.current.close();
    }
  }, []);

  const clearPollInterval = useCallback(() => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
  }, []);

  const startOAuth = async () => {
    setLoading(true);

    try {
      const { url } = await authApi.getOAuthUrl();

      authWindowRef.current = openAuthWindow(url);
      if (!authWindowRef.current) {
        throw new Error("Cannot open auth window");
      }

      pollIntervalRef.current = setInterval(() => {
        const { token, user, refreshToken } = useAuthStore.getState();
        if (token && user && refreshToken) {
          clearPollInterval();
          authWindowRef.current?.close();
          onSuccess(token, user, refreshToken);
          return;
        }
        if (!authWindowRef.current || authWindowRef.current.closed) {
          setLoading(false);
          clearPollInterval();
        }
      }, 500);
    } catch (err) {
      closeAuthWindow();
      setLoading(false);
      throw err;
    }
  };

  useEffect(() => {
    return () => {
      clearPollInterval();
      closeAuthWindow();
    };
  }, [closeAuthWindow, clearPollInterval]);

  return { startOAuth, loading };
}