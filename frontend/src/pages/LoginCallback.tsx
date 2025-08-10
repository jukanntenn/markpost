import React, { useEffect, useState, useRef } from "react";
import { useLocation } from "react-router-dom";
import { storage } from "../utils";
import Spinner from "react-bootstrap/Spinner";
import { anno } from "../utils/api";

const LoginCallback: React.FC = () => {
  const location = useLocation();
  const [loading, setLoading] = useState(true);
  const [_, setError] = useState<string | null>(null);
  const isProcessing = useRef(false); // 使用ref防止重复执行

  const handleCallback = async () => {
    // 如果已经在处理中，直接返回
    if (isProcessing.current) {
      return;
    }

    // 设置处理标志
    isProcessing.current = true;

    const params = new URLSearchParams(location.search);

    const code = params.get("code");
    const state = params.get("state");

    const oauthState = storage.get<string>("oauth_state");

    try {
      const requestConfig = {
        headers: {
          "X-Oauth-State": oauthState,
        },
        params: {
          state,
        },
      };

      const res = await anno.post(
        "/api/oauth/login",
        {
          code,
        },
        requestConfig
      );

      // 确保数据结构与密码登录一致
      const loginData = {
        access_token: res.data.access_token,
        refresh_token: res.data.refresh_token,
        user: res.data.user,
      };

      // 存储到 storage
      storage.set("login", loginData);

      // 发送成功消息给父窗口
      if (window.opener && !window.opener.closed) {
        window.opener.postMessage(
          {
            type: "oauth_result",
            success: true,
            data: loginData,
          },
          "*"
        );
      } else {
        // 如果没有父窗口，显示错误信息
        setError("无法与登录页面通信，请重新尝试登录");
      }
    } catch (err: any) {
      console.error("Auth callback error:", err);
      const errorMessage =
        err.response?.data?.message || err.code || "授权失败";
      setError(errorMessage);

      // 发送失败消息给父窗口
      if (window.opener && !window.opener.closed) {
        window.opener.postMessage(
          {
            type: "oauth_result",
            success: false,
            error: errorMessage,
          },
          "*"
        );
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // FIXME: 临时解决 strict 模式下触发两次的问题，需要找到更加优雅的解决方式
    handleCallback();
  }, []); // 空依赖数组确保只在组件挂载时执行一次

  if (loading) {
    return (
      <div className="d-flex justify-content-center mt-5">
        <div className="text-center">
          <Spinner animation="border" variant="primary" />
          <div className="mt-2">Login...</div>
        </div>
      </div>
    );
  }

  return "";
};

export default LoginCallback;
