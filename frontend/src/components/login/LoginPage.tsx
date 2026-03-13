"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useRouter } from "next/navigation";
import { storage, auth } from "@/utils";
import type { LoginResponse, OAuthUrlResponse } from "@/types/auth";
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
    document.title = "Login - Markpost";
  }, []);

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
      const res = await fetch("/api/oauth/url");
      if (!res.ok) throw new Error("Failed to get OAuth URL");
      const data = await res.json() as OAuthUrlResponse;
      const url = data.url;

      authWindowRef.current = openAuthWindow(url);
      if (!authWindowRef.current) {
        showErrorToast("Cannot open authentication window");
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
            showErrorToast("Login failed", event.data.message);
          }

          closeAuthWindow();
          window.removeEventListener("message", handleMessage);
          setLoadingGitHub(false);
        }
      };

      window.addEventListener("message", handleMessage);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Unknown error";
      showErrorToast("Login failed", message);
      closeAuthWindow();
      setLoadingGitHub(false);
    }
  }

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);

    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });

      if (!res.ok) {
        const error = await res.json();
        throw new Error(error.message || "Login failed");
      }

      const data = await res.json() as LoginResponse;

      if (!auth.checkLoginResponse(data)) {
        showErrorToast("Login failed", "Login error");
        return;
      }

      storage.set("login", data);
      router.push("/dashboard");
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Unknown error";
      if (err instanceof Error && err.message) {
        setError(message);
      } else {
        showErrorToast("Login failed", message);
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
    ? "Processing GitHub login..."
    : "Login with GitHub";

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
                <Label htmlFor="login-username">Username</Label>
                <Input
                  id="login-username"
                  type="text"
                  name="username"
                  value={formData.username}
                  onChange={handleInputChange}
                  placeholder="Enter your username"
                  required
                  disabled={loading}
                  autoComplete="username"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="login-password">Password</Label>
                <Input
                  id="login-password"
                  type="password"
                  name="password"
                  value={formData.password}
                  onChange={handleInputChange}
                  placeholder="Enter your password"
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
                    Signing in...
                  </span>
                ) : (
                  "Sign in"
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
                  Login with GitHub
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
