"use client";

import React, { useEffect, useState, useRef, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { authApi } from "@/lib/api/auth";
import { Loader2Icon } from "lucide-react";

const LoginSpinner = () => {
  return (
    <div className="flex justify-center pt-10">
      <div className="flex flex-col items-center gap-2 text-center text-sm text-muted-foreground">
        <Loader2Icon className="size-5 animate-spin" />
        <div>Processing authentication...</div>
      </div>
    </div>
  );
};

const LoginCallbackPage: React.FC = () => {
  const searchParams = useSearchParams();
  const [loading, setLoading] = useState(true);
  const processing = useRef(false);
  const setAuth = useAuthStore((state) => state.setAuth);

  const handleCallback = useCallback(async () => {
    if (processing.current) return;
    processing.current = true;

    let message = "";
    try {
      const code = searchParams.get("code");
      const state = searchParams.get("state");

      if (!code || !state) {
        throw new Error("Missing code or state");
      }

      const data = await authApi.loginWithGitHub(code, state);

      setAuth(data.token, data.user, data.refresh_token);
    } catch (err: unknown) {
      message = err instanceof Error ? err.message : "Unknown error";
    } finally {
      if (window.opener && !window.opener.closed) {
        window.opener.postMessage({ type: "oauth_result", message }, "*");
      }
      setLoading(false);
    }
  }, [searchParams, setAuth]);

  useEffect(() => {
    handleCallback();
  }, [handleCallback]);

  return loading ? <LoginSpinner /> : null;
};

export default LoginCallbackPage;
