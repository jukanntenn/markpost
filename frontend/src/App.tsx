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
import { ThemeProvider } from "./contexts/ThemeContext";
// Lazy load components
const Login = React.lazy(() => import("./pages/Login"));
const Dashboard = React.lazy(() => import("./pages/Dashboard"));
const Settings = React.lazy(() => import("./pages/Settings"));
const LoginCallbackPage = React.lazy(() => import("./pages/LoginCallback"));

// Protected Route component - 需要用户已登录才能访问
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const { isAuthenticated } = useContext(UserInfoContext);

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

// Public Route component - 如果用户已登录则重定向到 dashboard
const PublicRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useContext(UserInfoContext);

  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
};

function App() {
  // 检测当前路径，如果不在 /ui 下，重定向到 /ui/dashboard
  useEffect(() => {
    if (!window.location.pathname.startsWith("/ui")) {
      window.location.replace("/ui/dashboard");
    }
  }, []);

  return (
    <ThemeProvider>
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
              {/* 登录回调路由 - 不需要保护和布局 */}
              <Route path="login/callback" element={<LoginCallbackPage />} />

              <Route path="/" element={<Layout />}>
                {/* 公共路由 - 已登录用户访问时会重定向到 dashboard */}
                <Route
                  path="login"
                  element={
                    <PublicRoute>
                      <Login />
                    </PublicRoute>
                  }
                />

                {/* 受保护的路由 - 需要用户已登录才能访问 */}
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

                {/* 嵌套路由的默认重定向 */}
                <Route index element={<Navigate to="dashboard" replace />} />
              </Route>
            </Routes>
          </React.Suspense>
        </div>
      </Router>
    </ThemeProvider>
  );
}

export default App;
