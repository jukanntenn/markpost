"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";
import { authApi } from "@/lib/api/auth";
import { toast } from "sonner";
import { GithubIcon, Loader2Icon, TriangleAlertIcon } from "lucide-react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import LoginTitle from "./LoginTitle";
import LoginDivider from "./LoginDivider";

export function LoginPage() {
  const router = useRouter();
  const t = useTranslations("login");
  const tCommon = useTranslations("common");
  const [loading, setLoading] = useState(false);
  const [loadingGitHub, setLoadingGitHub] = useState(false);
  const [error, setError] = useState("");
  const [formData, setFormData] = useState({
    username: "",
    password: "",
  });
  const authWindowRef = useRef<Window | null>(null);
  const checkAuthWindowIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const setAuth = useAuthStore((state) => state.setAuth);

  const showErrorToast = (header: string, body?: string) => {
    toast.error(header, body ? { description: body } : undefined);
  };

  useEffect(() => {
    document.title = tCommon("pageTitle.login");
  }, [tCommon]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));

    setError("");
  };

  const authWindowClosed = () => {
    return !authWindowRef.current || authWindowRef.current.closed;
  };

  const openAuthWindow = (url: string) => {
    const width = 600;
    const height = 700;
    const left = (window.innerWidth - width) / 2 + window.screenX;
    const top = (window.innerHeight - height) / 2 + window.screenY;

    return window.open(
      url,
      "github_oauth",
      `width=${width},height=${height},left=${left},top=${top},resizable=yes,scrollbars=yes,status=yes`
    );
  };

  const closeAuthWindow = useCallback(() => {
    if (!authWindowClosed()) {
      authWindowRef.current!.close();
    }
  }, []);

  const clearCheckAuthWindowInterval = () => {
    if (checkAuthWindowIntervalRef.current) {
      clearInterval(checkAuthWindowIntervalRef.current);
      checkAuthWindowIntervalRef.current = null;
    }
  };

  async function handleGitHubLogin() {
    setLoadingGitHub(true);

    try {
      const data = await authApi.getOAuthUrl();
      const url = data.url;

      authWindowRef.current = openAuthWindow(url);
      if (!authWindowRef.current) {
        showErrorToast(t("cannotOpenAuthWindow"));
        return;
      }

      checkAuthWindowIntervalRef.current = setInterval(() => {
        if (authWindowClosed()) {
          clearCheckAuthWindowInterval();
          setLoadingGitHub(false);
        }
      }, 500);

      const handleMessage = async (event: MessageEvent) => {
        if (event.data?.type === "oauth_result") {
          if (checkAuthWindowIntervalRef.current) {
            clearCheckAuthWindowInterval();
          }

          if (event.data.message == "") {
            setTimeout(() => {
              router.push("/dashboard");
            }, 500);
          } else {
            showErrorToast(t("loginFailed"), event.data.message);
          }

          closeAuthWindow();
          window.removeEventListener("message", handleMessage);
          setLoadingGitHub(false);
        }
      };

      window.addEventListener("message", handleMessage);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t("unknownError");
      showErrorToast(t("loginFailed"), message);
      closeAuthWindow();
      setLoadingGitHub(false);
    }
  }

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);

    try {
      const data = await authApi.login(formData.username, formData.password);

      setAuth(data.token, data.user, data.refresh_token);
      router.push("/dashboard");
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t("unknownError");
      if (err instanceof Error && err.message) {
        setError(message);
      } else {
        showErrorToast(t("loginFailed"), message);
      }
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    return () => {
      clearCheckAuthWindowInterval();
      closeAuthWindow();
    };
  }, [closeAuthWindow]);

  const gitHubButtonText = loadingGitHub
    ? t("processingGitHubLogin")
    : t("githubLogin");

  return (
    <div className="flex min-h-svh items-center justify-center p-4">
      <div className="w-full max-w-md">
        <LoginTitle />

        <Card className="shadow-sm">
          <CardContent className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <TriangleAlertIcon />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <form onSubmit={handleLogin} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="login-username">{t("username")}</Label>
                <Input
                  id="login-username"
                  type="text"
                  name="username"
                  value={formData.username}
                  onChange={handleInputChange}
                  placeholder={t("usernamePlaceholder")}
                  required
                  disabled={loading}
                  autoComplete="username"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="login-password">{t("password")}</Label>
                <Input
                  id="login-password"
                  type="password"
                  name="password"
                  value={formData.password}
                  onChange={handleInputChange}
                  placeholder={t("passwordPlaceholder")}
                  required
                  disabled={loading}
                  autoComplete="current-password"
                />
              </div>

              <Button
                type="submit"
                className="w-full"
                disabled={loading || !formData.username || !formData.password}
              >
                {loading ? (
                  <span className="inline-flex items-center gap-2">
                    <Loader2Icon className="size-4 animate-spin" />
                    {t("signingIn")}
                  </span>
                ) : (
                  t("loginButton")
                )}
              </Button>
            </form>

            <LoginDivider />

            <Button
              type="button"
              variant="outline"
              className="w-full"
              onClick={handleGitHubLogin}
              disabled={loadingGitHub}
            >
              {loadingGitHub ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2Icon className="size-4 animate-spin" />
                  {gitHubButtonText}
                </span>
              ) : (
                <span className="inline-flex items-center gap-2">
                  <GithubIcon className="size-4" />
                  {t("githubLogin")}
                </span>
              )}
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export default LoginPage;
