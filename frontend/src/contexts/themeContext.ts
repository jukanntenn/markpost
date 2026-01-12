import { createContext } from "react";

export type ThemeMode = "light" | "dark" | "system";

export interface ThemeContextType {
  themeMode: ThemeMode;
  currentTheme: "light" | "dark";
  setThemeMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;
}

export const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

