"use client";

import { useEffect, useRef } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";
import { authApi } from "@/lib/api";
import { consumeExpectedOAuthState } from "@/hooks/useGitHubOAuth";
import { Spinner } from "@/components/ui/spinner";

// LoginCallbackPage handles the same-page OAuth redirect callback (auth.md §7).
// Flow: parse code/state (or error) from the URL → on error redirect to
// /login → second-layer state check vs sessionStorage → POST /oauth/login →
// setAuth → redirect to /dashboard. All failure paths redirect to /login.
export default function LoginCallbackPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const t = useTranslations("loginCallback");
  const setAuth = useAuthStore((state) => state.setAuth);
  const processing = useRef(false);

  useEffect(() => {
    if (processing.current) return;
    processing.current = true;

    const code = searchParams.get("code");
    const state = searchParams.get("state");
    const error = searchParams.get("error");

    // GitHub returned an error (e.g. user denied authorization).
    if (error) {
      router.replace("/login");
      return;
    }

    // Missing required parameters.
    if (!code || !state) {
      router.replace("/login");
      return;
    }

    // Front-end second-layer state check (backend is the primary defense).
    const expectedState = consumeExpectedOAuthState();
    if (state !== expectedState) {
      router.replace("/login");
      return;
    }

    authApi
      .loginWithGitHub(code, state)
      .then((data) => {
        setAuth(data.token, data.user, data.refresh_token);
        router.replace("/dashboard");
      })
      .catch(() => {
        router.replace("/login");
      });
  }, [searchParams, router, setAuth]);

  return (
    <div className="flex justify-center pt-10">
      <div className="flex flex-col items-center gap-2 text-center text-sm text-muted-foreground">
        <Spinner className="size-5" />
        <div>{t("loading")}</div>
      </div>
    </div>
  );
}
