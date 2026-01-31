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
import { useDeliveryChannels } from "../hooks/swr/useDeliveryChannels";
import { useCreateDeliveryChannel } from "../hooks/swr/useCreateDeliveryChannel";
import { useUpdateDeliveryChannel } from "../hooks/swr/useUpdateDeliveryChannel";
import { useDeleteDeliveryChannel } from "../hooks/swr/useDeleteDeliveryChannel";
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
  const [deliveryError, setDeliveryError] = useState("");
  const [deliverySuccess, setDeliverySuccess] = useState("");
  const [newChannelName, setNewChannelName] = useState("");
  const [newChannelWebhookURL, setNewChannelWebhookURL] = useState("");
  const [editingChannelID, setEditingChannelID] = useState<number | null>(null);
  const [editChannelName, setEditChannelName] = useState("");
  const [editChannelWebhookURL, setEditChannelWebhookURL] = useState("");

  const { trigger, isMutating, reset } = useChangePassword();
  const {
    data: deliveryChannelsData,
    error: deliveryChannelsLoadError,
    isLoading: isDeliveryChannelsLoading,
    mutate: mutateDeliveryChannels,
  } = useDeliveryChannels();
  const { trigger: createDeliveryChannel, isMutating: isCreatingDeliveryChannel } =
    useCreateDeliveryChannel();
  const { trigger: updateDeliveryChannel, isMutating: isUpdatingDeliveryChannel } =
    useUpdateDeliveryChannel();
  const { trigger: deleteDeliveryChannel, isMutating: isDeletingDeliveryChannel } =
    useDeleteDeliveryChannel();

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

  const maskWebhookURL = (raw: string) => {
    try {
      const u = new URL(raw);
      const last = u.pathname.split("/").filter(Boolean).pop() ?? "";
      const tail = last.slice(-6);
      return `${u.host}/…${tail}`;
    } catch {
      return raw;
    }
  };

  const beginEditChannel = (channel: {
    id: number;
    name: string;
    webhook_url: string;
  }) => {
    setEditingChannelID(channel.id);
    setEditChannelName(channel.name ?? "");
    setEditChannelWebhookURL(channel.webhook_url ?? "");
    setDeliveryError("");
    setDeliverySuccess("");
  };

  const cancelEditChannel = () => {
    setEditingChannelID(null);
    setEditChannelName("");
    setEditChannelWebhookURL("");
  };

  const handleCreateChannel = async (e: React.FormEvent) => {
    e.preventDefault();
    setDeliveryError("");
    setDeliverySuccess("");

    try {
      await createDeliveryChannel({
        kind: "feishu",
        name: newChannelName,
        webhook_url: newChannelWebhookURL,
        enabled: true,
      });
      setNewChannelName("");
      setNewChannelWebhookURL("");
      setDeliverySuccess(t("settings.deliveryChannelCreated"));
      await mutateDeliveryChannels();
    } catch (err: unknown) {
      setDeliveryError(api.getErrorMessage(err, t("settings.deliveryChannelCreateFailed")));
    }
  };

  const handleToggleChannel = async (id: number, enabled: boolean) => {
    setDeliveryError("");
    setDeliverySuccess("");

    try {
      await updateDeliveryChannel({ id, enabled });
      await mutateDeliveryChannels();
    } catch (err: unknown) {
      setDeliveryError(api.getErrorMessage(err, t("settings.deliveryChannelUpdateFailed")));
    }
  };

  const handleSaveChannel = async () => {
    if (editingChannelID == null) return;
    setDeliveryError("");
    setDeliverySuccess("");

    try {
      await updateDeliveryChannel({
        id: editingChannelID,
        name: editChannelName,
        webhook_url: editChannelWebhookURL,
      });
      setDeliverySuccess(t("settings.deliveryChannelUpdated"));
      cancelEditChannel();
      await mutateDeliveryChannels();
    } catch (err: unknown) {
      setDeliveryError(api.getErrorMessage(err, t("settings.deliveryChannelUpdateFailed")));
    }
  };

  const handleDeleteChannel = async (id: number) => {
    setDeliveryError("");
    setDeliverySuccess("");

    try {
      await deleteDeliveryChannel({ id });
      setDeliverySuccess(t("settings.deliveryChannelDeleted"));
      await mutateDeliveryChannels();
    } catch (err: unknown) {
      setDeliveryError(api.getErrorMessage(err, t("settings.deliveryChannelDeleteFailed")));
    }
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
                <div className="flex-grow-1">
                  <h6 className="mb-0 text-body">{t("settings.deliveryChannels")}</h6>
                </div>
              </div>
            </Card.Header>
            <Card.Body className="p-4 p-md-5">
              {deliveryChannelsLoadError && (
                <Alert variant="danger" className="mb-4 border-0">
                  {api.getErrorMessage(deliveryChannelsLoadError, t("settings.deliveryChannelsLoadFailed"))}
                </Alert>
              )}
              {deliveryError && (
                <Alert variant="danger" className="mb-4 border-0">
                  {deliveryError}
                </Alert>
              )}
              {deliverySuccess && (
                <Alert variant="success" className="mb-4 border-0">
                  <CheckCircle size={16} className="me-2" />
                  {deliverySuccess}
                </Alert>
              )}

              <div className="mb-4">
                <div className="text-muted small fw-semibold mb-2">
                  {t("settings.deliveryChannelsList")}
                </div>
                {isDeliveryChannelsLoading ? (
                  <div className="d-flex align-items-center gap-2 text-muted small">
                    <Spinner animation="border" size="sm" />
                    {t("settings.deliveryChannelsLoading")}
                  </div>
                ) : (deliveryChannelsData?.channels?.length ?? 0) === 0 ? (
                  <div className="text-muted small">{t("settings.deliveryChannelsEmpty")}</div>
                ) : (
                  <div className="d-flex flex-column gap-3">
                    {deliveryChannelsData?.channels?.map((ch) => (
                      <div
                        key={ch.id}
                        className="border rounded-3 p-3 d-flex flex-column gap-2"
                      >
                        <div className="d-flex align-items-start justify-content-between gap-3">
                          <div className="flex-grow-1">
                            <div className="fw-semibold text-body">
                              {ch.name?.trim() ? ch.name : t("settings.deliveryChannelUnnamed")}
                            </div>
                            <div className="text-muted small">
                              {t("settings.deliveryChannelType")}: {ch.kind}
                            </div>
                            <div className="text-muted small">
                              {t("settings.deliveryChannelWebhook")}: {maskWebhookURL(ch.webhook_url)}
                            </div>
                          </div>

                          <div className="d-flex flex-column align-items-end gap-2">
                            <Form.Check
                              type="switch"
                              id={`delivery-channel-enabled-${ch.id}`}
                              label={t("settings.deliveryChannelEnabled")}
                              checked={ch.enabled}
                              onChange={() => handleToggleChannel(ch.id, !ch.enabled)}
                              disabled={
                                isUpdatingDeliveryChannel ||
                                isDeletingDeliveryChannel ||
                                isCreatingDeliveryChannel
                              }
                            />
                            <div className="d-flex gap-2">
                              <Button
                                variant="outline-secondary"
                                size="sm"
                                onClick={() => beginEditChannel(ch)}
                                disabled={
                                  editingChannelID !== null ||
                                  isUpdatingDeliveryChannel ||
                                  isDeletingDeliveryChannel ||
                                  isCreatingDeliveryChannel
                                }
                              >
                                {t("settings.deliveryChannelEdit")}
                              </Button>
                              <Button
                                variant="outline-danger"
                                size="sm"
                                onClick={() => handleDeleteChannel(ch.id)}
                                disabled={
                                  editingChannelID !== null ||
                                  isDeletingDeliveryChannel ||
                                  isUpdatingDeliveryChannel ||
                                  isCreatingDeliveryChannel
                                }
                              >
                                {t("settings.deliveryChannelDelete")}
                              </Button>
                            </div>
                          </div>
                        </div>

                        {editingChannelID === ch.id && (
                          <div className="border-top pt-3 d-flex flex-column gap-3">
                            <Form.Group>
                              <Form.Label className="text-muted small fw-semibold mb-2">
                                {t("settings.deliveryChannelName")}
                              </Form.Label>
                              <Form.Control
                                value={editChannelName}
                                onChange={(e) => setEditChannelName(e.target.value)}
                                placeholder={t("settings.deliveryChannelNamePlaceholder")}
                                className="py-2 px-3 border-1"
                                style={{ borderRadius: "8px" }}
                              />
                            </Form.Group>
                            <Form.Group>
                              <Form.Label className="text-muted small fw-semibold mb-2">
                                {t("settings.deliveryChannelWebhookURL")}
                              </Form.Label>
                              <Form.Control
                                value={editChannelWebhookURL}
                                onChange={(e) => setEditChannelWebhookURL(e.target.value)}
                                placeholder={t("settings.deliveryChannelWebhookPlaceholder")}
                                className="py-2 px-3 border-1"
                                style={{ borderRadius: "8px" }}
                              />
                            </Form.Group>
                            <div className="d-flex justify-content-end gap-2">
                              <Button
                                variant="outline-secondary"
                                onClick={cancelEditChannel}
                                disabled={isUpdatingDeliveryChannel}
                              >
                                {t("settings.deliveryChannelCancel")}
                              </Button>
                              <Button
                                variant="primary"
                                onClick={handleSaveChannel}
                                disabled={
                                  isUpdatingDeliveryChannel ||
                                  editChannelWebhookURL.trim().length === 0
                                }
                              >
                                {isUpdatingDeliveryChannel ? (
                                  <>
                                    <Spinner
                                      as="span"
                                      animation="border"
                                      size="sm"
                                      role="status"
                                      aria-hidden="true"
                                      className="me-2"
                                    />
                                    {t("settings.deliveryChannelSaving")}
                                  </>
                                ) : (
                                  t("settings.deliveryChannelSave")
                                )}
                              </Button>
                            </div>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="border-top pt-4">
                <div className="text-muted small fw-semibold mb-2">
                  {t("settings.deliveryChannelAdd")}
                </div>
                <Form onSubmit={handleCreateChannel}>
                  <Form.Group className="mb-3">
                    <Form.Label className="text-muted small fw-semibold mb-2">
                      {t("settings.deliveryChannelName")}
                    </Form.Label>
                    <Form.Control
                      value={newChannelName}
                      onChange={(e) => setNewChannelName(e.target.value)}
                      placeholder={t("settings.deliveryChannelNamePlaceholder")}
                      disabled={isCreatingDeliveryChannel || editingChannelID !== null}
                      className="py-2 px-3 border-1"
                      style={{ borderRadius: "8px" }}
                    />
                  </Form.Group>
                  <Form.Group className="mb-3">
                    <Form.Label className="text-muted small fw-semibold mb-2">
                      {t("settings.deliveryChannelWebhookURL")}
                    </Form.Label>
                    <Form.Control
                      value={newChannelWebhookURL}
                      onChange={(e) => setNewChannelWebhookURL(e.target.value)}
                      placeholder={t("settings.deliveryChannelWebhookPlaceholder")}
                      required
                      disabled={isCreatingDeliveryChannel || editingChannelID !== null}
                      className="py-2 px-3 border-1"
                      style={{ borderRadius: "8px" }}
                    />
                  </Form.Group>

                  <div className="d-grid">
                    <Button
                      variant="primary"
                      type="submit"
                      className="py-2 fw-semibold"
                      style={{ borderRadius: "8px" }}
                      disabled={
                        isCreatingDeliveryChannel ||
                        editingChannelID !== null ||
                        newChannelWebhookURL.trim().length === 0
                      }
                    >
                      {isCreatingDeliveryChannel ? (
                        <>
                          <Spinner
                            as="span"
                            animation="border"
                            size="sm"
                            role="status"
                            aria-hidden="true"
                            className="me-2"
                          />
                          {t("settings.deliveryChannelCreating")}
                        </>
                      ) : (
                        t("settings.deliveryChannelCreate")
                      )}
                    </Button>
                  </div>
                </Form>
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
