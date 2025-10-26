import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App.tsx";
import { ToastsProvider as BootstrapToastsProvider } from "react-bootstrap-toasts";
import { UserInfoProvider } from "./components/UserInfoProvider";
import { ThemeProvider } from "./contexts/ThemeContext";
import "./i18n";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <UserInfoProvider>
      <ThemeProvider>
        <BootstrapToastsProvider
          toastContainerProps={{ position: "top-end", className: "p-4" }}
          limit={5}
        >
          <App />
        </BootstrapToastsProvider>
      </ThemeProvider>
    </UserInfoProvider>
  </StrictMode>
);
