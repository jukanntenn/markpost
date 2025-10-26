import i18n from "../i18n";

export const getCurrentLanguage = (): string => {
  return i18n.language || "en";
};

export const getAcceptLanguageHeader = (): string => {
  const lang = getCurrentLanguage();

  const langMap: Record<string, string> = {
    en: "en-US,en;q=0.9",
    zh: "zh-CN,zh;q=0.9,en;q=0.8",
  };

  return langMap[lang] || langMap.en;
};
