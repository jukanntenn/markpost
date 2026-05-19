"use client";

import { useEffect, useState, useRef, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";
import { authApi } from "@/lib/api";
import { Spinner } from "@/components/ui/spinner";

export default function LoginCallbackPage() {
  const searchParams = useSearchParams();
  const t = useTranslations("loginCallback");
  const [loading, setLoading] = useState(true);
  const processing = useRef(false);
  const setAuth = useAuthStore((state) => state.setAuth);

  const handleCallback = useCallback(async () => {
    if (processing.current) return;
    processing.current = true;

    try {
      const code = searchParams.get("code");
      const state = searchParams.get("state");

      if (!code || !state) {
        throw new Error("Missing code or state");
      }

      const data = await authApi.loginWithGitHub(code, state);

      setAuth(data.token, data.user, data.refresh_token);
      window.close();
    } catch {
      setLoading(false);
    }
  }, [searchParams, setAuth]);

  useEffect(() => {
    handleCallback();
  }, [handleCallback]);

  if (!loading) return null;
  return (
    <div className="flex justify-center pt-10">
      <div className="flex flex-col items-center gap-2 text-center text-sm text-muted-foreground">
        <Spinner className="size-5" />
        <div>{t("loading")}</div>
      </div>
    </div>
  );
}
