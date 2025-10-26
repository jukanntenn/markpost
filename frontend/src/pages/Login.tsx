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
import { storage, auth } from "../utils";
import type { LoginResponse, OAuthUrlResponse } from "../types/auth";
import axios from "axios";
import { useToasts } from "react-bootstrap-toasts";

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
  const checkAuthWindowIntervalRef = useRef<number | null>(null);

  const toasts = useToasts();
  const showErrorToast = (header: string, body: string) => {
    toasts.danger({
      headerContent: <span className="me-auto">{header}</span>,
      bodyContent: body,
    });
  };

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

  const closeAuthWindow = () => {
    if (!authWindowClosed()) {
      authWindowRef.current!.close();
    }
  };

  const clearCheckAuthWindowInterval = () => {
    if (checkAuthWindowIntervalRef.current) {
      clearInterval(checkAuthWindowIntervalRef.current);
      checkAuthWindowIntervalRef.current = null;
    }
  };

  async function handleGitHubLogin() {
    setLoadingGitHub(true);

    try {
      // TODO: add state field in response
      const res = await anno.get<OAuthUrlResponse>("/api/oauth/url");
      const url = res.data.url;
      const state = new URL(url).searchParams.get("state");
      storage.set("oauth_state", state);

      authWindowRef.current = openAuthWindow(url);
      if (!authWindowRef.current) {
        showErrorToast(t("login.cannotOpenAuthWindow"), "");
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
          storage.remove("oauth_state");
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
      storage.remove("oauth_state");
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
      if (axios.isAxiosError(err) && err.response?.data?.message) {
        setError(err.response?.data?.message);
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
  }, []);

  const gitHubButtonText = loadingGitHub
    ? t("login.processingGitHubLogin")
    : t("login.githubLogin");

  return (
    <div className="mt-5">
      <Container>
        <Row className="justify-content-center">
          <Col xs={12} sm={10} md={8} lg={6} xl={5}>
            {/* Title */}
            <div className="text-center mb-5">
              <div className="d-inline-flex align-items-center justify-content-center">
                <img
                  src="markpost.svg"
                  alt="Markpost"
                  height="48"
                  className="me-2"
                />
                <span className="fs-2 fw-bold text-body">Markpost</span>
              </div>
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
                <Form onSubmit={handleLogin} className="mb-4">
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
                      disabled={loading}
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
                      disabled={loading}
                      className="py-3 px-3 border-1"
                    />
                  </Form.Group>

                  <div className="d-grid gap-2">
                    <Button
                      variant="primary"
                      type="submit"
                      disabled={
                        loading || !formData.username || !formData.password
                      }
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
                    disabled={loadingGitHub}
                    className="py-3 fw-semibold"
                  >
                    {loadingGitHub ? (
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
