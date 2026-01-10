import { useState, useEffect } from "react";
import {
  Card,
  Button,
  Container,
  Form,
  Alert,
  Spinner,
  Row,
  Col,
  OverlayTrigger,
  Tooltip,
} from "react-bootstrap";
import { Gear, Lock, CheckCircle, InfoCircle } from "react-bootstrap-icons";
import * as api from "../utils/api";

import { useChangePassword } from "../hooks/swr/useChangePassword";
import { useTranslation } from "react-i18next";
import LanguageToggle from "../components/LanguageToggle";

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
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [showValidation, setShowValidation] = useState(false);

  const { trigger, isMutating, reset } = useChangePassword();

  useEffect(() => {
    document.title = t("common.pageTitle.settings");
  }, [t]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
    setError("");
    setSuccess("");
    setShowValidation(false);
  };

  const validatePassword = (password: string) => {
    return password.length >= 6;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    setShowValidation(true);

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

    try {
      await trigger({
        current_password: formData.current_password,
        new_password: formData.new_password,
      });

      setSuccess(t("settings.passwordChangeSuccess"));

      setFormData({
        current_password: "",
        new_password: "",
        confirm_password: "",
      });
      setShowValidation(false);
      reset();
    } catch (err: unknown) {
      console.error("Password change failed:", err);
      setError(api.getErrorMessage(err, t("settings.passwordChangeFailed")));
    }
  };

  const passwordValid = validatePassword(formData.new_password);
  const passwordsMatch =
    formData.new_password === formData.confirm_password &&
    formData.new_password.length > 0;

  return (
    <Container className="py-4">


      <Row className="justify-content-center">
        <Col xs={12} sm={10} md={8} lg={6} xl={5}>
          <Card className="border-0 shadow-lg">
            <Card.Header className="bg-body border-0 pt-3 px-4 pb-2">
              <div className="d-flex align-items-center">
                <Gear size={18} className="me-2 text-body" />
                <div className="flex-grow-1">
                  <h6 className="mb-0 text-body">
                    {t("settings.applicationSettings")}
                  </h6>
                </div>
              </div>
            </Card.Header>
            <Card.Body className="p-4 p-md-5">
              <div className="mb-3">
                <div className="d-flex align-items-center justify-content-between">
                  <div className="d-flex align-items-center gap-2">
                    <span className="form-label fw-semibold text-muted small mb-0">
                      {t("settings.language")}
                    </span>
                    <OverlayTrigger
                      placement="top"
                      overlay={<Tooltip id="language-help">{t("settings.languageDescription")}</Tooltip>}
                    >
                      <Button
                        variant="link"
                        className="p-0 text-muted d-inline-flex align-items-center"
                        aria-label={t("settings.languageDescription")}
                        style={{ lineHeight: 0 }}
                      >
                        <InfoCircle size={16} />
                      </Button>
                    </OverlayTrigger>
                  </div>
                  <LanguageToggle />
                </div>
              </div>
            </Card.Body>
          </Card>
        </Col>
      </Row>
      <Row className="justify-content-center mt-5">
        <Col xs={12} sm={10} md={8} lg={6} xl={5}>
           <Card className="border-0 shadow-lg">
            <Card.Header className="bg-body border-0 pt-3 px-4 pb-2">
              <div className="d-flex align-items-center">
                <Lock size={18} className="me-2 text-body" />
                <div className="flex-grow-1">
                  <h6 className="mb-0 text-body">
                    {t("settings.changePassword")}
                  </h6>
                </div>
              </div>
            </Card.Header>
            <Card.Body className="p-4 p-md-5">
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
                    disabled={isMutating}
                    className="py-3 px-3 border-1"
                    style={{ borderRadius: "8px" }}
                  />
                  <div className="form-text mt-1">
                    {t("settings.currentPasswordHelp")}
                  </div>
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
                    disabled={isMutating}
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
                    disabled={isMutating}
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
                      {t("settings.confirmPasswordRequired")}
                    </Form.Control.Feedback>
                  )}
                </Form.Group>

                <div className="d-grid gap-2">
                  <Button
                    variant="primary"
                    type="submit"
                    disabled={
                      isMutating ||
                      !formData.new_password ||
                      !formData.confirm_password
                    }
                    className="py-3 fw-semibold"
                    style={{ borderRadius: "8px" }}
                  >
                    {isMutating ? (
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
