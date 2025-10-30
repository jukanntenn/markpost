import { useEffect, useRef, useState } from "react";
import { Modal, Button, Form, Alert, Spinner } from "react-bootstrap";
import { useTranslation } from "react-i18next";
import { createTestPost } from "../utils/api";
import { useToasts } from "react-bootstrap-toasts";

interface CreateTestPostModalProps {
  show: boolean;
  postKey: string;
  onHide: () => void;
  onSuccess: () => void;
}

function CreateTestPostModal({ show, postKey, onHide, onSuccess }: CreateTestPostModalProps) {
  const { t } = useTranslation();
  const toasts = useToasts();
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>("");
  const titleRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (show) {
      setError("");
      setTimeout(() => titleRef.current?.focus(), 0);
    } else {
      setTitle("");
      setBody("");
      setError("");
      setLoading(false);
    }
  }, [show]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!body.trim()) {
      setError(t("createTestPost.errorEmptyBody"));
      return;
    }
    try {
      setLoading(true);
      setError("");
      await createTestPost(postKey, title.trim(), body);
      toasts.success({
        headerContent: <span className="me-auto">{t("createTestPost.successHeader")}</span>,
        bodyContent: t("createTestPost.successBody"),
      });
      onSuccess();
    } catch (err: any) {
      const msg = err?.response?.data?.error || t("createTestPost.errorServer");
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal show={show} onHide={onHide} backdrop keyboard size="lg" centered>
      <Form onSubmit={handleSubmit}>
        <Modal.Header closeButton>
          <Modal.Title>{t("createTestPost.title")}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          {error && (
            <Alert variant="danger" className="mb-3">
              {error}
            </Alert>
          )}
          <Form.Group className="mb-3">
            <Form.Label className="text-muted small fw-semibold">
              {t("createTestPost.titleLabel")}
            </Form.Label>
            <Form.Control
              ref={titleRef}
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("createTestPost.titlePlaceholder")}
              disabled={loading}
              className="py-2 px-3 border-1"
            />
          </Form.Group>
          <Form.Group>
            <Form.Label className="text-muted small fw-semibold">
              {t("createTestPost.bodyLabel")}
            </Form.Label>
            <Form.Control
              as="textarea"
              rows={8}
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder={t("createTestPost.bodyPlaceholder")}
              disabled={loading}
              className="py-2 px-3 border-1"
            />
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={onHide} disabled={loading}>
            {t("createTestPost.cancel")}
          </Button>
          <Button variant="primary" type="submit" disabled={loading || !body.trim()}>
            {loading ? (
              <span className="d-inline-flex align-items-center gap-2">
                <Spinner size="sm" animation="border" />
                {t("createTestPost.creating")}
              </span>
            ) : (
              t("createTestPost.create")
            )}
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  );
}

export default CreateTestPostModal;
