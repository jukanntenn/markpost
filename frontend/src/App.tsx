import React, { useContext, useEffect } from "react";
import {
  BrowserRouter as Router,
  Routes,
  Route,
  Navigate,
} from "react-router-dom";
import "bootstrap/dist/css/bootstrap.min.css";
import Layout from "./components/Layout";
import { UserInfoContext } from "./components/UserInfoProvider";

const Login = React.lazy(() => import("./pages/Login"));
const Dashboard = React.lazy(() => import("./pages/Dashboard"));
const Settings = React.lazy(() => import("./pages/Settings"));
const LoginCallbackPage = React.lazy(() => import("./pages/LoginCallback"));
const NotFound = React.lazy(() => import("./pages/NotFound"));
const Posts = React.lazy(() => import("./pages/Posts"));

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const { isAuthenticated } = useContext(UserInfoContext);

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

const PublicRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useContext(UserInfoContext);

  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
};

function App() {
  useEffect(() => {
    // handle case when user visit http://localhost directly
    if (!window.location.pathname.startsWith("/ui")) {
      window.location.replace("/ui/dashboard");
    }
  }, []);

  return (
    <Router basename="/ui">
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
            <Route path="login/callback" element={<LoginCallbackPage />} />

            <Route path="/" element={<Layout />}>
              <Route index element={<Navigate to="dashboard" replace />} />
              <Route
                path="login"
                element={
                  <PublicRoute>
                    <Login />
                  </PublicRoute>
                }
              />
              <Route
                path="dashboard"
                element={
                  <ProtectedRoute>
                    <Dashboard />
                  </ProtectedRoute>
                }
              />
              <Route
                path="settings"
                element={
                  <ProtectedRoute>
                    <Settings />
                  </ProtectedRoute>
                }
              />
              <Route
                path="posts"
                element={
                  <ProtectedRoute>
                    <Posts />
                  </ProtectedRoute>
                }
              />
              <Route path="*" element={<NotFound />} />
            </Route>
            <Route path="*" element={<NotFound />} />
          </Routes>
        </React.Suspense>
      </div>
    </Router>
  );
}

export default App;
