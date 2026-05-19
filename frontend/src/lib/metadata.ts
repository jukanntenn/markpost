import { Metadata } from "next";
import { getTranslations } from "next-intl/server";

export function buildPageMetadata(key: string) {
  return async (): Promise<Metadata> => {
    const t = await getTranslations("common.pageTitle");
    return { title: t(key) };
  };
}
