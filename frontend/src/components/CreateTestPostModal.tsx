import { useEffect, useRef, useState } from "react";
import { Modal, Button, Form, Alert, Spinner } from "react-bootstrap";
import { useTranslation } from "react-i18next";
import { useCreateTestPost } from "../hooks/swr/useCreateTestPost";
import * as api from "../utils/api";
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
  const [error, setError] = useState<string>("");
  const titleRef = useRef<HTMLInputElement | null>(null);

  const { trigger, isMutating, reset } = useCreateTestPost(postKey);

  useEffect(() => {
    if (show) {
      setError("");
      setTimeout(() => titleRef.current?.focus(), 0);
    } else {
      setTitle("");
      setBody("");
      setError("");
    }
  }, [show]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!body.trim()) {
      setError(t("createTestPost.errorEmptyBody"));
      return;
    }
    try {
      await trigger({ title: title.trim(), body });
      toasts.success({
        headerContent: <span className="me-auto">{t("createTestPost.successHeader")}</span>,
        bodyContent: t("createTestPost.successBody"),
      });
      onSuccess();
      reset();
    } catch (err: unknown) {
      const msg = api.getErrorMessage(err, t("createTestPost.errorServer"));
      setError(msg);
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
              disabled={isMutating}
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
              disabled={isMutating}
              className="py-2 px-3 border-1"
            />
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={onHide} disabled={isMutating}>
            {t("createTestPost.cancel")}
          </Button>
          <Button variant="primary" type="submit" disabled={isMutating || !body.trim()}>
            {isMutating ? (
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
