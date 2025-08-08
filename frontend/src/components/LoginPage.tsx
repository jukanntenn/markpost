import React, { useEffect } from "react";
import { Container, Row, Col, Card, Alert } from "react-bootstrap";
import { useAuth } from "../hooks/useAuth";
import GitHubLoginButton from "./GitHubLoginButton";
import LoadingSpinner from "./LoadingSpinner";

const LoginPage: React.FC = () => {
  const { login, loading, error, clearError } = useAuth();

  // Clear error when component mounts
  useEffect(() => {
    clearError();
  }, [clearError]);

  const handleLogin = async () => {
    try {
      await login();
    } catch (err) {
      // Error is handled by useAuth hook
      console.error("Login failed:", err);
    }
  };

  return (
    <div className="min-vh-100 d-flex align-items-center justify-content-center bg-light">
      <Container fluid className="px-4">
        <Row className="justify-content-center">
          <Col xs={12} sm={10} md={8} lg={6} xl={5}>
            <Card className="border-0 shadow">
              <Card.Body className="p-4">
                {/* Header */}
                <div className="text-center mb-4">
                  <div className="mb-3">
                    <svg
                      width="40"
                      height="40"
                      viewBox="0 0 24 24"
                      fill="currentColor"
                      className="text-primary"
                    >
                      <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
                    </svg>
                  </div>
                  <h3 className="fw-bold mb-2">Welcome Back</h3>
                  <p className="text-muted mb-0">
                    Sign in to your account using GitHub
                  </p>
                </div>

                {/* Error Alert */}
                {error && (
                  <Alert
                    variant="danger"
                    dismissible
                    onClose={clearError}
                    className="mb-3 border-0"
                  >
                    <div className="d-flex align-items-center">
                      <svg
                        width="16"
                        height="16"
                        fill="currentColor"
                        className="me-2 flex-shrink-0"
                        viewBox="0 0 16 16"
                      >
                        <path d="M8 15A7 7 0 1 1 8 1a7 7 0 0 1 0 14zm0 1A8 8 0 1 0 8 0a8 8 0 0 0 0 16z" />
                        <path d="M7.002 11a1 1 0 1 1 2 0 1 1 0 0 1-2 0zM7.1 4.995a.905.905 0 1 1 1.8 0l-.35 3.507a.552.552 0 0 1-1.1 0L7.1 4.995z" />
                      </svg>
                      <span className="small">{error}</span>
                    </div>
                  </Alert>
                )}

                {/* Login Button */}
                <div className="d-grid">
                  <GitHubLoginButton
                    onClick={handleLogin}
                    loading={loading}
                    disabled={loading}
                  />
                </div>

                {/* Loading Spinner */}
                {loading && (
                  <div className="mt-3 text-center">
                    <LoadingSpinner text="Redirecting to GitHub..." size="sm" />
                  </div>
                )}

                {/* Footer */}
                <div className="text-center mt-3">
                  <small className="text-muted">
                    By signing in, you agree to our terms of service and privacy
                    policy
                  </small>
                </div>
              </Card.Body>
            </Card>
          </Col>
        </Row>
      </Container>
    </div>
  );
};

export default LoginPage;
