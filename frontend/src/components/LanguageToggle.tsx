import { useTranslation } from "react-i18next";
import { LanguagesIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const LanguageToggle = () => {
  const { i18n, t } = useTranslation();

  const resolved = i18n.resolvedLanguage || i18n.language || "en";
  const isEnglish = resolved.startsWith("en");
  const isChinese = resolved.startsWith("zh");

  const handleLanguageChange = (lng: string) => {
    i18n.changeLanguage(lng);
  };

  const getCurrentLanguageLabel = () => {
    if (isEnglish) return t("language.english");
    if (isChinese) return t("language.chinese");
    return t("language.english");
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          className="gap-2"
          aria-label={t("language.changeLanguage")}
          title={t("language.changeLanguage")}
        >
          <LanguagesIcon className="size-4" />
          <span>{getCurrentLanguageLabel()}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuRadioGroup
          value={isChinese ? "zh" : "en"}
          onValueChange={(v) => handleLanguageChange(v)}
        >
          <DropdownMenuRadioItem value="en">
            {t("language.english")}
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="zh">
            {t("language.chinese")}
          </DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

export default LanguageToggle;
