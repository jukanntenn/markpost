import { Container, Row, Col, Card, Button } from "react-bootstrap";
import { useNavigate } from "react-router-dom";
import { useContext } from "react";
import { UserInfoContext } from "../components/UserInfoProvider";
import { useTranslation } from "react-i18next";

function NotFound() {
  const navigate = useNavigate();
  const { isAuthenticated } = useContext(UserInfoContext);
  const { t } = useTranslation();

  return (
    <Container className="py-5">
      <Row className="justify-content-center">
        <Col md={8} lg={6}>
          <Card className="shadow-sm">
            <Card.Body className="text-center p-5">
              <h1 className="display-6 mb-3">{t("notFound.title")}</h1>
              <p className="text-muted mb-4">
                {isAuthenticated
                  ? t("notFound.pageNotFound")
                  : t("notFound.pageNotFoundLoginRequired")}
              </p>
              {isAuthenticated ? (
                <Button variant="primary" onClick={() => navigate("/dashboard")}>
                  {t("notFound.backToDashboard")}
                </Button>
              ) : (
                <Button variant="primary" onClick={() => navigate("/login")}>
                  {t("notFound.goToLogin")}
                </Button>
              )}
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </Container>
  );
}

export default NotFound;
