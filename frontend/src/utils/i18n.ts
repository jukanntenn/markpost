import i18n from "../i18n";

export const getCurrentLanguage = (): string => {
  const lang = i18n.resolvedLanguage || i18n.language || "en";
  if (lang.startsWith("zh")) return "zh";
  if (lang.startsWith("en")) return "en";
  return "en";
};

export const getAcceptLanguageHeader = (): string => {
  const base = getCurrentLanguage();

  const langMap: Record<string, string> = {
    en: "en-US,en;q=0.9",
    zh: "zh-CN,zh;q=0.9,en;q=0.8",
  };

  return langMap[base] || langMap.en;
};
