import React, { createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';

export type ThemeMode = 'light' | 'dark' | 'system';

interface ThemeContextType {
  themeMode: ThemeMode;
  currentTheme: 'light' | 'dark';
  setThemeMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

interface ThemeProviderProps {
  children: ReactNode;
}

const getSystemTheme = (): 'light' | 'dark' => {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return 'light';
};

const getEffectiveTheme = (mode: ThemeMode): 'light' | 'dark' => {
  return mode === 'system' ? getSystemTheme() : mode;
};

export const ThemeProvider: React.FC<ThemeProviderProps> = ({ children }) => {
  const [themeMode, setThemeModeState] = useState<ThemeMode>(() => {
    // Get saved theme mode from localStorage
    const saved = localStorage.getItem('theme-mode') as ThemeMode;
    return saved || 'system';
  });

  const [currentTheme, setCurrentTheme] = useState<'light' | 'dark'>(() => {
    return getEffectiveTheme(themeMode);
  });

  // Update effective theme when themeMode or system preference changes
  useEffect(() => {
    const effectiveTheme = getEffectiveTheme(themeMode);
    setCurrentTheme(effectiveTheme);

    // Apply to document
    document.documentElement.setAttribute('data-bs-theme', effectiveTheme);

    // Apply body classes for Bootstrap compatibility
    document.body.classList.remove('bg-light', 'bg-dark', 'text-light', 'text-dark');
    if (effectiveTheme === 'dark') {
      document.body.classList.add('bg-dark', 'text-light');
    } else {
      document.body.classList.add('bg-light', 'text-dark');
    }
  }, [themeMode]);

  // Listen for system theme changes
  useEffect(() => {
    if (themeMode === 'system') {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

      const handleChange = () => {
        const newTheme = mediaQuery.matches ? 'dark' : 'light';
        setCurrentTheme(newTheme);
        document.documentElement.setAttribute('data-bs-theme', newTheme);
        document.body.classList.remove('bg-light', 'bg-dark', 'text-light', 'text-dark');
        if (newTheme === 'dark') {
          document.body.classList.add('bg-dark', 'text-light');
        } else {
          document.body.classList.add('bg-light', 'text-dark');
        }
      };

      mediaQuery.addEventListener('change', handleChange);
      return () => mediaQuery.removeEventListener('change', handleChange);
    }
  }, [themeMode]);

  const setThemeMode = (mode: ThemeMode) => {
    setThemeModeState(mode);
    localStorage.setItem('theme-mode', mode);
  };

  const toggleTheme = () => {
    // Cycle through modes: system -> light -> dark -> system
    const modes: ThemeMode[] = ['system', 'light', 'dark'];
    const currentIndex = modes.indexOf(themeMode);
    const nextMode = modes[(currentIndex + 1) % modes.length];
    setThemeMode(nextMode);
  };

  return (
    <ThemeContext.Provider value={{ themeMode, currentTheme, setThemeMode, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useTheme = (): ThemeContextType => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};