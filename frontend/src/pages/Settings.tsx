import { useState } from "react";
import {
  Card,
  Button,
  Container,
  Form,
  Alert,
  Spinner,
  Row,
  Col,
} from "react-bootstrap";
import { Gear, Shield, ArrowLeft, CheckCircle } from "react-bootstrap-icons";
import { auth } from "../utils/api";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";

interface PasswordChangeRequest {
  current_password: string;
  new_password: string;
  confirm_password: string;
}

function Settings() {
  const { t } = useTranslation();
  const [formData, setFormData] = useState<PasswordChangeRequest>({
    current_password: "",
    new_password: "",
    confirm_password: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [showValidation, setShowValidation] = useState(false);
  const navigate = useNavigate();

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
    // Clear error and success messages when user starts typing
    setError("");
    setSuccess("");
    setShowValidation(false);
  };

  const validatePassword = (password: string) => {
    return password.length >= 6;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Client-side validation
    setShowValidation(true);

    if (!formData.current_password.trim()) {
      setError(t("settings.enterCurrentPassword"));
      return;
    }

    if (!validatePassword(formData.new_password)) {
      setError(t("settings.passwordMinLength"));
      return;
    }

    if (formData.new_password !== formData.confirm_password) {
      setError(t("settings.passwordsNotMatch"));
      return;
    }

    if (formData.current_password === formData.new_password) {
      setError(t("settings.passwordSameAsCurrent"));
      return;
    }

    setLoading(true);
    setError("");
    setSuccess("");

    try {
      await auth.post("/api/auth/change-password", {
        current_password: formData.current_password,
        new_password: formData.new_password,
      });

      setSuccess(t("settings.passwordChangeSuccess"));

      // Clear form and reset validation state
      setFormData({
        current_password: "",
        new_password: "",
        confirm_password: "",
      });
      setShowValidation(false);

      // Keep user on settings page to see success message
      // No automatic navigation - let user decide when to leave
    } catch (err: any) {
      console.error("Password change failed:", err);
      if (err.response?.data?.message) {
        setError(err.response.data.message);
      } else if (err.response?.data?.error) {
        setError(err.response.data.error);
      } else {
        setError(t("settings.passwordChangeFailed"));
      }
    } finally {
      setLoading(false);
    }
  };

  const passwordValid = validatePassword(formData.new_password);
  const passwordsMatch =
    formData.new_password === formData.confirm_password &&
    formData.new_password.length > 0;

  return (
    <Container className="py-4">
      {/* Page Header */}
      <div className="mb-5 text-center">
        <div className="d-inline-flex align-items-center justify-content-center bg-primary bg-gradient rounded-circle p-3 mb-3">
          <Gear size={32} className="text-white" />
        </div>
        <h1 className="display-5 fw-bold text-body mb-3">
          {t("settings.title")}
        </h1>
        <p className="lead text-muted">{t("settings.subtitle")}</p>

        {/* Back to Dashboard */}
        <Button
          variant="outline-secondary"
          onClick={() => navigate("/dashboard")}
          className="mt-3 mb-4"
        >
          <ArrowLeft size={16} className="me-2" />
          {t("settings.backToDashboard")}
        </Button>
      </div>

      <Row className="justify-content-center">
        <Col xs={12} sm={10} md={8} lg={6} xl={5}>
          {/* Password Change Card */}
          <Card className="border-0 shadow-lg">
            <Card.Header className="bg-body border-0 pt-4 px-4 pb-3">
              <div className="d-flex align-items-center">
                <div className="bg-warning bg-gradient rounded-circle p-3 me-3">
                  <Shield size={24} className="text-white" />
                </div>
                <div className="flex-grow-1">
                  <h4 className="mb-1 fw-bold text-body">
                    {t("settings.title")}
                  </h4>
                  <p className="text-muted mb-0 small">
                    {t("settings.subtitle")}
                  </p>
                </div>
              </div>
            </Card.Header>

            <Card.Body className="p-4 p-md-5">
              {/* Error and Success Alerts */}
              {error && (
                <Alert variant="danger" className="mb-4 border-0">
                  {error}
                </Alert>
              )}
              {success && (
                <Alert variant="success" className="mb-4 border-0">
                  <CheckCircle size={16} className="me-2" />
                  {success}
                </Alert>
              )}

              <Form onSubmit={handleSubmit}>
                <Form.Group className="mb-4">
                  <Form.Label className="text-muted small fw-semibold mb-2">
                    {t("settings.currentPassword")}
                  </Form.Label>
                  <Form.Control
                    type="password"
                    name="current_password"
                    value={formData.current_password}
                    onChange={handleInputChange}
                    placeholder={t("settings.currentPasswordPlaceholder")}
                    required
                    disabled={loading}
                    className="py-3 px-3 border-1"
                    style={{ borderRadius: "8px" }}
                    isInvalid={
                      showValidation &&
                      !formData.current_password.trim() &&
                      !success
                    }
                  />
                  {showValidation &&
                    !formData.current_password.trim() &&
                    !success && (
                      <Form.Control.Feedback type="invalid" className="d-block">
                        {t("settings.enterCurrentPassword")}
                      </Form.Control.Feedback>
                    )}
                </Form.Group>

                <Form.Group className="mb-4">
                  <Form.Label className="text-muted small fw-semibold mb-2">
                    {t("settings.newPassword")}
                  </Form.Label>
                  <Form.Control
                    type="password"
                    name="new_password"
                    value={formData.new_password}
                    onChange={handleInputChange}
                    placeholder={t("settings.newPasswordPlaceholder")}
                    required
                    disabled={loading}
                    className="py-3 px-3 border-1"
                    style={{ borderRadius: "8px" }}
                    isInvalid={
                      showValidation &&
                      !passwordValid &&
                      formData.new_password.length > 0 &&
                      !success
                    }
                    isValid={showValidation && passwordValid && !success}
                  />
                  {showValidation &&
                    formData.new_password.length > 0 &&
                    !success && (
                      <Form.Control.Feedback
                        type={passwordValid ? "valid" : "invalid"}
                        className="d-block"
                      >
                        {passwordValid
                          ? t("settings.passwordStrengthValid")
                          : t("settings.passwordMinLength")}
                      </Form.Control.Feedback>
                    )}
                  {showValidation && !formData.new_password && !success && (
                    <Form.Control.Feedback type="invalid" className="d-block">
                      {t("settings.enterNewPassword")}
                    </Form.Control.Feedback>
                  )}
                </Form.Group>

                <Form.Group className="mb-4">
                  <Form.Label className="text-muted small fw-semibold mb-2">
                    {t("settings.confirmPassword")}
                  </Form.Label>
                  <Form.Control
                    type="password"
                    name="confirm_password"
                    value={formData.confirm_password}
                    onChange={handleInputChange}
                    placeholder={t("settings.confirmPasswordPlaceholder")}
                    required
                    disabled={loading}
                    className="py-3 px-3 border-1"
                    style={{ borderRadius: "8px" }}
                    isInvalid={
                      showValidation &&
                      formData.confirm_password.length > 0 &&
                      !passwordsMatch &&
                      !success
                    }
                    isValid={showValidation && passwordsMatch && !success}
                  />
                  {showValidation &&
                    formData.confirm_password.length > 0 &&
                    !success && (
                      <Form.Control.Feedback
                        type={passwordsMatch ? "valid" : "invalid"}
                        className="d-block"
                      >
                        {passwordsMatch
                          ? t("settings.passwordsMatch")
                          : t("settings.passwordsNotMatch")}
                      </Form.Control.Feedback>
                    )}
                  {showValidation && !formData.confirm_password && !success && (
                    <Form.Control.Feedback type="invalid" className="d-block">
                      {t("settings.confirmNewPasswordRequired")}
                    </Form.Control.Feedback>
                  )}
                </Form.Group>

                <div className="d-grid gap-2">
                  <Button
                    variant="primary"
                    type="submit"
                    disabled={
                      loading ||
                      !formData.current_password ||
                      !formData.new_password ||
                      !formData.confirm_password
                    }
                    className="py-3 fw-semibold"
                    style={{ borderRadius: "8px" }}
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
                        {t("settings.changingPassword")}
                      </>
                    ) : (
                      <>
                        <Shield size={16} className="me-2" />
                        {t("settings.changePassword")}
                      </>
                    )}
                  </Button>
                </div>
              </Form>
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </Container>
  );
}

export default Settings;
