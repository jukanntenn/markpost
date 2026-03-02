import React, { useContext, useEffect } from "react";
import {
  BrowserRouter as Router,
  Routes,
  Route,
  Navigate,
} from "react-router-dom";
import { SWRConfig } from "swr";
import Layout from "./components/Layout";
import { UserInfoContext } from "./components/UserInfoContext";
import { swrConfig } from "./swr/config";
import { Loader2Icon } from "lucide-react";

const Login = React.lazy(() => import("./pages/Login"));
const Dashboard = React.lazy(() => import("./pages/Dashboard"));
const Settings = React.lazy(() => import("./pages/Settings"));
const LoginCallbackPage = React.lazy(() => import("./pages/LoginCallback"));
const NotFound = React.lazy(() => import("./pages/NotFound"));
const Posts = React.lazy(() => import("./pages/Posts"));
const AdminDashboard = React.lazy(() => import("./pages/AdminDashboard"));
const AdminUsers = React.lazy(() => import("./pages/AdminUsers"));
const AdminPosts = React.lazy(() => import("./pages/AdminPosts"));
const AdminChannels = React.lazy(() => import("./pages/AdminChannels"));

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

const AdminRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated, isAdmin } = useContext(UserInfoContext);

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (!isAdmin()) {
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
      <SWRConfig value={swrConfig}>
        <div className="App">
          <React.Suspense
            fallback={
              <div className="flex min-h-svh items-center justify-center">
                <Loader2Icon className="size-6 animate-spin" />
                <span className="sr-only">Loading...</span>
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
                <Route
                  path="admin"
                  element={
                    <AdminRoute>
                      <AdminDashboard />
                    </AdminRoute>
                  }
                >
                  <Route index element={<Navigate to="users" replace />} />
                  <Route path="users" element={<AdminUsers />} />
                  <Route path="posts" element={<AdminPosts />} />
                  <Route path="channels" element={<AdminChannels />} />
                </Route>
                <Route path="*" element={<NotFound />} />
              </Route>
              <Route path="*" element={<NotFound />} />
            </Routes>
          </React.Suspense>
        </div>
      </SWRConfig>
    </Router>
  );
}

export default App;
