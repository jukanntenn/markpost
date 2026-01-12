import {
  Alert,
  Badge,
  Button,
  Card,
  Container,
  Spinner,
} from "react-bootstrap";
import { Book, Copy, Eye, EyeSlash, Key, FilePlus, JournalText } from "react-bootstrap-icons";
import { useState } from "react";

import { buildPostUrl } from "../utils/url";
import CreateTestPostModal from "../components/CreateTestPostModal";
import { usePostKey } from "../hooks/swr/usePostKey";
import { usePosts } from "../hooks/swr/usePosts";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";

const formatToLocalTime = (utcString: string): string => {
  if (!utcString) return "";

  const date = new Date(utcString);

  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");

  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
};

function Dashboard() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [showKey, setShowKey] = useState(false);
  const [copySuccess, setCopySuccess] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);

  const { data: postKeyData, isLoading: keyLoading, error: keyError } = usePostKey();
  const { data: postsData, isLoading: postsLoading, error: postsError, mutate: mutatePosts } = usePosts(1, 10, { refreshInterval: 3000 });

  const postKey = postKeyData?.post_key || "";
  const createdAt = postKeyData?.created_at || "";
  const recentPosts = postsData?.posts || [];

  const handleCopyKey = async () => {
    if (postKey) {
      try {
        await navigator.clipboard.writeText(postKey);
        setCopySuccess(true);
        setTimeout(() => setCopySuccess(false), 2000);
      } catch (err) {
        console.error("Failed to copy text: ", err);
      }
    }
  };

  return (
    <>
    <Container className="py-4">
      <div className="row g-4">
        <div className="col-12 col-xl-6">
          <div className="d-flex flex-column gap-4">
          <Card className="border-0 shadow-lg h-100">
            <Card.Header className="bg-body border-0 pt-3 px-4 pb-2">
              <div className="d-flex align-items-center">
                <Key size={18} className="me-2 text-body" />
                <div className="flex-grow-1">
                  <h6 className="mb-0 text-body">
                    {t("dashboard.postKey.title")}
                  </h6>
                </div>
              </div>
            </Card.Header>
            <Card.Body className="px-4 pb-4">
              {keyLoading ? (
                <div className="bg-body-tertiary rounded-3 p-4 border border-secondary-subtle text-center">
                  <Spinner animation="border" role="status" variant="primary">
                    <span className="visually-hidden">Loading...</span>
                  </Spinner>
                  <p className="mt-3 text-muted mb-0">
                    {t("dashboard.postKey.loadingKey")}
                  </p>
                </div>
              ) : keyError ? (
                <Alert variant="danger" className="mb-0">
                  <p>{t("dashboard.postKey.errorLoadingKey")}</p>
                </Alert>
              ) : (
                <div>
                  <div className="d-flex align-items-center justify-content-between">
                    <div className="flex-grow-1">
                      <div className="font-monospace fs-5 text-body">
                        {showKey ? postKey : "•".repeat(postKey.length)}
                      </div>
                      {copySuccess && (
                        <div className="mt-2">
                          <Badge bg="success" className="animate-fade-in">
                            {t("dashboard.postKey.copied")}
                          </Badge>
                        </div>
                      )}
                    </div>
                    <div className="d-flex gap-0 ms-3">
                      <Button
                        variant="link"
                        size="sm"
                        onClick={() => setShowKey(!showKey)}
                        className="d-flex align-items-center justify-content-center p-0 text-body"
                        style={{ width: "40px", height: "40px" }}
                        title={
                          showKey
                            ? t("dashboard.postKey.hideKey")
                            : t("dashboard.postKey.showKey")
                        }
                      >
                        {showKey ? <EyeSlash size={18} /> : <Eye size={18} />}
                      </Button>
                      <Button
                        variant="link"
                        size="sm"
                        onClick={handleCopyKey}
                        className="d-flex align-items-center justify-content-center p-0 text-body"
                        style={{ width: "40px", height: "40px" }}
                        title={t("dashboard.postKey.copyKey")}
                      >
                        <Copy size={18} />
                      </Button>
                      <Button
                        variant="link"
                        size="sm"
                        onClick={() => setShowCreateModal(true)}
                        className="d-flex align-items-center justify-content-center p-0 text-body"
                        style={{ width: "40px", height: "40px" }}
                        title={t("dashboard.postKey.createTestPostTip")}
                      >
                        <FilePlus size={18} />
                      </Button>
                    </div>
                  </div>
                </div>
              )}
              <div className="mt-3">
                <small className="text-muted">
                  {t("dashboard.postKey.createdAt")}{" "}
                  {formatToLocalTime(createdAt)}
                </small>
              </div>
            </Card.Body>
          </Card>
          <Card className="border-0 shadow-lg h-100">
            <Card.Header className="bg-body border-0 pt-3 px-4 pb-2">
              <div className="d-flex align-items-center">
                <Book size={18} className="me-2 text-body" />
                <div className="flex-grow-1">
                  <h6 className="mb-0 text-body">
                    {t("dashboard.documentation.title")}
                  </h6>
                </div>
              </div>
            </Card.Header>
            <Card.Body className="px-4 pb-4 text-center">
              <p className="text-muted mb-0">
                {t("dashboard.documentation.content")}{" "}
                <a
                  href="https://github.com/jukanntenn/markpost?tab=readme-ov-file#apis"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary text-decoration-none fw-medium"
                >
                  {t("dashboard.documentation.apiLink")}
                </a>{" "}
                {t("dashboard.documentation.content2")}
              </p>
            </Card.Body>
          </Card>
          </div>
        </div>

        <div className="col-12 col-xl-6">
          <Card className="border-0 shadow-lg h-100">
            <Card.Header className="bg-body border-0 pt-3 px-4 pb-2">
              <div className="d-flex align-items-center justify-content-between">
                <div className="d-flex align-items-center">
                  <JournalText size={18} className="me-2 text-body" />
                  <div className="flex-grow-1">
                    <h6 className="mb-0 text-body">{t("dashboard.recentPosts.title")}</h6>
                  </div>
                </div>
                <a
                  href="/posts"
                  onClick={(e) => {
                    e.preventDefault();
                    navigate("/posts");
                  }}
                  className="text-decoration-none small"
                >
                  {t("dashboard.recentPosts.viewAll")}
                </a>
              </div>
            </Card.Header>
            <Card.Body className="px-4 pb-4">
              {postsLoading ? (
                <div className="text-center">
                  <Spinner animation="border" role="status" variant="primary">
                    <span className="visually-hidden">Loading...</span>
                  </Spinner>
                  <p className="mt-3 text-muted mb-0">{t("dashboard.recentPosts.loading")}</p>
                </div>
              ) : postsError ? (
                <Alert variant="danger" className="mb-0">
                  <p>{t("dashboard.recentPosts.error")}</p>
                </Alert>
              ) : recentPosts.length === 0 ? (
                <div className="text-center">
                  <p className="text-muted mb-0">{t("dashboard.recentPosts.empty")}</p>
                </div>
              ) : (
                <div>
                  <ul className="list-unstyled mb-0">
                    {recentPosts.map((p) => (
                      <li key={p.id} className="py-2">
                        <div className="d-flex flex-column flex-md-row align-items-md-center justify-content-md-between">
                          <a
                            href={buildPostUrl(p.qid)}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-decoration-none fw-medium flex-grow-1"
                          >
                            {p.title}
                          </a>
                          <small className="text-muted mt-1 mt-md-0 ms-md-3">{formatToLocalTime(p.created_at)}</small>
                        </div>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </Card.Body>
          </Card>
        </div>
      </div>
    </Container>
    <CreateTestPostModal
      show={showCreateModal}
      postKey={postKey}
      onHide={() => setShowCreateModal(false)}
      onSuccess={() => {
        setShowCreateModal(false);
        mutatePosts();
      }}
    />
    </>
  );
}

export default Dashboard;
