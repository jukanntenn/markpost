import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App.tsx";
import { UserInfoProvider } from "./components/UserInfoProvider";
import { ThemeProvider } from "./contexts/ThemeProvider";
import "./index.css";
import "./i18n";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <UserInfoProvider>
      <ThemeProvider>
        <TooltipProvider delayDuration={200}>
          <App />
          <Toaster position="top-right" style={{ zIndex: 2000 }} />
        </TooltipProvider>
      </ThemeProvider>
    </UserInfoProvider>
  </StrictMode>
);
