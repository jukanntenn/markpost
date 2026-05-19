"use client";

import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import type { FormState } from "@/hooks/useChannelForm";

interface DeliveryChannelFormProps {
  form: FormState;
  onFormChange: (form: FormState) => void;
  onSubmit: (e: React.FormEvent) => void;
  onCancel: () => void;
  isSubmitting: boolean;
  isEditing: boolean;
}

export function DeliveryChannelForm({
  form,
  onFormChange,
  onSubmit,
  onCancel,
  isSubmitting,
  isEditing,
}: DeliveryChannelFormProps) {
  const t = useTranslations("settings");

  return (
    <>
      <Separator />
      <form onSubmit={onSubmit} className="space-y-3">
        <div className="space-y-2">
          <Label htmlFor="channel-name">{t("deliveryChannelName")}</Label>
          <Input
            id="channel-name"
            value={form.name}
            onChange={(e) => onFormChange({ ...form, name: e.target.value })}
            placeholder={t("deliveryChannelNamePlaceholder")}
            required
            disabled={isSubmitting}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="channel-webhook">{t("deliveryChannelWebhookURL")}</Label>
          <Input
            id="channel-webhook"
            value={form.webhookUrl}
            onChange={(e) => onFormChange({ ...form, webhookUrl: e.target.value })}
            placeholder={t("deliveryChannelWebhookPlaceholder")}
            required
            disabled={isSubmitting}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="channel-keywords">{t("deliveryChannelKeywords")}</Label>
          <Input
            id="channel-keywords"
            value={form.keywords}
            onChange={(e) => onFormChange({ ...form, keywords: e.target.value })}
            placeholder={t("deliveryChannelKeywordsPlaceholder")}
            disabled={isSubmitting}
          />
        </div>

        <div className="flex gap-2">
          <LoadingButton type="submit" loading={isSubmitting} loadingText={t("deliveryChannelSaving")}>
            {isEditing ? t("deliveryChannelSave") : t("deliveryChannelCreate")}
          </LoadingButton>
          <Button
            type="button"
            variant="outline"
            onClick={onCancel}
            disabled={isSubmitting}
          >
            {t("deliveryChannelCancel")}
          </Button>
        </div>
      </form>
    </>
  );
}
