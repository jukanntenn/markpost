"use client";

import { useTranslations } from "next-intl";

import { PageHeading } from "@/components/ui/page-heading";
import { AppSettingsCard } from "./AppSettingsCard";
import { PasswordChangeCard } from "./PasswordChangeCard";

export function SettingsPage() {
  const t = useTranslations("settings");

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <PageHeading>{t("title")}</PageHeading>

      <AppSettingsCard />

      <PasswordChangeCard />
    </div>
  );
}

export default SettingsPage;
