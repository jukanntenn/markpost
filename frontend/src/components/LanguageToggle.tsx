import React from "react";
import { Dropdown } from "react-bootstrap";
import { useTranslation } from "react-i18next";
import { Globe } from "react-bootstrap-icons";

const LanguageToggle: React.FC = () => {
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
    <Dropdown align="end">
      <Dropdown.Toggle
        variant="link"
        className="text-decoration-none p-2 d-flex align-items-center gap-2 text-body"
        id="language-dropdown"
        title={t("language.changeLanguage")}
        aria-label={t("language.changeLanguage")}
      >
        <Globe size={18} />
        <span>{getCurrentLanguageLabel()}</span>
      </Dropdown.Toggle>

      <Dropdown.Menu className="border-0 shadow-lg">
        <Dropdown.Item
          active={isEnglish}
          onClick={() => handleLanguageChange("en")}
          className="d-flex align-items-center gap-2"
        >
          <span className="fi fi-gb"></span>
          {t("language.english")}
        </Dropdown.Item>
        <Dropdown.Item
          active={isChinese}
          onClick={() => handleLanguageChange("zh")}
          className="d-flex align-items-center gap-2"
        >
          <span className="fi fi-cn"></span>
          {t("language.chinese")}
        </Dropdown.Item>
      </Dropdown.Menu>
    </Dropdown>
  );
};

export default LanguageToggle;
