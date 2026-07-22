"use client";

import { useTranslations } from "next-intl";
import { useLocaleContext } from "@/components/providers/LocaleProvider";
import { localeNames } from "@/i18n/constants";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";

export function AppSettingsCard() {
  const t = useTranslations("settings");
  const { locale, setLocale, availableLocales } = useLocaleContext();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("applicationSettings")}</CardTitle>
        <CardDescription>{t("languageDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-4">
          <Label htmlFor="locale-select">{t("language")}</Label>
          <select
            id="locale-select"
            value={locale}
            onChange={(e) => setLocale(e.target.value as typeof locale)}
            className="h-10 rounded-md border border-input bg-card px-3 py-2 text-sm text-foreground transition-[box-shadow] focus-visible:border-primary focus-visible:ring-ring focus-visible:ring-[3px] focus-visible:outline-none"
          >
            {availableLocales.map((l) => (
              <option key={l} value={l}>
                {localeNames[l]}
              </option>
            ))}
          </select>
        </div>
      </CardContent>
    </Card>
  );
}
