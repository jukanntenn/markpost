"use client";

import React, { useEffect, useState, useRef, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { storage, auth } from "@/utils";
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

  const handleCallback = useCallback(async () => {
    if (processing.current) return;
    processing.current = true;

    let message = "";
    try {
      const code = searchParams.get("code");
      const state = searchParams.get("state");
      
      const res = await fetch("/api/oauth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code, state }),
      });

      if (!res.ok) {
        const error = await res.json();
        throw new Error(error.message || "Authentication failed");
      }

      const data = await res.json();

      if (auth.checkLoginResponse(data)) {
        storage.set("login", data);
      } else {
        message = "Authentication error";
      }
    } catch (err: unknown) {
      message = err instanceof Error ? err.message : "Unknown error";
    } finally {
      if (window.opener && !window.opener.closed) {
        window.opener.postMessage({ type: "oauth_result", message }, "*");
      }
      setLoading(false);
    }
  }, [searchParams]);

  useEffect(() => {
    handleCallback();
  }, [handleCallback]);

  return loading ? <LoginSpinner /> : null;
};

export default LoginCallbackPage;
