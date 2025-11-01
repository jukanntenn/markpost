import {
  Alert,
  Badge,
  Button,
  Card,
  Container,
  Spinner,
} from "react-bootstrap";
import { Book, Copy, Eye, EyeSlash, Key, FilePlus } from "react-bootstrap-icons";
import { useEffect, useState } from "react";

import CreateTestPostModal from "../components/CreateTestPostModal";
import type { PostListItem } from "../types/posts";
import { auth } from "../utils/api";
import { fetchPosts } from "../utils/posts";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";

interface PostKeyResponse {
  post_key: string;
  created_at: string;
}

// Function to format UTC time to local browser timezone
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
  const [postKey, setPostKey] = useState<string>("");
  const [createdAt, setCreatedAt] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [copySuccess, setCopySuccess] = useState(false);
  const [recentPosts, setRecentPosts] = useState<PostListItem[]>([]);
  const [recentLoading, setRecentLoading] = useState(false);
  const [recentError, setRecentError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);

  useEffect(() => {
    document.title = t("common.pageTitle.dashboard");
  }, [t]);

  // Fetch post key from API
  useEffect(() => {
    const fetchPostKey = async () => {
      try {
        setLoading(true);
        setError(null);

        const response = await auth.get<PostKeyResponse>("/api/post_key");
        setPostKey(response.data.post_key);
        setCreatedAt(response.data.created_at);
      } catch (err) {
        console.error("Failed to fetch post key:", err);
        setError(t("dashboard.postKey.errorLoadingKey"));
      } finally {
        setLoading(false);
      }
    };

    fetchPostKey();
  }, []);

  const loadRecentPosts = async () => {
    try {
      setRecentLoading(true);
      setRecentError(null);
      const data = await fetchPosts(1, 10);
      setRecentPosts(data.posts);
    } catch {
      setRecentError(t("dashboard.recentPosts.error"));
    } finally {
      setRecentLoading(false);
    }
  };
  useEffect(() => {
    loadRecentPosts();
  }, []);

  // Copy to clipboard function
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
        {/* Left Column: Post Key + Docs stacked */}
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
              {loading ? (
                <div className="bg-body-tertiary rounded-3 p-4 border border-secondary-subtle text-center">
                  <Spinner animation="border" role="status" variant="primary">
                    <span className="visually-hidden">Loading...</span>
                  </Spinner>
                  <p className="mt-3 text-muted mb-0">
                    {t("dashboard.postKey.loadingKey")}
                  </p>
                </div>
              ) : error ? (
                <Alert variant="danger" className="mb-0">
                  <p>{error}</p>
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

        {/* Right Column: Recent Posts */}
        <div className="col-12 col-xl-6">
          <Card className="border-0 shadow-lg h-100">
            <Card.Header className="bg-body border-0 pt-3 px-4 pb-2">
              <div className="d-flex align-items-center justify-content-between">
                <div className="d-flex align-items-center">
                  <Book size={18} className="me-2 text-body" />
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
              {recentLoading ? (
                <div className="text-center">
                  <Spinner animation="border" role="status" variant="primary">
                    <span className="visually-hidden">Loading...</span>
                  </Spinner>
                  <p className="mt-3 text-muted mb-0">{t("dashboard.recentPosts.loading")}</p>
                </div>
              ) : recentError ? (
                <Alert variant="danger" className="mb-0">
                  <p>{recentError}</p>
                </Alert>
              ) : recentPosts.length === 0 ? (
                <div className="text-center">
                  <p className="text-muted mb-0">{t("dashboard.recentPosts.empty")}</p>
                </div>
              ) : (
                <div>
                  <ul className="list-unstyled mb-0">
                    {recentPosts.map((p) => (
                      <li key={p.id} className="py-2 d-flex justify-content-between align-items-center">
                        <a href={`/${p.id}`} target="_blank" rel="noopener noreferrer" className="text-decoration-none fw-medium">{p.title}</a>
                        <small className="text-muted">{formatToLocalTime(p.created_at)}</small>
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
        loadRecentPosts();
      }}
    />
    </>
  );
}

export default Dashboard;
