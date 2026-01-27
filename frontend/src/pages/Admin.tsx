import React, { useState } from "react";
import { useUsers, type User } from "../hooks/swr/useUsers";
import { useTranslation } from "react-i18next";
import Card from "react-bootstrap/Card";
import Table from "react-bootstrap/Table";
import Badge from "react-bootstrap/Badge";
import Button from "react-bootstrap/Button";
import Spinner from "react-bootstrap/Spinner";

const Admin: React.FC = () => {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const limit = 10;

  const { data, error, isLoading } = useUsers(page, limit);

  const totalPages = data ? Math.ceil(data.total / data.page_size) : 1;

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const renderTableRow = (user: User) => (
    <tr key={user.id}>
      <td>{user.id}</td>
      <td>{user.username}</td>
      <td>
        {user.role === "admin" ? (
          <Badge bg="danger">{t("admin.roleAdmin")}</Badge>
        ) : (
          <Badge bg="primary">{t("admin.roleUser")}</Badge>
        )}
      </td>
      <td>{user.github_id ?? "-"}</td>
      <td>{formatDate(user.created_at)}</td>
      <td>{formatDate(user.updated_at)}</td>
    </tr>
  );

  const renderMobileCard = (user: User) => (
    <Card key={user.id} className="mb-3">
      <Card.Body>
        <div className="d-flex justify-content-between align-items-start mb-2">
          <div>
            <strong>{user.username}</strong>
            <div className="text-muted small">ID: {user.id}</div>
          </div>
          {user.role === "admin" ? (
            <Badge bg="danger">{t("admin.roleAdmin")}</Badge>
          ) : (
            <Badge bg="primary">{t("admin.roleUser")}</Badge>
          )}
        </div>
        <div className="small">
          <div className="mb-1">
            <strong>{t("admin.githubId")}:</strong> {user.github_id ?? "-"}
          </div>
          <div className="mb-1">
            <strong>{t("admin.createdAt")}:</strong> {formatDate(user.created_at)}
          </div>
          <div>
            <strong>{t("admin.updatedAt")}:</strong> {formatDate(user.updated_at)}
          </div>
        </div>
      </Card.Body>
    </Card>
  );

  const renderContent = () => {
    if (isLoading) {
      return (
        <div className="text-center py-5">
          <Spinner animation="border" role="status">
            <span className="visually-hidden">{t("admin.loading")}</span>
          </Spinner>
        </div>
      );
    }

    if (error) {
      return (
        <div className="text-center py-5">
          <p className="text-danger">{t("admin.error")}</p>
        </div>
      );
    }

    if (!data || data.users.length === 0) {
      return (
        <div className="text-center py-5">
          <p className="text-muted">{t("admin.noUsers")}</p>
        </div>
      );
    }

    return (
      <>
        <div className="d-none d-md-block">
          <Table responsive hover>
            <thead>
              <tr>
                <th>{t("admin.id")}</th>
                <th>{t("admin.username")}</th>
                <th>{t("admin.role")}</th>
                <th>{t("admin.githubId")}</th>
                <th>{t("admin.createdAt")}</th>
                <th>{t("admin.updatedAt")}</th>
              </tr>
            </thead>
            <tbody>{data.users.map(renderTableRow)}</tbody>
          </Table>
        </div>

        <div className="d-md-none">{data.users.map(renderMobileCard)}</div>

        {totalPages > 1 && (
          <div className="d-flex justify-content-center mt-4">
            <div className="btn-group" role="group">
              <Button
                variant="outline-primary"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
              >
                {t("admin.previous")}
              </Button>
              <Button variant="outline-primary" disabled>
                {t("admin.page", { current: page, total: totalPages })}
              </Button>
              <Button
                variant="outline-primary"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
              >
                {t("admin.next")}
              </Button>
            </div>
          </div>
        )}

        <div className="text-center mt-3 text-muted small">
          {t("admin.totalUsers", { count: data.total })}
        </div>
      </>
    );
  };

  return (
    <div className="container py-4">
      <div className="row justify-content-center">
        <div className="col-12 col-lg-10">
          <Card>
            <Card.Header as="h5">{t("admin.title")}</Card.Header>
            <Card.Body>{renderContent()}</Card.Body>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default Admin;
