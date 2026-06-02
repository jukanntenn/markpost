"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";
import type { User } from "@/types/auth";
import { authApi } from "@/lib/api";
import { useGitHubOAuth } from "@/hooks/useGitHubOAuth";
import { GithubIcon } from "lucide-react";
import { Spinner } from "@/components/ui/spinner";

import { FormAlert } from "@/components/ui/form-alert";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
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
  const [error, setError] = useState("");
  const [formData, setFormData] = useState({
    username: "",
    password: "",
  });
  const setAuth = useAuthStore((state) => state.setAuth);

  const onOAuthSuccess = (token: string, user: User, refreshToken: string) => {
    setAuth(token, user, refreshToken);
    router.push("/dashboard");
  };

  const { startOAuth: handleGitHubLogin, loading: loadingGitHub } = useGitHubOAuth(onOAuthSuccess);

  function handleLoginError(err: unknown) {
    setError(err instanceof Error ? err.message : String(err));
  }

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

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);

    try {
      const data = await authApi.login(formData.username, formData.password);

      setAuth(data.token, data.user, data.refresh_token);
      router.push("/dashboard");
    } catch (err: unknown) {
      handleLoginError(err);
    } finally {
      setLoading(false);
    }
  }

  const gitHubButtonText = loadingGitHub
    ? t("processingGitHubLogin")
    : t("githubLogin");

  return (
    <div className="flex min-h-svh items-center justify-center p-4">
      <div className="w-full max-w-md">
        <LoginTitle />

        <Card>
          <CardContent className="space-y-4">
            <FormAlert message={error} />

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

              <LoadingButton
                type="submit"
                className="w-full"
                disabled={!formData.username || !formData.password}
                loading={loading}
                loadingText={t("signingIn")}
              >
                {t("loginButton")}
              </LoadingButton>
            </form>

            <LoginDivider />

            <Button
              type="button"
              variant="outline"
              className="w-full"
              onClick={async () => {
                try {
                  await handleGitHubLogin();
                } catch (err: unknown) {
                  handleLoginError(err);
                }
              }}
              disabled={loadingGitHub}
            >
              {loadingGitHub ? (
                <span className="inline-flex items-center gap-2">
                  <Spinner className="size-4" />
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
