"use client";

import { useTranslations } from "next-intl";
import { useTheme } from "next-themes";
import { MonitorIcon, MoonIcon, SunIcon } from "lucide-react";
import { Menu } from "@/components/ui/menu";
import { Button } from "@/components/ui/button";

export function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  const t = useTranslations("theme");

  const handleSelect = (mode: string) => {
    setTheme(mode as "light" | "dark" | "system");
  };

  const getThemeIcon = () => {
    switch (theme) {
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
    <Menu.Root>
      <Menu.Trigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={t("toggleTheme")}
            title={t("toggleTheme")}
          />
        }
      >
        {getThemeIcon()}
      </Menu.Trigger>
      <Menu.Popup>
        <Menu.RadioGroup value={theme} onValueChange={handleSelect}>
          <Menu.RadioItem value="light">
            <SunIcon className="size-4" />
            {t("light")}
          </Menu.RadioItem>
          <Menu.RadioItem value="dark">
            <MoonIcon className="size-4" />
            {t("dark")}
          </Menu.RadioItem>
          <Menu.RadioItem value="system">
            <MonitorIcon className="size-4" />
            {t("system")}
          </Menu.RadioItem>
        </Menu.RadioGroup>
      </Menu.Popup>
    </Menu.Root>
  );
}

export default ThemeToggle;
