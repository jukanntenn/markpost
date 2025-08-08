import React from "react";
import {
  BrowserRouter as Router,
  Routes,
  Route,
  Navigate,
} from "react-router-dom";
import "bootstrap/dist/css/bootstrap.min.css";
import "./App.css";

// Lazy load components
const LoginPage = React.lazy(() => import("./components/LoginPage"));
const DashboardPage = React.lazy(() => import("./components/DashboardPage"));
const AuthCallbackPage = React.lazy(
  () => import("./components/AuthCallbackPage")
);

// Protected Route component
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const token = localStorage.getItem("auth_token");

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

function App() {
  return (
    <Router>
      <div className="App">
        <React.Suspense
          fallback={
            <div className="d-flex justify-content-center align-items-center min-vh-100">
              <div className="spinner-border" role="status">
                <span className="visually-hidden">Loading...</span>
              </div>
            </div>
          }
        >
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/auth/callback" element={<AuthCallbackPage />} />
            <Route
              path="/dashboard"
              element={
                <ProtectedRoute>
                  <DashboardPage />
                </ProtectedRoute>
              }
            />
            <Route path="/" element={<Navigate to="/login" replace />} />
          </Routes>
        </React.Suspense>
      </div>
    </Router>
  );
}

export default App;
