import React from "react";
import { Container, Row, Col, Card, Button } from "react-bootstrap";
import { useAuth } from "../hooks/useAuth";

const DashboardPage: React.FC = () => {
  const { user, logout } = useAuth();

  return (
    <Container className="py-5">
      <Row className="justify-content-center">
        <Col md={8}>
          <Card>
            <Card.Header className="d-flex justify-content-between align-items-center">
              <h3 className="mb-0">Dashboard</h3>
              <Button variant="outline-danger" onClick={logout}>
                Logout
              </Button>
            </Card.Header>
            <Card.Body>
              <h4>Welcome, {user?.username}!</h4>
              <p className="text-muted">
                You have successfully logged in with GitHub.
              </p>
              <div className="mt-3">
                <strong>User Information:</strong>
                <ul className="mt-2">
                  <li>Username: {user?.username}</li>
                  <li>GitHub ID: {user?.github_id}</li>
                  <li>Post Key: {user?.post_key}</li>
                </ul>
              </div>
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </Container>
  );
};

export default DashboardPage;
