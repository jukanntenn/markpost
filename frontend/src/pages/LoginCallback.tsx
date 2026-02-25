import React, { useEffect, useState, useRef, useCallback } from "react";
import { useLocation } from "react-router-dom";
import { storage, auth } from "../utils";
import { Loader2Icon } from "lucide-react";
import { anno } from "../utils/api";
import { isAxiosError } from "axios";
import * as api from "../utils/api";
import { useTranslation } from "react-i18next";

const LoginSpinner = () => {
  const { t } = useTranslation();

  return (
    <div className="flex justify-center pt-10">
      <div className="flex flex-col items-center gap-2 text-center text-sm text-muted-foreground">
        <Loader2Icon className="size-5 animate-spin" />
        <div>{t("loginCallback.loading")}</div>
      </div>
    </div>
  );
};

const LoginCallback: React.FC = () => {
  const location = useLocation();
  const [loading, setLoading] = useState(true);
  const processing = useRef(false);
  const { t } = useTranslation();

  const handleCallback = useCallback(async () => {
    if (processing.current) return;
    processing.current = true;

    let message = "";
    try {
      const params = new URLSearchParams(location.search);
      const res = await anno.post(
        "/api/oauth/login",
        { code: params.get("code") },
        {
          params: { state: params.get("state") },
        }
      );

      if (auth.checkLoginResponse(res.data)) {
        storage.set("login", res.data);
      } else {
        message = t("loginCallback.authError");
      }
    } catch (err: unknown) {
      message = isAxiosError(err)
        ? api.getErrorMessage(err, t("loginCallback.authFailed"))
        : t("loginCallback.unknownError");
    } finally {
      if (window.opener && !window.opener.closed) {
        window.opener.postMessage({ type: "oauth_result", message }, "*");
      }
      setLoading(false);
    }
  }, [location.search, t]);

  useEffect(() => {
    // FIXME: 临时解决 strict 模式下触发两次的问题，需要找到更加优雅的解决方式
    handleCallback();
  }, [handleCallback]);

  return loading ? <LoginSpinner /> : "";
};

export default LoginCallback;
