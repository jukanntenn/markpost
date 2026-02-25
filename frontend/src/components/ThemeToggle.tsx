import { useTheme } from "../contexts/useTheme";
import { MonitorIcon, MoonIcon, SunIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";

const ThemeToggle = () => {
  const { themeMode, setThemeMode } = useTheme();
  const { t } = useTranslation();

  const handleSelect = (mode: "light" | "dark" | "system") => {
    setThemeMode(mode);
  };

  const getThemeIcon = () => {
    switch (themeMode) {
      case "light":
        return <SunIcon className="size-4" />;
      case "dark":
        return <MoonIcon className="size-4" />;
      case "system":
      default:
        return <MonitorIcon className="size-4" />;
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={t("theme.toggleTheme")}
          title={t("theme.toggleTheme")}
        >
          {getThemeIcon()}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuRadioGroup
          value={themeMode}
          onValueChange={(v) => handleSelect(v as "light" | "dark" | "system")}
        >
          <DropdownMenuRadioItem value="light">
            <SunIcon className="size-4" />
            {t("theme.light")}
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="dark">
            <MoonIcon className="size-4" />
            {t("theme.dark")}
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="system">
            <MonitorIcon className="size-4" />
            {t("theme.system")}
          </DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

export default ThemeToggle;
