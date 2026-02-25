import { useState, useEffect, useRef, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { anno } from "../utils/api";
import * as api from "../utils/api";
import { storage, auth } from "../utils";
import type { LoginResponse, OAuthUrlResponse } from "../types/auth";
import axios from "axios";
import LoginTitle from "../components/login/LoginTitle";
import LoginDivider from "../components/login/LoginDivider";
import { toast } from "sonner";
import { GithubIcon, Loader2Icon, TriangleAlertIcon } from "lucide-react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

function LoginPage() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [loadingGitHub, setLoadingGitHub] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [formData, setFormData] = useState({
    username: "",
    password: "",
  });
  const authWindowRef = useRef<Window | null>(null);
  const checkAuthWindowIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const showErrorToast = (header: string, body?: string) => {
    toast.error(header, body ? { description: body } : undefined);
  };

  useEffect(() => {
    document.title = t("common.pageTitle.login");
  }, [t]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));

    setError("");
    setSuccess("");
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
      const res = await anno.get<OAuthUrlResponse>("/api/oauth/url");
      const url = res.data.url;

      authWindowRef.current = openAuthWindow(url);
      if (!authWindowRef.current) {
        showErrorToast(t("login.cannotOpenAuthWindow"));
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

          // No news is good news
          if (event.data.message == "") {
            setTimeout(() => {
              window.location.href = "/ui/dashboard";
            }, 500);
          } else {
            showErrorToast(t("login.loginFailed"), event.data.message);
          }

          closeAuthWindow();
          window.removeEventListener("message", handleMessage);
          // no state storage
          setLoadingGitHub(false);
        }
      };

      window.addEventListener("message", handleMessage);
    } catch (err: unknown) {
      if (axios.isAxiosError(err) && err.response?.data?.message) {
        showErrorToast(t("login.loginFailed"), err.response?.data?.message);
      } else {
        showErrorToast(t("login.loginFailed"), t("login.unknownError"));
      }
      closeAuthWindow();
      // no state storage
      setLoadingGitHub(false);
    }
  }

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);

    try {
      const res = await anno.post<LoginResponse>("/api/auth/login", formData);
      const data = res.data;

      if (!auth.checkLoginResponse(data)) {
        showErrorToast(t("login.loginFailed"), t("login.loginError"));
        return;
      }

      storage.set("login", data);

      // FIXME: why navigate does not work here?
      // navigate("/dashboard", { replace: true });
      window.location.href = "/ui/dashboard";
  } catch (err: unknown) {
    if (axios.isAxiosError(err)) {
      const msg = api.getErrorMessage(err, t("login.unknownError"));
      if (err.response) {
        setError(msg);
      } else {
        showErrorToast(t("login.loginFailed"), msg);
      }
    } else {
      showErrorToast(t("login.loginFailed"), t("login.unknownError"));
    }
  } finally {
    setLoading(false);
  }
  }

  // 清理函数
  useEffect(() => {
    return () => {
      clearCheckAuthWindowInterval();
      closeAuthWindow();
    };
  }, [closeAuthWindow]);

  const gitHubButtonText = loadingGitHub
    ? t("login.processingGitHubLogin")
    : t("login.githubLogin");

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
            {success && (
              <Alert>
                <AlertDescription>{success}</AlertDescription>
              </Alert>
            )}

            <form onSubmit={handleLogin} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="login-username">{t("login.username")}</Label>
                <Input
                  id="login-username"
                  type="text"
                  name="username"
                  value={formData.username}
                  onChange={handleInputChange}
                  placeholder={t("login.usernamePlaceholder")}
                  required
                  disabled={loading}
                  autoComplete="username"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="login-password">{t("login.password")}</Label>
                <Input
                  id="login-password"
                  type="password"
                  name="password"
                  value={formData.password}
                  onChange={handleInputChange}
                  placeholder={t("login.passwordPlaceholder")}
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
                    {t("login.signingIn")}
                  </span>
                ) : (
                  t("login.loginButton")
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
                  {t("login.githubLogin")}
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
