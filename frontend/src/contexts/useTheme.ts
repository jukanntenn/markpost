import { useContext } from "react";
import { ThemeContext } from "./themeContext";
import type { ThemeContextType } from "./themeContext";

export const useTheme = (): ThemeContextType => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error("useTheme must be used within a ThemeProvider");
  }
  return context;
};

