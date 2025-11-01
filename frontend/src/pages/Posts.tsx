import { Alert, Button, Card, Container, Spinner, Table } from "react-bootstrap";
import { useEffect, useState } from "react";

import { Book } from "react-bootstrap-icons";
import type { PostListItem } from "../types/posts";
import { fetchPosts } from "../utils/posts";
import { useSearchParams } from "react-router-dom";
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

function Posts() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const [items, setItems] = useState<PostListItem[]>([]);
  const [page, setPage] = useState<number>(() => parseInt(searchParams.get("page") || "1", 10));
  const limit = 20;
  const [totalPages, setTotalPages] = useState<number>(1);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    document.title = t("common.pageTitle.allPosts");
  }, [t]);

  useEffect(() => {
    const load = async () => {
      try {
        setLoading(true);
        setError(null);
        const data = await fetchPosts(page, limit);
        setItems(data.posts);
        setTotalPages(data.pagination.total_pages || 1);
      } catch (err) {
        setError(t("posts.error"));
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [page]);

  const handlePageChange = (nextPage: number) => {
    if (nextPage < 1 || nextPage > totalPages) return;
    setPage(nextPage);
    setSearchParams({ page: String(nextPage) });
  };

  return (
    <Container className="py-4">
      <div className="row g-4">
        <div className="col-12">
          <Card className="border-0 shadow-sm">
            <Card.Header className="bg-body border-0 pt-3 px-4 pb-2">
                <div className="d-flex align-items-center justify-content-between">
                  <div className="d-flex align-items-center">
                    <Book size={18} className="me-2 text-body" />
                    <div className="flex-grow-1">
                      <h6 className="mb-0 text-body">{t("posts.title")}</h6>
                    </div>
                  </div>
                </div>
            </Card.Header>
            <Card.Body className="px-4 pb-4">
              {loading ? (
                <div className="bg-body-tertiary rounded-3 p-4 border border-secondary-subtle text-center">
                  <Spinner animation="border" role="status" variant="primary">
                    <span className="visually-hidden">Loading...</span>
                  </Spinner>
                  <p className="mt-3 text-muted mb-0">{t("posts.loading")}</p>
                </div>
              ) : error ? (
                <Alert variant="danger" className="mb-0">
                  <p>{error}</p>
                </Alert>
              ) : items.length === 0 ? (
                <div className="bg-body-tertiary rounded-3 p-4 border border-secondary-subtle text-center">
                  <p className="text-muted mb-0">{t("posts.empty")}</p>
                </div>
              ) : (
                <div className="bg-body-tertiary rounded-3 p-3 border border-secondary-subtle">
                  <Table hover responsive className="mb-3">
                    <thead>
                      <tr>
                        <th scope="col" style={{ width: "60%" }}>{t("posts.table.title")}</th>
                        <th scope="col">{t("posts.table.createdAt")}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {items.map((p) => (
                        <tr key={p.id}>
                          <td>
                            <a href={`/${p.id}`} target="_blank" rel="noopener noreferrer" className="text-decoration-none fw-medium">{p.title}</a>
                          </td>
                          <td>
                            <small className="text-muted">{formatToLocalTime(p.created_at)}</small>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </Table>
                  <div className="d-flex justify-content-between align-items-center">
                    <div className="d-flex align-items-center gap-2">
                      <Button
                        variant="outline-primary"
                        size="sm"
                        disabled={page <= 1}
                        onClick={() => handlePageChange(page - 1)}
                      >
                        {t("posts.pagination.prev")}
                      </Button>
                      <span className="px-2 text-muted small">{page} / {totalPages}</span>
                      <Button
                        variant="outline-primary"
                        size="sm"
                        disabled={page >= totalPages}
                        onClick={() => handlePageChange(page + 1)}
                      >
                        {t("posts.pagination.next")}
                      </Button>
                    </div>
                    <div>
                      <small className="text-muted">{page} / {totalPages}</small>
                    </div>
                  </div>
                </div>
              )}
            </Card.Body>
          </Card>
        </div>
      </div>
    </Container>
  );
}

export default Posts;
