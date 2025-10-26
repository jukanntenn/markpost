import React, { useEffect, useState, useRef } from "react";
import { useLocation } from "react-router-dom";
import { storage, auth } from "../utils";
import Spinner from "react-bootstrap/Spinner";
import { anno } from "../utils/api";
import { isAxiosError } from "axios";
import { useTranslation } from "react-i18next";

const LoginSpinner = () => {
  const { t } = useTranslation();

  return (
    <div className="d-flex justify-content-center mt-5">
      <div className="text-center">
        <Spinner animation="border" variant="primary" />
        <div className="mt-2">{t("loginCallback.loading")}</div>
      </div>
    </div>
  );
};

const LoginCallback: React.FC = () => {
  const location = useLocation();
  const [loading, setLoading] = useState(true);
  const processing = useRef(false);
  const { t } = useTranslation();

  const handleCallback = async () => {
    if (processing.current) return;
    processing.current = true;

    let message = "";
    try {
      const params = new URLSearchParams(location.search);
      const res = await anno.post(
        "/api/oauth/login",
        { code: params.get("code") },
        {
          headers: {
            "X-Oauth-State": storage.get<string>("oauth_state"),
          },
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
        ? err.response?.data?.message ||
          err.code ||
          t("loginCallback.authFailed")
        : t("loginCallback.unknownError");
    } finally {
      if (window.opener && !window.opener.closed) {
        window.opener.postMessage({ type: "oauth_result", message }, "*");
      }
      setLoading(false);
    }
  };

  useEffect(() => {
    // FIXME: 临时解决 strict 模式下触发两次的问题，需要找到更加优雅的解决方式
    handleCallback();
  }, []);

  return loading ? <LoginSpinner /> : "";
};

export default LoginCallback;
