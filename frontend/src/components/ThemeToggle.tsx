import React from "react";
import { useTheme } from "../contexts/ThemeContext";
import { Dropdown } from "react-bootstrap";
import { Sun, Moon, CircleHalf } from "react-bootstrap-icons";
import { useTranslation } from "react-i18next";

const ThemeToggle: React.FC = () => {
  const { themeMode, setThemeMode } = useTheme();
  const { t } = useTranslation();

  const handleSelect = (mode: "light" | "dark" | "system") => {
    setThemeMode(mode);
  };

  const getThemeIcon = () => {
    switch (themeMode) {
      case "light":
        return <Sun size={18} />;
      case "dark":
        return <Moon size={18} />;
      case "system":
      default:
        return <CircleHalf size={18} />;
    }
  };

  return (
    <Dropdown align="end">
      <Dropdown.Toggle
        variant="link"
        className="text-decoration-none p-2 d-flex align-items-center text-body"
        id="theme-dropdown"
        title={t("theme.toggleTheme")}
        aria-label={t("theme.toggleTheme")}
      >
        {getThemeIcon()}
      </Dropdown.Toggle>

      <Dropdown.Menu className="border-0 shadow-lg">
        <Dropdown.Item
          active={themeMode === "light"}
          onClick={() => handleSelect("light")}
          className="d-flex align-items-center gap-2"
        >
          <Sun size={16} />
          {t("theme.light")}
        </Dropdown.Item>
        <Dropdown.Item
          active={themeMode === "dark"}
          onClick={() => handleSelect("dark")}
          className="d-flex align-items-center gap-2"
        >
          <Moon size={16} />
          {t("theme.dark")}
        </Dropdown.Item>
        <Dropdown.Item
          active={themeMode === "system"}
          onClick={() => handleSelect("system")}
          className="d-flex align-items-center gap-2"
        >
          <CircleHalf size={16} />
          {t("theme.system")}
        </Dropdown.Item>
      </Dropdown.Menu>
    </Dropdown>
  );
};

export default ThemeToggle;
