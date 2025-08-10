import { useState, useEffect, useRef } from "react";
import {
  Button,
  Container,
  Card,
  Spinner,
  Row,
  Col,
  Form,
  Alert,
} from "react-bootstrap";
import { Github, Person, Lock } from "react-bootstrap-icons";
import { useTranslation } from "react-i18next";
import { anno } from "../utils/api";
import { storage } from "../utils";
import type { AuthResponse } from "../types/auth";

function LoginPage() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [formData, setFormData] = useState({
    username: "",
    password: "",
  });
  const authWindowRef = useRef<Window | null>(null);
  const checkIntervalRef = useRef<number | null>(null);
  const loginCheckIntervalRef = useRef<number | null>(null);

  function checkAuthWindow() {
    if (authWindowRef.current && authWindowRef.current.closed) {
      stopPolling();
      setLoading(false);

      // 检查是否已经登录
      const loginData = storage.get("login") as AuthResponse | null;
      if (loginData && loginData.user && loginData.access_token) {
        setSuccess(t("login.loginSuccess"));
        setTimeout(() => {
          window.location.href = "/ui/dashboard";
        }, 1000);
      } else {
        storage.remove("oauth_state");
      }
    }
  }

  function stopPolling() {
    if (checkIntervalRef.current) {
      clearInterval(checkIntervalRef.current);
      checkIntervalRef.current = null;
    }
    if (loginCheckIntervalRef.current) {
      clearInterval(loginCheckIntervalRef.current);
      loginCheckIntervalRef.current = null;
    }
  }

  async function handleGitHubLogin() {
    setLoading(true);
    setError("");

    try {
      const res = await anno.get("/api/oauth/url");
      const url = res.data.url;
      // 从 url 中提取 state 参数
      const state = new URL(url).searchParams.get("state");
      storage.set("oauth_state", state);

      // 计算弹窗位置和尺寸
      const width = 600;
      const height = 700;
      const left = (window.innerWidth - width) / 2 + window.screenX;
      const top = (window.innerHeight - height) / 2 + window.screenY;

      // 打开新的 tab 页
      authWindowRef.current = window.open(
        url,
        "github_oauth",
        `width=${width},height=${height},left=${left},top=${top},resizable=yes,scrollbars=yes,status=yes`
      );

      if (!authWindowRef.current) {
        setError(t("login.cannotOpenAuthWindow"));
        setLoading(false);
        return;
      }

      // 开始检查弹窗是否关闭
      checkIntervalRef.current = setInterval(checkAuthWindow, 1000);

      // 监听来自授权窗口的消息
      const handleMessage = async (event: MessageEvent) => {
        if (event.data?.type === "oauth_result") {
          stopPolling();
          setLoading(false);

          if (event.data.success) {
            setSuccess(t("login.loginSuccess"));

            // 检查 storage 中的数据
            const loginData = storage.get("login") as AuthResponse | null;

            if (loginData && loginData.user && loginData.access_token) {
              // 清理 oauth_state
              storage.remove("oauth_state");

              // 延迟跳转，让用户看到成功消息
              setTimeout(() => {
                window.location.href = "/ui/dashboard";
              }, 1000);
            } else {
              console.error("Invalid login data found:", loginData);
              setError(t("login.invalidLoginData"));
              storage.remove("oauth_state");
            }
          } else {
            // 清理 oauth_state
            storage.remove("oauth_state");
            setError(event.data.error || t("login.authFailed"));
          }

          // 关闭授权窗口
          if (authWindowRef.current && !authWindowRef.current.closed) {
            authWindowRef.current.close();
          }

          window.removeEventListener("message", handleMessage);
        }
      };

      window.addEventListener("message", handleMessage);
    } catch (err) {
      console.error("GitHub login failed:", err);
      setError(t("login.githubLoginFailed"));
      setLoading(false);
    }
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
    // 清除错误信息
    setError("");
    setSuccess("");
  };

  async function handlePasswordLogin(e: React.FormEvent) {
    e.preventDefault();
    setPasswordLoading(true);
    setError("");
    setSuccess("");

    try {
      const res = await anno.post("/api/auth/login", formData);

      // 存储登录信息（与现有系统保持一致）
      storage.set("login", {
        access_token: res.data.access_token,
        refresh_token: res.data.refresh_token,
        user: res.data.user,
      });

      setSuccess(t("login.loginSuccess"));

      // 延迟跳转，让用户看到成功消息
      setTimeout(() => {
        window.location.href = "/ui/dashboard";
      }, 1000);
    } catch (err: any) {
      console.error("Password login failed:", err);
      if (err.response?.data?.message) {
        setError(err.response.data.message);
      } else {
        setError(t("login.passwordLoginFailed"));
      }
    } finally {
      setPasswordLoading(false);
    }
  }

  // 清理函数
  useEffect(() => {
    return () => {
      stopPolling();
      if (authWindowRef.current && !authWindowRef.current.closed) {
        authWindowRef.current.close();
      }
    };
  }, []);

  const gitHubButtonText = loading
    ? t("login.processingGitHubLogin")
    : t("login.useGitHubLogin");

  return (
    <div className="min-vh-100 bg-body d-flex align-items-center">
      <Container>
        <Row className="justify-content-center">
          <Col xs={12} sm={10} md={8} lg={6} xl={5}>
            {/* Logo and Title */}
            <div className="text-center mb-5">
              <div className="d-inline-flex align-items-center justify-content-center bg-primary bg-gradient rounded-circle p-3 mb-4">
                <Github size={40} className="text-white" />
              </div>
              <h1 className="fw-bold text-body mb-2">{t("login.welcome")}</h1>
              <p className="text-muted">{t("login.subtitle")}</p>
            </div>

            {/* Login Card */}
            <Card className="border-0 shadow-lg">
              <Card.Body className="p-4 p-md-5">
                {/* Error and Success Alerts */}
                {error && (
                  <Alert variant="danger" className="mb-4 border-0">
                    {error}
                  </Alert>
                )}
                {success && (
                  <Alert variant="success" className="mb-4 border-0">
                    {success}
                  </Alert>
                )}

                {/* Password Login Form */}
                <Form onSubmit={handlePasswordLogin} className="mb-4">
                  <Form.Group className="mb-3">
                    <Form.Label className="text-muted small fw-semibold">
                      <Person size={14} className="me-1" />
                      {t("login.username")}
                    </Form.Label>
                    <Form.Control
                      type="text"
                      name="username"
                      value={formData.username}
                      onChange={handleInputChange}
                      placeholder={t("login.usernamePlaceholder")}
                      required
                      disabled={passwordLoading}
                      className="py-3 px-3 border-1"
                    />
                  </Form.Group>

                  <Form.Group className="mb-4">
                    <Form.Label className="text-muted small fw-semibold">
                      <Lock size={14} className="me-1" />
                      {t("login.password")}
                    </Form.Label>
                    <Form.Control
                      type="password"
                      name="password"
                      value={formData.password}
                      onChange={handleInputChange}
                      placeholder={t("login.passwordPlaceholder")}
                      required
                      disabled={passwordLoading}
                      className="py-3 px-3 border-1"
                    />
                  </Form.Group>

                  <div className="d-grid gap-2">
                    <Button
                      variant="primary"
                      type="submit"
                      disabled={
                        passwordLoading ||
                        !formData.username ||
                        !formData.password
                      }
                      className="py-3 fw-semibold"
                    >
                      {passwordLoading ? (
                        <>
                          <Spinner
                            as="span"
                            animation="border"
                            size="sm"
                            role="status"
                            aria-hidden="true"
                            className="me-2"
                          />
                          {t("login.signingIn")}
                        </>
                      ) : (
                        <>
                          <Person size={20} className="me-2" />
                          {t("login.loginButton")}
                        </>
                      )}
                    </Button>
                  </div>
                </Form>

                {/* Divider */}
                <div className="position-relative mb-4">
                  <hr className="text-muted" />
                  <span className="position-absolute top-50 start-50 translate-middle bg-body px-3 text-muted small">
                    {t("login.or")}
                  </span>
                </div>

                {/* GitHub Login Button */}
                <div className="d-grid gap-2">
                  <Button
                    variant="outline-secondary"
                    onClick={handleGitHubLogin}
                    disabled={loading}
                    className="py-3 fw-semibold"
                  >
                    {loading ? (
                      <>
                        <Spinner
                          as="span"
                          animation="border"
                          size="sm"
                          role="status"
                          aria-hidden="true"
                          className="me-2"
                        />
                        {gitHubButtonText}
                      </>
                    ) : (
                      <>
                        <Github size={16} className="me-2" />
                        {t("login.githubLogin")}
                      </>
                    )}
                  </Button>
                </div>
              </Card.Body>
            </Card>
          </Col>
        </Row>
      </Container>
    </div>
  );
}

export default LoginPage;
